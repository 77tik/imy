package ai

import (
	"fmt"
	"strings"
	"time"
)

// ChatMessage 聊天消息
type ChatMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatSummary 聊天总结
type ChatSummary struct {
	Topic      string   `json:"topic"`
	KeyPoints  []string `json:"key_points"`
	Sentiment  string   `json:"sentiment"`
	NextAction string   `json:"next_action"`
	Summary    string   `json:"summary"`
}

// ChatSummarizer 聊天记录总结器
type ChatSummarizer struct {
	client *AIClient
	prompt *PromptBuilder
}

// NewChatSummarizer 创建聊天记录总结器
func NewChatSummarizer(client *AIClient) *ChatSummarizer {
	return &ChatSummarizer{
		client: client,
		prompt: NewDefaultPromptBuilder(),
	}
}

// Summarize 总结聊天记录
func (cs *ChatSummarizer) Summarize(messages []ChatMessage) (*ChatSummary, error) {
	if len(messages) == 0 {
		return &ChatSummary{
			Topic:      "无聊天记录",
			KeyPoints:  []string{},
			Sentiment:  "中性",
			NextAction: "开始对话",
			Summary:    "暂无聊天记录",
		}, nil
	}

	chatHistory := cs.formatChatHistory(messages)
	prompt, err := cs.prompt.BuildPrompt("chat_summary", map[string]interface{}{
		"chat_history": chatHistory,
	})
	if err != nil {
		return nil, err
	}

	aiMessages := []Message{
		{Role: "user", Content: prompt},
	}

	response, err := cs.client.Chat(aiMessages)
	if err != nil {
		return nil, err
	}

	return cs.parseSummary(response)
}

// formatChatHistory 格式化聊天记录
func (cs *ChatSummarizer) formatChatHistory(messages []ChatMessage) string {
	var builder strings.Builder
	for _, msg := range messages {
		role := "用户"
		if msg.Role == "assistant" {
			role = "助手"
		}
		builder.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			msg.Timestamp.Format("15:04"), role, msg.Content))
	}
	return builder.String()
}

// parseSummary 解析总结响应
func (cs *ChatSummarizer) parseSummary(response string) (*ChatSummary, error) {
	summary := &ChatSummary{
		KeyPoints: []string{},
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "•") {
			point := strings.TrimPrefix(strings.TrimPrefix(line, "1."), "•")
			point = strings.TrimSpace(point)
			if point != "" {
				summary.KeyPoints = append(summary.KeyPoints, point)
			}
		} else if strings.Contains(line, "话题总结") || strings.Contains(line, "主题") {
			summary.Topic = line
		} else if strings.Contains(line, "情绪") || strings.Contains(line, "情感") {
			summary.Sentiment = line
		} else if strings.Contains(line, "建议") || strings.Contains(line, "行动") {
			summary.NextAction = line
		}
	}

	// 如果没有解析到主题，使用第一行作为主题
	if summary.Topic == "" && len(lines) > 0 {
		summary.Topic = lines[0]
	}

	summary.Summary = response
	return summary, nil
}

// SummarizeByTimeRange 按时间范围总结聊天记录
func (cs *ChatSummarizer) SummarizeByTimeRange(messages []ChatMessage, startTime, endTime time.Time) (*ChatSummary, error) {
	var filteredMessages []ChatMessage
	for _, msg := range messages {
		if msg.Timestamp.After(startTime) && msg.Timestamp.Before(endTime) {
			filteredMessages = append(filteredMessages, msg)
		}
	}
	return cs.Summarize(filteredMessages)
}

// QuickSummary 快速总结（简化版）
func (cs *ChatSummarizer) QuickSummary(messages []ChatMessage) string {
	if len(messages) == 0 {
		return "无聊天记录"
	}

	lastMsg := messages[len(messages)-1]
	return fmt.Sprintf("最后消息: %s", lastMsg.Content[:min(len(lastMsg.Content), 50)])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
