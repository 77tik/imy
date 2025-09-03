package ai

import (
	"strings"
)

// ModerationResult 审核结果
type ModerationResult struct {
	Type      string `json:"type"`
	Level     string `json:"level"`
	Action    string `json:"action"`
	Reason    string `json:"reason"`
	IsHarmful bool   `json:"is_harmful"`
}

// ContentModerator 消息审核器
type ContentModerator struct {
	client *AIClient
	prompt *PromptBuilder
}

// NewContentModerator 创建消息审核器
func NewContentModerator(client *AIClient) *ContentModerator {
	return &ContentModerator{
		client: client,
		prompt: NewDefaultPromptBuilder(),
	}
}

// Moderate 审核消息内容
func (cm *ContentModerator) Moderate(content string) (*ModerationResult, error) {
	if content == "" {
		return &ModerationResult{
			Type:      "无",
			Level:     "低",
			Action:    "通过",
			Reason:    "内容为空",
			IsHarmful: false,
		}, nil
	}

	prompt, err := cm.prompt.BuildPrompt("content_moderation", map[string]interface{}{
		"content": content,
	})
	if err != nil {
		return nil, err
	}

	messages := []Message{
		{Role: "user", Content: prompt},
	}

	response, err := cm.client.Chat(messages)
	if err != nil {
		return nil, err
	}

	return cm.parseResponse(response)
}

// parseResponse 解析审核响应
func (cm *ContentModerator) parseResponse(response string) (*ModerationResult, error) {
	result := &ModerationResult{
		IsHarmful: false,
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "违规类型：") {
			result.Type = strings.TrimPrefix(line, "违规类型：")
			result.IsHarmful = result.Type != "无"
		} else if strings.HasPrefix(line, "违规程度：") {
			result.Level = strings.TrimPrefix(line, "违规程度：")
		} else if strings.HasPrefix(line, "建议操作：") {
			result.Action = strings.TrimPrefix(line, "建议操作：")
		} else if strings.HasPrefix(line, "说明：") {
			result.Reason = strings.TrimPrefix(line, "说明：")
		}
	}

	return result, nil
}

// BatchModerate 批量审核消息
func (cm *ContentModerator) BatchModerate(contents []string) ([]*ModerationResult, error) {
	results := make([]*ModerationResult, len(contents))
	for i, content := range contents {
		result, err := cm.Moderate(content)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}
