package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// HTTPStoreRPCClient HTTP实现的Store RPC客户端
type HTTPStoreRPCClient struct {
	mu         sync.RWMutex
	address    string
	client     *http.Client
	connected  bool
	timeout    time.Duration
	headers    map[string]string
	retryCount int
}

// NewHTTPStoreRPCClient 创建HTTP RPC客户端
func NewHTTPStoreRPCClient(timeout time.Duration) *HTTPStoreRPCClient {
	return &HTTPStoreRPCClient{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout:    timeout,
		headers:    make(map[string]string),
		retryCount: 3,
	}
}

// Connect 连接到Store服务
func (c *HTTPStoreRPCClient) Connect(ctx context.Context, address string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.address = address
	
	// 执行健康检查验证连接
	req := &HealthCheckRequest{Ping: "ping"}
	_, err := c.healthCheck(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to connect to store %s: %w", address, err)
	}
	
	c.connected = true
	return nil
}

// Disconnect 断开连接
func (c *HTTPStoreRPCClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.connected = false
	c.address = ""
	return nil
}

// IsConnected 检查是否已连接
func (c *HTTPStoreRPCClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// SetHeader 设置请求头
func (c *HTTPStoreRPCClient) SetHeader(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers[key] = value
}

// SetRetryCount 设置重试次数
func (c *HTTPStoreRPCClient) SetRetryCount(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.retryCount = count
}

// makeRequest 发送RPC请求的通用方法
func (c *HTTPStoreRPCClient) makeRequest(ctx context.Context, method string, params interface{}) (*StoreRPCResponse, error) {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not connected")
	}
	address := c.address
	headers := make(map[string]string)
	for k, v := range c.headers {
		headers[k] = v
	}
	retryCount := c.retryCount
	c.mu.RUnlock()
	
	// 构建请求
	request := &StoreRPCRequest{
		RequestID: uuid.New().String(),
		Method:    method,
		Params:    make(map[string]interface{}),
		Timestamp: time.Now(),
		Timeout:   c.timeout,
	}
	
	// 序列化参数
	if params != nil {
		paramsBytes, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		err = json.Unmarshal(paramsBytes, &request.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}
	}
	
	// 序列化请求
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	var lastErr error
	for i := 0; i <= retryCount; i++ {
		// 创建HTTP请求
		httpReq, err := http.NewRequestWithContext(ctx, "POST", address+"/rpc", bytes.NewReader(requestBytes))
		if err != nil {
			lastErr = fmt.Errorf("failed to create HTTP request: %w", err)
			continue
		}
		
		// 设置请求头
		httpReq.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			httpReq.Header.Set(k, v)
		}
		
		// 发送请求
		resp, err := c.client.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("failed to send HTTP request: %w", err)
			if i < retryCount {
				time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
			}
			continue
		}
		
		// 读取响应
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}
		
		// 检查HTTP状态码
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP error: %d %s", resp.StatusCode, string(respBody))
			continue
		}
		
		// 解析响应
		var response StoreRPCResponse
		err = json.Unmarshal(respBody, &response)
		if err != nil {
			lastErr = fmt.Errorf("failed to unmarshal response: %w", err)
			continue
		}
		
		return &response, nil
	}
	
	return nil, lastErr
}

// parseResponse 解析响应数据的通用方法
func parseResponse[T any](response *StoreRPCResponse, result *T) error {
	if !response.Success {
		return fmt.Errorf("RPC error: %s", response.Error)
	}
	
	if response.Data == nil {
		return nil
	}
	
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal response data: %w", err)
	}
	
	err = json.Unmarshal(dataBytes, result)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response data: %w", err)
	}
	
	return nil
}

// Timeline操作方法

// GetTimeline 获取Timeline
func (c *HTTPStoreRPCClient) GetTimeline(ctx context.Context, req *GetTimelineRequest) (*GetTimelineResponse, error) {
	response, err := c.makeRequest(ctx, MethodGetTimeline, req)
	if err != nil {
		return nil, err
	}
	
	var result GetTimelineResponse
	err = parseResponse(response, &result)
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}

// CreateTimeline 创建Timeline
func (c *HTTPStoreRPCClient) CreateTimeline(ctx context.Context, req *CreateTimelineRequest) (*CreateTimelineResponse, error) {
	response, err := c.makeRequest(ctx, MethodCreateTimeline, req)
	if err != nil {
		return nil, err
	}
	
	var result CreateTimelineResponse
	err = parseResponse(response, &result)
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}

// DeleteTimeline 删除Timeline
func (c *HTTPStoreRPCClient) DeleteTimeline(ctx context.Context, req *DeleteTimelineRequest) (*DeleteTimelineResponse, error) {
	response, err := c.makeRequest(ctx, MethodDeleteTimeline, req)
	if err != nil {
		return nil, err
	}
	
	var result DeleteTimelineResponse
	err = parseResponse(response, &result)
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}

// MigrateTimeline 迁移Timeline
func (c *HTTPStoreRPCClient) MigrateTimeline(ctx context.Context, req *MigrateTimelineRequest) (*MigrateTimelineResponse, error) {
	response, err := c.makeRequest(ctx, MethodMigrateTimeline, req)
	if err != nil {
		return nil, err
	}
	
	var result MigrateTimelineResponse
	err = parseResponse(response, &result)
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}

// 消息操作方法

// AddMessage 添加消息
func (c *HTTPStoreRPCClient) AddMessage(ctx context.Context, req *AddMessageRequest) (*AddMessageResponse, error) {
	response, err := c.makeRequest(ctx, MethodAddMessage, req)
	if err != nil {
		return nil, err
	}
	
	var result AddMessageResponse
	err = parseResponse(response, &result)
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}

// GetMessages 获取消息
func (c *HTTPStoreRPCClient) GetMessages(ctx context.Context, req *GetMessagesRequest) (*GetMessagesResponse, error) {
	response, err := c.makeRequest(ctx, MethodGetMessages, req)
	if err != nil {
		return nil, err
	}
	
	var result GetMessagesResponse
	err = parseResponse(response, &result)
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}

// 块操作方法

// GetTimelineBlock 获取Timeline块
func (c *HTTPStoreRPCClient) GetTimelineBlock(ctx context.Context, req *GetTimelineBlockRequest) (*GetTimelineBlockResponse, error) {
	response, err := c.makeRequest(ctx, MethodGetTimelineBlock, req)
	if err != nil {
		return nil, err
	}
	
	var result GetTimelineBlockResponse
	err = parseResponse(response, &result)
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}

// Store状态方法

// GetStoreStats 获取Store统计
func (c *HTTPStoreRPCClient) GetStoreStats(ctx context.Context, req *GetStoreStatsRequest) (*GetStoreStatsResponse, error) {
	response, err := c.makeRequest(ctx, MethodGetStoreStats, req)
	if err != nil {
		return nil, err
	}
	
	var result GetStoreStatsResponse
	err = parseResponse(response, &result)
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}

// HealthCheck 健康检查
func (c *HTTPStoreRPCClient) HealthCheck(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error) {
	return c.healthCheck(ctx, req)
}

// healthCheck 内部健康检查方法
func (c *HTTPStoreRPCClient) healthCheck(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error) {
	response, err := c.makeRequest(ctx, MethodHealthCheck, req)
	if err != nil {
		return nil, err
	}
	
	var result HealthCheckResponse
	err = parseResponse(response, &result)
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}

// StoreRPCClientPool RPC客户端连接池
type StoreRPCClientPool struct {
	mu      sync.RWMutex
	clients map[string]StoreRPCClient
	timeout time.Duration
}

// NewStoreRPCClientPool 创建RPC客户端连接池
func NewStoreRPCClientPool(timeout time.Duration) *StoreRPCClientPool {
	return &StoreRPCClientPool{
		clients: make(map[string]StoreRPCClient),
		timeout: timeout,
	}
}

// GetClient 获取或创建客户端连接
func (p *StoreRPCClientPool) GetClient(ctx context.Context, storeID, address string) (StoreRPCClient, error) {
	p.mu.RLock()
	client, exists := p.clients[storeID]
	p.mu.RUnlock()
	
	if exists && client.IsConnected() {
		return client, nil
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// 双重检查
	client, exists = p.clients[storeID]
	if exists && client.IsConnected() {
		return client, nil
	}
	
	// 创建新客户端
	client = NewHTTPStoreRPCClient(p.timeout)
	err := client.Connect(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to store %s: %w", storeID, err)
	}
	
	p.clients[storeID] = client
	return client, nil
}

// RemoveClient 移除客户端连接
func (p *StoreRPCClientPool) RemoveClient(storeID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if client, exists := p.clients[storeID]; exists {
		client.Disconnect()
		delete(p.clients, storeID)
	}
}

// Close 关闭所有客户端连接
func (p *StoreRPCClientPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for _, client := range p.clients {
		client.Disconnect()
	}
	p.clients = make(map[string]StoreRPCClient)
}