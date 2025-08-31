package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StoreInfo Store节点信息
type StoreInfo struct {
	ID       string    `json:"id"`       // Store唯一标识
	Address  string    `json:"address"`  // Store服务地址
	Status   string    `json:"status"`   // 状态: active, inactive, unhealthy
	LastSeen time.Time `json:"lastSeen"` // 最后心跳时间
	Metadata map[string]interface{} `json:"metadata"` // 扩展元数据
}

// StoreRegistry Store注册中心接口
type StoreRegistry interface {
	// Register 注册Store节点
	Register(ctx context.Context, info *StoreInfo) error
	// Unregister 注销Store节点
	Unregister(ctx context.Context, storeID string) error
	// GetStore 获取指定Store信息
	GetStore(ctx context.Context, storeID string) (*StoreInfo, error)
	// ListStores 获取所有Store列表
	ListStores(ctx context.Context) ([]*StoreInfo, error)
	// ListActiveStores 获取活跃Store列表
	ListActiveStores(ctx context.Context) ([]*StoreInfo, error)
	// UpdateHeartbeat 更新心跳
	UpdateHeartbeat(ctx context.Context, storeID string) error
	// Watch 监听Store变化
	Watch(ctx context.Context) (<-chan StoreEvent, error)
}

// StoreEvent Store事件
type StoreEvent struct {
	Type  string     `json:"type"`  // 事件类型: register, unregister, heartbeat, unhealthy
	Store *StoreInfo `json:"store"` // Store信息
}

// InMemoryRegistry 内存实现的Store注册中心
type InMemoryRegistry struct {
	mu       sync.RWMutex
	stores   map[string]*StoreInfo
	watchers []chan StoreEvent
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewInMemoryRegistry 创建内存注册中心
func NewInMemoryRegistry() *InMemoryRegistry {
	ctx, cancel := context.WithCancel(context.Background())
	r := &InMemoryRegistry{
		stores:   make(map[string]*StoreInfo),
		watchers: make([]chan StoreEvent, 0),
		ctx:      ctx,
		cancel:   cancel,
	}
	
	// 启动健康检查协程
	go r.healthCheck()
	
	return r
}

// Register 注册Store节点
func (r *InMemoryRegistry) Register(ctx context.Context, info *StoreInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	info.Status = "active"
	info.LastSeen = time.Now()
	r.stores[info.ID] = info
	
	// 发送注册事件
	r.notifyWatchers(StoreEvent{
		Type:  "register",
		Store: info,
	})
	
	return nil
}

// Unregister 注销Store节点
func (r *InMemoryRegistry) Unregister(ctx context.Context, storeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	store, exists := r.stores[storeID]
	if !exists {
		return fmt.Errorf("store %s not found", storeID)
	}
	
	delete(r.stores, storeID)
	
	// 发送注销事件
	r.notifyWatchers(StoreEvent{
		Type:  "unregister",
		Store: store,
	})
	
	return nil
}

// GetStore 获取指定Store信息
func (r *InMemoryRegistry) GetStore(ctx context.Context, storeID string) (*StoreInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	store, exists := r.stores[storeID]
	if !exists {
		return nil, fmt.Errorf("store %s not found", storeID)
	}
	
	return store, nil
}

// ListStores 获取所有Store列表
func (r *InMemoryRegistry) ListStores(ctx context.Context) ([]*StoreInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	stores := make([]*StoreInfo, 0, len(r.stores))
	for _, store := range r.stores {
		stores = append(stores, store)
	}
	
	return stores, nil
}

// ListActiveStores 获取活跃Store列表
func (r *InMemoryRegistry) ListActiveStores(ctx context.Context) ([]*StoreInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	stores := make([]*StoreInfo, 0)
	for _, store := range r.stores {
		if store.Status == "active" {
			stores = append(stores, store)
		}
	}
	
	return stores, nil
}

// UpdateHeartbeat 更新心跳
func (r *InMemoryRegistry) UpdateHeartbeat(ctx context.Context, storeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	store, exists := r.stores[storeID]
	if !exists {
		return fmt.Errorf("store %s not found", storeID)
	}
	
	store.LastSeen = time.Now()
	if store.Status == "unhealthy" {
		store.Status = "active"
	}
	
	// 发送心跳事件
	r.notifyWatchers(StoreEvent{
		Type:  "heartbeat",
		Store: store,
	})
	
	return nil
}

// Watch 监听Store变化
func (r *InMemoryRegistry) Watch(ctx context.Context) (<-chan StoreEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	ch := make(chan StoreEvent, 100)
	r.watchers = append(r.watchers, ch)
	
	// 当context取消时，清理watcher
	go func() {
		<-ctx.Done()
		r.mu.Lock()
		defer r.mu.Unlock()
		
		// 移除watcher
		for i, watcher := range r.watchers {
			if watcher == ch {
				r.watchers = append(r.watchers[:i], r.watchers[i+1:]...)
				close(ch)
				break
			}
		}
	}()
	
	return ch, nil
}

// notifyWatchers 通知所有监听者
func (r *InMemoryRegistry) notifyWatchers(event StoreEvent) {
	for _, watcher := range r.watchers {
		select {
		case watcher <- event:
		default:
			// 如果channel满了，跳过这个watcher
		}
	}
}

// healthCheck 健康检查协程
func (r *InMemoryRegistry) healthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.checkUnhealthyStores()
		}
	}
}

// checkUnhealthyStores 检查不健康的Store
func (r *InMemoryRegistry) checkUnhealthyStores() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	for _, store := range r.stores {
		// 如果超过60秒没有心跳，标记为不健康
		if now.Sub(store.LastSeen) > 60*time.Second && store.Status == "active" {
			store.Status = "unhealthy"
			
			// 发送不健康事件
			r.notifyWatchers(StoreEvent{
				Type:  "unhealthy",
				Store: store,
			})
		}
	}
}

// Close 关闭注册中心
func (r *InMemoryRegistry) Close() {
	r.cancel()
}