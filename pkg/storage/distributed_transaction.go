package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TransactionStatus 事务状态
type TransactionStatus int

const (
	TransactionStatusPending TransactionStatus = iota
	TransactionStatusPrepared
	TransactionStatusCommitted
	TransactionStatusAborted
	TransactionStatusTimeout
)

func (s TransactionStatus) String() string {
	switch s {
	case TransactionStatusPending:
		return "pending"
	case TransactionStatusPrepared:
		return "prepared"
	case TransactionStatusCommitted:
		return "committed"
	case TransactionStatusAborted:
		return "aborted"
	case TransactionStatusTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

// TransactionOperation 事务操作类型
type TransactionOperation int

const (
	OpCreateTimeline TransactionOperation = iota
	OpDeleteTimeline
	OpAddMessage
	OpMigrateTimeline
	OpUpdateIndex
)

func (op TransactionOperation) String() string {
	switch op {
	case OpCreateTimeline:
		return "create_timeline"
	case OpDeleteTimeline:
		return "delete_timeline"
	case OpAddMessage:
		return "add_message"
	case OpMigrateTimeline:
		return "migrate_timeline"
	case OpUpdateIndex:
		return "update_index"
	default:
		return "unknown"
	}
}

// TransactionParticipant 事务参与者
type TransactionParticipant struct {
	StoreID   string                 `json:"store_id"`
	Operation TransactionOperation   `json:"operation"`
	Params    map[string]interface{} `json:"params"`
	Status    TransactionStatus      `json:"status"`
	Error     string                 `json:"error,omitempty"`
}

// DistributedTransaction 分布式事务
type DistributedTransaction struct {
	TransactionID string                    `json:"transaction_id"`
	CoordinatorID string                    `json:"coordinator_id"`
	Participants  []*TransactionParticipant `json:"participants"`
	Status        TransactionStatus         `json:"status"`
	CreatedAt     time.Time                 `json:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at"`
	Timeout       time.Duration             `json:"timeout"`
	Locks         []string                  `json:"locks"`
	mu            sync.RWMutex
}

// TransactionCoordinator 事务协调器接口
type TransactionCoordinator interface {
	// 开始事务
	BeginTransaction(ctx context.Context, participants []*TransactionParticipant, timeout time.Duration) (*DistributedTransaction, error)
	// 准备阶段
	PrepareTransaction(ctx context.Context, txnID string) error
	// 提交事务
	CommitTransaction(ctx context.Context, txnID string) error
	// 回滚事务
	AbortTransaction(ctx context.Context, txnID string) error
	// 获取事务状态
	GetTransactionStatus(ctx context.Context, txnID string) (*DistributedTransaction, error)
	// 清理超时事务
	CleanupTimeoutTransactions(ctx context.Context) error
}

// TransactionParticipantHandler 事务参与者处理器接口
type TransactionParticipantHandler interface {
	// 准备操作
	Prepare(ctx context.Context, txnID string, participant *TransactionParticipant) error
	// 提交操作
	Commit(ctx context.Context, txnID string, participant *TransactionParticipant) error
	// 回滚操作
	Abort(ctx context.Context, txnID string, participant *TransactionParticipant) error
}

// InMemoryTransactionCoordinator 内存事务协调器实现
type InMemoryTransactionCoordinator struct {
	transactions map[string]*DistributedTransaction
	handlers     map[string]TransactionParticipantHandler
	lockManager  DistributedLockManager
	storeID      string
	mu           sync.RWMutex
	cleanupCh    chan struct{}
}

// NewInMemoryTransactionCoordinator 创建内存事务协调器
func NewInMemoryTransactionCoordinator(storeID string, lockManager DistributedLockManager) *InMemoryTransactionCoordinator {
	coordinator := &InMemoryTransactionCoordinator{
		transactions: make(map[string]*DistributedTransaction),
		handlers:     make(map[string]TransactionParticipantHandler),
		lockManager:  lockManager,
		storeID:      storeID,
		cleanupCh:    make(chan struct{}),
	}
	
	// 启动清理超时事务的goroutine
	go coordinator.cleanupTimeoutTransactions()
	
	return coordinator
}

// RegisterHandler 注册事务参与者处理器
func (c *InMemoryTransactionCoordinator) RegisterHandler(storeID string, handler TransactionParticipantHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[storeID] = handler
}

// BeginTransaction 开始分布式事务
func (c *InMemoryTransactionCoordinator) BeginTransaction(ctx context.Context, participants []*TransactionParticipant, timeout time.Duration) (*DistributedTransaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// 生成事务ID
	txnID := fmt.Sprintf("%s_%d", c.storeID, time.Now().UnixNano())
	
	// 创建事务
	txn := &DistributedTransaction{
		TransactionID: txnID,
		CoordinatorID: c.storeID,
		Participants:  participants,
		Status:        TransactionStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Timeout:       timeout,
		Locks:         make([]string, 0),
	}
	
	// 初始化参与者状态
	for _, participant := range participants {
		participant.Status = TransactionStatusPending
	}
	
	// 获取必要的锁
	lockKeys := c.generateLockKeys(participants)
	for _, lockKey := range lockKeys {
		lock, err := c.lockManager.AcquireLock(ctx, lockKey, timeout)
		if err != nil {
			// 释放已获取的锁
			c.releaseLocks(ctx, txn.Locks)
			return nil, fmt.Errorf("failed to acquire lock %s: %w", lockKey, err)
		}
		txn.Locks = append(txn.Locks, lock.LockKey)
	}
	
	c.transactions[txnID] = txn
	return txn, nil
}

// PrepareTransaction 准备阶段
func (c *InMemoryTransactionCoordinator) PrepareTransaction(ctx context.Context, txnID string) error {
	c.mu.Lock()
	txn, exists := c.transactions[txnID]
	if !exists {
		c.mu.Unlock()
		return fmt.Errorf("transaction not found: %s", txnID)
	}
	c.mu.Unlock()
	
	txn.mu.Lock()
	defer txn.mu.Unlock()
	
	if txn.Status != TransactionStatusPending {
		return fmt.Errorf("transaction %s is not in pending status: %s", txnID, txn.Status)
	}
	
	// 检查是否超时
	if time.Since(txn.CreatedAt) > txn.Timeout {
		txn.Status = TransactionStatusTimeout
		return fmt.Errorf("transaction %s has timed out", txnID)
	}
	
	// 对所有参与者执行准备操作
	for _, participant := range txn.Participants {
		handler, exists := c.handlers[participant.StoreID]
		if !exists {
			participant.Status = TransactionStatusAborted
			participant.Error = fmt.Sprintf("handler not found for store %s", participant.StoreID)
			continue
		}
		
		if err := handler.Prepare(ctx, txnID, participant); err != nil {
			participant.Status = TransactionStatusAborted
			participant.Error = err.Error()
			return fmt.Errorf("prepare failed for participant %s: %w", participant.StoreID, err)
		}
		
		participant.Status = TransactionStatusPrepared
	}
	
	txn.Status = TransactionStatusPrepared
	txn.UpdatedAt = time.Now()
	return nil
}

// CommitTransaction 提交事务
func (c *InMemoryTransactionCoordinator) CommitTransaction(ctx context.Context, txnID string) error {
	c.mu.Lock()
	txn, exists := c.transactions[txnID]
	if !exists {
		c.mu.Unlock()
		return fmt.Errorf("transaction not found: %s", txnID)
	}
	c.mu.Unlock()
	
	txn.mu.Lock()
	defer txn.mu.Unlock()
	
	if txn.Status != TransactionStatusPrepared {
		return fmt.Errorf("transaction %s is not in prepared status: %s", txnID, txn.Status)
	}
	
	// 对所有参与者执行提交操作
	var commitErrors []error
	for _, participant := range txn.Participants {
		if participant.Status != TransactionStatusPrepared {
			continue
		}
		
		handler, exists := c.handlers[participant.StoreID]
		if !exists {
			commitErrors = append(commitErrors, fmt.Errorf("handler not found for store %s", participant.StoreID))
			continue
		}
		
		if err := handler.Commit(ctx, txnID, participant); err != nil {
			commitErrors = append(commitErrors, fmt.Errorf("commit failed for participant %s: %w", participant.StoreID, err))
			participant.Error = err.Error()
		} else {
			participant.Status = TransactionStatusCommitted
		}
	}
	
	if len(commitErrors) > 0 {
		txn.Status = TransactionStatusAborted
		return fmt.Errorf("commit failed with %d errors: %v", len(commitErrors), commitErrors)
	}
	
	txn.Status = TransactionStatusCommitted
	txn.UpdatedAt = time.Now()
	
	// 释放锁
	c.releaseLocks(ctx, txn.Locks)
	txn.Locks = nil
	
	return nil
}

// AbortTransaction 回滚事务
func (c *InMemoryTransactionCoordinator) AbortTransaction(ctx context.Context, txnID string) error {
	c.mu.Lock()
	txn, exists := c.transactions[txnID]
	if !exists {
		c.mu.Unlock()
		return fmt.Errorf("transaction not found: %s", txnID)
	}
	c.mu.Unlock()
	
	txn.mu.Lock()
	defer txn.mu.Unlock()
	
	if txn.Status == TransactionStatusCommitted {
		return fmt.Errorf("cannot abort committed transaction: %s", txnID)
	}
	
	// 对所有参与者执行回滚操作
	for _, participant := range txn.Participants {
		if participant.Status == TransactionStatusPending {
			continue
		}
		
		handler, exists := c.handlers[participant.StoreID]
		if !exists {
			continue
		}
		
		if err := handler.Abort(ctx, txnID, participant); err != nil {
			participant.Error = err.Error()
		}
		
		participant.Status = TransactionStatusAborted
	}
	
	txn.Status = TransactionStatusAborted
	txn.UpdatedAt = time.Now()
	
	// 释放锁
	c.releaseLocks(ctx, txn.Locks)
	txn.Locks = nil
	
	return nil
}

// GetTransactionStatus 获取事务状态
func (c *InMemoryTransactionCoordinator) GetTransactionStatus(ctx context.Context, txnID string) (*DistributedTransaction, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	txn, exists := c.transactions[txnID]
	if !exists {
		return nil, fmt.Errorf("transaction not found: %s", txnID)
	}
	
	txn.mu.RLock()
	defer txn.mu.RUnlock()
	
	// 返回事务的副本
	participants := make([]*TransactionParticipant, len(txn.Participants))
	for i, p := range txn.Participants {
		participants[i] = &TransactionParticipant{
			StoreID:   p.StoreID,
			Operation: p.Operation,
			Params:    p.Params,
			Status:    p.Status,
			Error:     p.Error,
		}
	}
	
	return &DistributedTransaction{
		TransactionID: txn.TransactionID,
		CoordinatorID: txn.CoordinatorID,
		Participants:  participants,
		Status:        txn.Status,
		CreatedAt:     txn.CreatedAt,
		UpdatedAt:     txn.UpdatedAt,
		Timeout:       txn.Timeout,
		Locks:         append([]string(nil), txn.Locks...),
	}, nil
}

// CleanupTimeoutTransactions 清理超时事务
func (c *InMemoryTransactionCoordinator) CleanupTimeoutTransactions(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	var timeoutTxns []string
	
	for txnID, txn := range c.transactions {
		txn.mu.RLock()
		if txn.Status == TransactionStatusPending || txn.Status == TransactionStatusPrepared {
			if now.Sub(txn.CreatedAt) > txn.Timeout {
				timeoutTxns = append(timeoutTxns, txnID)
			}
		}
		txn.mu.RUnlock()
	}
	
	// 回滚超时事务
	for _, txnID := range timeoutTxns {
		if err := c.AbortTransaction(ctx, txnID); err != nil {
			fmt.Printf("Warning: failed to abort timeout transaction %s: %v\n", txnID, err)
		}
	}
	
	return nil
}

// generateLockKeys 生成锁键
func (c *InMemoryTransactionCoordinator) generateLockKeys(participants []*TransactionParticipant) []string {
	lockKeySet := make(map[string]bool)
	
	for _, participant := range participants {
		switch participant.Operation {
		case OpCreateTimeline, OpDeleteTimeline:
			if timelineKey, ok := participant.Params["timeline_key"].(string); ok {
				lockKeySet[fmt.Sprintf("timeline:%s", timelineKey)] = true
			}
		case OpAddMessage:
			if timelineKey, ok := participant.Params["timeline_key"].(string); ok {
				lockKeySet[fmt.Sprintf("timeline:%s:messages", timelineKey)] = true
			}
		case OpMigrateTimeline:
			if timelineKey, ok := participant.Params["timeline_key"].(string); ok {
				lockKeySet[fmt.Sprintf("timeline:%s:migrate", timelineKey)] = true
			}
		case OpUpdateIndex:
			if indexKey, ok := participant.Params["index_key"].(string); ok {
				lockKeySet[fmt.Sprintf("index:%s", indexKey)] = true
			}
		}
	}
	
	lockKeys := make([]string, 0, len(lockKeySet))
	for key := range lockKeySet {
		lockKeys = append(lockKeys, key)
	}
	
	return lockKeys
}

// releaseLocks 释放锁
func (c *InMemoryTransactionCoordinator) releaseLocks(ctx context.Context, lockKeys []string) {
	for _, lockKey := range lockKeys {
		lockInfo, err := c.lockManager.GetLockInfo(ctx, lockKey)
		if err != nil {
			continue
		}
		
		lock := &DistributedLock{
			LockKey: lockInfo.LockKey,
			LockID:  lockInfo.LockID,
			OwnerID: lockInfo.OwnerID,
			StoreID: lockInfo.StoreID,
		}
		
		if err := c.lockManager.ReleaseLock(ctx, lock); err != nil {
			fmt.Printf("Warning: failed to release lock %s: %v\n", lockKey, err)
		}
	}
}

// cleanupTimeoutTransactions 清理超时事务的goroutine
func (c *InMemoryTransactionCoordinator) cleanupTimeoutTransactions() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := c.CleanupTimeoutTransactions(context.Background()); err != nil {
				fmt.Printf("Warning: failed to cleanup timeout transactions: %v\n", err)
			}
		case <-c.cleanupCh:
			return
		}
	}
}

// Close 关闭事务协调器
func (c *InMemoryTransactionCoordinator) Close() {
	close(c.cleanupCh)
}

// 事务便利方法

// ExecuteTransaction 执行分布式事务
func ExecuteTransaction(ctx context.Context, coordinator TransactionCoordinator, participants []*TransactionParticipant, timeout time.Duration) error {
	// 开始事务
	txn, err := coordinator.BeginTransaction(ctx, participants, timeout)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	// 准备阶段
	if err := coordinator.PrepareTransaction(ctx, txn.TransactionID); err != nil {
		// 准备失败，回滚事务
		if abortErr := coordinator.AbortTransaction(ctx, txn.TransactionID); abortErr != nil {
			fmt.Printf("Warning: failed to abort transaction %s: %v\n", txn.TransactionID, abortErr)
		}
		return fmt.Errorf("transaction prepare failed: %w", err)
	}
	
	// 提交阶段
	if err := coordinator.CommitTransaction(ctx, txn.TransactionID); err != nil {
		// 提交失败，回滚事务
		if abortErr := coordinator.AbortTransaction(ctx, txn.TransactionID); abortErr != nil {
			fmt.Printf("Warning: failed to abort transaction %s: %v\n", txn.TransactionID, abortErr)
		}
		return fmt.Errorf("transaction commit failed: %w", err)
	}
	
	return nil
}