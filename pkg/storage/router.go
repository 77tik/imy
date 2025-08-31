package storage

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"
	"time"
)

// StoreStatus Store状态常量
const (
	StoreStatusHealthy = "healthy"
	StoreStatusUnhealthy = "unhealthy"
)

// TimelineRouter Timeline路由器接口
type TimelineRouter interface {
	// 路由Timeline到指定Store
	RouteTimeline(timelineKey string) (string, error)
	// 获取Timeline的所有副本Store
	GetTimelineReplicas(timelineKey string) ([]string, error)
	// 添加Store节点
	AddStore(storeInfo *StoreInfo) error
	// 移除Store节点
	RemoveStore(storeID string) error
	// 更新Store负载信息
	UpdateStoreLoad(storeID string, load *StoreLoad) error
	// 获取最佳Store（负载均衡）
	GetBestStore() (string, error)
	// 重新平衡Timeline分布
	Rebalance() ([]*MigrationPlan, error)
}

// StoreLoad Store负载信息
type StoreLoad struct {
	StoreID         string    `json:"store_id"`
	TimelineCount   int       `json:"timeline_count"`   // Timeline数量
	BlockCount      int       `json:"block_count"`      // 块数量
	TotalSize       int64     `json:"total_size"`       // 总大小（字节）
	UsedCapacity    int64     `json:"used_capacity"`    // 已使用容量
	MaxCapacity     int64     `json:"max_capacity"`     // 最大容量
	CPUUsage        float64   `json:"cpu_usage"`        // CPU使用率
	MemoryUsage     float64   `json:"memory_usage"`     // 内存使用率
	NetworkLatency  int64     `json:"network_latency"`  // 网络延迟（毫秒）
	LastUpdate      time.Time `json:"last_update"`      // 最后更新时间
}

// MigrationPlan 迁移计划
type MigrationPlan struct {
	TimelineKey   string `json:"timeline_key"`
	SourceStoreID string `json:"source_store_id"`
	TargetStoreID string `json:"target_store_id"`
	Reason        string `json:"reason"`
	Priority      int    `json:"priority"` // 优先级，数字越小优先级越高
}

// ConsistentHashRouter 一致性哈希路由器
type ConsistentHashRouter struct {
	mu           sync.RWMutex
	stores       map[string]*StoreInfo // Store信息
	loads        map[string]*StoreLoad // Store负载信息
	hashRing     *HashRing             // 一致性哈希环
	replicas     int                   // 副本数量
	virtualNodes int                   // 虚拟节点数量
	loadThreshold float64              // 负载阈值
}

// HashRing 一致性哈希环
type HashRing struct {
	nodes    []uint32          // 排序的哈希值
	nodeMap  map[uint32]string // 哈希值到Store ID的映射
	virtuals int               // 每个物理节点的虚拟节点数
}

// NewConsistentHashRouter 创建一致性哈希路由器
func NewConsistentHashRouter(replicas, virtualNodes int, loadThreshold float64) *ConsistentHashRouter {
	return &ConsistentHashRouter{
		stores:        make(map[string]*StoreInfo),
		loads:         make(map[string]*StoreLoad),
		hashRing:      NewHashRing(virtualNodes),
		replicas:      replicas,
		virtualNodes:  virtualNodes,
		loadThreshold: loadThreshold,
	}
}

// NewHashRing 创建哈希环
func NewHashRing(virtualNodes int) *HashRing {
	return &HashRing{
		nodes:    make([]uint32, 0),
		nodeMap:  make(map[uint32]string),
		virtuals: virtualNodes,
	}
}

// AddNode 添加节点到哈希环
func (hr *HashRing) AddNode(nodeID string) {
	for i := 0; i < hr.virtuals; i++ {
		hash := hr.hash(fmt.Sprintf("%s:%d", nodeID, i))
		hr.nodes = append(hr.nodes, hash)
		hr.nodeMap[hash] = nodeID
	}
	sort.Slice(hr.nodes, func(i, j int) bool {
		return hr.nodes[i] < hr.nodes[j]
	})
}

// RemoveNode 从哈希环移除节点
func (hr *HashRing) RemoveNode(nodeID string) {
	newNodes := make([]uint32, 0)
	for _, hash := range hr.nodes {
		if hr.nodeMap[hash] != nodeID {
			newNodes = append(newNodes, hash)
		} else {
			delete(hr.nodeMap, hash)
		}
	}
	hr.nodes = newNodes
}

// GetNode 获取key对应的节点
func (hr *HashRing) GetNode(key string) string {
	if len(hr.nodes) == 0 {
		return ""
	}
	
	hash := hr.hash(key)
	idx := sort.Search(len(hr.nodes), func(i int) bool {
		return hr.nodes[i] >= hash
	})
	
	if idx == len(hr.nodes) {
		idx = 0
	}
	
	return hr.nodeMap[hr.nodes[idx]]
}

// GetNodes 获取key对应的多个节点（用于副本）
func (hr *HashRing) GetNodes(key string, count int) []string {
	if len(hr.nodeMap) == 0 {
		return []string{}
	}
	
	if count > len(hr.nodeMap) {
		count = len(hr.nodeMap)
	}
	
	hash := hr.hash(key)
	idx := sort.Search(len(hr.nodes), func(i int) bool {
		return hr.nodes[i] >= hash
	})
	
	if idx == len(hr.nodes) {
		idx = 0
	}
	
	result := make([]string, 0, count)
	seen := make(map[string]bool)
	
	for len(result) < count && len(seen) < len(hr.nodeMap) {
		nodeID := hr.nodeMap[hr.nodes[idx]]
		if !seen[nodeID] {
			result = append(result, nodeID)
			seen[nodeID] = true
		}
		idx = (idx + 1) % len(hr.nodes)
	}
	
	return result
}

// hash 计算哈希值
func (hr *HashRing) hash(key string) uint32 {
	h := sha256.Sum256([]byte(key))
	return uint32(h[0])<<24 | uint32(h[1])<<16 | uint32(h[2])<<8 | uint32(h[3])
}

// RouteTimeline 路由Timeline到指定Store
func (r *ConsistentHashRouter) RouteTimeline(timelineKey string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if len(r.stores) == 0 {
		return "", fmt.Errorf("no available stores")
	}
	
	// 首先尝试使用一致性哈希
	storeID := r.hashRing.GetNode(timelineKey)
	if storeID == "" {
		return "", fmt.Errorf("failed to route timeline")
	}
	
	// 检查Store是否健康且负载不过高
	store, exists := r.stores[storeID]
	if !exists || store.Status != StoreStatusHealthy {
		// 如果主Store不可用，选择备用Store
		return r.getBestAvailableStore()
	}
	
	// 检查负载
	load, hasLoad := r.loads[storeID]
	if hasLoad && r.isOverloaded(load) {
		// 如果负载过高，选择负载较低的Store
		return r.getBestAvailableStore()
	}
	
	return storeID, nil
}

// GetTimelineReplicas 获取Timeline的所有副本Store
func (r *ConsistentHashRouter) GetTimelineReplicas(timelineKey string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if len(r.stores) == 0 {
		return nil, fmt.Errorf("no available stores")
	}
	
	replicas := r.hashRing.GetNodes(timelineKey, r.replicas)
	
	// 过滤掉不健康的Store
	healthyReplicas := make([]string, 0, len(replicas))
	for _, storeID := range replicas {
		if store, exists := r.stores[storeID]; exists && store.Status == StoreStatusHealthy {
			healthyReplicas = append(healthyReplicas, storeID)
		}
	}
	
	return healthyReplicas, nil
}

// AddStore 添加Store节点
func (r *ConsistentHashRouter) AddStore(storeInfo *StoreInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.stores[storeInfo.ID] = storeInfo
	r.hashRing.AddNode(storeInfo.ID)
	
	return nil
}

// RemoveStore 移除Store节点
func (r *ConsistentHashRouter) RemoveStore(storeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.stores, storeID)
	delete(r.loads, storeID)
	r.hashRing.RemoveNode(storeID)
	
	return nil
}

// UpdateStoreLoad 更新Store负载信息
func (r *ConsistentHashRouter) UpdateStoreLoad(storeID string, load *StoreLoad) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	load.LastUpdate = time.Now()
	r.loads[storeID] = load
	
	return nil
}

// GetBestStore 获取最佳Store（负载均衡）
func (r *ConsistentHashRouter) GetBestStore() (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.getBestAvailableStore()
}

// getBestAvailableStore 获取最佳可用Store（内部方法，需要持有锁）
func (r *ConsistentHashRouter) getBestAvailableStore() (string, error) {
	if len(r.stores) == 0 {
		return "", fmt.Errorf("no available stores")
	}
	
	var bestStoreID string
	var bestScore float64 = -1
	
	for storeID, store := range r.stores {
		if store.Status != StoreStatusHealthy {
			continue
		}
		
		load, hasLoad := r.loads[storeID]
		if !hasLoad {
			// 如果没有负载信息，假设负载较低
			if bestStoreID == "" {
				bestStoreID = storeID
			}
			continue
		}
		
		// 计算Store评分（越高越好）
		score := r.calculateStoreScore(load)
		if score > bestScore {
			bestScore = score
			bestStoreID = storeID
		}
	}
	
	if bestStoreID == "" {
		return "", fmt.Errorf("no healthy stores available")
	}
	
	return bestStoreID, nil
}

// calculateStoreScore 计算Store评分
func (r *ConsistentHashRouter) calculateStoreScore(load *StoreLoad) float64 {
	// 容量利用率（越低越好）
	capacityRatio := float64(load.UsedCapacity) / float64(load.MaxCapacity)
	if capacityRatio > 1.0 {
		capacityRatio = 1.0
	}
	
	// CPU使用率（越低越好）
	cpuScore := 1.0 - load.CPUUsage
	if cpuScore < 0 {
		cpuScore = 0
	}
	
	// 内存使用率（越低越好）
	memoryScore := 1.0 - load.MemoryUsage
	if memoryScore < 0 {
		memoryScore = 0
	}
	
	// 网络延迟（越低越好，转换为评分）
	latencyScore := 1.0
	if load.NetworkLatency > 0 {
		latencyScore = 1.0 / (1.0 + float64(load.NetworkLatency)/1000.0)
	}
	
	// 综合评分（权重可调整）
	score := (1.0-capacityRatio)*0.3 + cpuScore*0.25 + memoryScore*0.25 + latencyScore*0.2
	
	return score
}

// isOverloaded 检查Store是否过载
func (r *ConsistentHashRouter) isOverloaded(load *StoreLoad) bool {
	// 检查容量
	if load.MaxCapacity > 0 {
		capacityRatio := float64(load.UsedCapacity) / float64(load.MaxCapacity)
		if capacityRatio > r.loadThreshold {
			return true
		}
	}
	
	// 检查CPU使用率
	if load.CPUUsage > r.loadThreshold {
		return true
	}
	
	// 检查内存使用率
	if load.MemoryUsage > r.loadThreshold {
		return true
	}
	
	return false
}

// Rebalance 重新平衡Timeline分布
func (r *ConsistentHashRouter) Rebalance() ([]*MigrationPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	plans := make([]*MigrationPlan, 0)
	
	// 找出过载的Store
	overloadedStores := make([]string, 0)
	underloadedStores := make([]string, 0)
	
	for storeID, store := range r.stores {
		if store.Status != StoreStatusHealthy {
			continue
		}
		
		load, hasLoad := r.loads[storeID]
		if !hasLoad {
			continue
		}
		
		if r.isOverloaded(load) {
			overloadedStores = append(overloadedStores, storeID)
		} else if r.calculateStoreScore(load) > 0.7 { // 负载较低的Store
			underloadedStores = append(underloadedStores, storeID)
		}
	}
	
	// 为过载的Store创建迁移计划
	for _, overloadedStore := range overloadedStores {
		if len(underloadedStores) == 0 {
			break
		}
		
		// 选择目标Store
		targetStore := underloadedStores[0]
		
		// TODO: 这里需要实际的Timeline列表来创建具体的迁移计划
		// 现在创建一个示例计划
		plan := &MigrationPlan{
			TimelineKey:   fmt.Sprintf("timeline_from_%s", overloadedStore),
			SourceStoreID: overloadedStore,
			TargetStoreID: targetStore,
			Reason:        "Load balancing",
			Priority:      1,
		}
		plans = append(plans, plan)
		
		// 轮换目标Store
		underloadedStores = append(underloadedStores[1:], underloadedStores[0])
	}
	
	return plans, nil
}

// LoadBalancingRouter 负载均衡路由器
type LoadBalancingRouter struct {
	mu           sync.RWMutex
	stores       map[string]*StoreInfo
	loads        map[string]*StoreLoad
	strategy     LoadBalancingStrategy
	roundRobinIdx int
}

// LoadBalancingStrategy 负载均衡策略
type LoadBalancingStrategy int

const (
	StrategyRoundRobin LoadBalancingStrategy = iota // 轮询
	StrategyLeastLoad                               // 最少负载
	StrategyWeightedRoundRobin                      // 加权轮询
	StrategyRandom                                  // 随机
)

// NewLoadBalancingRouter 创建负载均衡路由器
func NewLoadBalancingRouter(strategy LoadBalancingStrategy) *LoadBalancingRouter {
	return &LoadBalancingRouter{
		stores:   make(map[string]*StoreInfo),
		loads:    make(map[string]*StoreLoad),
		strategy: strategy,
	}
}

// RouteTimeline 路由Timeline（负载均衡）
func (r *LoadBalancingRouter) RouteTimeline(timelineKey string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	healthyStores := r.getHealthyStores()
	if len(healthyStores) == 0 {
		return "", fmt.Errorf("no healthy stores available")
	}
	
	switch r.strategy {
	case StrategyRoundRobin:
		return r.roundRobinSelect(healthyStores), nil
	case StrategyLeastLoad:
		return r.leastLoadSelect(healthyStores), nil
	case StrategyWeightedRoundRobin:
		return r.roundRobinSelect(healthyStores), nil
	case StrategyRandom:
		return r.randomSelect(healthyStores), nil
	default:
		return r.roundRobinSelect(healthyStores), nil
	}
}

// getHealthyStores 获取健康的Store列表
func (r *LoadBalancingRouter) getHealthyStores() []string {
	healthyStores := make([]string, 0)
	for storeID, store := range r.stores {
		if store.Status == StoreStatusHealthy {
			healthyStores = append(healthyStores, storeID)
		}
	}
	return healthyStores
}

// roundRobinSelect 轮询选择
func (r *LoadBalancingRouter) roundRobinSelect(stores []string) string {
	if len(stores) == 0 {
		return ""
	}
	storeID := stores[r.roundRobinIdx%len(stores)]
	r.roundRobinIdx++
	return storeID
}

// leastLoadSelect 最少负载选择
func (r *LoadBalancingRouter) leastLoadSelect(stores []string) string {
	if len(stores) == 0 {
		return ""
	}
	
	bestStore := stores[0]
	bestScore := -1.0
	
	for _, storeID := range stores {
		load, hasLoad := r.loads[storeID]
		if !hasLoad {
			return storeID // 没有负载信息的Store优先
		}
		
		score := r.calculateStoreScore(load)
		if score > bestScore {
			bestScore = score
			bestStore = storeID
		}
	}
	
	return bestStore
}

// randomSelect 随机选择
func (r *LoadBalancingRouter) randomSelect(stores []string) string {
	if len(stores) == 0 {
		return ""
	}
	// 简单的伪随机选择
	idx := int(time.Now().UnixNano()) % len(stores)
	return stores[idx]
}

// calculateStoreScore 计算Store评分（与一致性哈希路由器相同）
func (r *LoadBalancingRouter) calculateStoreScore(load *StoreLoad) float64 {
	capacityRatio := float64(load.UsedCapacity) / float64(load.MaxCapacity)
	if capacityRatio > 1.0 {
		capacityRatio = 1.0
	}
	
	cpuScore := 1.0 - load.CPUUsage
	if cpuScore < 0 {
		cpuScore = 0
	}
	
	memoryScore := 1.0 - load.MemoryUsage
	if memoryScore < 0 {
		memoryScore = 0
	}
	
	latencyScore := 1.0
	if load.NetworkLatency > 0 {
		latencyScore = 1.0 / (1.0 + float64(load.NetworkLatency)/1000.0)
	}
	
	score := (1.0-capacityRatio)*0.3 + cpuScore*0.25 + memoryScore*0.25 + latencyScore*0.2
	
	return score
}

// 实现TimelineRouter接口的其他方法
func (r *LoadBalancingRouter) GetTimelineReplicas(timelineKey string) ([]string, error) {
	// 负载均衡路由器不支持副本，返回单个Store
	storeID, err := r.RouteTimeline(timelineKey)
	if err != nil {
		return nil, err
	}
	return []string{storeID}, nil
}

func (r *LoadBalancingRouter) AddStore(storeInfo *StoreInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stores[storeInfo.ID] = storeInfo
	return nil
}

func (r *LoadBalancingRouter) RemoveStore(storeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.stores, storeID)
	delete(r.loads, storeID)
	return nil
}

func (r *LoadBalancingRouter) UpdateStoreLoad(storeID string, load *StoreLoad) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	load.LastUpdate = time.Now()
	r.loads[storeID] = load
	return nil
}

func (r *LoadBalancingRouter) GetBestStore() (string, error) {
	return r.RouteTimeline("")
}

func (r *LoadBalancingRouter) Rebalance() ([]*MigrationPlan, error) {
	// 负载均衡路由器不需要重新平衡
	return []*MigrationPlan{}, nil
}

// RouterManager 路由管理器
type RouterManager struct {
	mu           sync.RWMutex
	routers      map[string]TimelineRouter // 支持多种路由策略
	defaultName  string                     // 默认路由器
}

// NewRouterManager 创建路由管理器
func NewRouterManager() *RouterManager {
	return &RouterManager{
		routers: make(map[string]TimelineRouter),
	}
}

// RegisterRouter 注册路由器
func (rm *RouterManager) RegisterRouter(name string, router TimelineRouter) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.routers[name] = router
	if rm.defaultName == "" {
		rm.defaultName = name
	}
}

// SetDefaultRouter 设置默认路由器
func (rm *RouterManager) SetDefaultRouter(name string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if _, exists := rm.routers[name]; !exists {
		return fmt.Errorf("router %s not found", name)
	}
	rm.defaultName = name
	return nil
}

// GetRouter 获取路由器
func (rm *RouterManager) GetRouter(name string) (TimelineRouter, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	if name == "" {
		name = rm.defaultName
	}
	router, exists := rm.routers[name]
	if !exists {
		return nil, fmt.Errorf("router %s not found", name)
	}
	return router, nil
}

// RouteTimeline 使用默认路由器路由Timeline
func (rm *RouterManager) RouteTimeline(timelineKey string) (string, error) {
	router, err := rm.GetRouter("")
	if err != nil {
		return "", err
	}
	return router.RouteTimeline(timelineKey)
}