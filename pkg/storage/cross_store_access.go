package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CrossStoreAccessor 跨Store数据访问接口
type CrossStoreAccessor interface {
	// Timeline操作
	GetTimeline(ctx context.Context, timelineKey string) (*Timeline, error)
	CreateTimeline(ctx context.Context, timelineKey, timelineType string) error
	DeleteTimeline(ctx context.Context, timelineKey string) error
	
	// 消息操作
	AddMessage(ctx context.Context, timelineKey string, senderID uint32, data []byte, userIDs []string) error
	GetMessages(ctx context.Context, timelineKey string, startTime, endTime int64, limit int) ([]*Message, error)
	
	// Store状态
	GetStoreStats(ctx context.Context, storeID string) (*StoreStats, error)
	HealthCheck(ctx context.Context, storeID string) error
	
	// 数据迁移
	MigrateTimeline(ctx context.Context, timelineKey, targetStoreID string) error
}

// DistributedStoreAccessor 分布式Store访问器实现
type DistributedStoreAccessor struct {
	localStore    *Store
	rpcClientPool *StoreRPCClientPool
	globalIndex   GlobalIndexManager
	router        TimelineRouter
	storeRegistry StoreRegistry
	cacheManager  *CrossStoreCacheManager
	mu            sync.RWMutex
}

// StoreStats Store统计信息
type StoreStats struct {
	StoreID        string    `json:"store_id"`
	TimelineCount  int       `json:"timeline_count"`
	MessageCount   int64     `json:"message_count"`
	StorageSize    int64     `json:"storage_size"`
	LastHeartbeat  time.Time `json:"last_heartbeat"`
	Status         string    `json:"status"`
}

// CrossStoreCacheManager 跨Store缓存管理器
type CrossStoreCacheManager struct {
	timelineCache *TimelineCache
	messageCache  *MessageCache
	blockCache    *BlockCache
	mu            sync.RWMutex
}

// TimelineCache Timeline缓存
type TimelineCache struct {
	cache map[string]*Timeline
	mu    sync.RWMutex
}

// MessageCache 消息缓存
type MessageCache struct {
	cache map[string][]*Message
	mu    sync.RWMutex
}

// BlockCache 块缓存
type BlockCache struct {
	cache map[string]*TimelineBlock
	mu    sync.RWMutex
}

// NewDistributedStoreAccessor 创建分布式Store访问器
func NewDistributedStoreAccessor(
	localStore *Store,
	rpcClientPool *StoreRPCClientPool,
	globalIndex GlobalIndexManager,
	router TimelineRouter,
	storeRegistry StoreRegistry,
) *DistributedStoreAccessor {
	return &DistributedStoreAccessor{
		localStore:    localStore,
		rpcClientPool: rpcClientPool,
		globalIndex:   globalIndex,
		router:        router,
		storeRegistry: storeRegistry,
		cacheManager:  NewCrossStoreCacheManager(),
	}
}

// NewCrossStoreCacheManager 创建跨Store缓存管理器
func NewCrossStoreCacheManager() *CrossStoreCacheManager {
	return &CrossStoreCacheManager{
		timelineCache: &TimelineCache{cache: make(map[string]*Timeline)},
		messageCache:  &MessageCache{cache: make(map[string][]*Message)},
		blockCache:    &BlockCache{cache: make(map[string]*TimelineBlock)},
	}
}

// GetTimeline 获取Timeline
func (d *DistributedStoreAccessor) GetTimeline(ctx context.Context, timelineKey string) (*Timeline, error) {
	// 1. 检查缓存
	if timeline := d.cacheManager.GetTimeline(timelineKey); timeline != nil {
		return timeline, nil
	}
	
	// 2. 查找Timeline位置
	location, err := d.globalIndex.GetTimelineLocation(ctx, timelineKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline location: %w", err)
	}
	
	// 3. 确定主Store（从第一个Block获取）
	var primaryStoreID string
	if len(location.Blocks) > 0 {
		primaryStoreID = location.Blocks[0].StoreID
	} else {
		// 如果没有blocks，尝试从本地Store获取
		if timeline := d.localStore.GetOrCreateConvTimeline(timelineKey); timeline != nil {
			d.cacheManager.SetTimeline(timelineKey, timeline)
			return timeline, nil
		}
		if timeline := d.localStore.GetOrCreateUserTimeline(timelineKey); timeline != nil {
			d.cacheManager.SetTimeline(timelineKey, timeline)
			return timeline, nil
		}
		return nil, fmt.Errorf("timeline not found: %s", timelineKey)
	}
	
	// 4. 如果在本地Store
	if primaryStoreID == d.localStore.StoreID {
		// 尝试获取会话Timeline
		if timeline := d.localStore.GetOrCreateConvTimeline(timelineKey); timeline != nil {
			d.cacheManager.SetTimeline(timelineKey, timeline)
			return timeline, nil
		}
		// 尝试获取用户Timeline
		if timeline := d.localStore.GetOrCreateUserTimeline(timelineKey); timeline != nil {
			d.cacheManager.SetTimeline(timelineKey, timeline)
			return timeline, nil
		}
		return nil, fmt.Errorf("timeline not found locally: %s", timelineKey)
	}
	
	// 5. 远程访问
	timeline, err := d.getRemoteTimeline(ctx, primaryStoreID, timelineKey)
	if err != nil {
		return nil, err
	}
	
	// 6. 缓存结果
	if timeline != nil {
		d.cacheManager.SetTimeline(timelineKey, timeline)
	}
	
	return timeline, nil
}

// CreateTimeline 创建Timeline
func (d *DistributedStoreAccessor) CreateTimeline(ctx context.Context, timelineKey, timelineType string) error {
	// 1. 路由到目标Store
	targetStoreID, err := d.router.RouteTimeline(timelineKey)
	if err != nil {
		return fmt.Errorf("failed to route timeline: %w", err)
	}
	
	// 2. 如果在本地Store
	if targetStoreID == d.localStore.StoreID {
		// 根据Timeline类型创建
		if timelineType == "conv" {
			d.localStore.GetOrCreateConvTimeline(timelineKey)
		} else if timelineType == "user" {
			d.localStore.GetOrCreateUserTimeline(timelineKey)
		}
		// 更新全局索引
		return d.globalIndex.AddIndex(ctx, &GlobalStoreIndex{
			TimelineKey: timelineKey,
			StoreID:     targetStoreID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}
	
	// 3. 远程创建
	err = d.createRemoteTimeline(ctx, targetStoreID, timelineKey, timelineType)
	if err != nil {
		return err
	}
	
	// 4. 更新全局索引
	return d.globalIndex.AddIndex(ctx, &GlobalStoreIndex{
		TimelineKey: timelineKey,
		StoreID:     targetStoreID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
}

// DeleteTimeline 删除Timeline
func (d *DistributedStoreAccessor) DeleteTimeline(ctx context.Context, timelineKey string) error {
	// 1. 查找Timeline位置
	location, err := d.globalIndex.GetTimelineLocation(ctx, timelineKey)
	if err != nil {
		return fmt.Errorf("failed to get timeline location: %w", err)
	}
	
	// 2. 确定主Store（从第一个Block获取）
	var primaryStoreID string
	if len(location.Blocks) > 0 {
		primaryStoreID = location.Blocks[0].StoreID
	} else {
		return fmt.Errorf("timeline has no blocks")
	}
	
	// 3. 如果在本地Store
	if primaryStoreID == d.localStore.StoreID {
		// Store结构体没有直接的DeleteTimeline方法，这里只是标记删除
		// 实际删除逻辑需要在Store层面实现
	} else {
		// 4. 远程删除
		err = d.deleteRemoteTimeline(ctx, primaryStoreID, timelineKey)
		if err != nil {
			return err
		}
	}
	
	// 5. 从全局索引中移除
	err = d.globalIndex.RemoveIndex(ctx, timelineKey, primaryStoreID)
	if err != nil {
		return fmt.Errorf("failed to remove from global index: %w", err)
	}
	
	// 6. 清除缓存
	d.cacheManager.RemoveTimeline(timelineKey)
	
	return nil
}

// AddMessage 添加消息到Timeline
func (d *DistributedStoreAccessor) AddMessage(ctx context.Context, timelineKey string, senderID uint32, data []byte, userIDs []string) error {
	// 1. 查找Timeline位置
	location, err := d.globalIndex.GetTimelineLocation(ctx, timelineKey)
	if err != nil {
		return fmt.Errorf("failed to get timeline location: %w", err)
	}
	
	// 2. 确定主Store（从第一个Block获取）
	var primaryStoreID string
	if len(location.Blocks) > 0 {
		primaryStoreID = location.Blocks[0].StoreID
	} else {
		return fmt.Errorf("timeline has no blocks")
	}
	
	// 3. 如果在本地Store
	if primaryStoreID == d.localStore.StoreID {
		return d.localStore.AddMessage(timelineKey, senderID, data, userIDs)
	}
	
	// 4. 远程添加
	return d.addRemoteMessage(ctx, primaryStoreID, timelineKey, senderID, data, userIDs)
}

// GetMessages 获取消息列表
func (d *DistributedStoreAccessor) GetMessages(ctx context.Context, timelineKey string, startTime, endTime int64, limit int) ([]*Message, error) {
	// 1. 检查缓存
	cacheKey := fmt.Sprintf("%s:%d:%d:%d", timelineKey, startTime, endTime, limit)
	if messages := d.cacheManager.GetMessages(cacheKey); messages != nil {
		return messages, nil
	}
	
	// 2. 查找Timeline位置
	location, err := d.globalIndex.GetTimelineLocation(ctx, timelineKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline location: %w", err)
	}
	
	var messages []*Message
	
	// 3. 确定主Store（从第一个Block获取）
	var primaryStoreID string
	if len(location.Blocks) > 0 {
		primaryStoreID = location.Blocks[0].StoreID
	} else {
		return nil, fmt.Errorf("timeline has no blocks")
	}
	
	// 4. 如果在本地Store
	if primaryStoreID == d.localStore.StoreID {
		// 由于Store没有GetMessages方法，这里需要通过Timeline获取
		timeline, err := d.GetTimeline(ctx, timelineKey)
		if err != nil {
			return nil, err
		}
		// 从Timeline的blocks中获取消息
		for _, block := range timeline.Blocks {
			for _, msg := range block.Messages {
				msgTime := msg.CreateTime.Unix()
				if msgTime >= startTime && msgTime <= endTime {
					messages = append(messages, msg)
					if len(messages) >= limit {
						break
					}
				}
			}
			if len(messages) >= limit {
				break
			}
		}
	} else {
		// 5. 远程获取
		messages, err = d.getRemoteMessages(ctx, primaryStoreID, timelineKey, startTime, endTime, limit)
		if err != nil {
			return nil, err
		}
	}
	
	// 6. 缓存结果
	if messages != nil {
		d.cacheManager.SetMessages(cacheKey, messages)
	}
	
	return messages, nil
}

// GetStoreStats 获取Store统计信息
func (d *DistributedStoreAccessor) GetStoreStats(ctx context.Context, storeID string) (*StoreStats, error) {
	if storeID == d.localStore.StoreID {
		// 本地Store统计
		return &StoreStats{
			StoreID:       d.localStore.StoreID,
			TimelineCount: len(d.localStore.ConvTimelines) + len(d.localStore.UserTimelines),
			StorageSize:   0, // 需要计算实际存储大小
			LastHeartbeat: time.Now(),
			Status:        "healthy",
		}, nil
	}
	
	// 远程Store统计
	return d.getRemoteStoreStats(ctx, storeID)
}

// HealthCheck 健康检查
func (d *DistributedStoreAccessor) HealthCheck(ctx context.Context, storeID string) error {
	if storeID == d.localStore.StoreID {
		return nil // 本地Store总是健康的
	}
	
	// 远程健康检查
	return d.remoteHealthCheck(ctx, storeID)
}

// MigrateTimeline 迁移Timeline
func (d *DistributedStoreAccessor) MigrateTimeline(ctx context.Context, timelineKey, targetStoreID string) error {
	// 1. 查找当前Timeline位置
	location, err := d.globalIndex.GetTimelineLocation(ctx, timelineKey)
	if err != nil {
		return fmt.Errorf("failed to get timeline location: %w", err)
	}
	
	// 2. 确定当前主Store
	var currentStoreID string
	if len(location.Blocks) > 0 {
		currentStoreID = location.Blocks[0].StoreID
	} else {
		return fmt.Errorf("timeline has no blocks")
	}
	
	// 3. 如果已经在目标Store，无需迁移
	if currentStoreID == targetStoreID {
		return nil
	}
	
	// 4. 执行迁移
	err = d.executeMigration(ctx, timelineKey, currentStoreID, targetStoreID)
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}
	
	// 5. 更新全局索引
	err = d.globalIndex.UpdateIndex(ctx, &GlobalStoreIndex{
		TimelineKey: timelineKey,
		StoreID:     targetStoreID,
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to update global index: %w", err)
	}
	
	// 6. 清除缓存
	d.cacheManager.RemoveTimeline(timelineKey)
	
	return nil
}

// 缓存管理方法
func (c *CrossStoreCacheManager) GetTimeline(key string) *Timeline {
	c.timelineCache.mu.RLock()
	defer c.timelineCache.mu.RUnlock()
	return c.timelineCache.cache[key]
}

func (c *CrossStoreCacheManager) SetTimeline(key string, timeline *Timeline) {
	c.timelineCache.mu.Lock()
	defer c.timelineCache.mu.Unlock()
	c.timelineCache.cache[key] = timeline
}

func (c *CrossStoreCacheManager) RemoveTimeline(key string) {
	c.timelineCache.mu.Lock()
	defer c.timelineCache.mu.Unlock()
	delete(c.timelineCache.cache, key)
}

func (c *CrossStoreCacheManager) GetMessages(key string) []*Message {
	c.messageCache.mu.RLock()
	defer c.messageCache.mu.RUnlock()
	return c.messageCache.cache[key]
}

func (c *CrossStoreCacheManager) SetMessages(key string, messages []*Message) {
	c.messageCache.mu.Lock()
	defer c.messageCache.mu.Unlock()
	c.messageCache.cache[key] = messages
}

// 远程访问辅助方法（简化实现）
func (d *DistributedStoreAccessor) getRemoteTimeline(ctx context.Context, storeID, timelineKey string) (*Timeline, error) {
	// 这里需要实现RPC调用逻辑
	// 暂时返回错误，实际实现需要根据RPC接口定义
	return nil, fmt.Errorf("remote timeline access not implemented")
}

func (d *DistributedStoreAccessor) createRemoteTimeline(ctx context.Context, storeID, timelineKey, timelineType string) error {
	// 这里需要实现RPC调用逻辑
	return fmt.Errorf("remote timeline creation not implemented")
}

func (d *DistributedStoreAccessor) deleteRemoteTimeline(ctx context.Context, storeID, timelineKey string) error {
	// 这里需要实现RPC调用逻辑
	return fmt.Errorf("remote timeline deletion not implemented")
}

func (d *DistributedStoreAccessor) addRemoteMessage(ctx context.Context, storeID, timelineKey string, senderID uint32, data []byte, userIDs []string) error {
	// 这里需要实现RPC调用逻辑
	return fmt.Errorf("remote message addition not implemented")
}

func (d *DistributedStoreAccessor) getRemoteMessages(ctx context.Context, storeID, timelineKey string, startTime, endTime int64, limit int) ([]*Message, error) {
	// 这里需要实现RPC调用逻辑
	return nil, fmt.Errorf("remote message retrieval not implemented")
}

func (d *DistributedStoreAccessor) getRemoteStoreStats(ctx context.Context, storeID string) (*StoreStats, error) {
	// 这里需要实现RPC调用逻辑
	return nil, fmt.Errorf("remote store stats not implemented")
}

func (d *DistributedStoreAccessor) remoteHealthCheck(ctx context.Context, storeID string) error {
	// 这里需要实现RPC调用逻辑
	return fmt.Errorf("remote health check not implemented")
}

func (d *DistributedStoreAccessor) executeMigration(ctx context.Context, timelineKey, sourceStoreID, targetStoreID string) error {
	// 这里需要实现数据迁移逻辑
	return fmt.Errorf("timeline migration not implemented")
}