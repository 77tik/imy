# AI功能模块

简洁够用的AI功能模块，基于go-resty实现HTTP客户端，包含prompt生成器、消息审核、聊天记录总结和翻译功能。

## 快速开始

```go
import "imy/pkg/ai"

// 创建客户端
client := ai.NewAIClient("https://api.openai.com/v1", "your-api-key", "gpt-3.5-turbo")

// 消息审核
moderator := ai.NewContentModerator(client)
result, _ := moderator.Moderate("需要审核的消息")

// 聊天记录总结
summarizer := ai.NewChatSummarizer(client)
messages := []ai.ChatMessage{
    {Role: "user", Content: "你好", Timestamp: time.Now()},
}
summary, _ := summarizer.Summarize(messages)

// 翻译
translator := ai.NewTranslator(client)
translated, _ := translator.Translate("Hello", "中文")
```

## 安装

```bash
go get imy/pkg/ai
```

## 功能

- **Prompt生成器**: 内置常用模板，支持自定义扩展
- **消息审核**: 检测不当内容，返回审核结果
- **聊天记录总结**: 智能总结聊天内容
- **翻译功能**: 支持多语言翻译

## 配置

```go
client := ai.NewAIClient(baseURL, apiKey, model)
client.SetModel("gpt-4")
client.SetTimeout(60 * time.Second)
```

## 使用示例

查看 `example_test.go` 文件获取完整使用示例。