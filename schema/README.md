# 聊天系统数据库设计文档

## 表结构说明

### 1. conversation - 聊天会话表
- **id**: 会话ID，雪花ID
- **type**: 会话类型，1-单聊，2-群聊
- **remark**: 备注信息
- **pair_key**: 单聊配对唯一键，群聊为null，用于解决并发竞争

### 2. conversation_member - 会话成员表
- **conv_id**: 会话ID
- **user_id**: 用户ID
- **role**: 角色，1-成员，2-管理员
- **last_read_seq**: 最后读到的会话内序号

### 3. message - 聊天消息表
- **conv_id**: 会话ID
- **seq**: 会话内序号，严格递增
- **msg_id**: 全局消息ID，雪花ID
- **send_id**: 发送者ID
- **content**: 消息内容
- **content_type**: 内容类型，1-文本，2-图片，3-文件
- **status**: 消息状态，1-正常，2-撤回
- **recall_id**: 撤回操作者ID
- **recall_at**: 撤回时间

### 4. conversation_counter - 会话内序号表
- **conv_id**: 会话ID
- **latest_seq**: 当前会话最大序号

## 业务流程设计

### 1. 单聊会话建立

#### 流程说明
1. 检查两个用户是否已存在单聊会话
2. 如果不存在，创建新的单聊会话
3. 添加两个用户到会话成员表
4. 初始化会话计数器

#### SQL操作
```sql
-- 1. 生成单聊配对键（用户ID小的在前）
set @pair_key = concat(least(?, ?), '_', greatest(?, ?));

-- 2. 检查是否已存在单聊
select id from conversation 
where type = 1 and pair_key = @pair_key and deleted_at = 0;

-- 3. 如果不存在，创建单聊会话（利用unique约束防止并发重复创建）
start transaction;

insert into conversation (id, type, pair_key, remark) 
values (?, 1, @pair_key, '');

-- 4. 添加会话成员
insert into conversation_member (conv_id, user_id, role) 
values 
(?, ?, 1),  -- 用户1
(?, ?, 1);  -- 用户2

-- 5. 初始化会话计数器
insert into conversation_counter (conv_id, latest_seq) 
values (?, 0);

commit;
```

### 2. 群聊会话建立

#### 流程说明
1. 创建群聊会话（pair_key为null）
2. 添加创建者为管理员
3. 添加其他成员
4. 初始化会话计数器

#### SQL操作
```sql
start transaction;

-- 1. 创建群聊会话
insert into conversation (id, type, remark) 
values (?, 2, ?);

-- 2. 添加创建者为管理员
insert into conversation_member (conv_id, user_id, role) 
values (?, ?, 2);

-- 3. 批量添加其他成员
insert into conversation_member (conv_id, user_id, role) 
values 
(?, ?, 1),
(?, ?, 1);

-- 4. 初始化会话计数器
insert into conversation_counter (conv_id, latest_seq) 
values (?, 0);

commit;
```

### 3. 发送消息

#### 流程说明
1. 验证用户是否为会话成员
2. 获取下一个序号
3. 插入消息记录
4. 更新会话计数器

#### SQL操作
```sql
start transaction;

-- 1. 验证发送者权限
select 1 from conversation_member 
where conv_id = ? and user_id = ? and deleted_at = 0;

-- 2. 获取并更新序号（原子操作，防止并发）
insert into conversation_counter (conv_id, latest_seq)
values (?, 1)
on duplicate key update latest_seq = last_insert_id(latest_seq + 1);

set @seq := if(row_count() = 1, 1, last_insert_id());

-- 3. 插入消息
insert into message (conv_id, seq, msg_id, send_id, content, content_type, status)
values (?, @seq, ?, ?, ?, ?, 1);

commit;
```

### 4. 消息已读上报

#### 流程说明
1. 验证用户是否为会话成员
2. 更新用户的最后已读序号
3. 只能向前更新，不能回退

#### SQL操作
```sql
-- 更新最后已读序号（只能向前，不能回退）
update conversation_member 
set last_read_seq = greatest(last_read_seq, ?), 
    updated_at = now()
where conv_id = ? and user_id = ? and deleted_at = 0;
```

### 5. 消息撤回

#### 流程说明
1. 验证撤回权限（发送者本人或管理员）
2. 检查撤回时间限制（可选）
3. 更新消息状态为撤回

#### SQL操作
```sql
start transaction;

-- 1. 验证撤回权限
select m.send_id, cm.role 
from message m
left join conversation_member cm on cm.conv_id = m.conv_id and cm.user_id = ?
where m.conv_id = ? and m.seq = ? and m.deleted_at = 0;

-- 2. 执行撤回（只有发送者本人或管理员可以撤回）
update message 
set status = 2, 
    recall_id = ?, 
    recall_at = now(), 
    updated_at = now()
where conv_id = ? and seq = ? 
  and (send_id = ? or exists(
    select 1 from conversation_member 
    where conv_id = ? and user_id = ? and role = 2 and deleted_at = 0
  ))
  and status = 1 and deleted_at = 0;

commit;
```

### 6. 群聊成员变更

#### 6.1 添加成员
```sql
start transaction;

-- 1. 验证操作者权限（管理员才能添加成员）
select 1 from conversation_member 
where conv_id = ? and user_id = ? and role = 2 and deleted_at = 0;

-- 2. 添加新成员
insert into conversation_member (conv_id, user_id, role) 
values (?, ?, 1)
on duplicate key update deleted_at = 0, updated_at = now();

commit;
```

#### 6.2 移除成员
```sql
start transaction;

-- 1. 验证操作者权限
select 1 from conversation_member 
where conv_id = ? and user_id = ? and role = 2 and deleted_at = 0;

-- 2. 移除成员（软删除）
update conversation_member 
set deleted_at = 1, updated_at = now()
where conv_id = ? and user_id = ? and deleted_at = 0;

commit;
```

#### 6.3 转让管理员
```sql
start transaction;

-- 1. 验证当前用户是管理员
select 1 from conversation_member 
where conv_id = ? and user_id = ? and role = 2 and deleted_at = 0;

-- 2. 将当前管理员降为普通成员
update conversation_member 
set role = 1, updated_at = now()
where conv_id = ? and user_id = ? and deleted_at = 0;

-- 3. 将目标用户提升为管理员
update conversation_member 
set role = 2, updated_at = now()
where conv_id = ? and user_id = ? and deleted_at = 0;

commit;
```

### 7. 查询用户会话列表

```sql
-- 查询用户参与的所有会话
select c.id, c.type, c.remark, 
       cm.last_read_seq,
       cc.latest_seq,
       (cc.latest_seq - cm.last_read_seq) as unread_count
from conversation c
join conversation_member cm on cm.conv_id = c.id
left join conversation_counter cc on cc.conv_id = c.id
where cm.user_id = ? and cm.deleted_at = 0 and c.deleted_at = 0
order by c.updated_at desc;
```

### 8. 查询会话消息历史

```sql
-- 分页查询会话消息（按seq倒序）
select conv_id, seq, msg_id, send_id, content, content_type, 
       status, recall_id, recall_at, created_at
from message 
where conv_id = ? and deleted_at = 0
order by seq desc
limit ? offset ?;
```

## 并发与一致性保证

### 1. 单聊会话创建的并发控制

**问题**: 多个用户同时创建单聊会话可能导致重复创建

**解决方案**: 使用`pair_key`唯一约束

| 时间      | 连接A  | 连接B  | 结果 |
| ------- | ------ | ------ | ---- |
| t0      | `begin;` | | 开启事务A |
| t0+1ms  | `select id from conversation where pair_key = '7_9';` | | 返回空 |
| t0+2ms  | | `begin;` | 开启事务B |
| t0+3ms  | | `select id from conversation where pair_key = '7_9';` | 也返回空 |
| t0+5ms  | `insert into conversation(...) values(..., '7_9', ...);` | | 成功插入 |
| t0+7ms  | `commit;` | | A提交生效 |
| t0+8ms  | | `insert into conversation(...) values(..., '7_9', ...);` | **唯一约束冲突，插入失败** |
| t0+10ms | | `rollback;` | B回滚 |

### 2. 消息序号的并发控制

**问题**: 同一会话的并发消息发送需要保证序号严格递增且无重复

**解决方案**: 使用`conversation_counter`表的原子更新操作

```sql
-- 原子操作：获取下一个序号
insert into conversation_counter (conv_id, latest_seq)
values (?, 1)
on duplicate key update latest_seq = last_insert_id(latest_seq + 1);

set @seq := if(row_count() = 1, 1, last_insert_id());
```

**安全性保证**:
- 同一`conv_id`的并发操作会在`conversation_counter`表的同一行上排队（行级锁）
- `last_insert_id(latest_seq + 1)`确保每次获取的序号唯一且递增
- 整个过程在事务中执行，失败时序号更新会回滚，避免序号空洞

### 3. 已读状态的一致性

**设计原则**:
- 已读序号只能向前推进，不能回退
- 使用`greatest(last_read_seq, ?)`确保单调性
- 客户端重复上报不会影响数据一致性

### 4. 软删除的一致性

**设计原则**:
- 所有表都使用`deleted_at`字段进行软删除
- 查询时必须添加`deleted_at = 0`条件
- 删除操作设置`deleted_at = 1`而不是物理删除
- 支持数据恢复和审计追踪

## 原有并发竞争说明

### conversation会话表并发控制
主要是防止多用户并发创建单聊会话时，创建了多个的情况。初始状态：用户7和用户9没有建立单聊会话

| 时间      | 连接/服务  | 操作                                                                               | 结果/原因                               |
| ------- | ------ | -------------------------------------------------------------------------------- | ----------------------------------- |
| t0      | Conn A | `BEGIN;`                                                                         | 开启事务                                |
| t0+1ms  | Conn A | `SELECT id FROM conversation WHERE type=1 AND 包含成员(7,9);`                        | **返回空**（当前还没有）                      |
| t0+2ms  | Conn B | `BEGIN;`                                                                         | 开启事务                                |
| t0+3ms  | Conn B | `SELECT id FROM conversation WHERE type=1 AND 包含成员(7,9);`                        | **也返回空**（A 还没插入/提交）                 |
| t0+5ms  | Conn A | 生成 `conv_id=1001`，执行 `INSERT INTO conversation(id,type,...) VALUES (1001,1,...)` | 成功（没有约束拦它）                          |
| t0+7ms  | Conn A | `COMMIT;`                                                                        | A 提交生效                              |
| t0+8ms  | Conn B | 生成 `conv_id=1002`，执行 `INSERT INTO conversation(id,type,...) VALUES (1002,1,...)` | **仍成功**（因为没有唯一约束能阻止"另一条相同关系"的会话被插入） |
| t0+10ms | Conn B | `COMMIT;`                                                                        | B 提交生效                              |
| 结束      |        |                                                                                  | **结果：同一对(7,9) 产生了两个会话 1001、1002**   |

**解决方案**:
- 如果不设置唯一键约束，那么就真的可以创建出来
- 但是如果我设置了`pair_key null unique`，创建时即使A先创建了，B创建时会命中唯一键，不新插行
- 避免用锁，消耗太大

### conversation_counter 会话序列号表
并发下为同一会话分配一个严格单调的序号。消息插入之前，先从序列号表拿到序列号，再插入消息，这些都是在一个事务中执行的：

```sql
START TRANSACTION;

-- 1) 先尝试插入（第一次会插入 latest_seq=1；后续并发会走更新 + 自增）
INSERT INTO conversation_counter (conv_id, latest_seq)
VALUES (?, 1)
ON DUPLICATE KEY UPDATE latest_seq = LAST_INSERT_ID(latest_seq + 1);

-- 2) 拿到本次分配的 seq
SET @seq := IF(ROW_COUNT() = 1, 1, LAST_INSERT_ID());

-- 3) 写消息（message 的主键是 (conv_id, seq)）
INSERT INTO message (conv_id, seq, msg_id, send_id, content, content_type, status)
VALUES (?, @seq, ?, ?, ?, 0, 0);

COMMIT;
```

**为什么安全**:
- 同一个 conv_id 的并发会在 conversation_counter 同一行上排队（行级锁）
- 一次只允许一个把 latest_seq 改到下一个值，所以拿到的 @seq 严格单调且唯一
- 全过程在一个事务里，如果后续插入 message 失败并回滚，latest_seq 的更改也会回滚，不产生"空洞序号"
    