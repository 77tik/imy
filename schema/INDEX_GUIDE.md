# 聊天系统数据库索引设计指南

## 核心设计理念

按你的**"常用查询路径"来设计索引**，让一条查询"只走一个合适的索引、少回表、顺序读、多命中"，同时控制索引数量，别给高写入表乱加。

## 通用索引设计套路（8条黄金法则）

### 1. 一条核心查询 ≈ 一个核心索引
先写出最常见的 `where + order by + limit` 语句，再反推该查询需要的联合索引。

**示例：**
```sql
-- 查询语句
where conv_id=? and status=1 order by seq desc limit 50

-- 对应索引
create index idx_conv_status on message (conv_id, status, seq desc);
```

### 2. 索引列顺序遵循：等值列在前 → 范围/排序列在后
- **等值条件**的列放在最前面
- **范围查询**和**排序**的列放在后面
- 高选择度的列优先

### 3. 左前缀法则
MySQL BTree 只用得上索引的**最左前缀**：
- `(a,b,c)` 能服务 `a=?`、`a=? and b=?`、`a between ...`
- 但对 `b=?` 单独查询没用

### 4. 尽量"顺排 + 覆盖"，少回表
- 能顺着一个索引扫完就好（范围扫描）
- 尽量让索引同时满足 `where` 和 `order by`
- **覆盖索引**（索引里包含查询所需列）可以避免回表
- **不要把大字段**（text/json）塞进索引

### 5. 高写入表，索引要克制
每多一个二级索引，写放大+页分裂+锁竞争都会增加。像 `message` 这种高写表，**能用主键就别上多余索引**。

### 6. 低选择度的列不要单列索引
布尔/小枚举（如 `status`、`is_deleted`）单独建索引基本没用；**要和高选择度列组合**（如 `(conv_id, status)`）。

### 7. 等值放前，范围放后，排序看情况
```sql
-- 示例1
where a=? and b between ... order by b → (a,b)

-- 示例2  
where a=? order by created_at desc → (a, created_at desc) -- mysql8 支持 desc 索引
```

如果排序是 `seq` 而 `seq` 与时间强相关，就不必另做 `(a, created_at)`。

### 8. 检查冗余与覆盖
- InnoDB 的二级索引**自动包含主键列**，别重复把主键又加一遍
- 避免"等价索引"重复（比如已有 `(conv_id, seq)` 的主键，再建 `(conv_id)` 意义很小）

## 聊天系统表索引设计方案

### 1. conversation（会话表）

#### 现有索引
- `PRIMARY KEY (id)`
- `UNIQUE KEY (pair_key)`
- `KEY idx_deleted_at (deleted_at)`

#### 常用查询分析
```sql
-- 单聊 get-or-create
select id from conversation where pair_key = ? and deleted_at = 0;
-- → unique(pair_key) 已完美命中

-- 会话详情
select * from conversation where id = ?;
-- → PRIMARY KEY 命中
```

#### 索引建议
✅ **保留：**
- `PRIMARY KEY (id)` - 必须
- `UNIQUE KEY (pair_key)` - 防止单聊重复创建，核心索引

❓ **可选：**
- `(type, created_at)` - 仅管理后台按类型/时间筛选时考虑

❌ **可以移除：**
- `idx_deleted_at` - 布尔选择度低，除非确实经常扫大量已删除行

### 2. conversation_member（会话成员表）

#### 现有索引
- `PRIMARY KEY (conv_id, user_id)`
- `KEY idx_user_conv (user_id, conv_id)`

#### 常用查询分析
```sql
-- 某会话的成员列表
select * from conversation_member where conv_id = ? and deleted_at = 0;
-- → PRIMARY KEY 顺扫即可

-- 我参与的会话列表
select * from conversation_member where user_id = ? and deleted_at = 0;
-- → (user_id, conv_id) 命中
```

#### 索引建议
✅ **保留：**
- `PRIMARY KEY (conv_id, user_id)` - 必须
- `KEY idx_user_conv (user_id, conv_id)` - 查询用户会话列表必需

❌ **不需要：**
- `last_read_seq` 相关索引 - 上报已读是更新操作，不是查询；计算未读走 join conversation_counter

### 3. message（消息表 - 高写入表，索引克制）

#### 现有索引
- `PRIMARY KEY (conv_id, seq)` - 热路径，必须
- `UNIQUE KEY (msg_id)` - 幂等写，强烈建议
- `KEY idx_conv_time (conv_id, created_at, seq)` - 按时间查询
- `KEY idx_send_time (send_id, created_at)` - 按发送者查询
- `KEY idx_conv_status (conv_id, status, seq)` - 只看正常消息并分页

#### 常用查询分析
```sql
-- 会话消息分页（最核心查询）
select * from message 
where conv_id = ? and deleted_at = 0 
order by seq desc limit 50;
-- → PRIMARY KEY 足够，最快

-- 只看正常消息分页
select * from message 
where conv_id = ? and status = 1 and deleted_at = 0 
order by seq desc limit 50;
-- → (conv_id, status, seq) 很有用

-- 按时间范围拉取
select * from message 
where conv_id = ? and created_at between ? and ? 
order by created_at desc;
-- → idx_conv_time 才需要

-- 按发送者查最近消息
select * from message 
where send_id = ? and deleted_at = 0 
order by created_at desc limit 20;
-- → idx_send_time 合理
```

#### 索引建议
✅ **必须保留：**
- `PRIMARY KEY (conv_id, seq)` - 核心热路径
- `UNIQUE KEY (msg_id)` - 防重复，幂等性保证
- `KEY idx_conv_status (conv_id, status, seq desc)` - 过滤正常消息+分页
- `KEY idx_send_time (send_id, created_at desc)` - 按发送者查询

❓ **视情况保留：**
- `KEY idx_conv_time (conv_id, created_at, seq)` - 如果很少按时间范围查询，可以移除

❌ **绝对不要：**
- 不要为了"覆盖"而把 `content` 放进索引 - 大字段放索引是灾难

### 4. conversation_counter（会话计数器表）

#### 现有索引
- `PRIMARY KEY (conv_id)`
- `KEY idx_deleted_at (deleted_at)`

#### 使用场景
只用于发号与取 `latest_seq`，极少做复杂查询。

#### 索引建议
✅ **保留：**
- `PRIMARY KEY (conv_id)` - 足够

❌ **可以移除：**
- `idx_deleted_at` - 通常不需要"软删"计数器

## 索引列顺序决策法（三步法）

以 `where conv_id=? and status=1 order by seq desc limit 50` 为例：

### 第一步：等值列
- `conv_id` 放第一位（选择度高）
- `status` 如常用就放第二位

### 第二步：范围/排序列
- `seq` 放后面并配合 `desc`（MySQL8 支持 desc 索引）

### 第三步：必要字段
- 如果查询里只需要 `(conv_id, seq, status)` 这些"小字段"，可以让它成为覆盖索引
- 但有 `content` 就别想覆盖了

**最终索引：**
```sql
create index idx_conv_status on message (conv_id, status, seq desc);
```

## 什么时候不要加索引？

❌ **以下情况不要加索引：**

1. **几乎不用来过滤/排序/连接的列**（比如 `remark`）
2. **低选择度列的单列索引**（比如 `status`、`deleted_at`），除非和高选择度列组合
3. **频繁更新的列**（每次写入都会维护索引，写压会飙升），如无必要别加
4. **覆盖不了的"大字段"**（text/json）别放索引列里
5. **和已有索引完全重复/等价的索引**（浪费空间、拖慢写）

## 验证与调参方法

### 1. 性能分析
```sql
-- 查看执行计划
explain analyze <sql>;

-- 关注指标：
-- - 是否命中预期索引
-- - 行扫描量 (rows examined)
-- - 是否出现 filesort
-- - 是否使用了 using index (覆盖索引)
```

### 2. 安全试验
```sql
-- 不可见索引安全试验
alter table message alter index idx_conv_time invisible;
-- 观察一段时间没异常，再考虑 drop
alter table message drop index idx_conv_time;
```

### 3. 临时诊断
```sql
-- 强制使用指定索引
select ... from message force index (idx_conv_status) where ...;
```

### 4. 监控指标
- 关注写 QPS 高的表（message）的 `rows_written/second`
- 监控 `buffer pool` 命中率
- 索引越多，写放大越明显

## 典型查询索引决策表

| 典型查询 | 建议索引 | 备注 |
|---------|---------|------|
| 会话内最新/更多消息 | `PRIMARY KEY (conv_id, seq)` | 主路径；无需别的 |
| 只看正常消息分页 | `(conv_id, status, seq desc)` | 过滤 + 排序一次搞定 |
| 按发送者看 ta 的消息 | `(send_id, created_at desc)` | 后台/审计常用 |
| 按时间范围回放 | `(conv_id, created_at, seq)` | 若少用可不建 |
| get-or-create 单聊 | `UNIQUE(pair_key)` | 从根上消除并发竞态 |
| 我在的会话列表 | `(user_id, conv_id)` 在 conversation_member | 常用 |
| 会话成员列表 | `PRIMARY KEY (conv_id, user_id)` | 顺扫即可 |

## 推荐索引配置

### conversation 表
```sql
-- 保留
PRIMARY KEY (id),
UNIQUE KEY uk_pair_key (pair_key),

-- 可选（管理后台需要时）
KEY idx_type_time (type, created_at);
```

### conversation_member 表
```sql
-- 保留
PRIMARY KEY (conv_id, user_id),
KEY idx_user_conv (user_id, conv_id);
```

### message 表（核心，谨慎添加）
```sql
-- 必须
PRIMARY KEY (conv_id, seq),
UNIQUE KEY uk_msg_id (msg_id),

-- 强烈推荐
KEY idx_conv_status (conv_id, status, seq DESC),
KEY idx_send_time (send_id, created_at DESC),

-- 可选（按需）
KEY idx_conv_time (conv_id, created_at, seq);
```

### conversation_counter 表
```sql
-- 仅保留
PRIMARY KEY (conv_id);
```

## 总结

**一句话：** 索引不是"多多益善"，而是**用最少的索引覆盖最核心的读路径**。你的消息表写多读多，**主键 `(conv_id, seq)` + 少量必要二级索引**就够了；其它索引在 `explain analyze` 下真有价值再加。

**核心原则：**
- 高写入表索引要克制
- 一条核心查询对应一个核心索引
- 等值在前，范围在后
- 覆盖索引优于回表，但别放大字段
- 用数据说话，explain analyze 验证效果