package main

import (
	"context"
	"fmt"
	"log"

	"imy/pkg/storage"
)

func main() {
	ctx := context.Background()
	fmt.Println("=== 分布式存储系统使用示例 ===")

	// 运行所有示例
	runAllExamples(ctx)
}

func runAllExamples(ctx context.Context) {
	// 1. 基本使用示例
	storage.ExampleUsage()

	// 2. Store注册示例
	registry := storage.NewInMemoryRegistry()
	exampleRegisterStores(ctx, registry)

	// 3. 高级功能示例
	exampleCustomRouting(ctx)
	exampleLoadBalancing(ctx)

	fmt.Println("\n=== 所有示例执行完成 ===")
}

// Store注册示例
func exampleRegisterStores(ctx context.Context, registry storage.StoreRegistry) {
	fmt.Println("\n--- Store注册示例 ---")

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

// 自定义路由策略示例
func exampleCustomRouting(ctx context.Context) {
	fmt.Println("\n--- 自定义路由策略 ---")

	// 创建一致性哈希路由器
	hashRouter := storage.NewConsistentHashRouter(3, 150, 0.8) // 3个节点，150个虚拟节点，0.8负载因子

	// 添加Store节点
	stores := []*storage.StoreInfo{
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

// 负载均衡示例
func exampleLoadBalancing(ctx context.Context) {
	fmt.Println("\n--- 负载均衡 ---")

	// 创建负载均衡路由器（使用轮询策略）
	lbRouter := storage.NewLoadBalancingRouter(storage.StrategyRoundRobin)

	// 添加Store节点
	stores := []*storage.StoreInfo{
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