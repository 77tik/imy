package storage

import (
	"fmt"
	"log"
	"os"
	"time"
)

// 模拟一个完整的即时通讯应用场景
func RunUsageScenarios() {
	fmt.Println("=== 即时通讯应用使用场景演示 ===")
	
	// 初始化存储系统
	store := initializeStorage()
	
	// 场景1: 群聊消息发送
	fmt.Println("\n📱 场景1: 群聊消息发送")
	groupChatScenario(store)
	
	// 场景2: 用户上线获取未读消息
	fmt.Println("\n🔔 场景2: 用户上线获取未读消息")
	userOnlineScenario(store)
	
	// 场景3: 查看历史消息
	fmt.Println("\n📜 场景3: 查看历史消息")
	historyMessageScenario(store)
	
	// 场景4: 私聊消息
	fmt.Println("\n💬 场景4: 私聊消息")
	privateChatScenario(store)
	
	// 场景5: 消息已读状态管理
	fmt.Println("\n✅场景5: 消息已读状态管理")
	readStatusScenario(store)
	
	// 场景6: 系统消息推送
	fmt.Println("\n🔊 场景6: 系统消息推送")
	systemMessageScenario(store)
	
	fmt.Println("\n=== 演示完成 ===")
}

// 初始化存储系统
func initializeStorage() *Store {
	fmt.Println("初始化存储系统...")
	
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "imy_usage_demo")
	if err != nil {
		log.Fatalf("创建临时目录失败: %v", err)
	}
	
	// 配置存储参数
	config := &StoreConfig{
		MaxCapacity:     50000,  // 50KB Store容量
		TimelineMaxSize: 5,      // 每个块最多5条消息
		DataDir:         tempDir,
	}
	
	// 创建Store实例
	store, err := NewStore(config)
	if err != nil {
		log.Fatalf("创建Store失败: %v", err)
	}
	
	fmt.Printf("✓ 存储系统初始化完成，数据目录: %s\n", tempDir)
	return store
}

// 场景1: 群聊消息发送
func groupChatScenario(store *Store) {
	groupID := "tech_team_group"
	members := []string{"alice", "bob", "charlie", "david"}
	
	fmt.Printf("群聊ID: %s, 成员: %v\n", groupID, members)
	
	// 模拟多个用户发送消息
	messages := []struct {
		sender  uint32
		content string
	}{
		{1001, "大家好，今天的技术分享会几点开始？"},
		{1002, "下午2点，会议室A"},
		{1003, "好的，我会准时参加"},
		{1004, "我可能会晚到10分钟"},
		{1001, "没问题，我们等你"},
	}
	
	for i, msg := range messages {
		fmt.Printf("  [%d] 用户%d发送: %s\n", i+1, msg.sender, msg.content)
		
		err := store.AddMessage(groupID, msg.sender, []byte(msg.content), members)
		if err != nil {
			log.Printf("发送消息失败: %v", err)
			continue
		}
		
		// 模拟消息发送间隔
		time.Sleep(100 * time.Millisecond)
	}
	
	// 查看群聊Timeline状态
	groupTimeline := store.GetOrCreateConvTimeline(groupID)
	fmt.Printf("✓ 群聊消息发送完成，共创建了 %d 个消息块\n", len(groupTimeline.Blocks))
}

// 场景2: 用户上线获取未读消息
func userOnlineScenario(store *Store) {
	userID := "alice"
	
	fmt.Printf("用户 %s 上线...\n", userID)
	
	// 1. 获取当前检查点
	checkpoint := store.GetUserCheckpoint(userID)
	fmt.Printf("  当前检查点: %d\n", checkpoint)
	
	// 2. 获取未读消息
	unreadMessages, err := store.GetMessagesAfterCheckpoint(userID)
	if err != nil {
		log.Printf("获取未读消息失败: %v", err)
		return
	}
	
	fmt.Printf("  未读消息数量: %d\n", len(unreadMessages))
	
	// 3. 显示未读消息
	for i, msg := range unreadMessages {
		fmt.Printf("    [未读%d] SeqID:%d, 会话:%s, 发送者:%d, 内容:%s\n", 
			i+1, msg.SeqID, msg.ConvID, msg.SenderID, string(msg.Data))
	}
	
	// 4. 模拟用户阅读消息，更新检查点
	if len(unreadMessages) > 0 {
		lastSeqID := unreadMessages[len(unreadMessages)-1].SeqID
		store.UpdateUserCheckpoint(userID, lastSeqID)
		fmt.Printf("✓ 检查点已更新到: %d\n", lastSeqID)
	}
}

// 场景3: 查看历史消息
func historyMessageScenario(store *Store) {
	groupID := "tech_team_group"
	
	fmt.Printf("查看群聊 %s 的历史消息...\n", groupID)
	
	// 1. 获取最新的3条消息
	recentMessages, err := store.GetConvMessages(groupID, 3, 0)
	if err != nil {
		log.Printf("获取最新消息失败: %v", err)
		return
	}
	
	fmt.Printf("  最新的 %d 条消息:\n", len(recentMessages))
	for i, msg := range recentMessages {
		fmt.Printf("    [最新%d] SeqID:%d, 发送者:%d, 时间:%s, 内容:%s\n", 
			i+1, msg.SeqID, msg.SenderID, 
			msg.CreateTime.Format("15:04:05"), string(msg.Data))
	}
	
	// 2. 向前翻页，获取更早的消息
	if len(recentMessages) > 0 {
		earliest := recentMessages[0].SeqID
		olderMessages, err := store.GetConvMessages(groupID, 2, earliest)
		if err != nil {
			log.Printf("获取历史消息失败: %v", err)
			return
		}
		
		fmt.Printf("  更早的 %d 条消息:\n", len(olderMessages))
		for i, msg := range olderMessages {
			fmt.Printf("    [历史%d] SeqID:%d, 发送者:%d, 时间:%s, 内容:%s\n", 
				i+1, msg.SeqID, msg.SenderID, 
				msg.CreateTime.Format("15:04:05"), string(msg.Data))
		}
	}
	
	fmt.Println("✓ 历史消息查看完成")
}

// 场景4: 私聊消息
func privateChatScenario(store *Store) {
	privateChatID := "private_alice_bob"
	participants := []string{"alice", "bob"}
	
	fmt.Printf("私聊会话: %s, 参与者: %v\n", privateChatID, participants)
	
	// 模拟私聊对话
	privateMessages := []struct {
		sender  uint32
		content string
	}{
		{1001, "Hi Bob, 有空聊聊吗？"},
		{1002, "当然，什么事？"},
		{1001, "关于明天的项目演示，你准备得怎么样了？"},
		{1002, "基本准备好了，还有一些细节需要完善"},
		{1001, "需要我帮忙吗？"},
		{1002, "谢谢，如果有问题我会找你的"},
	}
	
	for i, msg := range privateMessages {
		fmt.Printf("  [私聊%d] 用户%d: %s\n", i+1, msg.sender, msg.content)
		
		err := store.AddMessage(privateChatID, msg.sender, []byte(msg.content), participants)
		if err != nil {
			log.Printf("发送私聊消息失败: %v", err)
			continue
		}
		
		time.Sleep(50 * time.Millisecond)
	}
	
	fmt.Println("✓ 私聊消息发送完成")
}

// 场景5: 消息已读状态管理
func readStatusScenario(store *Store) {
	userID := "bob"
	
	fmt.Printf("管理用户 %s 的消息已读状态...\n", userID)
	
	// 1. 查看当前检查点
	currentCheckpoint := store.GetUserCheckpoint(userID)
	fmt.Printf("  当前检查点: %d\n", currentCheckpoint)
	
	// 2. 获取未读消息
	unreadMessages, err := store.GetMessagesAfterCheckpoint(userID)
	if err != nil {
		log.Printf("获取未读消息失败: %v", err)
		return
	}
	
	fmt.Printf("  未读消息数量: %d\n", len(unreadMessages))
	
	// 3. 模拟用户逐条阅读消息
	for i, msg := range unreadMessages {
		fmt.Printf("    正在阅读消息 %d: %s\n", i+1, string(msg.Data))
		
		// 模拟阅读时间
		time.Sleep(200 * time.Millisecond)
		
		// 每读完一条消息就更新检查点
		store.UpdateUserCheckpoint(userID, msg.SeqID)
		fmt.Printf("      ✓ 检查点更新到: %d\n", msg.SeqID)
	}
	
	// 4. 验证所有消息已读
	finalUnread, _ := store.GetMessagesAfterCheckpoint(userID)
	fmt.Printf("✓ 消息已读状态管理完成，剩余未读消息: %d\n", len(finalUnread))
}

// 场景6: 系统消息推送
func systemMessageScenario(store *Store) {
	systemConvID := "system_notifications"
	allUsers := []string{"alice", "bob", "charlie", "david"}
	
	fmt.Printf("系统消息推送到所有用户...\n")
	
	// 模拟系统消息
	systemMessages := []string{
		"系统维护通知：今晚23:00-01:00进行系统维护",
		"新功能上线：现在支持文件传输功能",
		"安全提醒：请定期更新您的密码",
	}
	
	for i, content := range systemMessages {
		fmt.Printf("  [系统消息%d] %s\n", i+1, content)
		
		// 系统消息使用特殊的发送者ID (0)
		err := store.AddMessage(systemConvID, 0, []byte(content), allUsers)
		if err != nil {
			log.Printf("发送系统消息失败: %v", err)
			continue
		}
		
		time.Sleep(100 * time.Millisecond)
	}
	
	// 检查系统消息是否正确添加到用户时间线
	userTimeline := store.GetOrCreateUserTimeline("alice")
	fmt.Printf("✓ 系统消息推送完成，用户alice的时间线共有 %d 个块\n", len(userTimeline.Blocks))
	
	// 显示用户alice收到的最新消息
	aliceMessages, err := store.GetMessagesAfterCheckpoint("alice")
	if err == nil {
		fmt.Printf("  用户alice收到的新消息数量: %d\n", len(aliceMessages))
		for _, msg := range aliceMessages {
			if msg.SenderID == 0 { // 系统消息
				fmt.Printf("    [系统] %s\n", string(msg.Data))
			}
		}
	}
}