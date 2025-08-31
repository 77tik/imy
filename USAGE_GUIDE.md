# 分布式存储系统使用指南

本指南将详细介绍如何使用已实现的分布式存储系统。

## 系统概述

这是一个基于Timeline的分布式存储系统，具有以下核心特性：

- **分布式架构**：支持多个Store节点的水平扩展
- **自动负载均衡**：智能路由和数据分布
- **高可用性**：故障检测和自动恢复
- **一致性保证**：分布式锁和事务支持
- **性能优化**：多级缓存和性能监控

## 快速开始

### 1. 基本组件初始化

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "your-project/pkg/storage"
)

func main() {
    ctx := context.Background()
    
    // 1. 创建Store注册中心
    registry := storage.NewInMemoryRegistry()
    
    // 2. 创建全局索引
    globalIndex := storage.NewDistributedIndex(registry)
    
    // 3. 创建Store发现客户端
    discoveryClient := storage.NewStoreDiscoveryClient(registry)
    
    // 4. 创建Store配置
    config := &storage.StoreConfig{
        ID:       "store-1",
        Address:  "localhost:8080",
        DataDir:  "./data/store-1",
        Capacity: 10 * 1024 * 1024 * 1024, // 10GB
    }
    
    // 5. 创建Store实例
    store, err := storage.NewStore(config)
    if err != nil {
        log.Fatal("创建Store失败:", err)
    }
    
    fmt.Println("✓ 分布式存储系统初始化完成")
}
```

### 2. Store节点注册

```go
// 注册Store节点到注册中心
func registerStores(ctx context.Context, registry storage.StoreRegistry) {
    stores := []*storage.StoreInfo{
        {
            ID:      "store-1",
            Address: "192.168.1.10:8080",
            Status:  "healthy",
            Metadata: map[string]interface{}{
                "region":   "us-west-1",
                "capacity": 10737418240, // 10GB
                "tags":     []string{"ssd", "high-performance"},
            },
        },
        {
            ID:      "store-2",
            Address: "192.168.1.11:8080",
            Status:  "healthy",
            Metadata: map[string]interface{}{
                "region":   "us-west-1",
                "capacity": 21474836480, // 20GB
                "tags":     []string{"hdd", "large-capacity"},
            },
        },
    }
    
    for _, store := range stores {
        err := registry.Register(ctx, store)
        if err != nil {
            log.Printf("注册Store %s 失败: %v", store.ID, err)
            continue
        }
        fmt.Printf("✓ Store %s 注册成功\n", store.ID)
    }
}
```

### 3. Timeline操作

```go
// 基本Timeline操作
func basicTimelineOperations(ctx context.Context, store *storage.Store) {
    // 创建Timeline
    timeline := &storage.Timeline{
        Key:       "user:1001:messages",
        StoreID:   store.GetID(),
        CreatedAt: time.Now(),
    }
    
    err := store.CreateTimeline(ctx, timeline)
    if err != nil {
        log.Printf("创建Timeline失败: %v", err)
        return
    }
    fmt.Printf("✓ Timeline %s 创建成功\n", timeline.Key)
    
    // 添加消息
    messages := []*storage.Message{
        {
            ID:        "msg-1",
            Content:   []byte("Hello, World!"),
            Timestamp: time.Now(),
        },
        {
            ID:        "msg-2",
            Content:   []byte("分布式存储系统测试"),
            Timestamp: time.Now().Add(time.Second),
        },
    }
    
    for _, msg := range messages {
        err := store.AppendMessage(ctx, timeline.Key, msg)
        if err != nil {
            log.Printf("添加消息失败: %v", err)
            continue
        }
        fmt.Printf("✓ 消息 %s 添加成功\n", msg.ID)
    }
    
    // 查询消息
    query := &storage.Query{
        TimelineKey: timeline.Key,
        StartTime:   time.Now().Add(-time.Hour),
        EndTime:     time.Now().Add(time.Hour),
        Limit:       10,
    }
    
    results, err := store.QueryMessages(ctx, query)
    if err != nil {
        log.Printf("查询消息失败: %v", err)
        return
    }
    
    fmt.Printf("✓ 查询到 %d 条消息\n", len(results))
    for _, msg := range results {
        fmt.Printf("  - %s: %s\n", msg.ID, string(msg.Content))
    }
}
```

## 高级功能

### 1. 路由策略配置

#### 一致性哈希路由

```go
func setupConsistentHashRouting() {
    // 创建一致性哈希路由器
    router := storage.NewConsistentHashRouter(3, 150, 0.8)
    
    // 添加Store节点
    stores := []*storage.StoreInfo{
        {ID: "store-1", Address: "192.168.1.10:8080", Status: "healthy"},
        {ID: "store-2", Address: "192.168.1.11:8080", Status: "healthy"},
        {ID: "store-3", Address: "192.168.1.12:8080", Status: "healthy"},
    }
    
    for _, store := range stores {
        router.AddStore(store)
    }
    
    // 路由Timeline
    timelineKey := "user:1001:messages"
    storeID, err := router.RouteTimeline(timelineKey)
    if err != nil {
        log.Printf("路由失败: %v", err)
        return
    }
    
    fmt.Printf("Timeline %s 路由到 Store: %s\n", timelineKey, storeID)
}
```

#### 负载均衡路由

```go
func setupLoadBalancingRouting() {
    // 创建负载均衡路由器
    router := storage.NewLoadBalancingRouter(storage.StrategyRoundRobin)
    
    // 添加Store节点
    stores := []*storage.StoreInfo{
        {ID: "store-1", Address: "192.168.1.10:8080", Status: "healthy"},
        {ID: "store-2", Address: "192.168.1.11:8080", Status: "healthy"},
        {ID: "store-3", Address: "192.168.1.12:8080", Status: "healthy"},
    }
    
    for _, store := range stores {
        router.AddStore(store)
    }
    
    // 支持的负载均衡策略：
    // - StrategyRoundRobin: 轮询
    // - StrategyLeastLoad: 最少负载
    // - StrategyWeightedRoundRobin: 加权轮询
    // - StrategyRandom: 随机
}
```

### 2. 分片管理

```go
func setupShardManagement(registry storage.StoreRegistry, globalIndex storage.GlobalIndex) {
    // 创建分片策略
    policy := &storage.ShardPolicy{
        Strategy:             storage.ShardByLoad,
        MaxTimelinePerStore:  1000,
        MaxSizePerStore:      10 * 1024 * 1024 * 1024, // 10GB
        LoadBalanceThreshold: 0.8,
        ReplicationFactor:    1,
        AutoRebalance:        true,
        RebalanceInterval:    5 * time.Minute,
    }
    
    // 创建分片管理器
    shardManager := storage.NewTimelineShardManager(registry, globalIndex, policy)
    
    ctx := context.Background()
    
    // 获取分片推荐
    recommendation, err := shardManager.GetShardRecommendation(ctx, "user:1001:messages", 1024*1024)
    if err != nil {
        log.Printf("获取分片推荐失败: %v", err)
        return
    }
    
    fmt.Printf("推荐Store: %s (置信度: %.2f)\n", 
        recommendation.RecommendedStore, recommendation.Confidence)
    
    // 获取重平衡推荐
    rebalanceRecommendations, err := shardManager.GetRebalanceRecommendations(ctx)
    if err != nil {
        log.Printf("获取重平衡推荐失败: %v", err)
        return
    }
    
    for _, rec := range rebalanceRecommendations {
        fmt.Printf("重平衡推荐: %s 从 %s 迁移到 %s (优先级: %d)\n",
            rec.TimelineKey, rec.FromStore, rec.ToStore, rec.Priority)
    }
}
```

### 3. 缓存系统

```go
func setupCacheSystem() {
    // 创建多级缓存
    cacheManager := storage.NewCacheManager()
    
    // L1缓存配置（内存缓存）
    l1Config := &storage.CacheConfig{
        Level:    1,
        Type:     "memory",
        Size:     100 * 1024 * 1024, // 100MB
        TTL:      5 * time.Minute,
        MaxItems: 10000,
    }
    
    // L2缓存配置（Redis缓存）
    l2Config := &storage.CacheConfig{
        Level:    2,
        Type:     "redis",
        Size:     1024 * 1024 * 1024, // 1GB
        TTL:      30 * time.Minute,
        MaxItems: 100000,
        Options: map[string]interface{}{
            "address":  "localhost:6379",
            "password": "",
            "db":       0,
        },
    }
    
    // 添加缓存层
    cacheManager.AddCacheLevel(l1Config)
    cacheManager.AddCacheLevel(l2Config)
    
    ctx := context.Background()
    
    // 缓存数据
    key := "timeline:user:1001:messages"
    data := []byte("cached timeline data")
    
    err := cacheManager.Set(ctx, key, data, 10*time.Minute)
    if err != nil {
        log.Printf("缓存设置失败: %v", err)
        return
    }
    
    // 获取缓存数据
    cachedData, hit, err := cacheManager.Get(ctx, key)
    if err != nil {
        log.Printf("缓存获取失败: %v", err)
        return
    }
    
    if hit {
        fmt.Printf("缓存命中: %s\n", string(cachedData))
    } else {
        fmt.Println("缓存未命中")
    }
}
```

### 4. 性能监控

```go
func setupPerformanceMonitoring() {
    // 创建性能优化器
    optimizer := storage.NewPerformanceOptimizer()
    
    ctx := context.Background()
    
    // 获取性能指标
    metrics, err := optimizer.GetMetrics(ctx)
    if err != nil {
        log.Printf("获取性能指标失败: %v", err)
        return
    }
    
    fmt.Printf("性能指标:\n")
    fmt.Printf("  - 总请求数: %d\n", metrics.TotalRequests)
    fmt.Printf("  - 平均响应时间: %.2fms\n", metrics.AverageResponseTime)
    fmt.Printf("  - 缓存命中率: %.2f%%\n", metrics.CacheHitRate*100)
    fmt.Printf("  - 错误率: %.2f%%\n", metrics.ErrorRate*100)
    
    // 获取优化建议
    suggestions, err := optimizer.GetOptimizationSuggestions(ctx)
    if err != nil {
        log.Printf("获取优化建议失败: %v", err)
        return
    }
    
    fmt.Println("\n优化建议:")
    for _, suggestion := range suggestions {
        fmt.Printf("  - %s (优先级: %s)\n", suggestion.Description, suggestion.Priority)
    }
}
```

### 5. 跨Store数据访问

```go
func setupCrossStoreAccess(registry storage.StoreRegistry) {
    // 创建跨Store访问客户端
    crossStoreClient := storage.NewCrossStoreAccessClient(registry)
    
    ctx := context.Background()
    
    // 跨Store查询
    query := &storage.CrossStoreQuery{
        TimelineKeys: []string{"user:1001:messages", "user:1002:messages"},
        StartTime:    time.Now().Add(-time.Hour),
        EndTime:      time.Now(),
        Limit:        100,
    }
    
    results, err := crossStoreClient.QueryAcrossStores(ctx, query)
    if err != nil {
        log.Printf("跨Store查询失败: %v", err)
        return
    }
    
    fmt.Printf("跨Store查询结果: %d 条记录\n", len(results))
    
    // 获取Store统计信息
    stats, err := crossStoreClient.GetStoreStats(ctx, "store-1")
    if err != nil {
        log.Printf("获取Store统计失败: %v", err)
        return
    }
    
    fmt.Printf("Store统计信息:\n")
    fmt.Printf("  - Timeline数量: %d\n", stats.TimelineCount)
    fmt.Printf("  - 总大小: %d bytes\n", stats.TotalSize)
    fmt.Printf("  - 平均响应时间: %.2fms\n", stats.AverageResponseTime)
}
```

## 故障处理和恢复

### 1. 健康检查

```go
func setupHealthCheck(registry storage.StoreRegistry) {
    // 创建健康检查器
    healthChecker := storage.NewHealthChecker(registry)
    
    ctx := context.Background()
    
    // 检查所有Store的健康状态
    healthStatus, err := healthChecker.CheckAllStores(ctx)
    if err != nil {
        log.Printf("健康检查失败: %v", err)
        return
    }
    
    for storeID, status := range healthStatus {
        fmt.Printf("Store %s: %s\n", storeID, status.Status)
        if status.Status != "healthy" {
            fmt.Printf("  错误: %s\n", status.Error)
        }
    }
}
```

### 2. 故障恢复

```go
func setupFailureRecovery(registry storage.StoreRegistry, globalIndex storage.GlobalIndex) {
    // 创建故障恢复管理器
    recoveryManager := storage.NewFailureRecoveryManager(registry, globalIndex)
    
    ctx := context.Background()
    
    // 检测故障Store
    failedStores, err := recoveryManager.DetectFailedStores(ctx)
    if err != nil {
        log.Printf("检测故障Store失败: %v", err)
        return
    }
    
    // 执行故障恢复
    for _, storeID := range failedStores {
        fmt.Printf("检测到故障Store: %s，开始恢复...\n", storeID)
        
        err := recoveryManager.RecoverStore(ctx, storeID)
        if err != nil {
            log.Printf("Store %s 恢复失败: %v", storeID, err)
            continue
        }
        
        fmt.Printf("✓ Store %s 恢复成功\n", storeID)
    }
}
```

## 配置示例

### 1. 生产环境配置

```go
// 生产环境推荐配置
func getProductionConfig() *storage.SystemConfig {
    return &storage.SystemConfig{
        // Store配置
        StoreConfig: &storage.StoreConfig{
            MaxTimelinePerStore: 10000,
            MaxSizePerStore:     100 * 1024 * 1024 * 1024, // 100GB
            BlockSize:           4 * 1024 * 1024,          // 4MB
            CompressionEnabled:  true,
            EncryptionEnabled:   true,
        },
        
        // 分片配置
        ShardPolicy: &storage.ShardPolicy{
            Strategy:             storage.ShardByLoad,
            LoadBalanceThreshold: 0.75,
            ReplicationFactor:    3,
            AutoRebalance:        true,
            RebalanceInterval:    10 * time.Minute,
        },
        
        // 缓存配置
        CacheConfig: &storage.CacheConfig{
            Level:    1,
            Type:     "redis-cluster",
            Size:     10 * 1024 * 1024 * 1024, // 10GB
            TTL:      60 * time.Minute,
            MaxItems: 1000000,
        },
        
        // 性能配置
        PerformanceConfig: &storage.PerformanceConfig{
            MaxConcurrentRequests: 1000,
            RequestTimeout:        30 * time.Second,
            BatchSize:             100,
            EnableMetrics:         true,
            MetricsInterval:       time.Minute,
        },
    }
}
```

### 2. 开发环境配置

```go
// 开发环境配置
func getDevelopmentConfig() *storage.SystemConfig {
    return &storage.SystemConfig{
        StoreConfig: &storage.StoreConfig{
            MaxTimelinePerStore: 100,
            MaxSizePerStore:     1 * 1024 * 1024 * 1024, // 1GB
            BlockSize:           1 * 1024 * 1024,        // 1MB
            CompressionEnabled:  false,
            EncryptionEnabled:   false,
        },
        
        ShardPolicy: &storage.ShardPolicy{
            Strategy:             storage.ShardBySize,
            LoadBalanceThreshold: 0.9,
            ReplicationFactor:    1,
            AutoRebalance:        false,
        },
        
        CacheConfig: &storage.CacheConfig{
            Level:    1,
            Type:     "memory",
            Size:     100 * 1024 * 1024, // 100MB
            TTL:      5 * time.Minute,
            MaxItems: 1000,
        },
    }
}
```

## 最佳实践

### 1. 性能优化

- **批量操作**：使用批量API减少网络开销
- **缓存策略**：合理配置多级缓存提高读取性能
- **分片策略**：根据数据访问模式选择合适的分片策略
- **连接池**：使用连接池管理Store连接

### 2. 可靠性保证

- **副本配置**：生产环境建议设置3个副本
- **健康检查**：定期检查Store健康状态
- **故障恢复**：配置自动故障检测和恢复机制
- **数据备份**：定期备份重要数据

### 3. 监控和运维

- **性能监控**：监控关键性能指标
- **日志记录**：记录详细的操作日志
- **告警机制**：配置关键指标的告警
- **容量规划**：根据业务增长规划存储容量

### 4. 安全考虑

- **访问控制**：实施适当的访问控制策略
- **数据加密**：对敏感数据进行加密存储
- **网络安全**：使用TLS加密网络通信
- **审计日志**：记录所有重要操作的审计日志

## 故障排查

### 常见问题

1. **Store连接失败**
   - 检查网络连接
   - 验证Store地址和端口
   - 检查防火墙设置

2. **性能问题**
   - 检查缓存命中率
   - 分析慢查询日志
   - 监控系统资源使用情况

3. **数据不一致**
   - 检查副本同步状态
   - 验证分布式锁机制
   - 检查网络分区情况

4. **容量问题**
   - 监控存储使用率
   - 检查数据分布是否均匀
   - 考虑扩容或数据清理

### 调试工具

系统提供了多种调试和监控工具：

- **性能分析器**：分析系统性能瓶颈
- **数据一致性检查器**：验证数据一致性
- **负载分析器**：分析Store负载分布
- **网络诊断工具**：诊断网络连接问题

## 总结

这个分布式存储系统提供了完整的企业级功能，包括：

- ✅ 分布式架构和水平扩展
- ✅ 智能路由和负载均衡
- ✅ 自动故障检测和恢复
- ✅ 多级缓存和性能优化
- ✅ 数据一致性和事务支持
- ✅ 全面的监控和运维工具

通过本指南，您可以快速上手并在生产环境中部署使用这个分布式存储系统。如有问题，请参考故障排查部分或查看详细的API文档。