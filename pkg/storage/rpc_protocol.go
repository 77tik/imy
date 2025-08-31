package storage

import (
	"context"
	"time"
)

// StoreRPCRequest RPC请求基础结构
type StoreRPCRequest struct {
	RequestID   string                 `json:"requestId"`   // 请求ID
	Method      string                 `json:"method"`      // 方法名
	Params      map[string]interface{} `json:"params"`      // 参数
	Timestamp   time.Time              `json:"timestamp"`   // 时间戳
	Timeout     time.Duration          `json:"timeout"`     // 超时时间
	SourceStore string                 `json:"sourceStore"` // 源Store ID
}

// StoreRPCResponse RPC响应基础结构
type StoreRPCResponse struct {
	RequestID string                 `json:"requestId"` // 对应的请求ID
	Success   bool                   `json:"success"`   // 是否成功
	Data      map[string]interface{} `json:"data"`      // 响应数据
	Error     string                 `json:"error"`     // 错误信息
	Timestamp time.Time              `json:"timestamp"` // 响应时间戳
}

// Timeline相关RPC方法参数和响应

// GetTimelineRequest 获取Timeline请求
type GetTimelineRequest struct {
	TimelineKey string `json:"timelineKey"`
}

// GetTimelineResponse 获取Timeline响应
type GetTimelineResponse struct {
	Timeline *Timeline `json:"timeline"`
	Exists   bool      `json:"exists"`
}

// AddMessageRequest 添加消息请求
type AddMessageRequest struct {
	TimelineKey string   `json:"timelineKey"`
	Message     *Message `json:"message"`
}

// AddMessageResponse 添加消息响应
type AddMessageResponse struct {
	BlockID   string `json:"blockId"`
	Offset    int64  `json:"offset"`
	MessageID string `json:"messageId"`
}

// GetMessagesRequest 获取消息请求
type GetMessagesRequest struct {
	TimelineKey string `json:"timelineKey"`
	StartTime   int64  `json:"startTime"`
	EndTime     int64  `json:"endTime"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
}

// GetMessagesResponse 获取消息响应
type GetMessagesResponse struct {
	Messages []*Message `json:"messages"`
	Total    int        `json:"total"`
	HasMore  bool       `json:"hasMore"`
}

// CreateTimelineRequest 创建Timeline请求
type CreateTimelineRequest struct {
	TimelineKey string                 `json:"timelineKey"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// CreateTimelineResponse 创建Timeline响应
type CreateTimelineResponse struct {
	Timeline *Timeline `json:"timeline"`
	Created  bool      `json:"created"`
}

// DeleteTimelineRequest 删除Timeline请求
type DeleteTimelineRequest struct {
	TimelineKey string `json:"timelineKey"`
	Force       bool   `json:"force"` // 是否强制删除
}

// DeleteTimelineResponse 删除Timeline响应
type DeleteTimelineResponse struct {
	Deleted bool `json:"deleted"`
}

// GetTimelineBlockRequest 获取Timeline块请求
type GetTimelineBlockRequest struct {
	BlockID string `json:"blockId"`
}

// GetTimelineBlockResponse 获取Timeline块响应
type GetTimelineBlockResponse struct {
	Block  *TimelineBlock `json:"block"`
	Exists bool           `json:"exists"`
}

// MigrateTimelineRequest 迁移Timeline请求
type MigrateTimelineRequest struct {
	TimelineKey   string `json:"timelineKey"`
	TargetStoreID string `json:"targetStoreId"`
}

// MigrateTimelineResponse 迁移Timeline响应
type MigrateTimelineResponse struct {
	Success       bool     `json:"success"`
	MigratedBlocks []string `json:"migratedBlocks"`
}

// Store状态相关RPC方法

// GetStoreStatsRequest 获取Store统计请求
type GetStoreStatsRequest struct {
	IncludeTimelines bool `json:"includeTimelines"`
}

// GetStoreStatsResponse 获取Store统计响应
type GetStoreStatsResponse struct {
	StoreID       string   `json:"storeId"`
	TimelineCount int      `json:"timelineCount"`
	BlockCount    int      `json:"blockCount"`
	TotalSize     int64    `json:"totalSize"`
	Timelines     []string `json:"timelines,omitempty"`
	Uptime        int64    `json:"uptime"`
	LastUpdate    int64    `json:"lastUpdate"`
}

// HealthCheckRequest 健康检查请求
type HealthCheckRequest struct {
	Ping string `json:"ping"`
}

// HealthCheckResponse 健康检查响应
type HealthCheckResponse struct {
	Pong      string `json:"pong"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

// StoreRPCService Store RPC服务接口
type StoreRPCService interface {
	// Timeline操作
	GetTimeline(ctx context.Context, req *GetTimelineRequest) (*GetTimelineResponse, error)
	CreateTimeline(ctx context.Context, req *CreateTimelineRequest) (*CreateTimelineResponse, error)
	DeleteTimeline(ctx context.Context, req *DeleteTimelineRequest) (*DeleteTimelineResponse, error)
	MigrateTimeline(ctx context.Context, req *MigrateTimelineRequest) (*MigrateTimelineResponse, error)
	
	// 消息操作
	AddMessage(ctx context.Context, req *AddMessageRequest) (*AddMessageResponse, error)
	GetMessages(ctx context.Context, req *GetMessagesRequest) (*GetMessagesResponse, error)
	
	// 块操作
	GetTimelineBlock(ctx context.Context, req *GetTimelineBlockRequest) (*GetTimelineBlockResponse, error)
	
	// Store状态
	GetStoreStats(ctx context.Context, req *GetStoreStatsRequest) (*GetStoreStatsResponse, error)
	HealthCheck(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error)
}

// StoreRPCClient Store RPC客户端接口
type StoreRPCClient interface {
	// 连接管理
	Connect(ctx context.Context, address string) error
	Disconnect() error
	IsConnected() bool
	
	// Timeline操作
	GetTimeline(ctx context.Context, req *GetTimelineRequest) (*GetTimelineResponse, error)
	CreateTimeline(ctx context.Context, req *CreateTimelineRequest) (*CreateTimelineResponse, error)
	DeleteTimeline(ctx context.Context, req *DeleteTimelineRequest) (*DeleteTimelineResponse, error)
	MigrateTimeline(ctx context.Context, req *MigrateTimelineRequest) (*MigrateTimelineResponse, error)
	
	// 消息操作
	AddMessage(ctx context.Context, req *AddMessageRequest) (*AddMessageResponse, error)
	GetMessages(ctx context.Context, req *GetMessagesRequest) (*GetMessagesResponse, error)
	
	// 块操作
	GetTimelineBlock(ctx context.Context, req *GetTimelineBlockRequest) (*GetTimelineBlockResponse, error)
	
	// Store状态
	GetStoreStats(ctx context.Context, req *GetStoreStatsRequest) (*GetStoreStatsResponse, error)
	HealthCheck(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error)
}

// RPC方法常量
const (
	// Timeline操作方法
	MethodGetTimeline     = "GetTimeline"
	MethodCreateTimeline  = "CreateTimeline"
	MethodDeleteTimeline  = "DeleteTimeline"
	MethodMigrateTimeline = "MigrateTimeline"
	
	// 消息操作方法
	MethodAddMessage  = "AddMessage"
	MethodGetMessages = "GetMessages"
	
	// 块操作方法
	MethodGetTimelineBlock = "GetTimelineBlock"
	
	// Store状态方法
	MethodGetStoreStats = "GetStoreStats"
	MethodHealthCheck   = "HealthCheck"
)

// RPC错误码
const (
	ErrCodeSuccess          = 0
	ErrCodeInvalidRequest   = 1001
	ErrCodeMethodNotFound   = 1002
	ErrCodeInternalError    = 1003
	ErrCodeTimeout          = 1004
	ErrCodeTimelineNotFound = 2001
	ErrCodeBlockNotFound    = 2002
	ErrCodeInvalidMessage   = 2003
	ErrCodeStorageFull      = 2004
	ErrCodeMigrationFailed  = 2005
)

// RPC错误信息
var ErrMessages = map[int]string{
	ErrCodeSuccess:          "Success",
	ErrCodeInvalidRequest:   "Invalid request",
	ErrCodeMethodNotFound:   "Method not found",
	ErrCodeInternalError:    "Internal error",
	ErrCodeTimeout:          "Request timeout",
	ErrCodeTimelineNotFound: "Timeline not found",
	ErrCodeBlockNotFound:    "Block not found",
	ErrCodeInvalidMessage:   "Invalid message",
	ErrCodeStorageFull:      "Storage full",
	ErrCodeMigrationFailed:  "Migration failed",
}

// RPCError RPC错误结构
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// Error 实现error接口
func (e *RPCError) Error() string {
	if e.Detail != "" {
		return e.Message + ": " + e.Detail
	}
	return e.Message
}

// NewRPCError 创建RPC错误
func NewRPCError(code int, detail string) *RPCError {
	message, exists := ErrMessages[code]
	if !exists {
		message = "Unknown error"
	}
	return &RPCError{
		Code:    code,
		Message: message,
		Detail:  detail,
	}
}