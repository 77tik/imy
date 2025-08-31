package storage

import (
	"fmt"
	"testing"
)

func TestBlockStorageArchitecture(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	
	// 创建Store配置
	config := &StoreConfig{
		MaxCapacity:     1000,
		TimelineMaxSize: 3, // 每个块最多3条消息
		DataDir:         tempDir,
	}
	
	// 创建Store
	store, err := NewStore(config)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	
	// 测试基本的Timeline创建
	convID := "test_conv_1"
	convTimeline := store.GetOrCreateConvTimeline(convID)
	if convTimeline == nil {
		t.Fatal("Failed to create conv timeline")
	}
	
	if convTimeline.ID != convID {
		t.Errorf("Expected timeline ID %s, got %s", convID, convTimeline.ID)
	}
	
	if convTimeline.Type != "conv" {
		t.Errorf("Expected timeline type 'conv', got %s", convTimeline.Type)
	}
	
	// 测试用户Timeline创建
	userID := "user1"
	userTimeline := store.GetOrCreateUserTimeline(userID)
	if userTimeline == nil {
		t.Fatal("Failed to create user timeline")
	}
	
	if userTimeline.ID != userID {
		t.Errorf("Expected timeline ID %s, got %s", userID, userTimeline.ID)
	}
	
	if userTimeline.Type != "user" {
		t.Errorf("Expected timeline type 'user', got %s", userTimeline.Type)
	}
	
	// 测试checkpoint功能
	checkpoint := store.GetUserCheckpoint("user1")
	if checkpoint != 0 {
		t.Errorf("Initial checkpoint should be 0, got %d", checkpoint)
	}
	
	store.UpdateUserCheckpoint("user1", 3)
	checkpoint = store.GetUserCheckpoint("user1")
	if checkpoint != 3 {
		t.Errorf("Updated checkpoint should be 3, got %d", checkpoint)
	}
	
	t.Logf("Basic block storage architecture test passed successfully!")
}

func TestBlockPersistence(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	
	// 创建Store配置
	config := &StoreConfig{
		MaxCapacity:     1000,
		TimelineMaxSize: 2, // 每个块最多2条消息
		DataDir:         tempDir,
	}
	
	// 创建Store并添加消息
	store, err := NewStore(config)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	
	convID := "test_conv_persist"
	userIDs := []string{"user1"}
	
	// 添加3条消息，应该创建2个块（2+1）
	for i := 0; i < 3; i++ {
		data := []byte(fmt.Sprintf("persist message %d", i+1))
		err := store.AddMessage(convID, 1001, data, userIDs)
		if err != nil {
			t.Fatalf("Failed to add message %d: %v", i+1, err)
		}
	}
	
	// 保存时间线元数据
	convTimeline := store.GetOrCreateConvTimeline(convID)
	err = store.saveTimelineMetadata(convTimeline)
	if err != nil {
		t.Fatalf("Failed to save timeline metadata: %v", err)
	}
	
	// 创建新的Store实例来测试加载
	newStore, err := NewStore(config)
	if err != nil {
		t.Fatalf("Failed to create new store: %v", err)
	}
	
	// 创建新的时间线并加载数据
	newTimeline := &Timeline{
		ID:     convID,
		Type:   "conv",
		Blocks: make([]*TimelineBlock, 0),
	}
	
	err = newStore.loadTimeline(newTimeline)
	if err != nil {
		t.Fatalf("Failed to load timeline: %v", err)
	}
	
	// 验证加载的数据
	if len(newTimeline.Blocks) != 2 {
		t.Errorf("Expected 2 blocks after loading, got %d", len(newTimeline.Blocks))
	}
	
	// 验证第一个块的消息
	if len(newTimeline.Blocks[0].Messages) != 2 {
		t.Errorf("First block should have 2 messages, got %d", len(newTimeline.Blocks[0].Messages))
	}
	
	// 验证消息内容
	for i, msg := range newTimeline.Blocks[0].Messages {
		expected := fmt.Sprintf("persist message %d", i+1)
		if string(msg.Data) != expected {
			t.Errorf("Block 0 Message %d: expected %s, got %s", i, expected, string(msg.Data))
		}
	}
	
	t.Logf("Block persistence test passed successfully!")
}