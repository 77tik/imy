package storage

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// ShardStrategy 分片策略
type ShardStrategy string

const (
	ShardByHash       ShardStrategy = "hash"        // 基于哈希的分片
	ShardByLoad       ShardStrategy = "load"        // 基于负载的分片
	ShardBySize       ShardStrategy = "size"        // 基于大小的分片
	ShardByGeography  ShardStrategy = "geography"   // 基于地理位置的分片
)

// ShardPolicy 分片策略配置
type ShardPolicy struct {
	Strategy          ShardStrategy `json:"strategy"`
	MaxTimelinePerStore int         `json:"max_timeline_per_store"`  // 每个Store最大Timeline数
	MaxSizePerStore     int64       `json:"max_size_per_store"`      // 每个Store最大数据大小(字节)
	LoadBalanceThreshold float64    `json:"load_balance_threshold"`  // 负载均衡阈值(0.0-1.0)
	ReplicationFactor   int         `json:"replication_factor"`      // 副本因子
	AutoRebalance       bool        `json:"auto_rebalance"`          // 是否自动重平衡
	RebalanceInterval   time.Duration `json:"rebalance_interval"`    // 重平衡检查间隔
}

// DefaultShardPolicy 默认分片策略
func DefaultShardPolicy() *ShardPolicy {
	return &ShardPolicy{
		Strategy:             ShardByLoad,
		MaxTimelinePerStore:  1000,
		MaxSizePerStore:      10 * 1024 * 1024 * 1024, // 10GB
		LoadBalanceThreshold: 0.8,
		ReplicationFactor:    1,
		AutoRebalance:        true,
		RebalanceInterval:    5 * time.Minute,
	}
}

// ShardRecommendation 分片推荐
type ShardRecommendation struct {
	TimelineKey     string   `json:"timeline_key"`
	RecommendedStore string  `json:"recommended_store"`
	Reason          string   `json:"reason"`
	Confidence      float64  `json:"confidence"`      // 推荐置信度(0.0-1.0)
	Alternatives    []string `json:"alternatives"`    // 备选Store
}

// RebalanceRecommendation 重平衡推荐
type RebalanceRecommendation struct {
	TimelineKey string `json:"timeline_key"`
	FromStore   string `json:"from_store"`
	ToStore     string `json:"to_store"`
	Reason      string `json:"reason"`
	Priority    int    `json:"priority"`    // 优先级(1-10, 10最高)
	ExpectedGain float64 `json:"expected_gain"` // 预期收益
}

// ShardManager 分片管理器接口
type ShardManager interface {
	// GetShardRecommendation 获取新Timeline的分片推荐
	GetShardRecommendation(ctx context.Context, timelineKey string, estimatedSize int64) (*ShardRecommendation, error)
	
	// GetRebalanceRecommendations 获取重平衡推荐
	GetRebalanceRecommendations(ctx context.Context) ([]*RebalanceRecommendation, error)
	
	// UpdateShardPolicy 更新分片策略
	UpdateShardPolicy(policy *ShardPolicy) error
	
	// GetShardPolicy 获取当前分片策略
	GetShardPolicy() *ShardPolicy
	
	// StartAutoRebalance 启动自动重平衡
	StartAutoRebalance(ctx context.Context) error
	
	// StopAutoRebalance 停止自动重平衡
	StopAutoRebalance() error
	
	// GetShardStats 获取分片统计信息
	GetShardStats(ctx context.Context) (*ShardStats, error)
}

// ShardStats 分片统计信息
type ShardStats struct {
	TotalStores      int                         `json:"total_stores"`
	TotalTimelines   int                         `json:"total_timelines"`
	TotalSize        int64                       `json:"total_size"`
	AverageLoad      float64                     `json:"average_load"`
	LoadVariance     float64                     `json:"load_variance"`
	StoreStats       map[string]*ShardStoreStats `json:"store_stats"`
	LastRebalance    *time.Time                  `json:"last_rebalance,omitempty"`
	RebalanceCount   int                         `json:"rebalance_count"`
}

// ShardStoreStats Store分片统计信息
type ShardStoreStats struct {
	StoreID        string  `json:"store_id"`
	TimelineCount  int     `json:"timeline_count"`
	TotalSize      int64   `json:"total_size"`
	LoadFactor     float64 `json:"load_factor"`     // 负载因子(0.0-1.0)
	HealthScore    float64 `json:"health_score"`    // 健康评分(0.0-1.0)
	LastUpdate     time.Time `json:"last_update"`
}

// TimelineShardManager Timeline分片管理器实现
type TimelineShardManager struct {
	mu                sync.RWMutex
	policy            *ShardPolicy
	globalIndex       GlobalIndexManager
	storeRegistry     StoreRegistry
	routerManager     *RouterManager
	migrationManager  MigrationManager
	autoRebalanceStop chan struct{}
	autoRebalanceRunning bool
	stats             *ShardStats
}

// NewTimelineShardManager 创建Timeline分片管理器
func NewTimelineShardManager(
	globalIndex GlobalIndexManager,
	storeRegistry StoreRegistry,
	routerManager *RouterManager,
	migrationManager MigrationManager,
) *TimelineShardManager {
	return &TimelineShardManager{
		policy:           DefaultShardPolicy(),
		globalIndex:      globalIndex,
		storeRegistry:    storeRegistry,
		routerManager:    routerManager,
		migrationManager: migrationManager,
		stats:            &ShardStats{StoreStats: make(map[string]*ShardStoreStats)},
	}
}

// GetShardRecommendation 获取新Timeline的分片推荐
func (tsm *TimelineShardManager) GetShardRecommendation(ctx context.Context, timelineKey string, estimatedSize int64) (*ShardRecommendation, error) {
	tsm.mu.RLock()
	defer tsm.mu.RUnlock()
	
	// 获取所有可用的Store
	stores, err := tsm.storeRegistry.ListStores(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list stores: %w", err)
	}
	
	if len(stores) == 0 {
		return nil, fmt.Errorf("no available stores")
	}
	
	// 根据策略选择Store
	switch tsm.policy.Strategy {
	case ShardByHash:
		return tsm.recommendByHash(ctx, timelineKey, stores)
	case ShardByLoad:
		return tsm.recommendByLoad(ctx, timelineKey, estimatedSize, stores)
	case ShardBySize:
		return tsm.recommendBySize(ctx, timelineKey, estimatedSize, stores)
	default:
		return tsm.recommendByLoad(ctx, timelineKey, estimatedSize, stores)
	}
}

// recommendByHash 基于哈希的推荐
func (tsm *TimelineShardManager) recommendByHash(ctx context.Context, timelineKey string, stores []*StoreInfo) (*ShardRecommendation, error) {
	// 使用路由器的哈希算法
	router, err := tsm.routerManager.GetRouter("")
	if err != nil {
		return nil, fmt.Errorf("failed to get router: %w", err)
	}
	
	recommendedStore, err := router.RouteTimeline(timelineKey)
	if err != nil {
		return nil, fmt.Errorf("failed to route timeline: %w", err)
	}
	
	// 获取备选Store
	alternatives := make([]string, 0, len(stores)-1)
	for _, store := range stores {
		if store.ID != recommendedStore {
			alternatives = append(alternatives, store.ID)
		}
	}
	
	return &ShardRecommendation{
		TimelineKey:      timelineKey,
		RecommendedStore: recommendedStore,
		Reason:           "Consistent hash routing",
		Confidence:       0.9,
		Alternatives:     alternatives,
	}, nil
}

// recommendByLoad 基于负载的推荐
func (tsm *TimelineShardManager) recommendByLoad(ctx context.Context, timelineKey string, estimatedSize int64, stores []*StoreInfo) (*ShardRecommendation, error) {
	// 获取每个Store的负载信息
	type storeLoad struct {
		storeInfo *StoreInfo
		loadInfo  *StoreLoadInfo
		loadFactor float64
	}
	
	storeLoads := make([]*storeLoad, 0, len(stores))
	for _, store := range stores {
		if store.Status != "active" {
			continue // 跳过不活跃的Store
		}
		
		loadInfo, err := tsm.globalIndex.GetStoreLoad(ctx, store.ID)
		if err != nil {
			continue // 跳过无法获取负载信息的Store
		}
		
		// 计算负载因子
		loadFactor := tsm.calculateLoadFactor(loadInfo, estimatedSize)
		
		storeLoads = append(storeLoads, &storeLoad{
			storeInfo:  store,
			loadInfo:   loadInfo,
			loadFactor: loadFactor,
		})
	}
	
	if len(storeLoads) == 0 {
		return nil, fmt.Errorf("no healthy stores available")
	}
	
	// 按负载因子排序（升序）
	sort.Slice(storeLoads, func(i, j int) bool {
		return storeLoads[i].loadFactor < storeLoads[j].loadFactor
	})
	
	// 选择负载最低的Store
	bestStore := storeLoads[0]
	
	// 检查是否超过阈值
	if bestStore.loadFactor > tsm.policy.LoadBalanceThreshold {
		return nil, fmt.Errorf("all stores are overloaded")
	}
	
	// 获取备选Store
	alternatives := make([]string, 0, min(3, len(storeLoads)-1))
	for i := 1; i < len(storeLoads) && i <= 3; i++ {
		alternatives = append(alternatives, storeLoads[i].storeInfo.ID)
	}
	
	confidence := 1.0 - bestStore.loadFactor // 负载越低，置信度越高
	
	return &ShardRecommendation{
		TimelineKey:      timelineKey,
		RecommendedStore: bestStore.storeInfo.ID,
		Reason:           fmt.Sprintf("Lowest load factor: %.2f", bestStore.loadFactor),
		Confidence:       confidence,
		Alternatives:     alternatives,
	}, nil
}

// recommendBySize 基于大小的推荐
func (tsm *TimelineShardManager) recommendBySize(ctx context.Context, timelineKey string, estimatedSize int64, stores []*StoreInfo) (*ShardRecommendation, error) {
	// 过滤出有足够空间的Store
	type storeCapacity struct {
		storeInfo     *StoreInfo
		availableSize int64
		usageRatio    float64
	}
	
	validStores := make([]*storeCapacity, 0, len(stores))
	for _, store := range stores {
		if store.Status != "active" {
			continue
		}
		
		loadInfo, err := tsm.globalIndex.GetStoreLoad(ctx, store.ID)
		if err != nil {
			continue
		}
		
		availableSize := tsm.policy.MaxSizePerStore - loadInfo.TotalSize
		if availableSize < estimatedSize {
			continue // 空间不足
		}
		
		usageRatio := float64(loadInfo.TotalSize) / float64(tsm.policy.MaxSizePerStore)
		
		validStores = append(validStores, &storeCapacity{
			storeInfo:     store,
			availableSize: availableSize,
			usageRatio:    usageRatio,
		})
	}
	
	if len(validStores) == 0 {
		return nil, fmt.Errorf("no stores with sufficient space")
	}
	
	// 按使用率排序（升序）
	sort.Slice(validStores, func(i, j int) bool {
		return validStores[i].usageRatio < validStores[j].usageRatio
	})
	
	bestStore := validStores[0]
	
	// 获取备选Store
	alternatives := make([]string, 0, min(3, len(validStores)-1))
	for i := 1; i < len(validStores) && i <= 3; i++ {
		alternatives = append(alternatives, validStores[i].storeInfo.ID)
	}
	
	confidence := 1.0 - bestStore.usageRatio // 使用率越低，置信度越高
	
	return &ShardRecommendation{
		TimelineKey:      timelineKey,
		RecommendedStore: bestStore.storeInfo.ID,
		Reason:           fmt.Sprintf("Lowest usage ratio: %.2f, available: %d bytes", bestStore.usageRatio, bestStore.availableSize),
		Confidence:       confidence,
		Alternatives:     alternatives,
	}, nil
}

// calculateLoadFactor 计算负载因子
func (tsm *TimelineShardManager) calculateLoadFactor(loadInfo *StoreLoadInfo, additionalSize int64) float64 {
	// 综合考虑Timeline数量和数据大小
	timelineRatio := float64(loadInfo.TimelineCount) / float64(tsm.policy.MaxTimelinePerStore)
	sizeRatio := float64(loadInfo.TotalSize+additionalSize) / float64(tsm.policy.MaxSizePerStore)
	
	// 取较大值作为负载因子
	return math.Max(timelineRatio, sizeRatio)
}

// storeLoadData 存储负载数据（内部使用）
type storeLoadData struct {
	storeInfo  *StoreInfo
	loadInfo   *StoreLoadInfo
	loadFactor float64
	timelines  []string
}

// GetRebalanceRecommendations 获取重平衡推荐
func (tsm *TimelineShardManager) GetRebalanceRecommendations(ctx context.Context) ([]*RebalanceRecommendation, error) {
	tsm.mu.RLock()
	defer tsm.mu.RUnlock()
	
	if !tsm.policy.AutoRebalance {
		return nil, nil // 未启用自动重平衡
	}
	
	// 获取所有Store的负载信息
	stores, err := tsm.storeRegistry.ListStores(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list stores: %w", err)
	}
	
	storeLoads := make([]*storeLoadData, 0, len(stores))
	for _, store := range stores {
		if store.Status != "active" {
			continue
		}
		
		loadInfo, err := tsm.globalIndex.GetStoreLoad(ctx, store.ID)
		if err != nil {
			continue
		}
		
		timelines, err := tsm.globalIndex.ListTimelinesByStore(ctx, store.ID)
		if err != nil {
			continue
		}
		
		loadFactor := tsm.calculateLoadFactor(loadInfo, 0)
		
		storeLoads = append(storeLoads, &storeLoadData{
			storeInfo:  store,
			loadInfo:   loadInfo,
			loadFactor: loadFactor,
			timelines:  timelines,
		})
	}
	
	if len(storeLoads) < 2 {
		return nil, nil // 至少需要2个Store才能重平衡
	}
	
	// 按负载因子排序
	sort.Slice(storeLoads, func(i, j int) bool {
		return storeLoads[i].loadFactor > storeLoads[j].loadFactor
	})
	
	var recommendations []*RebalanceRecommendation
	
	// 检查是否需要重平衡
	highestLoad := storeLoads[0].loadFactor
	lowestLoad := storeLoads[len(storeLoads)-1].loadFactor
	
	if highestLoad-lowestLoad < 0.2 {
		return nil, nil // 负载差异不大，无需重平衡
	}
	
	// 从高负载Store迁移Timeline到低负载Store
	for i := 0; i < len(storeLoads)/2; i++ {
		highLoadStore := storeLoads[i]
		lowLoadStore := storeLoads[len(storeLoads)-1-i]
		
		if highLoadStore.loadFactor <= tsm.policy.LoadBalanceThreshold {
			break // 高负载Store已经在阈值内
		}
		
		// 选择要迁移的Timeline（选择较小的）
		for _, timelineKey := range highLoadStore.timelines {
			location, err := tsm.globalIndex.GetTimelineLocation(ctx, timelineKey)
			if err != nil {
				continue
			}
			
			// 估算迁移后的负载变化
			expectedGain := tsm.calculateMigrationGain(highLoadStore, lowLoadStore, location.TotalSize)
			
			if expectedGain > 0.1 { // 只有收益足够大才推荐迁移
				recommendations = append(recommendations, &RebalanceRecommendation{
					TimelineKey:  timelineKey,
					FromStore:    highLoadStore.storeInfo.ID,
					ToStore:      lowLoadStore.storeInfo.ID,
					Reason:       fmt.Sprintf("Load balancing: %.2f -> %.2f", highLoadStore.loadFactor, lowLoadStore.loadFactor),
					Priority:     int(expectedGain * 10),
					ExpectedGain: expectedGain,
				})
				
				// 更新负载信息用于下次计算
				highLoadStore.loadInfo.TotalSize -= location.TotalSize
				lowLoadStore.loadInfo.TotalSize += location.TotalSize
				highLoadStore.loadFactor = tsm.calculateLoadFactor(highLoadStore.loadInfo, 0)
				lowLoadStore.loadFactor = tsm.calculateLoadFactor(lowLoadStore.loadInfo, 0)
				
				break // 每次只迁移一个Timeline
			}
		}
	}
	
	// 按优先级排序
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority > recommendations[j].Priority
	})
	
	return recommendations, nil
}

// calculateMigrationGain 计算迁移收益
func (tsm *TimelineShardManager) calculateMigrationGain(highLoadStore, lowLoadStore *storeLoadData, timelineSize int64) float64 {
	// 计算迁移前的负载差异
	beforeGap := highLoadStore.loadFactor - lowLoadStore.loadFactor
	
	// 计算迁移后的负载
	newHighLoad := tsm.calculateLoadFactor(&StoreLoadInfo{
		TotalSize:     highLoadStore.loadInfo.TotalSize - timelineSize,
		TimelineCount: highLoadStore.loadInfo.TimelineCount - 1,
	}, 0)
	
	newLowLoad := tsm.calculateLoadFactor(&StoreLoadInfo{
		TotalSize:     lowLoadStore.loadInfo.TotalSize + timelineSize,
		TimelineCount: lowLoadStore.loadInfo.TimelineCount + 1,
	}, 0)
	
	// 计算迁移后的负载差异
	afterGap := math.Abs(newHighLoad - newLowLoad)
	
	// 收益 = 负载差异的减少程度
	return beforeGap - afterGap
}

// UpdateShardPolicy 更新分片策略
func (tsm *TimelineShardManager) UpdateShardPolicy(policy *ShardPolicy) error {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	
	tsm.policy = policy
	return nil
}

// GetShardPolicy 获取当前分片策略
func (tsm *TimelineShardManager) GetShardPolicy() *ShardPolicy {
	tsm.mu.RLock()
	defer tsm.mu.RUnlock()
	
	// 返回副本
	policyCopy := *tsm.policy
	return &policyCopy
}

// StartAutoRebalance 启动自动重平衡
func (tsm *TimelineShardManager) StartAutoRebalance(ctx context.Context) error {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	
	if tsm.autoRebalanceRunning {
		return fmt.Errorf("auto rebalance is already running")
	}
	
	tsm.autoRebalanceStop = make(chan struct{})
	tsm.autoRebalanceRunning = true
	
	go tsm.autoRebalanceLoop(ctx)
	
	return nil
}

// StopAutoRebalance 停止自动重平衡
func (tsm *TimelineShardManager) StopAutoRebalance() error {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	
	if !tsm.autoRebalanceRunning {
		return fmt.Errorf("auto rebalance is not running")
	}
	
	close(tsm.autoRebalanceStop)
	tsm.autoRebalanceRunning = false
	
	return nil
}

// autoRebalanceLoop 自动重平衡循环
func (tsm *TimelineShardManager) autoRebalanceLoop(ctx context.Context) {
	ticker := time.NewTicker(tsm.policy.RebalanceInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-tsm.autoRebalanceStop:
			return
		case <-ticker.C:
			tsm.performAutoRebalance(ctx)
		}
	}
}

// performAutoRebalance 执行自动重平衡
func (tsm *TimelineShardManager) performAutoRebalance(ctx context.Context) {
	recommendations, err := tsm.GetRebalanceRecommendations(ctx)
	if err != nil {
		fmt.Printf("Failed to get rebalance recommendations: %v\n", err)
		return
	}
	
	if len(recommendations) == 0 {
		return // 无需重平衡
	}
	
	// 执行优先级最高的重平衡
	recommendation := recommendations[0]
	
	_, err = tsm.migrationManager.StartMigration(ctx, recommendation.TimelineKey, recommendation.ToStore)
	if err != nil {
		fmt.Printf("Failed to start migration for %s: %v\n", recommendation.TimelineKey, err)
		return
	}
	
	// 更新统计信息
	tsm.mu.Lock()
	tsm.stats.RebalanceCount++
	now := time.Now()
	tsm.stats.LastRebalance = &now
	tsm.mu.Unlock()
	
	fmt.Printf("Started auto rebalance: %s from %s to %s\n", 
		recommendation.TimelineKey, recommendation.FromStore, recommendation.ToStore)
}

// GetShardStats 获取分片统计信息
func (tsm *TimelineShardManager) GetShardStats(ctx context.Context) (*ShardStats, error) {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	
	// 获取所有Store信息
	stores, err := tsm.storeRegistry.ListStores(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list stores: %w", err)
	}
	
	stats := &ShardStats{
		TotalStores:    len(stores),
		StoreStats:     make(map[string]*ShardStoreStats),
		LastRebalance:  tsm.stats.LastRebalance,
		RebalanceCount: tsm.stats.RebalanceCount,
	}
	
	var totalTimelines int
	var totalSize int64
	var loadFactors []float64
	
	for _, store := range stores {
		loadInfo, err := tsm.globalIndex.GetStoreLoad(ctx, store.ID)
		if err != nil {
			continue
		}
		
		loadFactor := tsm.calculateLoadFactor(loadInfo, 0)
		healthScore := 1.0
		if store.Status != "active" {
			healthScore = 0.0
		}
		
		stats.StoreStats[store.ID] = &ShardStoreStats{
			StoreID:       store.ID,
			TimelineCount: loadInfo.TimelineCount,
			TotalSize:     loadInfo.TotalSize,
			LoadFactor:    loadFactor,
			HealthScore:   healthScore,
			LastUpdate:    loadInfo.LastUpdate,
		}
		
		totalTimelines += loadInfo.TimelineCount
		totalSize += loadInfo.TotalSize
		loadFactors = append(loadFactors, loadFactor)
	}
	
	stats.TotalTimelines = totalTimelines
	stats.TotalSize = totalSize
	
	// 计算平均负载和方差
	if len(loadFactors) > 0 {
		var sum float64
		for _, lf := range loadFactors {
			sum += lf
		}
		stats.AverageLoad = sum / float64(len(loadFactors))
		
		// 计算方差
		var variance float64
		for _, lf := range loadFactors {
			variance += math.Pow(lf-stats.AverageLoad, 2)
		}
		stats.LoadVariance = variance / float64(len(loadFactors))
	}
	
	return stats, nil
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}