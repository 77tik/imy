package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// GlobalStoreIndex 全局Store索引条目
type GlobalStoreIndex struct {
	TimelineKey string    `json:"timelineKey"` // Timeline键
	StoreID     string    `json:"storeId"`     // Store ID
	BlockID     string    `json:"blockId"`     // Block ID
	Offset      int64     `json:"offset"`      // 在Store中的偏移量
	Size        int64     `json:"size"`        // 数据大小
	CreatedAt   time.Time `json:"createdAt"`   // 创建时间
	UpdatedAt   time.Time `json:"updatedAt"`   // 更新时间
}

// TimelineLocation Timeline位置信息
type TimelineLocation struct {
	TimelineKey string               `json:"timelineKey"`
	Blocks      []*GlobalStoreIndex  `json:"blocks"`      // 所有块的索引
	StoreMap    map[string][]*GlobalStoreIndex `json:"storeMap"`    // 按Store分组的块索引
	TotalSize   int64                `json:"totalSize"`   // 总大小
	BlockCount  int                  `json:"blockCount"`  // 块数量
	LastUpdate  time.Time            `json:"lastUpdate"`  // 最后更新时间
}

// GlobalIndexManager 全局索引管理器接口
type GlobalIndexManager interface {
	// AddIndex 添加索引条目
	AddIndex(ctx context.Context, index *GlobalStoreIndex) error
	// RemoveIndex 移除索引条目
	RemoveIndex(ctx context.Context, timelineKey, blockID string) error
	// GetTimelineLocation 获取Timeline位置信息
	GetTimelineLocation(ctx context.Context, timelineKey string) (*TimelineLocation, error)
	// ListTimelinesByStore 获取指定Store上的所有Timeline
	ListTimelinesByStore(ctx context.Context, storeID string) ([]string, error)
	// UpdateIndex 更新索引条目
	UpdateIndex(ctx context.Context, index *GlobalStoreIndex) error
	// MigrateTimeline 迁移Timeline到新Store
	MigrateTimeline(ctx context.Context, timelineKey, fromStoreID, toStoreID string) error
	// GetStoreLoad 获取Store负载信息
	GetStoreLoad(ctx context.Context, storeID string) (*StoreLoadInfo, error)
	// Watch 监听索引变化
	Watch(ctx context.Context, timelineKey string) (<-chan IndexEvent, error)
}

// StoreLoadInfo Store负载信息
type StoreLoadInfo struct {
	StoreID       string `json:"storeId"`
	TimelineCount int    `json:"timelineCount"` // Timeline数量
	BlockCount    int    `json:"blockCount"`    // 块数量
	TotalSize     int64  `json:"totalSize"`     // 总数据大小
	LastUpdate    time.Time `json:"lastUpdate"`
}

// IndexEvent 索引事件
type IndexEvent struct {
	Type        string             `json:"type"`        // 事件类型: add, remove, update, migrate
	TimelineKey string             `json:"timelineKey"`
	Index       *GlobalStoreIndex  `json:"index"`
	OldStoreID  string             `json:"oldStoreId,omitempty"` // 迁移时的原Store ID
}

// InMemoryGlobalIndex 内存实现的全局索引管理器
type InMemoryGlobalIndex struct {
	mu           sync.RWMutex
	timelineIndex map[string]*TimelineLocation           // Timeline -> Location
	storeIndex    map[string]map[string]*GlobalStoreIndex // StoreID -> TimelineKey -> Index
	loadInfo      map[string]*StoreLoadInfo               // StoreID -> LoadInfo
	watchers      map[string][]chan IndexEvent            // TimelineKey -> Watchers
}

// NewInMemoryGlobalIndex 创建内存全局索引管理器
func NewInMemoryGlobalIndex() *InMemoryGlobalIndex {
	return &InMemoryGlobalIndex{
		timelineIndex: make(map[string]*TimelineLocation),
		storeIndex:    make(map[string]map[string]*GlobalStoreIndex),
		loadInfo:      make(map[string]*StoreLoadInfo),
		watchers:      make(map[string][]chan IndexEvent),
	}
}

// AddIndex 添加索引条目
func (g *InMemoryGlobalIndex) AddIndex(ctx context.Context, index *GlobalStoreIndex) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	index.UpdatedAt = time.Now()
	
	// 更新Timeline索引
	location, exists := g.timelineIndex[index.TimelineKey]
	if !exists {
		location = &TimelineLocation{
			TimelineKey: index.TimelineKey,
			Blocks:      make([]*GlobalStoreIndex, 0),
			StoreMap:    make(map[string][]*GlobalStoreIndex),
			TotalSize:   0,
			BlockCount:  0,
			LastUpdate:  time.Now(),
		}
		g.timelineIndex[index.TimelineKey] = location
	}
	
	// 添加到blocks列表
	location.Blocks = append(location.Blocks, index)
	
	// 添加到storeMap
	if location.StoreMap[index.StoreID] == nil {
		location.StoreMap[index.StoreID] = make([]*GlobalStoreIndex, 0)
	}
	location.StoreMap[index.StoreID] = append(location.StoreMap[index.StoreID], index)
	
	// 更新统计信息
	location.TotalSize += index.Size
	location.BlockCount++
	location.LastUpdate = time.Now()
	
	// 更新Store索引
	if g.storeIndex[index.StoreID] == nil {
		g.storeIndex[index.StoreID] = make(map[string]*GlobalStoreIndex)
	}
	g.storeIndex[index.StoreID][index.TimelineKey+":"+index.BlockID] = index
	
	// 更新Store负载信息
	g.updateStoreLoad(index.StoreID)
	
	// 通知监听者
	g.notifyWatchers(index.TimelineKey, IndexEvent{
		Type:        "add",
		TimelineKey: index.TimelineKey,
		Index:       index,
	})
	
	return nil
}

// RemoveIndex 移除索引条目
func (g *InMemoryGlobalIndex) RemoveIndex(ctx context.Context, timelineKey, blockID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	location, exists := g.timelineIndex[timelineKey]
	if !exists {
		return fmt.Errorf("timeline %s not found", timelineKey)
	}
	
	// 查找并移除索引
	var removedIndex *GlobalStoreIndex
	for i, index := range location.Blocks {
		if index.BlockID == blockID {
			removedIndex = index
			// 从blocks列表移除
			location.Blocks = append(location.Blocks[:i], location.Blocks[i+1:]...)
			
			// 从storeMap移除
			storeBlocks := location.StoreMap[index.StoreID]
			for j, storeIndex := range storeBlocks {
				if storeIndex.BlockID == blockID {
					location.StoreMap[index.StoreID] = append(storeBlocks[:j], storeBlocks[j+1:]...)
					break
				}
			}
			
			// 更新统计信息
			location.TotalSize -= index.Size
			location.BlockCount--
			location.LastUpdate = time.Now()
			
			// 从Store索引移除
			delete(g.storeIndex[index.StoreID], timelineKey+":"+blockID)
			
			// 更新Store负载信息
			g.updateStoreLoad(index.StoreID)
			
			break
		}
	}
	
	if removedIndex == nil {
		return fmt.Errorf("block %s not found in timeline %s", blockID, timelineKey)
	}
	
	// 如果Timeline没有块了，移除整个Timeline
	if location.BlockCount == 0 {
		delete(g.timelineIndex, timelineKey)
	}
	
	// 通知监听者
	g.notifyWatchers(timelineKey, IndexEvent{
		Type:        "remove",
		TimelineKey: timelineKey,
		Index:       removedIndex,
	})
	
	return nil
}

// GetTimelineLocation 获取Timeline位置信息
func (g *InMemoryGlobalIndex) GetTimelineLocation(ctx context.Context, timelineKey string) (*TimelineLocation, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	location, exists := g.timelineIndex[timelineKey]
	if !exists {
		return nil, fmt.Errorf("timeline %s not found", timelineKey)
	}
	
	return location, nil
}

// ListTimelinesByStore 获取指定Store上的所有Timeline
func (g *InMemoryGlobalIndex) ListTimelinesByStore(ctx context.Context, storeID string) ([]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	storeIndexes, exists := g.storeIndex[storeID]
	if !exists {
		return []string{}, nil
	}
	
	timelineSet := make(map[string]bool)
	for key := range storeIndexes {
		// 从 "timelineKey:blockID" 格式中提取timelineKey
		parts := splitTimelineKey(key)
		if len(parts) > 0 {
			timelineSet[parts[0]] = true
		}
	}
	
	timelines := make([]string, 0, len(timelineSet))
	for timeline := range timelineSet {
		timelines = append(timelines, timeline)
	}
	
	return timelines, nil
}

// UpdateIndex 更新索引条目
func (g *InMemoryGlobalIndex) UpdateIndex(ctx context.Context, index *GlobalStoreIndex) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	index.UpdatedAt = time.Now()
	
	// 查找并更新索引
	location, exists := g.timelineIndex[index.TimelineKey]
	if !exists {
		return fmt.Errorf("timeline %s not found", index.TimelineKey)
	}
	
	for i, existingIndex := range location.Blocks {
		if existingIndex.BlockID == index.BlockID {
			// 更新统计信息
			location.TotalSize = location.TotalSize - existingIndex.Size + index.Size
			location.LastUpdate = time.Now()
			
			// 更新索引
			location.Blocks[i] = index
			
			// 更新storeMap
			for j, storeIndex := range location.StoreMap[index.StoreID] {
				if storeIndex.BlockID == index.BlockID {
					location.StoreMap[index.StoreID][j] = index
					break
				}
			}
			
			// 更新Store索引
			g.storeIndex[index.StoreID][index.TimelineKey+":"+index.BlockID] = index
			
			// 更新Store负载信息
			g.updateStoreLoad(index.StoreID)
			
			// 通知监听者
			g.notifyWatchers(index.TimelineKey, IndexEvent{
				Type:        "update",
				TimelineKey: index.TimelineKey,
				Index:       index,
			})
			
			return nil
		}
	}
	
	return fmt.Errorf("block %s not found in timeline %s", index.BlockID, index.TimelineKey)
}

// MigrateTimeline 迁移Timeline到新Store
func (g *InMemoryGlobalIndex) MigrateTimeline(ctx context.Context, timelineKey, fromStoreID, toStoreID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	location, exists := g.timelineIndex[timelineKey]
	if !exists {
		return fmt.Errorf("timeline %s not found", timelineKey)
	}
	
	// 更新所有相关的索引条目
	for _, index := range location.Blocks {
		if index.StoreID == fromStoreID {
			// 从原Store索引移除
			delete(g.storeIndex[fromStoreID], timelineKey+":"+index.BlockID)
			
			// 更新StoreID
			index.StoreID = toStoreID
			index.UpdatedAt = time.Now()
			
			// 添加到新Store索引
			if g.storeIndex[toStoreID] == nil {
				g.storeIndex[toStoreID] = make(map[string]*GlobalStoreIndex)
			}
			g.storeIndex[toStoreID][timelineKey+":"+index.BlockID] = index
		}
	}
	
	// 更新storeMap
	blocks := location.StoreMap[fromStoreID]
	delete(location.StoreMap, fromStoreID)
	location.StoreMap[toStoreID] = blocks
	location.LastUpdate = time.Now()
	
	// 更新Store负载信息
	g.updateStoreLoad(fromStoreID)
	g.updateStoreLoad(toStoreID)
	
	// 通知监听者
	g.notifyWatchers(timelineKey, IndexEvent{
		Type:        "migrate",
		TimelineKey: timelineKey,
		OldStoreID:  fromStoreID,
	})
	
	return nil
}

// GetStoreLoad 获取Store负载信息
func (g *InMemoryGlobalIndex) GetStoreLoad(ctx context.Context, storeID string) (*StoreLoadInfo, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	loadInfo, exists := g.loadInfo[storeID]
	if !exists {
		return &StoreLoadInfo{
			StoreID:       storeID,
			TimelineCount: 0,
			BlockCount:    0,
			TotalSize:     0,
			LastUpdate:    time.Now(),
		}, nil
	}
	
	return loadInfo, nil
}

// Watch 监听索引变化
func (g *InMemoryGlobalIndex) Watch(ctx context.Context, timelineKey string) (<-chan IndexEvent, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	ch := make(chan IndexEvent, 100)
	if g.watchers[timelineKey] == nil {
		g.watchers[timelineKey] = make([]chan IndexEvent, 0)
	}
	g.watchers[timelineKey] = append(g.watchers[timelineKey], ch)
	
	// 当context取消时，清理watcher
	go func() {
		<-ctx.Done()
		g.mu.Lock()
		defer g.mu.Unlock()
		
		// 移除watcher
		watchers := g.watchers[timelineKey]
		for i, watcher := range watchers {
			if watcher == ch {
				g.watchers[timelineKey] = append(watchers[:i], watchers[i+1:]...)
				close(ch)
				break
			}
		}
		
		// 如果没有监听者了，清理map条目
		if len(g.watchers[timelineKey]) == 0 {
			delete(g.watchers, timelineKey)
		}
	}()
	
	return ch, nil
}

// updateStoreLoad 更新Store负载信息
func (g *InMemoryGlobalIndex) updateStoreLoad(storeID string) {
	storeIndexes, exists := g.storeIndex[storeID]
	if !exists {
		g.loadInfo[storeID] = &StoreLoadInfo{
			StoreID:       storeID,
			TimelineCount: 0,
			BlockCount:    0,
			TotalSize:     0,
			LastUpdate:    time.Now(),
		}
		return
	}
	
	timelineSet := make(map[string]bool)
	var totalSize int64
	blockCount := len(storeIndexes)
	
	for key, index := range storeIndexes {
		parts := splitTimelineKey(key)
		if len(parts) > 0 {
			timelineSet[parts[0]] = true
		}
		totalSize += index.Size
	}
	
	g.loadInfo[storeID] = &StoreLoadInfo{
		StoreID:       storeID,
		TimelineCount: len(timelineSet),
		BlockCount:    blockCount,
		TotalSize:     totalSize,
		LastUpdate:    time.Now(),
	}
}

// notifyWatchers 通知监听者
func (g *InMemoryGlobalIndex) notifyWatchers(timelineKey string, event IndexEvent) {
	watchers, exists := g.watchers[timelineKey]
	if !exists {
		return
	}
	
	for _, watcher := range watchers {
		select {
		case watcher <- event:
		default:
			// 如果channel满了，跳过这个watcher
		}
	}
}

// splitTimelineKey 分割Timeline键
func splitTimelineKey(key string) []string {
	// 简单的字符串分割实现
	result := make([]string, 0)
	start := 0
	for i, char := range key {
		if char == ':' {
			result = append(result, key[start:i])
			start = i + 1
		}
	}
	if start < len(key) {
		result = append(result, key[start:])
	}
	return result
}