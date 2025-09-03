package ai

import (
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

// AIClient 基于resty的HTTP客户端
type AIClient struct {
	client  *resty.Client
	baseURL string
	apiKey  string
	model   string
}

// NewAIClient 创建新的AI客户端
func NewAIClient(baseURL, apiKey, model string) *AIClient {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("Content-Type", "application/json")
	if apiKey != "" {
		client.SetHeader("Authorization", "Bearer "+apiKey)
	}

	return &AIClient{
		client:  client,
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
	}
}

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

// ChatResponse 聊天响应结构
type ChatResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Choice 选择结构
type Choice struct {
	Message Message `json:"message"`
}

// Usage 使用量结构
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Chat 发送聊天请求
func (c *AIClient) Chat(messages []Message) (string, error) {
	req := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}

	var resp ChatResponse
	_, err := c.client.R().
		SetBody(req).
		SetResult(&resp).
		Post(c.baseURL + "/chat/completions")

	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response choices")
}

// ChatStream 流式聊天（简化版）
func (c *AIClient) ChatStream(messages []Message) (<-chan string, error) {
	// 简化实现，返回单个响应
	resp, err := c.Chat(messages)
	if err != nil {
		return nil, err
	}

	ch := make(chan string, 1)
	ch <- resp
	close(ch)
	return ch, nil
}

// SetModel 设置模型
func (c *AIClient) SetModel(model string) {
	c.model = model
}

// SetTimeout 设置超时时间
func (c *AIClient) SetTimeout(timeout time.Duration) {
	c.client.SetTimeout(timeout)
}
