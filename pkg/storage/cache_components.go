package storage

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"
)

// Prefetcher 预取器接口
type Prefetcher struct {
	cacheManager *MultiLevelCacheManager
	prefetchQueue chan string
	stopCh       chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
	patterns     map[string][]string // 预取模式
}

// NewPrefetcher 创建预取器
func NewPrefetcher(cacheManager *MultiLevelCacheManager) *Prefetcher {
	p := &Prefetcher{
		cacheManager:  cacheManager,
		prefetchQueue: make(chan string, 1000),
		stopCh:        make(chan struct{}),
		patterns:      make(map[string][]string),
	}
	
	// 启动预取工作协程
	p.wg.Add(1)
	go p.prefetchWorker()
	
	return p
}

// TriggerPrefetch 触发预取
func (p *Prefetcher) TriggerPrefetch(key string) {
	select {
	case p.prefetchQueue <- key:
	default:
		// 队列满时忽略
	}
}

// WarmCache 预热缓存
func (p *Prefetcher) WarmCache(ctx context.Context, keys []string) error {
	for _, key := range keys {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case p.prefetchQueue <- key:
		default:
			// 队列满时继续下一个
		}
	}
	return nil
}

// Stop 停止预取器
func (p *Prefetcher) Stop() {
	close(p.stopCh)
	p.wg.Wait()
}

// prefetchWorker 预取工作协程
func (p *Prefetcher) prefetchWorker() {
	defer p.wg.Done()
	
	for {
		select {
		case <-p.stopCh:
			return
		case key := <-p.prefetchQueue:
			p.processPrefetch(key)
		}
	}
}

// processPrefetch 处理预取
func (p *Prefetcher) processPrefetch(key string) {
	// 基于访问模式预取相关数据
	p.mu.RLock()
	relatedKeys, exists := p.patterns[key]
	p.mu.RUnlock()
	
	if exists {
		for _, relatedKey := range relatedKeys {
			// 检查是否已在缓存中
			if _, found, _ := p.cacheManager.Get(context.Background(), relatedKey); !found {
				// 这里应该从数据源加载数据，暂时跳过
				// TODO: 实现从Store加载数据的逻辑
			}
		}
	}
}

// Compressor 压缩器接口
type Compressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
}

// GzipCompressor Gzip压缩器
type GzipCompressor struct{}

// NewGzipCompressor 创建Gzip压缩器
func NewGzipCompressor() *GzipCompressor {
	return &GzipCompressor{}
}

// Compress 压缩数据
func (gc *GzipCompressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	
	if err := writer.Close(); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// Decompress 解压数据
func (gc *GzipCompressor) Decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	
	return io.ReadAll(reader)
}

// Serializer 序列化器接口
type Serializer interface {
	Serialize(value interface{}) ([]byte, error)
	Deserialize(data []byte, target interface{}) error
}

// JSONSerializer JSON序列化器
type JSONSerializer struct{}

// NewJSONSerializer 创建JSON序列化器
func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{}
}

// Serialize 序列化
func (js *JSONSerializer) Serialize(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

// Deserialize 反序列化
func (js *JSONSerializer) Deserialize(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}

// BatchManager 批处理管理器
type BatchManager struct {
	cacheManager *MultiLevelCacheManager
	batchQueue   chan *BatchItem
	stopCh       chan struct{}
	wg           sync.WaitGroup
	batchSize    int
	flushInterval time.Duration
}

// BatchItem 批处理项
type BatchItem struct {
	Key   string
	Value interface{}
	TTL   time.Duration
	Op    string // "set" or "delete"
}

// NewBatchManager 创建批处理管理器
func NewBatchManager(cacheManager *MultiLevelCacheManager) *BatchManager {
	bm := &BatchManager{
		cacheManager:  cacheManager,
		batchQueue:    make(chan *BatchItem, 10000),
		stopCh:        make(chan struct{}),
		batchSize:     100,
		flushInterval: 5 * time.Second,
	}
	
	// 启动批处理工作协程
	bm.wg.Add(1)
	go bm.batchWorker()
	
	return bm
}

// ScheduleWrite 调度写入
func (bm *BatchManager) ScheduleWrite(key string, value interface{}, ttl time.Duration) {
	item := &BatchItem{
		Key:   key,
		Value: value,
		TTL:   ttl,
		Op:    "set",
	}
	
	select {
	case bm.batchQueue <- item:
	default:
		// 队列满时直接执行
		bm.executeItem(item)
	}
}

// ScheduleDelete 调度删除
func (bm *BatchManager) ScheduleDelete(key string) {
	item := &BatchItem{
		Key: key,
		Op:  "delete",
	}
	
	select {
	case bm.batchQueue <- item:
	default:
		// 队列满时直接执行
		bm.executeItem(item)
	}
}

// Stop 停止批处理管理器
func (bm *BatchManager) Stop() {
	close(bm.stopCh)
	bm.wg.Wait()
}

// batchWorker 批处理工作协程
func (bm *BatchManager) batchWorker() {
	defer bm.wg.Done()
	
	batch := make([]*BatchItem, 0, bm.batchSize)
	ticker := time.NewTicker(bm.flushInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-bm.stopCh:
			// 处理剩余批次
			if len(batch) > 0 {
				bm.executeBatch(batch)
			}
			return
			
		case item := <-bm.batchQueue:
			batch = append(batch, item)
			if len(batch) >= bm.batchSize {
				bm.executeBatch(batch)
				batch = batch[:0] // 重置切片
			}
			
		case <-ticker.C:
			if len(batch) > 0 {
				bm.executeBatch(batch)
				batch = batch[:0] // 重置切片
			}
		}
	}
}

// executeBatch 执行批次
func (bm *BatchManager) executeBatch(batch []*BatchItem) {
	for _, item := range batch {
		bm.executeItem(item)
	}
}

// executeItem 执行单个项目
func (bm *BatchManager) executeItem(item *BatchItem) {
	switch item.Op {
	case "set":
		// 写入L2和L3缓存
		if bm.cacheManager.l2Cache != nil {
			bm.cacheManager.l2Cache.Set(item.Key, item.Value, item.TTL)
		}
		if bm.cacheManager.l3Cache != nil {
			bm.cacheManager.l3Cache.Set(item.Key, item.Value, item.TTL)
		}
		
	case "delete":
		if bm.cacheManager.l2Cache != nil {
			bm.cacheManager.l2Cache.Delete(item.Key)
		}
		if bm.cacheManager.l3Cache != nil {
			bm.cacheManager.l3Cache.Delete(item.Key)
		}
	}
}