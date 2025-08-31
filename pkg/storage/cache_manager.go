package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CacheLevel 缓存级别
type CacheLevel int

const (
	L1Cache CacheLevel = iota // 内存缓存
	L2Cache                   // 本地磁盘缓存
	L3Cache                   // 分布式缓存
)

// CacheEntry 缓存条目
type CacheEntry struct {
	Key        string
	Value      interface{}
	ExpireTime time.Time
	AccessTime time.Time
	HitCount   int64
	Size       int64
}

// CacheStats 缓存统计
type CacheStats struct {
	Hits        int64
	Misses      int64
	Evictions   int64
	TotalSize   int64
	EntryCount  int64
	HitRatio    float64
}

// CachePolicy 缓存策略
type CachePolicy struct {
	MaxSize    int64         // 最大缓存大小
	TTL        time.Duration // 生存时间
	EvictPolicy string       // 淘汰策略: LRU, LFU, FIFO
	WritePolicy string       // 写策略: WriteThrough, WriteBack, WriteAround
}

// CacheManager 多级缓存管理器接口
type CacheManager interface {
	// Get 获取缓存值
	Get(ctx context.Context, key string) (interface{}, bool, error)
	
	// Set 设置缓存值
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	
	// Delete 删除缓存
	Delete(ctx context.Context, key string) error
	
	// Clear 清空指定级别的缓存
	Clear(ctx context.Context, level CacheLevel) error
	
	// GetStats 获取缓存统计
	GetStats(level CacheLevel) *CacheStats
	
	// UpdatePolicy 更新缓存策略
	UpdatePolicy(level CacheLevel, policy *CachePolicy) error
	
	// Warm 预热缓存
	Warm(ctx context.Context, keys []string) error
	
	// Close 关闭缓存管理器
	Close() error
}

// MultiLevelCacheManager 多级缓存管理器实现
type MultiLevelCacheManager struct {
	l1Cache Cache // L1: 内存缓存
	l2Cache Cache // L2: 本地磁盘缓存
	l3Cache Cache // L3: 分布式缓存
	
	policies map[CacheLevel]*CachePolicy
	stats    map[CacheLevel]*CacheStats
	mu       sync.RWMutex
	
	// 性能优化相关
	prefetcher   *Prefetcher
	compressor   Compressor
	serializer   Serializer
	batchManager *BatchManager
}

// Cache 缓存接口
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration) error
	Delete(key string) error
	Clear() error
	Size() int64
	Stats() *CacheStats
}

// NewMultiLevelCacheManager 创建多级缓存管理器
func NewMultiLevelCacheManager(l1, l2, l3 Cache) *MultiLevelCacheManager {
	mcm := &MultiLevelCacheManager{
		l1Cache: l1,
		l2Cache: l2,
		l3Cache: l3,
		policies: make(map[CacheLevel]*CachePolicy),
		stats:    make(map[CacheLevel]*CacheStats),
	}
	
	// 初始化默认策略
	mcm.policies[L1Cache] = &CachePolicy{
		MaxSize:     100 * 1024 * 1024, // 100MB
		TTL:         5 * time.Minute,
		EvictPolicy: "LRU",
		WritePolicy: "WriteThrough",
	}
	
	mcm.policies[L2Cache] = &CachePolicy{
		MaxSize:     1024 * 1024 * 1024, // 1GB
		TTL:         30 * time.Minute,
		EvictPolicy: "LRU",
		WritePolicy: "WriteBack",
	}
	
	mcm.policies[L3Cache] = &CachePolicy{
		MaxSize:     10 * 1024 * 1024 * 1024, // 10GB
		TTL:         2 * time.Hour,
		EvictPolicy: "LFU",
		WritePolicy: "WriteAround",
	}
	
	// 初始化统计
	for level := L1Cache; level <= L3Cache; level++ {
		mcm.stats[level] = &CacheStats{}
	}
	
	// 初始化性能优化组件
	mcm.prefetcher = NewPrefetcher(mcm)
	mcm.compressor = NewGzipCompressor()
	mcm.serializer = NewJSONSerializer()
	mcm.batchManager = NewBatchManager(mcm)
	
	return mcm
}

// Get 多级缓存获取
func (mcm *MultiLevelCacheManager) Get(ctx context.Context, key string) (interface{}, bool, error) {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()
	
	// L1缓存查找
	if value, found := mcm.l1Cache.Get(key); found {
		mcm.stats[L1Cache].Hits++
		mcm.updateHitRatio(L1Cache)
		return value, true, nil
	}
	mcm.stats[L1Cache].Misses++
	
	// L2缓存查找
	if mcm.l2Cache != nil {
		if value, found := mcm.l2Cache.Get(key); found {
			mcm.stats[L2Cache].Hits++
			// 提升到L1缓存
			mcm.l1Cache.Set(key, value, mcm.policies[L1Cache].TTL)
			mcm.updateHitRatio(L2Cache)
			return value, true, nil
		}
		mcm.stats[L2Cache].Misses++
	}
	
	// L3缓存查找
	if mcm.l3Cache != nil {
		if value, found := mcm.l3Cache.Get(key); found {
			mcm.stats[L3Cache].Hits++
			// 提升到L2和L1缓存
			if mcm.l2Cache != nil {
				mcm.l2Cache.Set(key, value, mcm.policies[L2Cache].TTL)
			}
			mcm.l1Cache.Set(key, value, mcm.policies[L1Cache].TTL)
			mcm.updateHitRatio(L3Cache)
			return value, true, nil
		}
		mcm.stats[L3Cache].Misses++
	}
	
	// 触发预取
	go mcm.prefetcher.TriggerPrefetch(key)
	
	return nil, false, nil
}

// Set 多级缓存设置
func (mcm *MultiLevelCacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	mcm.mu.Lock()
	defer mcm.mu.Unlock()
	
	// 序列化和压缩
	processedValue, err := mcm.processValue(value)
	if err != nil {
		return fmt.Errorf("failed to process value: %w", err)
	}
	
	// 根据写策略决定写入行为
	l1Policy := mcm.policies[L1Cache]
	
	switch l1Policy.WritePolicy {
	case "WriteThrough":
		// 同时写入所有级别
		mcm.l1Cache.Set(key, processedValue, ttl)
		if mcm.l2Cache != nil {
			mcm.l2Cache.Set(key, processedValue, ttl)
		}
		if mcm.l3Cache != nil {
			mcm.l3Cache.Set(key, processedValue, ttl)
		}
		
	case "WriteBack":
		// 只写入L1，延迟写入其他级别
		mcm.l1Cache.Set(key, processedValue, ttl)
		go mcm.batchManager.ScheduleWrite(key, processedValue, ttl)
		
	case "WriteAround":
		// 跳过L1，直接写入L2和L3
		if mcm.l2Cache != nil {
			mcm.l2Cache.Set(key, processedValue, ttl)
		}
		if mcm.l3Cache != nil {
			mcm.l3Cache.Set(key, processedValue, ttl)
		}
	}
	
	return nil
}

// Delete 删除缓存
func (mcm *MultiLevelCacheManager) Delete(ctx context.Context, key string) error {
	mcm.mu.Lock()
	defer mcm.mu.Unlock()
	
	mcm.l1Cache.Delete(key)
	if mcm.l2Cache != nil {
		mcm.l2Cache.Delete(key)
	}
	if mcm.l3Cache != nil {
		mcm.l3Cache.Delete(key)
	}
	
	return nil
}

// Clear 清空指定级别缓存
func (mcm *MultiLevelCacheManager) Clear(ctx context.Context, level CacheLevel) error {
	mcm.mu.Lock()
	defer mcm.mu.Unlock()
	
	switch level {
	case L1Cache:
		return mcm.l1Cache.Clear()
	case L2Cache:
		if mcm.l2Cache != nil {
			return mcm.l2Cache.Clear()
		}
	case L3Cache:
		if mcm.l3Cache != nil {
			return mcm.l3Cache.Clear()
		}
	}
	
	return nil
}

// GetStats 获取缓存统计
func (mcm *MultiLevelCacheManager) GetStats(level CacheLevel) *CacheStats {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()
	
	return mcm.stats[level]
}

// UpdatePolicy 更新缓存策略
func (mcm *MultiLevelCacheManager) UpdatePolicy(level CacheLevel, policy *CachePolicy) error {
	mcm.mu.Lock()
	defer mcm.mu.Unlock()
	
	mcm.policies[level] = policy
	return nil
}

// Warm 预热缓存
func (mcm *MultiLevelCacheManager) Warm(ctx context.Context, keys []string) error {
	return mcm.prefetcher.WarmCache(ctx, keys)
}

// Close 关闭缓存管理器
func (mcm *MultiLevelCacheManager) Close() error {
	if mcm.prefetcher != nil {
		mcm.prefetcher.Stop()
	}
	if mcm.batchManager != nil {
		mcm.batchManager.Stop()
	}
	return nil
}

// processValue 处理值（序列化和压缩）
func (mcm *MultiLevelCacheManager) processValue(value interface{}) (interface{}, error) {
	// 序列化
	data, err := mcm.serializer.Serialize(value)
	if err != nil {
		return nil, err
	}
	
	// 压缩（如果数据较大）
	if len(data) > 1024 { // 1KB阈值
		compressed, err := mcm.compressor.Compress(data)
		if err != nil {
			return nil, err
		}
		return compressed, nil
	}
	
	return data, nil
}

// updateHitRatio 更新命中率
func (mcm *MultiLevelCacheManager) updateHitRatio(level CacheLevel) {
	stats := mcm.stats[level]
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRatio = float64(stats.Hits) / float64(total)
	}
}