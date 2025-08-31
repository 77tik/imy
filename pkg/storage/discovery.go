package storage

import (
	"context"
	"fmt"
	"log"
	"time"
)

// StoreDiscoveryClient Store服务发现客户端
type StoreDiscoveryClient struct {
	registry     StoreRegistry
	storeInfo    *StoreInfo
	heartbeatCtx context.Context
	heartbeatCancel context.CancelFunc
	isRegistered bool
}

// NewStoreDiscoveryClient 创建服务发现客户端
func NewStoreDiscoveryClient(registry StoreRegistry, storeInfo *StoreInfo) *StoreDiscoveryClient {
	return &StoreDiscoveryClient{
		registry:  registry,
		storeInfo: storeInfo,
	}
}

// Start 启动服务发现客户端
func (c *StoreDiscoveryClient) Start(ctx context.Context) error {
	// 注册Store
	if err := c.registry.Register(ctx, c.storeInfo); err != nil {
		return fmt.Errorf("failed to register store: %w", err)
	}
	
	c.isRegistered = true
	log.Printf("Store %s registered successfully at %s", c.storeInfo.ID, c.storeInfo.Address)
	
	// 启动心跳
	c.heartbeatCtx, c.heartbeatCancel = context.WithCancel(ctx)
	go c.startHeartbeat()
	
	return nil
}

// Stop 停止服务发现客户端
func (c *StoreDiscoveryClient) Stop() error {
	// 停止心跳
	if c.heartbeatCancel != nil {
		c.heartbeatCancel()
	}
	
	// 注销Store
	if c.isRegistered {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := c.registry.Unregister(ctx, c.storeInfo.ID); err != nil {
			log.Printf("Failed to unregister store %s: %v", c.storeInfo.ID, err)
			return err
		}
		
		log.Printf("Store %s unregistered successfully", c.storeInfo.ID)
		c.isRegistered = false
	}
	
	return nil
}

// startHeartbeat 启动心跳协程
func (c *StoreDiscoveryClient) startHeartbeat() {
	ticker := time.NewTicker(15 * time.Second) // 每15秒发送一次心跳
	defer ticker.Stop()
	
	for {
		select {
		case <-c.heartbeatCtx.Done():
			return
		case <-ticker.C:
			if err := c.sendHeartbeat(); err != nil {
				log.Printf("Failed to send heartbeat for store %s: %v", c.storeInfo.ID, err)
			}
		}
	}
}

// sendHeartbeat 发送心跳
func (c *StoreDiscoveryClient) sendHeartbeat() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return c.registry.UpdateHeartbeat(ctx, c.storeInfo.ID)
}

// UpdateMetadata 更新Store元数据
func (c *StoreDiscoveryClient) UpdateMetadata(metadata map[string]interface{}) error {
	c.storeInfo.Metadata = metadata
	
	// 重新注册以更新元数据
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return c.registry.Register(ctx, c.storeInfo)
}

// GetStoreInfo 获取当前Store信息
func (c *StoreDiscoveryClient) GetStoreInfo() *StoreInfo {
	return c.storeInfo
}

// StoreWatcher Store监听器
type StoreWatcher struct {
	registry StoreRegistry
	eventCh  <-chan StoreEvent
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewStoreWatcher 创建Store监听器
func NewStoreWatcher(registry StoreRegistry) *StoreWatcher {
	return &StoreWatcher{
		registry: registry,
	}
}

// Start 启动监听器
func (w *StoreWatcher) Start(ctx context.Context) error {
	w.ctx, w.cancel = context.WithCancel(ctx)
	
	eventCh, err := w.registry.Watch(w.ctx)
	if err != nil {
		return fmt.Errorf("failed to start watching: %w", err)
	}
	
	w.eventCh = eventCh
	return nil
}

// Stop 停止监听器
func (w *StoreWatcher) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

// Events 获取事件通道
func (w *StoreWatcher) Events() <-chan StoreEvent {
	return w.eventCh
}

// StoreManager Store管理器
type StoreManager struct {
	registry StoreRegistry
	watcher  *StoreWatcher
	stores   map[string]*StoreInfo
}

// NewStoreManager 创建Store管理器
func NewStoreManager(registry StoreRegistry) *StoreManager {
	return &StoreManager{
		registry: registry,
		watcher:  NewStoreWatcher(registry),
		stores:   make(map[string]*StoreInfo),
	}
}

// Start 启动Store管理器
func (m *StoreManager) Start(ctx context.Context) error {
	// 启动监听器
	if err := m.watcher.Start(ctx); err != nil {
		return err
	}
	
	// 初始化加载所有Store
	if err := m.loadStores(ctx); err != nil {
		return err
	}
	
	// 启动事件处理协程
	go m.handleEvents()
	
	return nil
}

// Stop 停止Store管理器
func (m *StoreManager) Stop() {
	m.watcher.Stop()
}

// loadStores 加载所有Store
func (m *StoreManager) loadStores(ctx context.Context) error {
	stores, err := m.registry.ListStores(ctx)
	if err != nil {
		return err
	}
	
	for _, store := range stores {
		m.stores[store.ID] = store
	}
	
	log.Printf("Loaded %d stores", len(stores))
	return nil
}

// handleEvents 处理Store事件
func (m *StoreManager) handleEvents() {
	for event := range m.watcher.Events() {
		switch event.Type {
		case "register":
			m.stores[event.Store.ID] = event.Store
			log.Printf("Store %s registered", event.Store.ID)
			
		case "unregister":
			delete(m.stores, event.Store.ID)
			log.Printf("Store %s unregistered", event.Store.ID)
			
		case "heartbeat":
			if store, exists := m.stores[event.Store.ID]; exists {
				store.LastSeen = event.Store.LastSeen
				store.Status = event.Store.Status
			}
			
		case "unhealthy":
			if store, exists := m.stores[event.Store.ID]; exists {
				store.Status = "unhealthy"
				log.Printf("Store %s marked as unhealthy", event.Store.ID)
			}
		}
	}
}

// GetActiveStores 获取活跃的Store列表
func (m *StoreManager) GetActiveStores() []*StoreInfo {
	stores := make([]*StoreInfo, 0)
	for _, store := range m.stores {
		if store.Status == "active" {
			stores = append(stores, store)
		}
	}
	return stores
}

// GetStore 获取指定Store
func (m *StoreManager) GetStore(storeID string) (*StoreInfo, bool) {
	store, exists := m.stores[storeID]
	return store, exists
}

// GetAllStores 获取所有Store
func (m *StoreManager) GetAllStores() map[string]*StoreInfo {
	return m.stores
}