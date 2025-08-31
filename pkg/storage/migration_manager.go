package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MigrationStatus 迁移状态
type MigrationStatus string

const (
	MigrationPending    MigrationStatus = "pending"    // 等待中
	MigrationRunning    MigrationStatus = "running"    // 进行中
	MigrationCompleted  MigrationStatus = "completed"  // 已完成
	MigrationFailed     MigrationStatus = "failed"     // 失败
	MigrationCancelled  MigrationStatus = "cancelled"  // 已取消
)

// MigrationTask 迁移任务
type MigrationTask struct {
	ID           string          `json:"id"`
	TimelineKey  string          `json:"timeline_key"`
	SourceStore  string          `json:"source_store"`
	TargetStore  string          `json:"target_store"`
	Status       MigrationStatus `json:"status"`
	Progress     float64         `json:"progress"`     // 0.0 - 1.0
	StartTime    time.Time       `json:"start_time"`
	EndTime      *time.Time      `json:"end_time,omitempty"`
	Error        string          `json:"error,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// MigrationManager 迁移管理器接口
type MigrationManager interface {
	// StartMigration 开始迁移Timeline
	StartMigration(ctx context.Context, timelineKey, targetStoreID string) (*MigrationTask, error)
	
	// GetMigrationStatus 获取迁移状态
	GetMigrationStatus(ctx context.Context, taskID string) (*MigrationTask, error)
	
	// CancelMigration 取消迁移
	CancelMigration(ctx context.Context, taskID string) error
	
	// ListMigrations 列出迁移任务
	ListMigrations(ctx context.Context, status MigrationStatus) ([]*MigrationTask, error)
	
	// CleanupCompletedMigrations 清理已完成的迁移任务
	CleanupCompletedMigrations(ctx context.Context, olderThan time.Duration) error
}

// TimelineMigrationManager Timeline迁移管理器实现
type TimelineMigrationManager struct {
	mu                sync.RWMutex
	tasks             map[string]*MigrationTask
	localStore        *Store
	globalIndex       GlobalIndexManager
	rpcClientPool     *StoreRPCClientPool
	crossStoreAccess  *DistributedStoreAccessor
	lockManager       DistributedLockManager
	storeID           string
	runningTasks      map[string]context.CancelFunc // 正在运行的任务取消函数
}

// NewTimelineMigrationManager 创建Timeline迁移管理器
func NewTimelineMigrationManager(
	localStore *Store,
	globalIndex GlobalIndexManager,
	rpcClientPool *StoreRPCClientPool,
	crossStoreAccess *DistributedStoreAccessor,
	lockManager DistributedLockManager,
	storeID string,
) *TimelineMigrationManager {
	return &TimelineMigrationManager{
		tasks:            make(map[string]*MigrationTask),
		localStore:       localStore,
		globalIndex:      globalIndex,
		rpcClientPool:    rpcClientPool,
		crossStoreAccess: crossStoreAccess,
		lockManager:      lockManager,
		storeID:          storeID,
		runningTasks:     make(map[string]context.CancelFunc),
	}
}

// StartMigration 开始迁移Timeline
func (tmm *TimelineMigrationManager) StartMigration(ctx context.Context, timelineKey, targetStoreID string) (*MigrationTask, error) {
	// 获取当前Timeline位置
	location, err := tmm.globalIndex.GetTimelineLocation(ctx, timelineKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline location: %w", err)
	}
	
	// 从第一个块获取源Store ID
	if len(location.Blocks) == 0 {
		return nil, fmt.Errorf("timeline has no blocks: %s", timelineKey)
	}
	
	sourceStoreID := location.Blocks[0].StoreID
	if sourceStoreID == targetStoreID {
		return nil, fmt.Errorf("timeline is already on target store")
	}
	
	// 创建迁移任务
	taskID := fmt.Sprintf("migration_%s_%d", timelineKey, time.Now().UnixNano())
	task := &MigrationTask{
		ID:          taskID,
		TimelineKey: timelineKey,
		SourceStore: sourceStoreID,
		TargetStore: targetStoreID,
		Status:      MigrationPending,
		Progress:    0.0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	tmm.mu.Lock()
	tmm.tasks[taskID] = task
	tmm.mu.Unlock()
	
	// 启动异步迁移
	go tmm.executeMigration(ctx, task)
	
	return task, nil
}

// executeMigration 执行迁移
func (tmm *TimelineMigrationManager) executeMigration(parentCtx context.Context, task *MigrationTask) {
	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	
	tmm.mu.Lock()
	tmm.runningTasks[task.ID] = cancel
	tmm.mu.Unlock()
	
	defer func() {
		tmm.mu.Lock()
		delete(tmm.runningTasks, task.ID)
		tmm.mu.Unlock()
	}()
	
	// 更新任务状态为运行中
	tmm.updateTaskStatus(task.ID, MigrationRunning, 0.0, "")
	task.StartTime = time.Now()
	
	// 获取迁移锁
	lockKey := fmt.Sprintf("migration:%s", task.TimelineKey)
	err := WithLock(ctx, tmm.lockManager, lockKey, 30*time.Minute, func() error {
		return tmm.performMigration(ctx, task)
	})
	
	if err != nil {
		tmm.updateTaskStatus(task.ID, MigrationFailed, task.Progress, err.Error())
	} else {
		tmm.updateTaskStatus(task.ID, MigrationCompleted, 1.0, "")
	}
	
	now := time.Now()
	task.EndTime = &now
}

// performMigration 执行具体的迁移操作
func (tmm *TimelineMigrationManager) performMigration(ctx context.Context, task *MigrationTask) error {
	// 步骤1: 获取源Timeline数据 (20%)
	tmm.updateTaskStatus(task.ID, MigrationRunning, 0.1, "Getting source timeline")
	
	sourceTimeline, err := tmm.crossStoreAccess.GetTimeline(ctx, task.TimelineKey)
	if err != nil {
		return fmt.Errorf("failed to get source timeline: %w", err)
	}
	
	tmm.updateTaskStatus(task.ID, MigrationRunning, 0.2, "Got source timeline")
	
	// 步骤2: 在目标Store创建Timeline (40%)
	tmm.updateTaskStatus(task.ID, MigrationRunning, 0.3, "Creating target timeline")
	
	err = tmm.crossStoreAccess.CreateTimeline(ctx, task.TimelineKey, sourceTimeline.Type)
	if err != nil {
		return fmt.Errorf("failed to create target timeline: %w", err)
	}
	
	tmm.updateTaskStatus(task.ID, MigrationRunning, 0.4, "Created target timeline")
	
	// 步骤3: 迁移消息数据 (70%)
	tmm.updateTaskStatus(task.ID, MigrationRunning, 0.5, "Migrating messages")
	
	// 获取所有消息（这里简化处理，实际应该分批处理）
	messages, err := tmm.crossStoreAccess.GetMessages(ctx, task.TimelineKey, 0, time.Now().Unix(), 10000)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}
	
	// 批量添加消息到目标Store
	for i, msg := range messages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		err = tmm.crossStoreAccess.AddMessage(ctx, task.TimelineKey, msg.SenderID, msg.Data, nil)
		if err != nil {
			return fmt.Errorf("failed to add message %d: %w", i, err)
		}
		
		// 更新进度
		progress := 0.5 + 0.2*float64(i+1)/float64(len(messages))
		tmm.updateTaskStatus(task.ID, MigrationRunning, progress, fmt.Sprintf("Migrated %d/%d messages", i+1, len(messages)))
	}
	
	tmm.updateTaskStatus(task.ID, MigrationRunning, 0.7, "Messages migrated")
	
	// 步骤4: 更新全局索引 (90%)
	tmm.updateTaskStatus(task.ID, MigrationRunning, 0.8, "Updating global index")
	
	err = tmm.globalIndex.MigrateTimeline(ctx, task.TimelineKey, task.SourceStore, task.TargetStore)
	if err != nil {
		return fmt.Errorf("failed to update global index: %w", err)
	}
	
	tmm.updateTaskStatus(task.ID, MigrationRunning, 0.9, "Global index updated")
	
	// 步骤5: 清理源Store数据 (100%)
	tmm.updateTaskStatus(task.ID, MigrationRunning, 0.95, "Cleaning up source store")
	
	err = tmm.crossStoreAccess.DeleteTimeline(ctx, task.TimelineKey)
	if err != nil {
		// 记录警告但不失败，因为数据已经迁移成功
		fmt.Printf("Warning: failed to cleanup source timeline: %v\n", err)
	}
	
	tmm.updateTaskStatus(task.ID, MigrationRunning, 1.0, "Migration completed")
	return nil
}

// updateTaskStatus 更新任务状态
func (tmm *TimelineMigrationManager) updateTaskStatus(taskID string, status MigrationStatus, progress float64, message string) {
	tmm.mu.Lock()
	defer tmm.mu.Unlock()
	
	if task, exists := tmm.tasks[taskID]; exists {
		task.Status = status
		task.Progress = progress
		if message != "" {
			task.Error = message
		}
		task.UpdatedAt = time.Now()
	}
}

// GetMigrationStatus 获取迁移状态
func (tmm *TimelineMigrationManager) GetMigrationStatus(ctx context.Context, taskID string) (*MigrationTask, error) {
	tmm.mu.RLock()
	defer tmm.mu.RUnlock()
	
	task, exists := tmm.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("migration task not found: %s", taskID)
	}
	
	// 返回任务副本
	taskCopy := *task
	return &taskCopy, nil
}

// CancelMigration 取消迁移
func (tmm *TimelineMigrationManager) CancelMigration(ctx context.Context, taskID string) error {
	tmm.mu.Lock()
	defer tmm.mu.Unlock()
	
	task, exists := tmm.tasks[taskID]
	if !exists {
		return fmt.Errorf("migration task not found: %s", taskID)
	}
	
	if task.Status != MigrationRunning && task.Status != MigrationPending {
		return fmt.Errorf("cannot cancel migration in status: %s", task.Status)
	}
	
	// 取消正在运行的任务
	if cancel, exists := tmm.runningTasks[taskID]; exists {
		cancel()
	}
	
	task.Status = MigrationCancelled
	task.UpdatedAt = time.Now()
	now := time.Now()
	task.EndTime = &now
	
	return nil
}

// ListMigrations 列出迁移任务
func (tmm *TimelineMigrationManager) ListMigrations(ctx context.Context, status MigrationStatus) ([]*MigrationTask, error) {
	tmm.mu.RLock()
	defer tmm.mu.RUnlock()
	
	var result []*MigrationTask
	for _, task := range tmm.tasks {
		if status == "" || task.Status == status {
			taskCopy := *task
			result = append(result, &taskCopy)
		}
	}
	
	return result, nil
}

// CleanupCompletedMigrations 清理已完成的迁移任务
func (tmm *TimelineMigrationManager) CleanupCompletedMigrations(ctx context.Context, olderThan time.Duration) error {
	tmm.mu.Lock()
	defer tmm.mu.Unlock()
	
	cutoff := time.Now().Add(-olderThan)
	for taskID, task := range tmm.tasks {
		if (task.Status == MigrationCompleted || task.Status == MigrationFailed || task.Status == MigrationCancelled) &&
			task.UpdatedAt.Before(cutoff) {
			delete(tmm.tasks, taskID)
		}
	}
	
	return nil
}