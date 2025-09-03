package ai

import (
	"fmt"
	"time"
)

// ExampleBasicUsage 基础使用示例
func ExampleBasicUsage() {
	// 1. 创建AI客户端
	client := NewAIClient("https://api.openai.com/v1", "your-api-key", "gpt-3.5-turbo")
	
	// 2. 消息审核示例
	moderator := NewContentModerator(client)
	result, err := moderator.Moderate("你好，这是一个测试消息")
	if err != nil {
		fmt.Printf("审核错误: %v\n", err)
	} else {
		fmt.Printf("审核结果: %+v\n", result)
	}

	// 3. 聊天记录总结示例
	summarizer := NewChatSummarizer(client)
	messages := []ChatMessage{
		{Role: "user", Content: "你好，我想学习Go语言", Timestamp: time.Now()},
		{Role: "assistant", Content: "很高兴帮助你学习Go语言！我们可以从基础开始", Timestamp: time.Now()},
		{Role: "user", Content: "太好了，从变量开始吧", Timestamp: time.Now()},
	}
	summary, err := summarizer.Summarize(messages)
	if err != nil {
		fmt.Printf("总结错误: %v\n", err)
	} else {
		fmt.Printf("聊天记录总结: %s\n", summary.Summary)
	}

	// 4. 翻译示例
	translator := NewTranslator(client)
	translation, err := translator.Translate("Hello, world!", "中文")
	if err != nil {
		fmt.Printf("翻译错误: %v\n", err)
	} else {
		fmt.Printf("翻译结果: %s\n", translation.Translated)
	}
}

// ExampleQuickStart 快速开始示例
func ExampleQuickStart() {
	// 快速创建所有服务
	client := NewAIClient("https://api.openai.com/v1", "your-api-key", "gpt-3.5-turbo")
	
	// 使用内置功能
	moderator := NewContentModerator(client)
	translator := NewTranslator(client)
	
	// 快速审核
	result, _ := moderator.Moderate("这是一条测试消息")
	fmt.Printf("审核结果: %s\n", result.Action)
	
	// 快速翻译
	translated := translator.QuickTranslate("Hello", "中文")
	fmt.Printf("翻译结果: %s\n", translated)
}

// ExamplePromptBuilder prompt生成器使用示例
func ExamplePromptBuilder() {
	// 创建prompt生成器
	builder := NewDefaultPromptBuilder()
	
	// 使用内置模板
	prompt, err := builder.BuildPrompt("content_moderation", map[string]interface{}{
		"content": "需要审核的内容",
	})
	if err != nil {
		fmt.Printf("构建prompt错误: %v\n", err)
	} else {
		fmt.Printf("生成的prompt: %s\n", prompt)
	}
	
	// 添加自定义模板
	builder.AddTemplate("custom", "这是一个自定义模板: {{.data}}")
	customPrompt, _ := builder.BuildPrompt("custom", map[string]interface{}{
		"data": "测试数据",
	})
	fmt.Printf("自定义prompt: %s\n", customPrompt)
}

// ExampleBatchProcessing 批量处理示例
func ExampleBatchProcessing() {
	client := NewAIClient("https://api.openai.com/v1", "your-api-key", "gpt-3.5-turbo")
	moderator := NewContentModerator(client)
	
	// 批量审核
	contents := []string{"消息1", "消息2", "消息3"}
	results, err := moderator.BatchModerate(contents)
	if err != nil {
		fmt.Printf("批量审核错误: %v\n", err)
	} else {
		for i, result := range results {
			fmt.Printf("消息%d审核结果: %s\n", i+1, result.Action)
		}
	}
}

// ExampleConfig 配置示例
func ExampleConfig() {
	// 创建客户端
	client := NewAIClient("https://api.openai.com/v1", "your-api-key", "gpt-3.5-turbo")
	
	// 修改配置
	client.SetModel("gpt-4")
	client.SetTimeout(60 * time.Second)
	
	// 使用配置后的客户端
	translator := NewTranslator(client)
	result, _ := translator.Translate("Hello", "中文")
	fmt.Printf("使用gpt-4翻译: %s\n", result.Translated)
}

// ExampleIntegration 集成使用示例
func ExampleIntegration() {
	// 创建统一的AI服务
	client := NewAIClient("https://api.openai.com/v1", "your-api-key", "gpt-3.5-turbo")
	
	moderator := NewContentModerator(client)
	summarizer := NewChatSummarizer(client)
	translator := NewTranslator(client)
	
	// 模拟完整的工作流程
	messages := []ChatMessage{
		{Role: "user", Content: "Hello, how are you?", Timestamp: time.Now()},
		{Role: "assistant", Content: "I'm fine, thank you!", Timestamp: time.Now()},
	}
	
	// 1. 审核用户消息
	for _, msg := range messages {
		if msg.Role == "user" {
			result, _ := moderator.Moderate(msg.Content)
			fmt.Printf("消息审核: %s\n", result.Action)
		}
	}
	
	// 2. 总结聊天
	summary, _ := summarizer.Summarize(messages)
	fmt.Printf("聊天总结: %s\n", summary.Summary)
	
	// 3. 翻译总结
	translated, _ := translator.Translate(summary.Summary, "中文")
	fmt.Printf("翻译总结: %s\n", translated.Translated)
}