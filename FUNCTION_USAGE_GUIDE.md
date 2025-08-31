# 块存储架构函数使用指南

本指南详细说明了块存储架构中每个函数的具体用法、参数说明、返回值以及适用场景。

## 目录

1. [初始化函数](#初始化函数)
2. [Timeline管理函数](#timeline管理函数)
3. [消息操作函数](#消息操作函数)
4. [Checkpoint管理函数](#checkpoint管理函数)
5. [查询函数](#查询函数)
6. [内部函数](#内部函数)
7. [实际使用场景](#实际使用场景)

---

## 初始化函数

### `NewStore(config *StoreConfig) (*Store, error)`

**用途**: 创建新的存储实例

**参数**:
- `config *StoreConfig`: 存储配置
  - `MaxCapacity int64`: Store最大容量（字节）
  - `TimelineMaxSize int`: 每个Timeline块的最大消息数
  - `DataDir string`: 数据存储目录

**返回值**:
- `*Store`: 存储实例
- `error`: 错误信息

**使用场景**:
- 应用启动时初始化存储系统
- 创建新的存储实例用于测试

**示例**:
```go
config := &storage.StoreConfig{
    MaxCapacity:     1024 * 1024, // 1MB
    TimelineMaxSize: 10,          // 每块10条消息
    DataDir:         "/data/imy",
}
store, err := storage.NewStore(config)
if err != nil {
    log.Fatal(err)
}
```

**适用场景**:
- ✅ 应用程序启动
- ✅ 单元测试初始化
- ✅ 数据迁移工具
- ❌ 频繁调用（应该复用实例）

---

## Timeline管理函数

### `GetOrCreateConvTimeline(convID string) *Timeline`

**用途**: 获取或创建会话Timeline

**参数**:
- `convID string`: 会话ID

**返回值**:
- `*Timeline`: Timeline实例

**使用场景**:
- 发送消息前获取会话Timeline
- 查看会话消息历史
- 会话管理

**示例**:
```go
// 群聊场景
groupTimeline := store.GetOrCreateConvTimeline("group_123")

// 私聊场景
privateTimeline := store.GetOrCreateConvTimeline("private_alice_bob")
```

**适用场景**:
- ✅ 发送消息前
- ✅ 查看消息历史
- ✅ 会话初始化
- ✅ 消息统计

### `GetOrCreateUserTimeline(userID string) *Timeline`

**用途**: 获取或创建用户Timeline

**参数**:
- `userID string`: 用户ID

**返回值**:
- `*Timeline`: Timeline实例

**使用场景**:
- 用户收到消息时更新个人Timeline
- 获取用户未读消息
- 用户消息统计

**示例**:
```go
// 获取用户Timeline
userTimeline := store.GetOrCreateUserTimeline("alice")

// 检查用户消息数量
fmt.Printf("用户消息块数: %d\n", len(userTimeline.Blocks))
```

**适用场景**:
- ✅ 消息推送到用户
- ✅ 未读消息查询
- ✅ 用户消息统计
- ✅ 离线消息处理

---

## 消息操作函数

### `AddMessage(convID string, senderID uint32, data []byte, participants []string) error`

**用途**: 添加消息到会话和所有参与者的Timeline

**参数**:
- `convID string`: 会话ID
- `senderID uint32`: 发送者ID
- `data []byte`: 消息内容
- `participants []string`: 参与者列表

**返回值**:
- `error`: 错误信息

**使用场景**:
- 用户发送消息
- 系统消息推送
- 机器人消息

**示例**:
```go
// 群聊消息
err := store.AddMessage(
    "tech_group", 
    1001, 
    []byte("大家好！"), 
    []string{"alice", "bob", "charlie"}
)

// 私聊消息
err := store.AddMessage(
    "private_alice_bob", 
    1001, 
    []byte("你好"), 
    []string{"alice", "bob"}
)

// 系统消息（senderID为0）
err := store.AddMessage(
    "system_notifications", 
    0, 
    []byte("系统维护通知"), 
    []string{"alice", "bob", "charlie"}
)
```

**适用场景**:
- ✅ 用户发送文本消息
- ✅ 用户发送图片/文件
- ✅ 系统通知推送
- ✅ 机器人自动回复
- ❌ 批量导入历史消息（应使用专门的导入函数）

---

## Checkpoint管理函数

### `GetUserCheckpoint(userID string) uint64`

**用途**: 获取用户当前的消息检查点

**参数**:
- `userID string`: 用户ID

**返回值**:
- `uint64`: 检查点SeqID

**使用场景**:
- 计算未读消息数量
- 用户上线时获取状态
- 消息同步

**示例**:
```go
// 获取用户检查点
checkpoint := store.GetUserCheckpoint("alice")
fmt.Printf("用户alice的检查点: %d\n", checkpoint)

// 检查是否有未读消息
if checkpoint == 0 {
    fmt.Println("新用户，没有历史记录")
}
```

**适用场景**:
- ✅ 用户登录时
- ✅ 计算未读消息
- ✅ 消息同步状态检查
- ✅ 离线消息处理

### `UpdateUserCheckpoint(userID string, seqID uint64)`

**用途**: 更新用户的消息检查点

**参数**:
- `userID string`: 用户ID
- `seqID uint64`: 新的检查点SeqID

**返回值**: 无

**使用场景**:
- 用户阅读消息后更新已读状态
- 消息确认
- 同步状态更新

**示例**:
```go
// 用户阅读消息后更新检查点
store.UpdateUserCheckpoint("alice", 12345)

// 批量标记已读
lastMessage := messages[len(messages)-1]
store.UpdateUserCheckpoint("alice", lastMessage.SeqID)
```

**适用场景**:
- ✅ 用户阅读消息
- ✅ 消息确认回执
- ✅ 离线消息同步完成
- ✅ 自动标记已读

---

## 查询函数

### `GetMessagesAfterCheckpoint(userID string) ([]*Message, error)`

**用途**: 获取用户检查点之后的所有未读消息

**参数**:
- `userID string`: 用户ID

**返回值**:
- `[]*Message`: 未读消息列表
- `error`: 错误信息

**使用场景**:
- 用户上线获取未读消息
- 推送通知计数
- 离线消息同步

**示例**:
```go
// 获取未读消息
unreadMessages, err := store.GetMessagesAfterCheckpoint("alice")
if err != nil {
    log.Printf("获取未读消息失败: %v", err)
    return
}

fmt.Printf("未读消息数量: %d\n", len(unreadMessages))
for _, msg := range unreadMessages {
    fmt.Printf("会话:%s, 发送者:%d, 内容:%s\n", 
        msg.ConvID, msg.SenderID, string(msg.Data))
}
```

**适用场景**:
- ✅ 用户登录时
- ✅ 推送通知
- ✅ 未读消息提醒
- ✅ 消息同步

### `GetConvMessages(convID string, limit int, beforeSeqID uint64) ([]*Message, error)`

**用途**: 获取会话中的消息，支持分页

**参数**:
- `convID string`: 会话ID
- `limit int`: 返回消息数量限制
- `beforeSeqID uint64`: 获取此SeqID之前的消息（0表示获取最新消息）

**返回值**:
- `[]*Message`: 消息列表（按时间倒序）
- `error`: 错误信息

**使用场景**:
- 查看消息历史
- 消息分页加载
- 搜索特定时间段消息

**示例**:
```go
// 获取最新的20条消息
recentMessages, err := store.GetConvMessages("group_123", 20, 0)

// 向前翻页，获取更早的消息
if len(recentMessages) > 0 {
    earliestSeqID := recentMessages[0].SeqID
    olderMessages, err := store.GetConvMessages("group_123", 20, earliestSeqID)
}

// 显示消息
for _, msg := range recentMessages {
    fmt.Printf("[%s] 用户%d: %s\n", 
        msg.CreateTime.Format("15:04:05"), 
        msg.SenderID, 
        string(msg.Data))
}
```

**适用场景**:
- ✅ 聊天界面消息加载
- ✅ 消息历史查看
- ✅ 消息搜索结果
- ✅ 消息导出

---

## 内部函数

### `Timeline.AddMessage(msg *Message) error`

**用途**: 向Timeline添加消息（内部函数）

**参数**:
- `msg *Message`: 消息对象

**返回值**:
- `error`: 错误信息

**使用场景**:
- Store内部调用
- 不建议直接使用

### `Timeline.GetMessages(limit int, beforeSeqID uint64) []*Message`

**用途**: 从Timeline获取消息（内部函数）

**参数**:
- `limit int`: 消息数量限制
- `beforeSeqID uint64`: 起始SeqID

**返回值**:
- `[]*Message`: 消息列表

**使用场景**:
- Store内部调用
- 不建议直接使用

---

## 实际使用场景

### 场景1: 即时通讯应用启动

```go
// 1. 初始化存储
config := &storage.StoreConfig{
    MaxCapacity:     100 * 1024 * 1024, // 100MB
    TimelineMaxSize: 50,                 // 每块50条消息
    DataDir:         "/data/chat",
}
store, err := storage.NewStore(config)

// 2. 加载用户数据
for _, userID := range activeUsers {
    checkpoint := store.GetUserCheckpoint(userID)
    unreadCount := len(store.GetMessagesAfterCheckpoint(userID))
    fmt.Printf("用户%s: 检查点%d, 未读%d\n", userID, checkpoint, unreadCount)
}
```

### 场景2: 用户发送消息

```go
// 接收到用户消息
func handleUserMessage(convID string, senderID uint32, content string, participants []string) {
    // 1. 添加消息到存储
    err := store.AddMessage(convID, senderID, []byte(content), participants)
    if err != nil {
        log.Printf("消息存储失败: %v", err)
        return
    }
    
    // 2. 推送给在线用户
    for _, userID := range participants {
        if isUserOnline(userID) {
            pushMessageToUser(userID, convID, content)
        }
    }
    
    // 3. 发送推送通知给离线用户
    for _, userID := range participants {
        if !isUserOnline(userID) {
            unreadCount := len(store.GetMessagesAfterCheckpoint(userID))
            sendPushNotification(userID, unreadCount)
        }
    }
}
```

### 场景3: 用户上线处理

```go
// 用户上线
func handleUserOnline(userID string) {
    // 1. 获取未读消息
    unreadMessages, err := store.GetMessagesAfterCheckpoint(userID)
    if err != nil {
        log.Printf("获取未读消息失败: %v", err)
        return
    }
    
    // 2. 发送未读消息给客户端
    for _, msg := range unreadMessages {
        sendMessageToClient(userID, msg)
    }
    
    // 3. 更新在线状态
    setUserOnline(userID, true)
}
```

### 场景4: 消息历史加载

```go
// 加载聊天历史
func loadChatHistory(convID string, page int, pageSize int) ([]*Message, error) {
    var beforeSeqID uint64 = 0
    
    // 如果不是第一页，需要计算起始位置
    if page > 1 {
        // 获取前面页数的消息来确定起始位置
        skipCount := (page - 1) * pageSize
        tempMessages, err := store.GetConvMessages(convID, skipCount, 0)
        if err != nil {
            return nil, err
        }
        if len(tempMessages) > 0 {
            beforeSeqID = tempMessages[len(tempMessages)-1].SeqID
        }
    }
    
    // 获取当前页消息
    return store.GetConvMessages(convID, pageSize, beforeSeqID)
}
```

### 场景5: 消息已读状态管理

```go
// 标记消息已读
func markMessagesAsRead(userID string, convID string, lastReadSeqID uint64) {
    // 1. 更新用户检查点
    store.UpdateUserCheckpoint(userID, lastReadSeqID)
    
    // 2. 通知其他参与者（如果需要已读回执）
    notifyReadReceipt(convID, userID, lastReadSeqID)
    
    // 3. 更新未读计数缓存
    updateUnreadCountCache(userID)
}
```

### 场景6: 系统消息推送

```go
// 系统广播消息
func broadcastSystemMessage(content string, targetUsers []string) {
    systemConvID := "system_broadcast"
    
    // 使用senderID=0表示系统消息
    err := store.AddMessage(systemConvID, 0, []byte(content), targetUsers)
    if err != nil {
        log.Printf("系统消息发送失败: %v", err)
        return
    }
    
    // 推送给在线用户
    for _, userID := range targetUsers {
        if isUserOnline(userID) {
            pushSystemMessage(userID, content)
        }
    }
}
```

## 性能优化建议

### 1. 合理设置配置参数

```go
// 根据业务场景调整参数
config := &storage.StoreConfig{
    MaxCapacity:     500 * 1024 * 1024, // 根据服务器内存调整
    TimelineMaxSize: 100,                // 根据消息频率调整
    DataDir:         "/fast-ssd/data",   // 使用SSD存储
}
```

### 2. 批量操作

```go
// 避免频繁的单条消息查询
// ❌ 不好的做法
for _, userID := range users {
    messages, _ := store.GetMessagesAfterCheckpoint(userID)
    // 处理消息
}

// ✅ 更好的做法
type UserMessages struct {
    UserID   string
    Messages []*Message
}

var userMessagesList []UserMessages
for _, userID := range users {
    messages, _ := store.GetMessagesAfterCheckpoint(userID)
    userMessagesList = append(userMessagesList, UserMessages{
        UserID:   userID,
        Messages: messages,
    })
}
// 批量处理
```

### 3. 缓存策略

```go
// 缓存热点数据
var (
    checkpointCache = make(map[string]uint64)
    cacheMutex      sync.RWMutex
)

func getCachedCheckpoint(userID string) uint64 {
    cacheMutex.RLock()
    defer cacheMutex.RUnlock()
    
    if checkpoint, exists := checkpointCache[userID]; exists {
        return checkpoint
    }
    
    // 从存储获取并缓存
    checkpoint := store.GetUserCheckpoint(userID)
    cacheMutex.Lock()
    checkpointCache[userID] = checkpoint
    cacheMutex.Unlock()
    
    return checkpoint
}
```

## 错误处理最佳实践

### 1. 消息发送失败处理

```go
func sendMessageWithRetry(convID string, senderID uint32, content []byte, participants []string) error {
    maxRetries := 3
    var lastErr error
    
    for i := 0; i < maxRetries; i++ {
        err := store.AddMessage(convID, senderID, content, participants)
        if err == nil {
            return nil
        }
        
        lastErr = err
        log.Printf("消息发送失败，重试 %d/%d: %v", i+1, maxRetries, err)
        time.Sleep(time.Duration(i+1) * time.Second)
    }
    
    return fmt.Errorf("消息发送最终失败: %v", lastErr)
}
```

### 2. 存储空间不足处理

```go
func handleStorageError(err error) {
    if strings.Contains(err.Error(), "storage capacity exceeded") {
        // 触发数据清理或扩容
        go cleanupOldData()
        
        // 通知管理员
        notifyAdmin("存储空间不足，需要扩容")
        
        // 临时降级服务
        enableEmergencyMode()
    }
}
```

这个指南涵盖了块存储架构中所有主要函数的详细用法和适用场景。通过这些示例和最佳实践，你可以更好地理解和使用这个存储系统。