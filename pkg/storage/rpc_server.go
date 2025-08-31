package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// HTTPStoreRPCServer HTTP实现的Store RPC服务端
type HTTPStoreRPCServer struct {
	mu       sync.RWMutex
	store    *Store
	server   *http.Server
	handlers map[string]RPCHandler
	running  bool
	middlewares []Middleware
}

// RPCHandler RPC处理函数类型
type RPCHandler func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// Middleware 中间件类型
type Middleware func(next http.Handler) http.Handler

// NewHTTPStoreRPCServer 创建HTTP RPC服务端
func NewHTTPStoreRPCServer(store *Store) *HTTPStoreRPCServer {
	server := &HTTPStoreRPCServer{
		store:    store,
		handlers: make(map[string]RPCHandler),
	}
	
	// 注册默认处理器
	server.registerDefaultHandlers()
	
	return server
}

// registerDefaultHandlers 注册默认的RPC处理器
func (s *HTTPStoreRPCServer) registerDefaultHandlers() {
	// Timeline操作
	s.handlers[MethodGetTimeline] = s.handleGetTimeline
	s.handlers[MethodCreateTimeline] = s.handleCreateTimeline
	s.handlers[MethodDeleteTimeline] = s.handleDeleteTimeline
	s.handlers[MethodMigrateTimeline] = s.handleMigrateTimeline
	
	// 消息操作
	s.handlers[MethodAddMessage] = s.handleAddMessage
	s.handlers[MethodGetMessages] = s.handleGetMessages
	
	// 块操作
	s.handlers[MethodGetTimelineBlock] = s.handleGetTimelineBlock
	
	// Store状态
	s.handlers[MethodGetStoreStats] = s.handleGetStoreStats
	s.handlers[MethodHealthCheck] = s.handleHealthCheck
}

// RegisterHandler 注册自定义RPC处理器
func (s *HTTPStoreRPCServer) RegisterHandler(method string, handler RPCHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

// AddMiddleware 添加中间件
func (s *HTTPStoreRPCServer) AddMiddleware(middleware Middleware) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.middlewares = append(s.middlewares, middleware)
}

// Start 启动RPC服务
func (s *HTTPStoreRPCServer) Start(address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.running {
		return fmt.Errorf("server is already running")
	}
	
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", s.handleRPC)
	mux.HandleFunc("/health", s.handleHealth)
	
	// 应用中间件
	var handler http.Handler = mux
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		handler = s.middlewares[i](handler)
	}
	
	s.server = &http.Server{
		Addr:    address,
		Handler: handler,
	}
	
	s.running = true
	
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("RPC server error: %v", err)
		}
	}()
	
	return nil
}

// Stop 停止RPC服务
func (s *HTTPStoreRPCServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.running {
		return nil
	}
	
	s.running = false
	
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	
	return nil
}

// IsRunning 检查服务是否运行中
func (s *HTTPStoreRPCServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// handleRPC 处理RPC请求
func (s *HTTPStoreRPCServer) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeErrorResponse(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	
	// 解析请求
	var request StoreRPCRequest
	err = json.Unmarshal(body, &request)
	if err != nil {
		s.writeErrorResponse(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}
	
	// 验证请求
	if request.Method == "" {
		s.writeErrorResponse(w, "Method is required", http.StatusBadRequest)
		return
	}
	
	// 查找处理器
	s.mu.RLock()
	handler, exists := s.handlers[request.Method]
	s.mu.RUnlock()
	
	if !exists {
		s.writeRPCErrorResponse(w, request.RequestID, ErrCodeMethodNotFound, "Method not found: "+request.Method)
		return
	}
	
	// 创建上下文
	ctx := r.Context()
	if request.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, request.Timeout)
		defer cancel()
	}
	
	// 执行处理器
	result, err := handler(ctx, request.Params)
	if err != nil {
		s.writeRPCErrorResponse(w, request.RequestID, ErrCodeInternalError, err.Error())
		return
	}
	
	// 构建响应
	response := &StoreRPCResponse{
		RequestID: request.RequestID,
		Success:   true,
		Timestamp: time.Now(),
	}
	
	if result != nil {
		// 序列化结果
		resultBytes, err := json.Marshal(result)
		if err != nil {
			s.writeRPCErrorResponse(w, request.RequestID, ErrCodeInternalError, "Failed to marshal result")
			return
		}
		
		var resultMap map[string]interface{}
		err = json.Unmarshal(resultBytes, &resultMap)
		if err != nil {
			s.writeRPCErrorResponse(w, request.RequestID, ErrCodeInternalError, "Failed to unmarshal result")
			return
		}
		
		response.Data = resultMap
	}
	
	// 发送响应
	s.writeJSONResponse(w, response, http.StatusOK)
}

// handleHealth 处理健康检查请求
func (s *HTTPStoreRPCServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"store_id":  s.store.StoreID,
	}
	s.writeJSONResponse(w, response, http.StatusOK)
}

// writeJSONResponse 写入JSON响应
func (s *HTTPStoreRPCServer) writeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

// writeErrorResponse 写入错误响应
func (s *HTTPStoreRPCServer) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := map[string]string{
		"error": message,
	}
	s.writeJSONResponse(w, response, statusCode)
}

// writeRPCErrorResponse 写入RPC错误响应
func (s *HTTPStoreRPCServer) writeRPCErrorResponse(w http.ResponseWriter, requestID string, errorCode int, errorMessage string) {
	response := &StoreRPCResponse{
		RequestID: requestID,
		Success:   false,
		Error:     errorMessage,
		Timestamp: time.Now(),
	}
	s.writeJSONResponse(w, response, http.StatusOK)
}

// parseParams 解析参数的通用方法
func parseParams[T any](params map[string]interface{}, result *T) error {
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}
	
	err = json.Unmarshal(paramsBytes, result)
	if err != nil {
		return fmt.Errorf("failed to unmarshal params: %w", err)
	}
	
	return nil
}

// Timeline操作处理器

// handleGetTimeline 处理获取Timeline请求
func (s *HTTPStoreRPCServer) handleGetTimeline(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	var req GetTimelineRequest
	err := parseParams(params, &req)
	if err != nil {
		return nil, err
	}
	
	timeline, exists := s.store.ConvTimelines[req.TimelineKey]
	if !exists {
		// 尝试加载Timeline
		timeline = s.store.GetOrCreateConvTimeline(req.TimelineKey)
	}
	
	return &GetTimelineResponse{
		Timeline: timeline,
		Exists:   timeline != nil,
	}, nil
}

// handleCreateTimeline 处理创建Timeline请求
func (s *HTTPStoreRPCServer) handleCreateTimeline(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	var req CreateTimelineRequest
	err := parseParams(params, &req)
	if err != nil {
		return nil, err
	}
	
	// 检查Timeline是否已存在
	if _, exists := s.store.ConvTimelines[req.TimelineKey]; exists {
		return &CreateTimelineResponse{
			Timeline: s.store.ConvTimelines[req.TimelineKey],
			Created:  false,
		}, nil
	}
	
	// 创建新Timeline
	timeline := s.store.GetOrCreateConvTimeline(req.TimelineKey)
	
	// TODO: 设置元数据 - Timeline结构体需要添加Metadata字段
	// if req.Metadata != nil {
	//     for k, v := range req.Metadata {
	//         timeline.Metadata[k] = v
	//     }
	//     // 保存元数据
	//     err = s.store.saveTimelineMetadata(timeline)
	//     if err != nil {
	//         return nil, fmt.Errorf("failed to save timeline metadata: %w", err)
	//     }
	// }
	
	return &CreateTimelineResponse{
		Timeline: timeline,
		Created:  true,
	}, nil
}

// handleDeleteTimeline 处理删除Timeline请求
func (s *HTTPStoreRPCServer) handleDeleteTimeline(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	var req DeleteTimelineRequest
	err := parseParams(params, &req)
	if err != nil {
		return nil, err
	}
	
	// 检查Timeline是否存在
	_, exists := s.store.ConvTimelines[req.TimelineKey]
	if !exists {
		return &DeleteTimelineResponse{Deleted: false}, nil
	}
	
	// TODO: 实现删除Timeline文件和块的逻辑
	// err = s.store.deleteTimeline(timeline)
	// if err != nil && !req.Force {
	//     return nil, fmt.Errorf("failed to delete timeline: %w", err)
	// }
	
	// 从内存中移除
	delete(s.store.ConvTimelines, req.TimelineKey)
	
	return &DeleteTimelineResponse{Deleted: true}, nil
}

// handleMigrateTimeline 处理迁移Timeline请求
func (s *HTTPStoreRPCServer) handleMigrateTimeline(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	var req MigrateTimelineRequest
	err := parseParams(params, &req)
	if err != nil {
		return nil, err
	}
	
	// TODO: 实现Timeline迁移逻辑
	// 这里需要与目标Store协调，传输Timeline数据
	
	return &MigrateTimelineResponse{
		Success:        false,
		MigratedBlocks: []string{},
	}, fmt.Errorf("timeline migration not implemented yet")
}

// 消息操作处理器

// handleAddMessage 处理添加消息请求
func (s *HTTPStoreRPCServer) handleAddMessage(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	var req AddMessageRequest
	err := parseParams(params, &req)
	if err != nil {
		return nil, err
	}
	
	// 获取或创建Timeline
	timeline := s.store.GetOrCreateConvTimeline(req.TimelineKey)
	
	// 添加消息 - 使用Store的AddMessage方法
	err = s.store.AddMessage(req.TimelineKey, req.Message.SenderID, req.Message.Data, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to add message: %w", err)
	}
	
	// 返回响应 - 这里简化处理，实际应该返回具体的块ID和偏移量
	return &AddMessageResponse{
		BlockID:   timeline.CurrentBlock.BlockID,
		Offset:    int64(len(timeline.CurrentBlock.Messages)),
		MessageID: fmt.Sprintf("%d", req.Message.SeqID),
	}, nil
}

// handleGetMessages 处理获取消息请求
func (s *HTTPStoreRPCServer) handleGetMessages(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	var req GetMessagesRequest
	err := parseParams(params, &req)
	if err != nil {
		return nil, err
	}
	
	// 获取Timeline
	_, exists := s.store.ConvTimelines[req.TimelineKey]
	if !exists {
		return &GetMessagesResponse{
			Messages: []*Message{},
			Total:    0,
			HasMore:  false,
		}, nil
	}
	
	// TODO: 实现消息查询逻辑
	// 这里需要根据时间范围、限制和偏移量查询消息
	
	return &GetMessagesResponse{
		Messages: []*Message{},
		Total:    0,
		HasMore:  false,
	}, nil
}

// 块操作处理器

// handleGetTimelineBlock 处理获取Timeline块请求
func (s *HTTPStoreRPCServer) handleGetTimelineBlock(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	var req GetTimelineBlockRequest
	err := parseParams(params, &req)
	if err != nil {
		return nil, err
	}
	
	// 从缓存中查找块
	block, exists := s.store.TimelineBlocks[req.BlockID]
	if !exists {
		return &GetTimelineBlockResponse{
			Block:  nil,
			Exists: false,
		}, nil
	}
	
	return &GetTimelineBlockResponse{
		Block:  block,
		Exists: true,
	}, nil
}

// Store状态处理器

// handleGetStoreStats 处理获取Store统计请求
func (s *HTTPStoreRPCServer) handleGetStoreStats(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	var req GetStoreStatsRequest
	err := parseParams(params, &req)
	if err != nil {
		return nil, err
	}
	
	timelineCount := len(s.store.ConvTimelines) + len(s.store.UserTimelines)
	blockCount := len(s.store.TimelineBlocks)
	
	response := &GetStoreStatsResponse{
		StoreID:       s.store.StoreID,
		TimelineCount: timelineCount,
		BlockCount:    blockCount,
		TotalSize:     s.store.CurrentCapacity,
		Uptime:        0, // TODO: 添加Store创建时间字段来计算uptime
		LastUpdate:    time.Now().Unix(),
	}
	
	if req.IncludeTimelines {
		timelines := make([]string, 0, timelineCount)
		for key := range s.store.ConvTimelines {
			timelines = append(timelines, key)
		}
		for key := range s.store.UserTimelines {
			timelines = append(timelines, key)
		}
		response.Timelines = timelines
	}
	
	return response, nil
}

// handleHealthCheck 处理健康检查请求
func (s *HTTPStoreRPCServer) handleHealthCheck(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	var req HealthCheckRequest
	err := parseParams(params, &req)
	if err != nil {
		return nil, err
	}
	
	return &HealthCheckResponse{
		Pong:      "pong",
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
	}, nil
}

// 中间件

// LoggingMiddleware 日志中间件
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// CORSMiddleware CORS中间件
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(requestsPerSecond int) Middleware {
	// TODO: 实现限流逻辑
	return func(next http.Handler) http.Handler {
		return next
	}
}