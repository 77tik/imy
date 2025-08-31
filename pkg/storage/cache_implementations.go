package storage

import (
	"container/list"
	"sync"
	"time"
)

// MemoryCache 内存缓存实现（LRU）
type MemoryCache struct {
	mu       sync.RWMutex
	cache    map[string]*list.Element
	lruList  *list.List
	maxSize  int64
	curSize  int64
	stats    *CacheStats
}

// memoryCacheItem 内存缓存项
type memoryCacheItem struct {
	key        string
	value      interface{}
	expireTime time.Time
	size       int64
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(maxSize int64) *MemoryCache {
	return &MemoryCache{
		cache:   make(map[string]*list.Element),
		lruList: list.New(),
		maxSize: maxSize,
		stats:   &CacheStats{},
	}
}

// Get 获取缓存值
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	elem, exists := mc.cache[key]
	if !exists {
		mc.stats.Misses++
		return nil, false
	}
	
	item := elem.Value.(*memoryCacheItem)
	
	// 检查是否过期
	if !item.expireTime.IsZero() && time.Now().After(item.expireTime) {
		mc.removeElement(elem)
		mc.stats.Misses++
		return nil, false
	}
	
	// 移动到链表头部（LRU）
	mc.lruList.MoveToFront(elem)
	mc.stats.Hits++
	
	return item.value, true
}

// Set 设置缓存值
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	var expireTime time.Time
	if ttl > 0 {
		expireTime = time.Now().Add(ttl)
	}
	
	size := mc.estimateSize(value)
	item := &memoryCacheItem{
		key:        key,
		value:      value,
		expireTime: expireTime,
		size:       size,
	}
	
	// 如果键已存在，更新值
	if elem, exists := mc.cache[key]; exists {
		oldItem := elem.Value.(*memoryCacheItem)
		mc.curSize = mc.curSize - oldItem.size + size
		elem.Value = item
		mc.lruList.MoveToFront(elem)
		return nil
	}
	
	// 检查容量限制
	for mc.curSize+size > mc.maxSize && mc.lruList.Len() > 0 {
		mc.evictLRU()
	}
	
	// 添加新项
	elem := mc.lruList.PushFront(item)
	mc.cache[key] = elem
	mc.curSize += size
	mc.stats.EntryCount++
	mc.stats.TotalSize = mc.curSize
	
	return nil
}

// Delete 删除缓存
func (mc *MemoryCache) Delete(key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if elem, exists := mc.cache[key]; exists {
		mc.removeElement(elem)
	}
	
	return nil
}

// Clear 清空缓存
func (mc *MemoryCache) Clear() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.cache = make(map[string]*list.Element)
	mc.lruList = list.New()
	mc.curSize = 0
	mc.stats.EntryCount = 0
	mc.stats.TotalSize = 0
	
	return nil
}

// Size 获取当前大小
func (mc *MemoryCache) Size() int64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.curSize
}

// Stats 获取统计信息
func (mc *MemoryCache) Stats() *CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	stats := *mc.stats
	stats.TotalSize = mc.curSize
	stats.EntryCount = int64(len(mc.cache))
	
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRatio = float64(stats.Hits) / float64(total)
	}
	
	return &stats
}

// evictLRU 淘汰最近最少使用的项
func (mc *MemoryCache) evictLRU() {
	elem := mc.lruList.Back()
	if elem != nil {
		mc.removeElement(elem)
		mc.stats.Evictions++
	}
}

// removeElement 移除元素
func (mc *MemoryCache) removeElement(elem *list.Element) {
	item := elem.Value.(*memoryCacheItem)
	delete(mc.cache, item.key)
	mc.lruList.Remove(elem)
	mc.curSize -= item.size
	mc.stats.EntryCount--
}

// estimateSize 估算值的大小
func (mc *MemoryCache) estimateSize(value interface{}) int64 {
	// 简单估算，实际应用中可以更精确
	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	default:
		return 64 // 默认64字节
	}
}

// DiskCache 磁盘缓存实现（简化版）
type DiskCache struct {
	mu      sync.RWMutex
	baseDir string
	stats   *CacheStats
}

// NewDiskCache 创建磁盘缓存
func NewDiskCache(baseDir string) *DiskCache {
	return &DiskCache{
		baseDir: baseDir,
		stats:   &CacheStats{},
	}
}

// Get 获取缓存值
func (dc *DiskCache) Get(key string) (interface{}, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	// 简化实现：实际应该从磁盘读取文件
	// 这里返回false表示未找到
	dc.stats.Misses++
	return nil, false
}

// Set 设置缓存值
func (dc *DiskCache) Set(key string, value interface{}, ttl time.Duration) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	// 简化实现：实际应该写入磁盘文件
	// 这里只更新统计
	dc.stats.EntryCount++
	return nil
}

// Delete 删除缓存
func (dc *DiskCache) Delete(key string) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	// 简化实现：实际应该删除磁盘文件
	dc.stats.EntryCount--
	return nil
}

// Clear 清空缓存
func (dc *DiskCache) Clear() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	// 简化实现：实际应该清空磁盘目录
	dc.stats.EntryCount = 0
	dc.stats.TotalSize = 0
	return nil
}

// Size 获取当前大小
func (dc *DiskCache) Size() int64 {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.stats.TotalSize
}

// Stats 获取统计信息
func (dc *DiskCache) Stats() *CacheStats {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.stats
}

// DistributedCache 分布式缓存实现（简化版）
type DistributedCache struct {
	mu      sync.RWMutex
	nodes   []string
	stats   *CacheStats
}

// NewDistributedCache 创建分布式缓存
func NewDistributedCache(nodes []string) *DistributedCache {
	return &DistributedCache{
		nodes: nodes,
		stats: &CacheStats{},
	}
}

// Get 获取缓存值
func (dc *DistributedCache) Get(key string) (interface{}, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	// 简化实现：实际应该通过网络从远程节点获取
	// 这里返回false表示未找到
	dc.stats.Misses++
	return nil, false
}

// Set 设置缓存值
func (dc *DistributedCache) Set(key string, value interface{}, ttl time.Duration) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	// 简化实现：实际应该通过网络写入远程节点
	// 这里只更新统计
	dc.stats.EntryCount++
	return nil
}

// Delete 删除缓存
func (dc *DistributedCache) Delete(key string) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	// 简化实现：实际应该通过网络从远程节点删除
	dc.stats.EntryCount--
	return nil
}

// Clear 清空缓存
func (dc *DistributedCache) Clear() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	// 简化实现：实际应该清空所有远程节点
	dc.stats.EntryCount = 0
	dc.stats.TotalSize = 0
	return nil
}

// Size 获取当前大小
func (dc *DistributedCache) Size() int64 {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.stats.TotalSize
}

// Stats 获取统计信息
func (dc *DistributedCache) Stats() *CacheStats {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.stats
}