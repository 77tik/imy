package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DistributedLockManager 分布式锁管理器接口
type DistributedLockManager interface {
	// 获取锁
	AcquireLock(ctx context.Context, lockKey string, ttl time.Duration) (*DistributedLock, error)
	// 释放锁
	ReleaseLock(ctx context.Context, lock *DistributedLock) error
	// 续期锁
	RenewLock(ctx context.Context, lock *DistributedLock, ttl time.Duration) error
	// 检查锁状态
	IsLocked(ctx context.Context, lockKey string) (bool, error)
	// 获取锁信息
	GetLockInfo(ctx context.Context, lockKey string) (*LockInfo, error)
}

// DistributedLock 分布式锁
type DistributedLock struct {
	LockKey   string    `json:"lock_key"`
	LockID    string    `json:"lock_id"`
	OwnerID   string    `json:"owner_id"`
	StoreID   string    `json:"store_id"`
	AcquiredAt time.Time `json:"acquired_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	TTL        time.Duration `json:"ttl"`
	manager    DistributedLockManager
}

// LockInfo 锁信息
type LockInfo struct {
	LockKey    string    `json:"lock_key"`
	LockID     string    `json:"lock_id"`
	OwnerID    string    `json:"owner_id"`
	StoreID    string    `json:"store_id"`
	AcquiredAt time.Time `json:"acquired_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	IsActive   bool      `json:"is_active"`
}

// InMemoryDistributedLockManager 内存分布式锁管理器实现
type InMemoryDistributedLockManager struct {
	locks     map[string]*LockInfo
	storeID   string
	mu        sync.RWMutex
	cleanupCh chan struct{}
}

// NewInMemoryDistributedLockManager 创建内存分布式锁管理器
func NewInMemoryDistributedLockManager(storeID string) *InMemoryDistributedLockManager {
	manager := &InMemoryDistributedLockManager{
		locks:     make(map[string]*LockInfo),
		storeID:   storeID,
		cleanupCh: make(chan struct{}),
	}
	
	// 启动清理过期锁的goroutine
	go manager.cleanupExpiredLocks()
	
	return manager
}

// AcquireLock 获取分布式锁
func (m *InMemoryDistributedLockManager) AcquireLock(ctx context.Context, lockKey string, ttl time.Duration) (*DistributedLock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 检查是否已存在锁
	if existingLock, exists := m.locks[lockKey]; exists {
		// 检查锁是否过期
		if time.Now().Before(existingLock.ExpiresAt) {
			return nil, fmt.Errorf("lock already acquired by %s", existingLock.OwnerID)
		}
		// 锁已过期，删除
		delete(m.locks, lockKey)
	}
	
	// 创建新锁
	lockID := fmt.Sprintf("%s_%d", m.storeID, time.Now().UnixNano())
	ownerID := fmt.Sprintf("%s_%d", m.storeID, time.Now().UnixNano())
	now := time.Now()
	
	lockInfo := &LockInfo{
		LockKey:    lockKey,
		LockID:     lockID,
		OwnerID:    ownerID,
		StoreID:    m.storeID,
		AcquiredAt: now,
		ExpiresAt:  now.Add(ttl),
		IsActive:   true,
	}
	
	m.locks[lockKey] = lockInfo
	
	return &DistributedLock{
		LockKey:    lockKey,
		LockID:     lockID,
		OwnerID:    ownerID,
		StoreID:    m.storeID,
		AcquiredAt: now,
		ExpiresAt:  now.Add(ttl),
		TTL:        ttl,
		manager:    m,
	}, nil
}

// ReleaseLock 释放分布式锁
func (m *InMemoryDistributedLockManager) ReleaseLock(ctx context.Context, lock *DistributedLock) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	existingLock, exists := m.locks[lock.LockKey]
	if !exists {
		return fmt.Errorf("lock not found: %s", lock.LockKey)
	}
	
	// 验证锁的所有者
	if existingLock.OwnerID != lock.OwnerID {
		return fmt.Errorf("lock owned by different owner: %s", existingLock.OwnerID)
	}
	
	// 删除锁
	delete(m.locks, lock.LockKey)
	return nil
}

// RenewLock 续期分布式锁
func (m *InMemoryDistributedLockManager) RenewLock(ctx context.Context, lock *DistributedLock, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	existingLock, exists := m.locks[lock.LockKey]
	if !exists {
		return fmt.Errorf("lock not found: %s", lock.LockKey)
	}
	
	// 验证锁的所有者
	if existingLock.OwnerID != lock.OwnerID {
		return fmt.Errorf("lock owned by different owner: %s", existingLock.OwnerID)
	}
	
	// 检查锁是否过期
	if time.Now().After(existingLock.ExpiresAt) {
		return fmt.Errorf("lock has expired")
	}
	
	// 续期锁
	now := time.Now()
	existingLock.ExpiresAt = now.Add(ttl)
	lock.ExpiresAt = now.Add(ttl)
	lock.TTL = ttl
	
	return nil
}

// IsLocked 检查是否被锁定
func (m *InMemoryDistributedLockManager) IsLocked(ctx context.Context, lockKey string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	lockInfo, exists := m.locks[lockKey]
	if !exists {
		return false, nil
	}
	
	// 检查锁是否过期
	if time.Now().After(lockInfo.ExpiresAt) {
		return false, nil
	}
	
	return true, nil
}

// GetLockInfo 获取锁信息
func (m *InMemoryDistributedLockManager) GetLockInfo(ctx context.Context, lockKey string) (*LockInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	lockInfo, exists := m.locks[lockKey]
	if !exists {
		return nil, fmt.Errorf("lock not found: %s", lockKey)
	}
	
	// 检查锁是否过期
	if time.Now().After(lockInfo.ExpiresAt) {
		lockInfo.IsActive = false
	}
	
	// 返回锁信息的副本
	return &LockInfo{
		LockKey:    lockInfo.LockKey,
		LockID:     lockInfo.LockID,
		OwnerID:    lockInfo.OwnerID,
		StoreID:    lockInfo.StoreID,
		AcquiredAt: lockInfo.AcquiredAt,
		ExpiresAt:  lockInfo.ExpiresAt,
		IsActive:   lockInfo.IsActive,
	}, nil
}

// cleanupExpiredLocks 清理过期锁
func (m *InMemoryDistributedLockManager) cleanupExpiredLocks() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			now := time.Now()
			for key, lockInfo := range m.locks {
				if now.After(lockInfo.ExpiresAt) {
					delete(m.locks, key)
				}
			}
			m.mu.Unlock()
		case <-m.cleanupCh:
			return
		}
	}
}

// Close 关闭锁管理器
func (m *InMemoryDistributedLockManager) Close() {
	close(m.cleanupCh)
}

// DistributedLock 方法

// Release 释放锁
func (l *DistributedLock) Release(ctx context.Context) error {
	return l.manager.ReleaseLock(ctx, l)
}

// Renew 续期锁
func (l *DistributedLock) Renew(ctx context.Context, ttl time.Duration) error {
	return l.manager.RenewLock(ctx, l, ttl)
}

// IsExpired 检查锁是否过期
func (l *DistributedLock) IsExpired() bool {
	return time.Now().After(l.ExpiresAt)
}

// TimeToExpire 获取锁剩余时间
func (l *DistributedLock) TimeToExpire() time.Duration {
	remaining := time.Until(l.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// 锁的便利方法

// WithLock 使用锁执行函数
func WithLock(ctx context.Context, manager DistributedLockManager, lockKey string, ttl time.Duration, fn func() error) error {
	lock, err := manager.AcquireLock(ctx, lockKey, ttl)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	
	defer func() {
		if releaseErr := lock.Release(ctx); releaseErr != nil {
			// 记录释放锁失败的错误，但不影响主要逻辑
			fmt.Printf("Warning: failed to release lock %s: %v\n", lockKey, releaseErr)
		}
	}()
	
	return fn()
}

// TryWithLock 尝试获取锁并执行函数，如果获取失败则立即返回
func TryWithLock(ctx context.Context, manager DistributedLockManager, lockKey string, ttl time.Duration, fn func() error) error {
	locked, err := manager.IsLocked(ctx, lockKey)
	if err != nil {
		return fmt.Errorf("failed to check lock status: %w", err)
	}
	
	if locked {
		return fmt.Errorf("resource is locked: %s", lockKey)
	}
	
	return WithLock(ctx, manager, lockKey, ttl, fn)
}

// WithAutoRenewLock 使用自动续期的锁执行函数
func WithAutoRenewLock(ctx context.Context, manager DistributedLockManager, lockKey string, ttl time.Duration, renewInterval time.Duration, fn func() error) error {
	lock, err := manager.AcquireLock(ctx, lockKey, ttl)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	
	// 创建用于停止续期的context
	renewCtx, cancelRenew := context.WithCancel(ctx)
	defer cancelRenew()
	
	// 启动自动续期goroutine
	go func() {
		ticker := time.NewTicker(renewInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				if err := lock.Renew(renewCtx, ttl); err != nil {
					fmt.Printf("Warning: failed to renew lock %s: %v\n", lockKey, err)
					return
				}
			case <-renewCtx.Done():
				return
			}
		}
	}()
	
	defer func() {
		if releaseErr := lock.Release(ctx); releaseErr != nil {
			fmt.Printf("Warning: failed to release lock %s: %v\n", lockKey, releaseErr)
		}
	}()
	
	return fn()
}