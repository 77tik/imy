package storage

import (
	"context"
	"fmt"
	"time"
)

// DistributedStorageManager 分布式存储管理器
type DistributedStorageManager struct {
	localStore       *Store
	globalIndex      GlobalIndexManager
	routerManager    *RouterManager
	storeRegistry    StoreRegistry
	rpcClientPool    *StoreRPCClientPool
	crossStoreAccess *DistributedStoreAccessor
	lockManager      DistributedLockManager
	txnCoordinator   TransactionCoordinator
	storeID          string
}

// NewDistributedStorageManager 创建分布式存储管理器
func NewDistributedStorageManager(
	localStore *Store,
	globalIndex GlobalIndexManager,
	routerManager *RouterManager,
	storeRegistry StoreRegistry,
	rpcClientPool *StoreRPCClientPool,
	storeID string,
) *DistributedStorageManager {
	// 创建分布式锁管理器
	lockManager := NewInMemoryDistributedLockManager(storeID)
	
	// 创建事务协调器
	txnCoordinator := NewInMemoryTransactionCoordinator(storeID, lockManager)
	
	// 获取默认路由器作为TimelineRouter
	defaultRouter, err := routerManager.GetRouter("")
	if err != nil {
		// 如果没有默认路由器，创建一个简单的路由器
		defaultRouter = NewConsistentHashRouter(1, 100, 0.8)
	}
	
	// 创建跨Store访问器
	crossStoreAccess := NewDistributedStoreAccessor(
		localStore,
		rpcClientPool,
		globalIndex,
		defaultRouter,
		storeRegistry,
	)
	
	return &DistributedStorageManager{
		localStore:       localStore,
		globalIndex:      globalIndex,
		routerManager:    routerManager,
		storeRegistry:    storeRegistry,
		rpcClientPool:    rpcClientPool,
		crossStoreAccess: crossStoreAccess,
		lockManager:      lockManager,
		txnCoordinator:   txnCoordinator,
		storeID:          storeID,
	}
}

// CreateTimelineWithTransaction 使用事务创建Timeline
func (dsm *DistributedStorageManager) CreateTimelineWithTransaction(ctx context.Context, timelineKey string, timelineType string) error {
	// 确定目标Store
	targetStoreID, err := dsm.routerManager.RouteTimeline(timelineKey)
	if err != nil {
		return fmt.Errorf("failed to route timeline: %w", err)
	}
	
	// 创建事务参与者
	participants := []*TransactionParticipant{
		{
			StoreID:   targetStoreID,
			Operation: OpCreateTimeline,
			Params: map[string]interface{}{
				"timeline_key":  timelineKey,
				"timeline_type": timelineType,
			},
		},
		{
			StoreID:   dsm.storeID, // 本地Store负责更新全局索引
			Operation: OpUpdateIndex,
			Params: map[string]interface{}{
				"index_key":     timelineKey,
				"target_store":  targetStoreID,
				"operation":     "add",
			},
		},
	}
	
	// 执行事务
	return ExecuteTransaction(ctx, dsm.txnCoordinator, participants, 30*time.Second)
}

// DeleteTimelineWithTransaction 使用事务删除Timeline
func (dsm *DistributedStorageManager) DeleteTimelineWithTransaction(ctx context.Context, timelineKey string) error {
	// 获取Timeline位置
	location, err := dsm.globalIndex.GetTimelineLocation(ctx, timelineKey)
	if err != nil {
		return fmt.Errorf("failed to get timeline location: %w", err)
	}
	
	// 确定主Store
	var targetStoreID string
	if len(location.Blocks) > 0 {
		targetStoreID = location.Blocks[0].StoreID
	} else {
		return fmt.Errorf("timeline not found: %s", timelineKey)
	}
	
	// 创建事务参与者
	participants := []*TransactionParticipant{
		{
			StoreID:   targetStoreID,
			Operation: OpDeleteTimeline,
			Params: map[string]interface{}{
				"timeline_key": timelineKey,
			},
		},
		{
			StoreID:   dsm.storeID, // 本地Store负责更新全局索引
			Operation: OpUpdateIndex,
			Params: map[string]interface{}{
				"index_key":  timelineKey,
				"operation": "remove",
			},
		},
	}
	
	// 执行事务
	return ExecuteTransaction(ctx, dsm.txnCoordinator, participants, 30*time.Second)
}

// AddMessageWithTransaction 使用事务添加消息
func (dsm *DistributedStorageManager) AddMessageWithTransaction(ctx context.Context, timelineKey string, senderID string, data []byte, userIDs []string) error {
	// 获取Timeline位置
	location, err := dsm.globalIndex.GetTimelineLocation(ctx, timelineKey)
	if err != nil {
		return fmt.Errorf("failed to get timeline location: %w", err)
	}
	
	// 确定主Store
	var targetStoreID string
	if len(location.Blocks) > 0 {
		targetStoreID = location.Blocks[0].StoreID
	} else {
		return fmt.Errorf("timeline not found: %s", timelineKey)
	}
	
	// 创建事务参与者
	participants := []*TransactionParticipant{
		{
			StoreID:   targetStoreID,
			Operation: OpAddMessage,
			Params: map[string]interface{}{
				"timeline_key": timelineKey,
				"sender_id":    senderID,
				"data":         data,
				"user_ids":     userIDs,
			},
		},
	}
	
	// 执行事务
	return ExecuteTransaction(ctx, dsm.txnCoordinator, participants, 15*time.Second)
}

// MigrateTimelineWithTransaction 使用事务迁移Timeline
func (dsm *DistributedStorageManager) MigrateTimelineWithTransaction(ctx context.Context, timelineKey string, targetStoreID string) error {
	// 获取Timeline位置
	location, err := dsm.globalIndex.GetTimelineLocation(ctx, timelineKey)
	if err != nil {
		return fmt.Errorf("failed to get timeline location: %w", err)
	}
	
	// 确定源Store
	var sourceStoreID string
	if len(location.Blocks) > 0 {
		sourceStoreID = location.Blocks[0].StoreID
	} else {
		return fmt.Errorf("timeline not found: %s", timelineKey)
	}
	
	if sourceStoreID == targetStoreID {
		return fmt.Errorf("timeline is already on target store: %s", targetStoreID)
	}
	
	// 创建事务参与者
	participants := []*TransactionParticipant{
		{
			StoreID:   sourceStoreID,
			Operation: OpMigrateTimeline,
			Params: map[string]interface{}{
				"timeline_key":     timelineKey,
				"target_store_id": targetStoreID,
				"operation":        "export",
			},
		},
		{
			StoreID:   targetStoreID,
			Operation: OpMigrateTimeline,
			Params: map[string]interface{}{
				"timeline_key":     timelineKey,
				"source_store_id": sourceStoreID,
				"operation":        "import",
			},
		},
		{
			StoreID:   dsm.storeID, // 本地Store负责更新全局索引
			Operation: OpUpdateIndex,
			Params: map[string]interface{}{
				"index_key":        timelineKey,
				"old_store_id":     sourceStoreID,
				"new_store_id":     targetStoreID,
				"operation":        "migrate",
			},
		},
	}
	
	// 执行事务
	return ExecuteTransaction(ctx, dsm.txnCoordinator, participants, 60*time.Second)
}

// GetTimelineWithLock 使用锁获取Timeline
func (dsm *DistributedStorageManager) GetTimelineWithLock(ctx context.Context, timelineKey string) (*Timeline, error) {
	lockKey := fmt.Sprintf("timeline:%s:read", timelineKey)
	
	var timeline *Timeline
	err := WithLock(ctx, dsm.lockManager, lockKey, 10*time.Second, func() error {
		var err error
		timeline, err = dsm.crossStoreAccess.GetTimeline(ctx, timelineKey)
		return err
	})
	
	return timeline, err
}

// GetMessagesWithLock 使用锁获取消息
func (dsm *DistributedStorageManager) GetMessagesWithLock(ctx context.Context, timelineKey string, startTime, endTime int64, limit int) ([]*Message, error) {
	lockKey := fmt.Sprintf("timeline:%s:messages:read", timelineKey)
	
	var messages []*Message
	err := WithLock(ctx, dsm.lockManager, lockKey, 10*time.Second, func() error {
		var err error
		messages, err = dsm.crossStoreAccess.GetMessages(ctx, timelineKey, startTime, endTime, limit)
		return err
	})
	
	return messages, err
}

// GetStoreStats 获取Store统计信息
func (dsm *DistributedStorageManager) GetStoreStats(ctx context.Context, storeID string) (*StoreStats, error) {
	return dsm.crossStoreAccess.GetStoreStats(ctx, storeID)
}

// HealthCheck 健康检查
func (dsm *DistributedStorageManager) HealthCheck(ctx context.Context, storeID string) error {
	return dsm.crossStoreAccess.HealthCheck(ctx, storeID)
}

// RegisterTransactionHandler 注册事务处理器
func (dsm *DistributedStorageManager) RegisterTransactionHandler(storeID string, handler TransactionParticipantHandler) {
	if coordinator, ok := dsm.txnCoordinator.(*InMemoryTransactionCoordinator); ok {
		coordinator.RegisterHandler(storeID, handler)
	}
}

// GetLockManager 获取锁管理器
func (dsm *DistributedStorageManager) GetLockManager() DistributedLockManager {
	return dsm.lockManager
}

// GetTransactionCoordinator 获取事务协调器
func (dsm *DistributedStorageManager) GetTransactionCoordinator() TransactionCoordinator {
	return dsm.txnCoordinator
}

// GetCrossStoreAccessor 获取跨Store访问器
func (dsm *DistributedStorageManager) GetCrossStoreAccessor() *DistributedStoreAccessor {
	return dsm.crossStoreAccess
}

// Close 关闭分布式存储管理器
func (dsm *DistributedStorageManager) Close() error {
	// 关闭锁管理器
	if lockManager, ok := dsm.lockManager.(*InMemoryDistributedLockManager); ok {
		lockManager.Close()
	}
	
	// 关闭事务协调器
	if txnCoordinator, ok := dsm.txnCoordinator.(*InMemoryTransactionCoordinator); ok {
		txnCoordinator.Close()
	}
	
	return nil
}

// DefaultTransactionHandler 默认事务处理器实现
type DefaultTransactionHandler struct {
	localStore    *Store
	globalIndex   GlobalIndexManager
	rpcClientPool *StoreRPCClientPool
	storeID       string
}

// NewDefaultTransactionHandler 创建默认事务处理器
func NewDefaultTransactionHandler(
	localStore *Store,
	globalIndex GlobalIndexManager,
	rpcClientPool *StoreRPCClientPool,
	storeID string,
) *DefaultTransactionHandler {
	return &DefaultTransactionHandler{
		localStore:    localStore,
		globalIndex:   globalIndex,
		rpcClientPool: rpcClientPool,
		storeID:       storeID,
	}
}

// Prepare 准备操作
func (h *DefaultTransactionHandler) Prepare(ctx context.Context, txnID string, participant *TransactionParticipant) error {
	switch participant.Operation {
	case OpCreateTimeline:
		// 验证Timeline不存在
		timelineKey := participant.Params["timeline_key"].(string)
		_, err := h.globalIndex.GetTimelineLocation(ctx, timelineKey)
		if err == nil {
			return fmt.Errorf("timeline already exists: %s", timelineKey)
		}
		return nil
		
	case OpDeleteTimeline:
		// 验证Timeline存在
		timelineKey := participant.Params["timeline_key"].(string)
		_, err := h.globalIndex.GetTimelineLocation(ctx, timelineKey)
		if err != nil {
			return fmt.Errorf("timeline not found: %s", timelineKey)
		}
		return nil
		
	case OpAddMessage:
		// 验证Timeline存在
		timelineKey := participant.Params["timeline_key"].(string)
		_, err := h.globalIndex.GetTimelineLocation(ctx, timelineKey)
		if err != nil {
			return fmt.Errorf("timeline not found: %s", timelineKey)
		}
		return nil
		
	case OpMigrateTimeline:
		// 验证迁移条件
		timelineKey := participant.Params["timeline_key"].(string)
		_, err := h.globalIndex.GetTimelineLocation(ctx, timelineKey)
		if err != nil {
			return fmt.Errorf("timeline not found: %s", timelineKey)
		}
		return nil
		
	case OpUpdateIndex:
		// 索引更新总是可以准备
		return nil
		
	default:
		return fmt.Errorf("unsupported operation: %s", participant.Operation)
	}
}

// Commit 提交操作
func (h *DefaultTransactionHandler) Commit(ctx context.Context, txnID string, participant *TransactionParticipant) error {
	switch participant.Operation {
	case OpCreateTimeline:
		timelineKey := participant.Params["timeline_key"].(string)
		timelineType := participant.Params["timeline_type"].(string)
		
		if participant.StoreID == h.storeID {
			// 本地创建
			if timelineType == "conversation" {
				_ = h.localStore.GetOrCreateConvTimeline(timelineKey)
				return nil
			} else {
				_ = h.localStore.GetOrCreateUserTimeline(timelineKey)
				return nil
			}
		} else {
			// 远程创建
			client, err := h.rpcClientPool.GetClient(ctx, participant.StoreID, "")
			if err != nil {
				return err
			}
			// 这里需要实现RPC调用
			_ = client
			return fmt.Errorf("remote timeline creation not implemented")
		}
		
	case OpDeleteTimeline:
		// 删除Timeline的实现
		return fmt.Errorf("timeline deletion not implemented")
		
	case OpAddMessage:
		timelineKey := participant.Params["timeline_key"].(string)
		senderID := participant.Params["sender_id"].(uint32)
		data := participant.Params["data"].([]byte)
		userIDs := participant.Params["user_ids"].([]string)
		
		if participant.StoreID == h.storeID {
			// 本地添加消息
			return h.localStore.AddMessage(timelineKey, senderID, data, userIDs)
		} else {
			// 远程添加消息
			client, err := h.rpcClientPool.GetClient(ctx, participant.StoreID, "")
			if err != nil {
				return err
			}
			// 这里需要实现RPC调用
			_ = client
			return fmt.Errorf("remote message addition not implemented")
		}
		
	case OpMigrateTimeline:
		// Timeline迁移的实现
		return fmt.Errorf("timeline migration not implemented")
		
	case OpUpdateIndex:
		indexKey := participant.Params["index_key"].(string)
		operation := participant.Params["operation"].(string)
		
		switch operation {
		case "add":
			targetStore := participant.Params["target_store"].(string)
			return h.globalIndex.AddIndex(ctx, &GlobalStoreIndex{
				TimelineKey: indexKey,
				StoreID:     targetStore,
				BlockID:     fmt.Sprintf("%s_block_1", indexKey),
				Offset:      0,
				Size:        0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			})
		case "remove":
			return h.globalIndex.RemoveIndex(ctx, indexKey, "")
		case "migrate":
			// 迁移索引更新的实现
			return fmt.Errorf("index migration not implemented")
		default:
			return fmt.Errorf("unsupported index operation: %s", operation)
		}
		
	default:
		return fmt.Errorf("unsupported operation: %s", participant.Operation)
	}
}

// Abort 回滚操作
func (h *DefaultTransactionHandler) Abort(ctx context.Context, txnID string, participant *TransactionParticipant) error {
	// 大多数情况下，准备阶段的操作是只读的，不需要回滚
	// 如果有需要清理的资源，在这里实现
	return nil
}