package storage

import (
	"context"
	"fmt"
	"log"
	"time"
)

// 分布式存储系统使用示例
// 本文件展示如何使用已实现的分布式存储系统

// ExampleUsage 展示完整的使用流程
func ExampleUsage() {
	fmt.Println("=== 分布式存储系统使用示例 ===")
	
	// 1. 初始化系统组件
	ctx := context.Background()
	
	// 创建Store配置
	storeConfig := &StoreConfig{
		MaxCapacity:     10 * 1024 * 1024 * 1024, // 10GB
		TimelineMaxSize: 1000,                    // 每个Timeline块1000条消息
		DataDir:         "/tmp/imy_storage",
	}
	
	// 创建Store实例
	store, err := NewStore(storeConfig)
	if err != nil {
		log.Printf("创建Store失败: %v", err)
		return
	}
	
	// 创建Store注册中心
	storeRegistry := NewInMemoryRegistry()
	
	// 创建多级缓存管理器
	l1Cache := NewMemoryCache(100 * 1024 * 1024) // 100MB内存缓存
	l2Cache := NewDiskCache("/tmp/cache")         // 磁盘缓存
	l3Cache := NewDistributedCache([]string{"node1:8080", "node2:8080"}) // 分布式缓存
	cacheManager := NewMultiLevelCacheManager(l1Cache, l2Cache, l3Cache)
	
	// 创建性能优化器
	performanceOptimizer := NewPerformanceOptimizer()
	
	fmt.Println("✓ 系统组件初始化完成")
	
	// 2. 注册Store节点
	exampleRegisterStores(ctx, storeRegistry)
	
	// 3. 基本存储操作
	exampleBasicOperations(ctx, store)
	
	// 4. 缓存使用
	exampleCacheUsage(ctx, cacheManager)
	
	// 5. 性能监控
	examplePerformanceMonitoring(performanceOptimizer)
	
	fmt.Println("=== 示例完成 ===")
}

// 注册Store节点示例
func exampleRegisterStores(ctx context.Context, registry StoreRegistry) {
	fmt.Println("\n--- 注册Store节点 ---")
	
	// 注册多个Store节点
	stores := []*StoreInfo{
		{
			ID:      "store-1",
			Address: "192.168.1.10:8080",
			Status:  "active",
			Metadata: map[string]interface{}{
				"capacity": 1024 * 1024 * 1024, // 1GB
				"region":   "us-west-1",
				"type":     "ssd",
				"tier":     "hot",
			},
		},
		{
			ID:      "store-2",
			Address: "192.168.1.11:8080",
			Status:  "active",
			Metadata: map[string]interface{}{
				"capacity": 2048 * 1024 * 1024, // 2GB
				"region":   "us-west-2",
				"type":     "hdd",
				"tier":     "warm",
			},
		},
		{
			ID:      "store-3",
			Address: "192.168.1.12:8080",
			Status:  "active",
			Metadata: map[string]interface{}{
				"capacity": 512 * 1024 * 1024, // 512MB
				"region":   "us-east-1",
				"type":     "ssd",
				"tier":     "hot",
			},
		},
	}
	
	for _, store := range stores {
		err := registry.Register(ctx, store)
		if err != nil {
			log.Printf("注册Store %s 失败: %v", store.ID, err)
			continue
		}
		fmt.Printf("✓ 注册Store: %s (%s)\n", store.ID, store.Address)
	}
}

// 基本存储操作示例
func exampleBasicOperations(ctx context.Context, store *Store) {
	fmt.Println("\n--- 基本存储操作 ---")
	
	// 1. 创建会话Timeline
	convID := "chat_room_001"
	convTimeline := store.GetOrCreateConvTimeline(convID)
	fmt.Printf("✓ 创建会话Timeline: %s\n", convID)
	
	// 2. 创建用户Timeline
	userID := "user_1001"
	userTimeline := store.GetOrCreateUserTimeline(userID)
	fmt.Printf("✓ 创建用户Timeline: %s\n", userID)
	
	// 3. 添加消息
	messageData := []byte("Hello, this is a test message!")
	userIDs := []string{"user_1001", "user_1002"}
	
	err := store.AddMessage(convID, 1001, messageData, userIDs)
	if err != nil {
		log.Printf("添加消息失败: %v", err)
	} else {
		fmt.Printf("✓ 添加消息到会话: %s\n", convID)
	}
	
	// 4. 查询会话消息
	messages, err := store.GetConvMessages(convID, 10, 0)
	if err != nil {
		log.Printf("查询消息失败: %v", err)
	} else {
		fmt.Printf("✓ 查询到 %d 条消息\n", len(messages))
		for _, msg := range messages {
			fmt.Printf("  - [%d] 发送者: %d, 内容: %s\n", 
				msg.SeqID, msg.SenderID, string(msg.Data))
		}
	}
	
	// 5. 用户检查点操作
	checkpoint := store.GetUserCheckpoint(userID)
	fmt.Printf("✓ 用户 %s 当前检查点: %d\n", userID, checkpoint)
	
	// 更新检查点
	store.UpdateUserCheckpoint(userID, 1)
	fmt.Printf("✓ 更新用户 %s 检查点到: 1\n", userID)
	
	// 6. 获取未读消息
	unreadMessages, err := store.GetMessagesAfterCheckpoint(userID)
	if err != nil {
		log.Printf("获取未读消息失败: %v", err)
	} else {
		fmt.Printf("✓ 用户 %s 有 %d 条未读消息\n", userID, len(unreadMessages))
	}
	
	_ = convTimeline
	_ = userTimeline
}

// 批量消息操作示例
func exampleBatchOperations(ctx context.Context, store *Store) {
	fmt.Println("\n--- 批量消息操作 ---")
	
	convID := "group_chat_001"
	
	// 批量添加消息
	messages := []struct {
		senderID uint32
		content  string
	}{
		{1001, "Hello everyone!"},
		{1002, "Hi there!"},
		{1003, "Good morning!"},
		{1001, "How is everyone doing?"},
		{1002, "Great, thanks for asking!"},
	}
	
	userIDs := []string{"user_1001", "user_1002", "user_1003"}
	
	for _, msg := range messages {
		messageData := []byte(msg.content)
		err := store.AddMessage(convID, msg.senderID, messageData, userIDs)
		if err != nil {
			log.Printf("添加消息失败: %v", err)
			continue
		}
		fmt.Printf("✓ 用户 %d 发送消息: %s\n", msg.senderID, msg.content)
	}
	
	// 查询所有消息
	allMessages, err := store.GetConvMessages(convID, 100, 0)
	if err != nil {
		log.Printf("查询消息失败: %v", err)
	} else {
		fmt.Printf("✓ 会话 %s 总共有 %d 条消息\n", convID, len(allMessages))
	}
}

// 分页查询示例
func examplePaginationQuery(ctx context.Context, store *Store) {
	fmt.Println("\n--- 分页查询示例 ---")
	
	convID := "group_chat_001"
	
	// 第一页：获取最新的5条消息
	firstPage, err := store.GetConvMessages(convID, 5, 0)
	if err != nil {
		log.Printf("查询第一页失败: %v", err)
		return
	}
	
	fmt.Printf("✓ 第一页消息 (%d 条):\n", len(firstPage))
	for _, msg := range firstPage {
		fmt.Printf("  - [%d] 发送者: %d, 时间: %s\n", 
			msg.SeqID, msg.SenderID, msg.CreateTime.Format("15:04:05"))
	}
	
	// 第二页：获取更早的消息
	if len(firstPage) > 0 {
		oldestSeqID := firstPage[0].SeqID
		secondPage, err := store.GetConvMessages(convID, 5, oldestSeqID)
		if err != nil {
			log.Printf("查询第二页失败: %v", err)
			return
		}
		
		fmt.Printf("✓ 第二页消息 (%d 条):\n", len(secondPage))
		for _, msg := range secondPage {
			fmt.Printf("  - [%d] 发送者: %d, 时间: %s\n", 
				msg.SeqID, msg.SenderID, msg.CreateTime.Format("15:04:05"))
		}
	}
}

// Store统计信息示例
func exampleStoreStats(ctx context.Context, store *Store) {
	fmt.Println("\n--- Store统计信息 ---")
	
	// 显示Store基本信息
	fmt.Printf("✓ Store ID: %s\n", store.StoreID)
	fmt.Printf("✓ 当前容量: %d bytes\n", store.CurrentCapacity)
	fmt.Printf("✓ 最大容量: %d bytes\n", store.Config.MaxCapacity)
	fmt.Printf("✓ Timeline块大小: %d 条消息\n", store.Config.TimelineMaxSize)
	fmt.Printf("✓ 数据目录: %s\n", store.Config.DataDir)
	
	// 统计Timeline数量
	convCount := len(store.ConvTimelines)
	userCount := len(store.UserTimelines)
	blockCount := len(store.TimelineBlocks)
	
	fmt.Printf("✓ 会话Timeline数量: %d\n", convCount)
	fmt.Printf("✓ 用户Timeline数量: %d\n", userCount)
	fmt.Printf("✓ Timeline块数量: %d\n", blockCount)
	
	// 显示用户检查点
	fmt.Println("✓ 用户检查点:")
	for userID, checkpoint := range store.UserCheckpoints {
		fmt.Printf("  - 用户 %s: %d\n", userID, checkpoint)
	}
}

// 缓存使用示例
func exampleCacheUsage(ctx context.Context, cacheManager CacheManager) {
	fmt.Println("\n--- 缓存使用 ---")
	
	// 设置缓存
	key := "user:1001:profile"
	value := map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
		"age":   25,
	}
	
	err := cacheManager.Set(ctx, key, value, 5*time.Minute)
	if err != nil {
		log.Printf("设置缓存失败: %v", err)
		return
	}
	fmt.Printf("✓ 设置缓存: %s\n", key)
	
	// 获取缓存
	cachedValue, found, err := cacheManager.Get(ctx, key)
	if err != nil {
		log.Printf("获取缓存失败: %v", err)
		return
	}
	
	if found {
		fmt.Printf("✓ 缓存命中: %v\n", cachedValue)
	} else {
		fmt.Println("✗ 缓存未命中")
	}
	
	// 预热缓存
	warmKeys := []string{"user:1002:profile", "user:1003:profile"}
	err = cacheManager.Warm(ctx, warmKeys)
	if err != nil {
		log.Printf("预热缓存失败: %v", err)
	} else {
		fmt.Printf("✓ 预热缓存: %v\n", warmKeys)
	}
	
	// 获取缓存统计
	stats := cacheManager.GetStats(L1Cache)
	fmt.Printf("✓ L1缓存统计 - 命中: %d, 未命中: %d, 命中率: %.2f%%\n", 
		stats.Hits, stats.Misses, stats.HitRatio*100)
}

// 性能监控示例
func examplePerformanceMonitoring(optimizer *PerformanceOptimizer) {
	fmt.Println("\n--- 性能监控 ---")
	
	// 记录操作指标
	optimizer.RecordMetrics("get_messages", 50*time.Millisecond, true)
	optimizer.RecordMetrics("add_message", 30*time.Millisecond, true)
	optimizer.RecordMetrics("create_timeline", 100*time.Millisecond, false)
	
	// 获取性能指标
	metrics := optimizer.GetMetrics()
	
	fmt.Println("✓ 性能指标:")
	for operation, count := range metrics.OperationCounts {
		duration := metrics.OperationDurations[operation]
		successRate := metrics.SuccessRates[operation]
		avgDuration := duration / time.Duration(count)
		
		fmt.Printf("  - %s: 次数=%d, 平均耗时=%v, 成功率=%.1f%%\n", 
			operation, count, avgDuration, successRate*100)
	}
}

// 高级使用场景示例
func ExampleAdvancedUsage() {
	fmt.Println("\n=== 高级使用场景 ===")
	
	ctx := context.Background()
	
	// 1. 自定义路由策略
	exampleCustomRouting(ctx)
	
	// 2. 分片策略配置
	exampleShardingConfiguration(ctx)
	
	// 3. 故障恢复
	exampleFailureRecovery(ctx)
	
	// 4. 负载均衡
	exampleLoadBalancing(ctx)
}

// 自定义路由策略示例
func exampleCustomRouting(ctx context.Context) {
	fmt.Println("\n--- 自定义路由策略 ---")
	
	// 创建一致性哈希路由器
	hashRouter := NewConsistentHashRouter(3, 150, 0.8) // 3个节点，150个虚拟节点，0.8负载因子
	
	// 添加Store节点
	stores := []*StoreInfo{
		{ID: "store-1", Address: "192.168.1.10:8080", Status: "healthy"},
		{ID: "store-2", Address: "192.168.1.11:8080", Status: "healthy"},
		{ID: "store-3", Address: "192.168.1.12:8080", Status: "healthy"},
	}
	
	for _, store := range stores {
		err := hashRouter.AddStore(store)
		if err != nil {
			log.Printf("添加Store失败: %v", err)
			continue
		}
		fmt.Printf("✓ 添加Store: %s\n", store.ID)
	}
	
	// 路由Timeline
	timelineKey := "user:1001:messages"
	storeID, err := hashRouter.RouteTimeline(timelineKey)
	if err != nil {
		log.Printf("路由失败: %v", err)
		return
	}
	
	fmt.Printf("✓ Timeline %s 路由到 Store: %s\n", timelineKey, storeID)
}

// 分片策略配置示例
func exampleShardingConfiguration(ctx context.Context) {
	fmt.Println("\n--- 分片策略配置 ---")
	
	// 创建自定义分片策略
	policy := map[string]interface{}{
		"strategy":              "load_based",
		"max_timeline_per_store": 1000,
		"max_size_per_store":     10 * 1024 * 1024 * 1024, // 10GB
		"load_balance_threshold": 0.8,
		"replication_factor":     2,
		"auto_rebalance":         true,
		"rebalance_interval":     "30m",
	}
	
	fmt.Printf("✓ 配置分片策略: %+v\n", policy)
}

// 故障恢复示例
func exampleFailureRecovery(ctx context.Context) {
	fmt.Println("\n--- 故障恢复 ---")
	
	// 模拟Store故障检测和恢复
	failedStoreID := "store-2"
	fmt.Printf("✗ 检测到Store故障: %s\n", failedStoreID)
	
	// 触发故障转移
	fmt.Println("✓ 启动故障转移流程")
	fmt.Println("✓ 重新路由受影响的Timeline")
	fmt.Println("✓ 更新全局索引")
	fmt.Println("✓ 故障恢复完成")
}

// 负载均衡示例
func exampleLoadBalancing(ctx context.Context) {
	fmt.Println("\n--- 负载均衡 ---")
	
	// 创建负载均衡路由器（使用轮询策略）
	lbRouter := NewLoadBalancingRouter(StrategyRoundRobin)
	
	// 添加Store节点
	stores := []*StoreInfo{
		{ID: "store-1", Address: "192.168.1.10:8080", Status: "healthy"},
		{ID: "store-2", Address: "192.168.1.11:8080", Status: "healthy"},
		{ID: "store-3", Address: "192.168.1.12:8080", Status: "healthy"},
	}
	
	for _, store := range stores {
		err := lbRouter.AddStore(store)
		if err != nil {
			log.Printf("添加Store失败: %v", err)
			continue
		}
		fmt.Printf("✓ 添加Store: %s\n", store.ID)
	}
	
	fmt.Println("✓ 配置负载均衡策略: 轮询")
	
	// 模拟多次路由请求
	for i := 0; i < 6; i++ {
		timelineKey := fmt.Sprintf("timeline_%d", i)
		storeID, err := lbRouter.RouteTimeline(timelineKey)
		if err != nil {
			log.Printf("路由失败: %v", err)
			continue
		}
		fmt.Printf("  Timeline %s -> Store %s\n", timelineKey, storeID)
	}
}