# 块存储架构详细指南

## 概述

这个块存储架构实现了一个高效的消息存储系统，核心思想是将消息存储在固定大小的块中，块存储在固定容量的Store中。通过offset机制可以跨Store定位和拼接块内容。

## 核心数据结构

### 1. StoreConfig - Store配置

```go
type StoreConfig struct {
    MaxCapacity     int64  // Store最大容量（字节）
    TimelineMaxSize int64  // Timeline块最大大小（消息数量）
    DataDir         string // 数据目录
}
```

**用途**: 定义Store的基本配置参数
**使用场景**: 初始化Store时必须提供

### 2. StoreIndex - Store索引信息

```go
type StoreIndex struct {
    StoreID   string `json:"store_id"`
    Offset    int64  `json:"offset"`
    Size      int64  `json:"size"`
    CreatedAt int64  `json:"created_at"`
}
```

**用途**: 记录Timeline块在Store中的位置信息
**使用场景**: 
- 跨Store查找特定的Timeline块
- 实现块的快速定位
- 支持分布式存储扩展

### 3. TimelineBlock - Timeline块

```go
type TimelineBlock struct {
    BlockID   string        `json:"block_id"`
    StoreID   string        `json:"store_id"`
    Offset    int64         `json:"offset"`
    Size      int64         `json:"size"`
    Messages  []*Message    `json:"-"` // 内存中的消息缓存
    IsFull    bool          `json:"is_full"`
    NextBlock *TimelineBlock `json:"-"` // 下一个块的引用
    mu        sync.RWMutex
}
```

**用途**: 存储固定数量的消息，是存储的基本单元
**使用场景**:
- 消息达到块大小限制时自动创建新块
- 块满时持久化到磁盘
- 通过链表结构连接多个块

### 4. Store - 存储管理器

```go
type Store struct {
    Config          *StoreConfig
    StoreID         string
    CurrentCapacity int64
    ConvTimelines   map[string]*Timeline  // 会话时间线
    UserTimelines   map[string]*Timeline  // 用户时间线
    UserCheckpoints map[string]int64      // 用户检查点
    StoreIndex      map[string][]*StoreIndex
    TimelineBlocks  map[string]*TimelineBlock
    seqGenerator    int64
    mu              sync.RWMutex
}
```

**用途**: 管理所有Timeline和块，提供统一的存储接口
**使用场景**: 作为整个存储系统的入口点

### 5. Timeline - 时间线

```go
type Timeline struct {
    ID          string           `json:"id"`
    Type        string           `json:"type"` // "conv" 或 "user"
    Blocks      []*TimelineBlock `json:"blocks"`
    CurrentBlock *TimelineBlock  `json:"-"`
    LastSeqID   int64            `json:"last_seq_id"`
    mu          sync.RWMutex
}
```

**用途**: 管理一系列相关的消息块
**使用场景**:
- 会话消息的时间线管理
- 用户消息的时间线管理

## 核心函数详解

### 初始化和配置函数

#### `NewStore(config *StoreConfig) (*Store, error)`

**功能**: 创建新的Store实例
**参数**: 
- `config`: Store配置信息
**返回**: Store实例和错误信息
**使用场景**: 
- 应用启动时初始化存储系统
- 创建新的存储分区

**示例**:
```go
config := &StoreConfig{
    MaxCapacity:     10000,
    TimelineMaxSize: 100,
    DataDir:         "/data/storage",
}
store, err := NewStore(config)
```

### Timeline管理函数

#### `GetOrCreateConvTimeline(convID string) *Timeline`

**功能**: 获取或创建会话时间线
**参数**: 
- `convID`: 会话ID
**返回**: Timeline实例
**使用场景**: 
- 用户发送消息到特定会话时
- 需要查询会话历史消息时
- 会话相关的所有操作

**示例**:
```go
// 用户在会话中发送消息
convTimeline := store.GetOrCreateConvTimeline("chat_room_001")
```

#### `GetOrCreateUserTimeline(userID string) *Timeline`

**功能**: 获取或创建用户时间线
**参数**: 
- `userID`: 用户ID
**返回**: Timeline实例
**使用场景**: 
- 用户需要同步所有相关消息时
- 实现用户消息推送功能
- 用户离线消息管理

**示例**:
```go
// 用户上线时获取所有相关消息
userTimeline := store.GetOrCreateUserTimeline("user_123")
```

### 消息操作函数

#### `AddMessage(convID string, senderID uint32, data []byte, userIDs []string) error`

**功能**: 添加消息到存储系统
**参数**: 
- `convID`: 会话ID
- `senderID`: 发送者ID
- `data`: 消息内容
- `userIDs`: 相关用户ID列表
**返回**: 错误信息
**使用场景**: 
- 用户发送新消息时
- 系统消息推送时
- 批量消息导入时

**示例**:
```go
// 用户发送消息
err := store.AddMessage(
    "chat_room_001", 
    1001, 
    []byte("Hello, world!"), 
    []string{"user_123", "user_456"}
)
```

#### `GetConvMessages(convID string, limit int, beforeSeqID int64) ([]*Message, error)`

**功能**: 获取会话消息
**参数**: 
- `convID`: 会话ID
- `limit`: 消息数量限制
- `beforeSeqID`: 获取此序列号之前的消息（0表示最新消息）
**返回**: 消息列表和错误信息
**使用场景**: 
- 用户查看会话历史
- 分页加载历史消息
- 消息搜索功能

**示例**:
```go
// 获取最新的20条消息
messages, err := store.GetConvMessages("chat_room_001", 20, 0)

// 获取序列号1000之前的10条消息（向前翻页）
messages, err := store.GetConvMessages("chat_room_001", 10, 1000)
```

### Checkpoint管理函数

#### `GetUserCheckpoint(userID string) int64`

**功能**: 获取用户的消息检查点
**参数**: 
- `userID`: 用户ID
**返回**: 检查点序列号
**使用场景**: 
- 用户上线时确定未读消息起点
- 消息同步状态管理
- 离线消息推送

#### `UpdateUserCheckpoint(userID string, seqID int64)`

**功能**: 更新用户的消息检查点
**参数**: 
- `userID`: 用户ID
- `seqID`: 新的检查点序列号
**使用场景**: 
- 用户阅读消息后更新已读状态
- 消息确认机制
- 同步状态维护

#### `GetMessagesAfterCheckpoint(userID string) ([]*Message, error)`

**功能**: 获取用户检查点之后的所有消息
**参数**: 
- `userID`: 用户ID
**返回**: 消息列表和错误信息
**使用场景**: 
- 用户上线时获取未读消息
- 离线消息推送
- 消息同步功能

**示例**:
```go
// 用户上线流程
checkpoint := store.GetUserCheckpoint("user_123")
unreadMessages, err := store.GetMessagesAfterCheckpoint("user_123")

// 用户阅读消息后更新检查点
store.UpdateUserCheckpoint("user_123", latestSeqID)
```

### Timeline内部函数

#### `(tl *Timeline) AddMessage(msg *Message, store *Store) error`

**功能**: 向Timeline添加消息
**参数**: 
- `msg`: 消息对象
- `store`: Store实例
**返回**: 错误信息
**使用场景**: 
- Store.AddMessage内部调用
- 直接操作Timeline时使用

#### `(tl *Timeline) createNewBlock(store *Store) error`

**功能**: 为Timeline创建新的块
**参数**: 
- `store`: Store实例
**返回**: 错误信息
**使用场景**: 
- 当前块已满时自动调用
- Timeline初始化时调用

### 持久化函数

#### `saveTimelineBlock(block *TimelineBlock) error`

**功能**: 将Timeline块持久化到磁盘
**参数**: 
- `block`: 要保存的块
**返回**: 错误信息
**使用场景**: 
- 块满时自动持久化
- 系统关闭时保存未满的块
- 内存压力时释放块到磁盘

#### `loadTimelineBlock(blockID string) (*TimelineBlock, error)`

**功能**: 从磁盘加载Timeline块
**参数**: 
- `blockID`: 块ID
**返回**: 块对象和错误信息
**使用场景**: 
- 系统启动时恢复数据
- 访问历史消息时按需加载
- 内存不足时的懒加载

#### `saveTimelineMetadata(tl *Timeline) error`

**功能**: 保存Timeline元数据
**参数**: 
- `tl`: Timeline对象
**返回**: 错误信息
**使用场景**: 
- Timeline结构变化时
- 定期备份元数据
- 系统关闭时保存状态

#### `loadTimeline(tl *Timeline) error`

**功能**: 加载Timeline及其所有块
**参数**: 
- `tl`: Timeline对象
**返回**: 错误信息
**使用场景**: 
- 系统启动时恢复Timeline
- 冷启动时加载历史数据

### 工具函数

#### `NextSeqID() int64`

**功能**: 生成下一个全局唯一的序列号
**返回**: 序列号
**使用场景**: 
- 每条消息都需要唯一的序列号
- 消息排序和去重
- 检查点机制

## 使用流程示例

### 1. 系统初始化

```go
// 1. 创建配置
config := &StoreConfig{
    MaxCapacity:     1000000, // 1MB
    TimelineMaxSize: 100,     // 每块100条消息
    DataDir:         "/data/imy",
}

// 2. 初始化Store
store, err := NewStore(config)
if err != nil {
    log.Fatal(err)
}
```

### 2. 发送消息流程

```go
// 用户在群聊中发送消息
convID := "group_chat_001"
senderID := uint32(1001)
messageData := []byte("大家好！")
userIDs := []string{"user_001", "user_002", "user_003"}

// 添加消息（自动处理会话和用户时间线）
err := store.AddMessage(convID, senderID, messageData, userIDs)
if err != nil {
    log.Printf("发送消息失败: %v", err)
}
```

### 3. 用户上线流程

```go
userID := "user_001"

// 1. 获取用户检查点
checkpoint := store.GetUserCheckpoint(userID)

// 2. 获取未读消息
unreadMessages, err := store.GetMessagesAfterCheckpoint(userID)
if err != nil {
    log.Printf("获取未读消息失败: %v", err)
    return
}

// 3. 推送未读消息给用户
for _, msg := range unreadMessages {
    // 推送消息到客户端
    pushMessageToClient(userID, msg)
}

// 4. 更新检查点
if len(unreadMessages) > 0 {
    lastSeqID := unreadMessages[len(unreadMessages)-1].SeqID
    store.UpdateUserCheckpoint(userID, lastSeqID)
}
```

### 4. 查看历史消息流程

```go
convID := "group_chat_001"

// 1. 获取最新的20条消息
recentMessages, err := store.GetConvMessages(convID, 20, 0)
if err != nil {
    log.Printf("获取最新消息失败: %v", err)
    return
}

// 2. 用户向前翻页，获取更早的消息
if len(recentMessages) > 0 {
    earliestSeqID := recentMessages[0].SeqID
    olderMessages, err := store.GetConvMessages(convID, 20, earliestSeqID)
    if err != nil {
        log.Printf("获取历史消息失败: %v", err)
        return
    }
}
```

## 架构优势

### 1. 扩展性
- **水平扩展**: 通过多个Store实现分布式存储
- **容量管理**: 固定容量限制便于资源规划
- **索引机制**: 支持跨Store的快速定位

### 2. 性能优化
- **块级持久化**: 只在块满时写入磁盘，减少I/O
- **内存管理**: 按需加载块，避免大文件全量读取
- **并发安全**: 细粒度锁设计，提高并发性能

### 3. 数据安全
- **增量备份**: 块级别的持久化策略
- **故障恢复**: 通过元数据快速恢复Timeline结构
- **数据完整性**: Gob二进制格式确保数据一致性

### 4. 功能完整性
- **消息路由**: 同时支持会话和用户时间线
- **检查点机制**: 支持离线消息和已读状态
- **历史查询**: 支持分页和时间范围查询

## 注意事项

1. **并发安全**: 所有公共函数都是线程安全的
2. **内存使用**: 当前块保存在内存中，历史块按需加载
3. **磁盘空间**: 需要监控Store容量，及时清理或扩展
4. **错误处理**: 所有函数都返回详细的错误信息
5. **配置调优**: TimelineMaxSize和MaxCapacity需要根据实际场景调整

这个架构为高并发的即时通讯系统提供了强大而灵活的存储基础。