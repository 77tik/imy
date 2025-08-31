package storage

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// StoreConfig Store配置
type StoreConfig struct {
	MaxCapacity     int64  // Store最大容量（字节）
	TimelineMaxSize int64  // Timeline块最大大小（消息数量）
	DataDir         string // 数据目录
}

// StoreIndex Store索引信息
type StoreIndex struct {
	StoreID   string `json:"store_id"`   // 标识在哪个store
	Offset    int64  `json:"offset"`     // store中的偏移，精准找到timelineBlock
	Size      int64  `json:"size"`       // 该timelineBlock的大小
	CreatedAt int64  `json:"created_at"` // 创建时间戳
}

// TimelineBlock Timeline块信息
type TimelineBlock struct {
	BlockID   string         `json:"block_id"`
	StoreID   string         `json:"store_id"`
	Offset    int64          `json:"offset"`
	Size      int64          `json:"size"`
	Messages  []*Message     `json:"-"` // 内存中的消息缓存
	IsFull    bool           `json:"is_full"`
	NextBlock *TimelineBlock `json:"-"` // 下一个块的引用
	mu        sync.RWMutex
}

// Store 管理所有的 Timeline
type Store struct {
	Config          *StoreConfig // Store配置
	StoreID         string       // 当前Store ID
	CurrentCapacity int64        // 当前已使用容量
	// 会话存储库：ConvID -> Timeline
	ConvTimelines map[string]*Timeline
	// 用户同步库：UserID -> Timeline
	UserTimelines map[string]*Timeline
	// 用户 checkpoint：UserID -> SeqID
	UserCheckpoints map[string]int64
	StoreIndex      map[string][]*StoreIndex  // Timeline的Store索引，一个Timeline可能由位于不同store的tblock组成
	TimelineBlocks  map[string]*TimelineBlock // Timeline块缓存
	// 全局序列号生成器
	seqGenerator int64
	// 读写锁
	mu sync.RWMutex
}

// Timeline 时间线存储
type Timeline struct {
	ID           string           `json:"id"`
	Type         string           `json:"type"`   // "conv" 或 "user"
	Blocks       []*TimelineBlock `json:"blocks"` // Timeline块列表
	CurrentBlock *TimelineBlock   `json:"-"`      // 当前活跃块
	LastSeqID    int64            `json:"last_seq_id"`
	mu           sync.RWMutex
}

// Message 消息结构
type Message struct {
	SeqID      int64     `json:"seq_id"`
	ConvID     string    `json:"conv_id"`
	SenderID   uint32    `json:"sender_id"`
	CreateTime time.Time `json:"create_time"`
	Data       []byte    `json:"data"`
}

// NewStore 创建新的存储实例
func NewStore(config *StoreConfig) (*Store, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, err
	}

	// 生成Store ID
	storeID := fmt.Sprintf("store_%d", time.Now().UnixNano())

	return &Store{
		Config:          config,
		StoreID:         storeID,
		CurrentCapacity: 0,
		ConvTimelines:   make(map[string]*Timeline),
		UserTimelines:   make(map[string]*Timeline),
		UserCheckpoints: make(map[string]int64),
		StoreIndex:      make(map[string][]*StoreIndex),
		TimelineBlocks:  make(map[string]*TimelineBlock),
		seqGenerator:    0,
	}, nil
}

// NextSeqID 生成下一个序列号
func (s *Store) NextSeqID() int64 {
	return atomic.AddInt64(&s.seqGenerator, 1)
}

// GetOrCreateConvTimeline 获取或创建会话时间线
func (s *Store) GetOrCreateConvTimeline(convID string) *Timeline {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tl, exists := s.ConvTimelines[convID]; exists {
		return tl
	}

	tl := &Timeline{
		ID:        convID,
		Type:      "conv",
		Blocks:    make([]*TimelineBlock, 0),
		LastSeqID: 0,
	}

	// 尝试从文件加载
	s.loadTimeline(tl)

	s.ConvTimelines[convID] = tl
	return tl
}

// GetOrCreateUserTimeline 获取或创建用户时间线
func (s *Store) GetOrCreateUserTimeline(userID string) *Timeline {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tl, exists := s.UserTimelines[userID]; exists {
		return tl
	}

	tl := &Timeline{
		ID:        userID,
		Type:      "user",
		Blocks:    make([]*TimelineBlock, 0),
		LastSeqID: 0,
	}

	// 尝试从文件加载
	s.loadTimeline(tl)

	s.UserTimelines[userID] = tl
	return tl
}

// AddMessage 添加消息到会话和相关用户的时间线
func (s *Store) AddMessage(convID string, senderID uint32, data []byte, userIDs []string) error {
	seqID := s.NextSeqID()
	msg := &Message{
		SeqID:      seqID,
		ConvID:     convID,
		SenderID:   senderID,
		CreateTime: time.Now(),
		Data:       data,
	}

	// 添加到会话时间线
	convTL := s.GetOrCreateConvTimeline(convID)
	if err := convTL.AddMessage(msg, s); err != nil {
		return err
	}

	// 添加到所有相关用户的时间线
	for _, userID := range userIDs {
		userTL := s.GetOrCreateUserTimeline(userID)
		if err := userTL.AddMessage(msg, s); err != nil {
			return err
		}
	}

	// 持久化Timeline元数据
	if err := s.saveTimelineMetadata(convTL); err != nil {
		return err
	}

	for _, userID := range userIDs {
		userTL := s.GetOrCreateUserTimeline(userID)
		if err := s.saveTimelineMetadata(userTL); err != nil {
			return err
		}
	}

	return nil
}

// GetUserCheckpoint 获取用户的 checkpoint
func (s *Store) GetUserCheckpoint(userID string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.UserCheckpoints[userID]
}

// UpdateUserCheckpoint 更新用户的 checkpoint
func (s *Store) UpdateUserCheckpoint(userID string, seqID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.UserCheckpoints[userID] = seqID
}

// GetMessagesAfterCheckpoint 获取用户 checkpoint 之后的消息
func (s *Store) GetMessagesAfterCheckpoint(userID string) ([]*Message, error) {
	checkpoint := s.GetUserCheckpoint(userID)
	userTL := s.GetOrCreateUserTimeline(userID)

	userTL.mu.RLock()
	defer userTL.mu.RUnlock()

	var result []*Message
	// 遍历所有块获取消息
	for _, block := range userTL.Blocks {
		block.mu.RLock()
		for _, msg := range block.Messages {
			if msg.SeqID > checkpoint {
				result = append(result, msg)
			}
		}
		block.mu.RUnlock()
	}

	return result, nil
}

// GetConvMessages 获取会话的历史消息（分页）
func (s *Store) GetConvMessages(convID string, limit int, beforeSeqID int64) ([]*Message, error) {
	convTL := s.GetOrCreateConvTimeline(convID)

	convTL.mu.RLock()
	defer convTL.mu.RUnlock()

	var result []*Message
	count := 0

	// 收集所有消息
	var allMessages []*Message
	for _, block := range convTL.Blocks {
		block.mu.RLock()
		allMessages = append(allMessages, block.Messages...)
		block.mu.RUnlock()
	}

	// 从后往前遍历，获取最新的消息
	for i := len(allMessages) - 1; i >= 0 && count < limit; i-- {
		msg := allMessages[i]
		if beforeSeqID == 0 || msg.SeqID < beforeSeqID {
			result = append([]*Message{msg}, result...) // 保持时间顺序
			count++
		}
	}

	return result, nil
}

// AddMessage 向时间线添加消息
func (tl *Timeline) AddMessage(msg *Message, store *Store) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	// 如果没有当前块或当前块已满，创建新块
	if tl.CurrentBlock == nil || tl.CurrentBlock.IsFull {
		if err := tl.createNewBlock(store); err != nil {
			return err
		}
	}

	// 添加消息到当前块
	tl.CurrentBlock.mu.Lock()
	tl.CurrentBlock.Messages = append(tl.CurrentBlock.Messages, msg)
	tl.CurrentBlock.Size++

	// 检查块是否已满
	var blockToSave *TimelineBlock
	if tl.CurrentBlock.Size >= store.Config.TimelineMaxSize {
		tl.CurrentBlock.IsFull = true
		blockToSave = tl.CurrentBlock
	}
	tl.CurrentBlock.mu.Unlock()

	// 在释放Timeline锁之前保存需要持久化的块
	if blockToSave != nil {
		// 临时释放Timeline锁来避免死锁
		tl.mu.Unlock()
		if err := store.saveTimelineBlock(blockToSave); err != nil {
			tl.mu.Lock() // 重新获取锁以保持defer的一致性
			return err
		}
		tl.mu.Lock() // 重新获取锁
	}

	tl.LastSeqID = msg.SeqID
	return nil
}

// createNewBlock 创建新的Timeline块
func (tl *Timeline) createNewBlock(store *Store) error {
	// 生成块ID
	blockID := fmt.Sprintf("%s_%s_%d", tl.Type, tl.ID, time.Now().UnixNano())

	// 检查Store容量
	if store.CurrentCapacity >= store.Config.MaxCapacity {
		return fmt.Errorf("store capacity exceeded")
	}

	// 创建新块
	newBlock := &TimelineBlock{
		BlockID:  blockID,
		StoreID:  store.StoreID,
		Offset:   store.CurrentCapacity,
		Size:     0,
		Messages: make([]*Message, 0),
		IsFull:   false,
	}

	// 如果有当前块，建立链接
	if tl.CurrentBlock != nil {
		tl.CurrentBlock.NextBlock = newBlock
	}

	// 更新Timeline
	tl.Blocks = append(tl.Blocks, newBlock)
	tl.CurrentBlock = newBlock

	// 更新Store索引
	storeIndex := &StoreIndex{
		StoreID:   store.StoreID,
		Offset:    newBlock.Offset,
		Size:      0,
		CreatedAt: time.Now().Unix(),
	}

	timelineKey := fmt.Sprintf("%s_%s", tl.Type, tl.ID)
	store.StoreIndex[timelineKey] = append(store.StoreIndex[timelineKey], storeIndex)
	store.TimelineBlocks[blockID] = newBlock

	return nil
}

// 元数据文件路径生成
func (s *Store) getTimelineMetaFilePath(tl *Timeline) string {
	filename := fmt.Sprintf("%s_%s.meta", tl.Type, tl.ID)
	return filepath.Join(s.Config.DataDir, filename)
}

// Store文件路径生成
func (s *Store) getStoreFilePath() string {
	filename := fmt.Sprintf("%s.store", s.StoreID)
	return filepath.Join(s.Config.DataDir, filename)
}

// Timeline块文件路径生成
func (s *Store) getTimelineBlockFilePath(blockID string) string {
	filename := fmt.Sprintf("block_%s.gob", blockID)
	return filepath.Join(s.Config.DataDir, filename)
}

// saveTimelineBlock 保存Timeline块到文件
func (s *Store) saveTimelineBlock(block *TimelineBlock) error {
	block.mu.RLock()
	defer block.mu.RUnlock()

	filePath := s.getTimelineBlockFilePath(block.BlockID)

	// 打开文件进行写入
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建gob编码器
	encoder := gob.NewEncoder(file)

	// 保存所有消息
	for _, msg := range block.Messages {
		if err := encoder.Encode(msg); err != nil {
			return err
		}
	}

	// 更新Store容量
	s.CurrentCapacity += block.Size

	return nil
}

// loadTimelineBlock 从文件加载Timeline块
func (s *Store) loadTimelineBlock(blockID string) (*TimelineBlock, error) {
	filePath := s.getTimelineBlockFilePath(blockID)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 文件不存在
		}
		return nil, err
	}
	defer file.Close()

	// 创建gob解码器
	decoder := gob.NewDecoder(file)

	var messages []*Message
	// 逐个解码消息
	for {
		var msg Message
		err := decoder.Decode(&msg)
		if err != nil {
			if err.Error() == "EOF" {
				break // 文件结束
			}
			return nil, err
		}

		messages = append(messages, &msg)
	}

	// 创建Timeline块
	block := &TimelineBlock{
		BlockID:  blockID,
		StoreID:  s.StoreID,
		Messages: messages,
		Size:     int64(len(messages)),
		IsFull:   true, // 从文件加载的块默认为已满
	}

	return block, nil
}

// saveTimeline 保存Timeline元数据（块架构下不再需要保存消息到单个文件）
func (s *Store) saveTimeline(tl *Timeline) error {
	// 在新架构下，消息保存在各个块中，这里只需要保存元数据
	return s.saveTimelineMetadata(tl)
}

// saveTimelineMetadata 保存时间线元数据
func (s *Store) saveTimelineMetadata(tl *Timeline) error {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	metadata := struct {
		ID        string   `json:"id"`
		Type      string   `json:"type"`
		LastSeqID int64    `json:"last_seq_id"`
		BlockIDs  []string `json:"block_ids"`
	}{
		ID:        tl.ID,
		Type:      tl.Type,
		LastSeqID: tl.LastSeqID,
		BlockIDs:  make([]string, 0),
	}

	// 收集所有块ID
	for _, block := range tl.Blocks {
		metadata.BlockIDs = append(metadata.BlockIDs, block.BlockID)
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	metaPath := s.getTimelineMetaFilePath(tl)
	return os.WriteFile(metaPath, data, 0644)
}

// loadTimeline 从文件加载时间线
func (s *Store) loadTimeline(tl *Timeline) error {
	// 先加载元数据
	if err := s.loadTimelineMetadata(tl); err != nil {
		return err
	}

	// 加载所有块
	return s.loadTimelineBlocks(tl)
}

// loadTimelineMetadata 加载时间线元数据
func (s *Store) loadTimelineMetadata(tl *Timeline) error {
	metaPath := s.getTimelineMetaFilePath(tl)

	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，使用默认值
		}
		return err
	}

	var metadata struct {
		ID        string   `json:"id"`
		Type      string   `json:"type"`
		LastSeqID int64    `json:"last_seq_id"`
		BlockIDs  []string `json:"block_ids"`
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	tl.LastSeqID = metadata.LastSeqID
	// 存储块ID信息，稍后用于加载块

	// 更新全局序列号生成器
	if metadata.LastSeqID > atomic.LoadInt64(&s.seqGenerator) {
		atomic.StoreInt64(&s.seqGenerator, metadata.LastSeqID)
	}

	return nil
}

// loadTimelineBlocks 加载时间线的所有块
func (s *Store) loadTimelineBlocks(tl *Timeline) error {
	// 从元数据中获取块ID列表
	metaPath := s.getTimelineMetaFilePath(tl)

	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在
		}
		return err
	}

	var metadata struct {
		BlockIDs []string `json:"block_ids"`
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	// 加载每个块
	for _, blockID := range metadata.BlockIDs {
		block, err := s.loadTimelineBlock(blockID)
		if err != nil {
			return err
		}
		if block != nil {
			tl.Blocks = append(tl.Blocks, block)
			s.TimelineBlocks[blockID] = block

			// 设置当前块（最后一个未满的块）
			if !block.IsFull {
				tl.CurrentBlock = block
			}
		}
	}

	return nil
}
