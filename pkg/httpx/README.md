# HTTPx 包使用指南

## 概述

`httpx` 包是一个增强的 HTTP 处理库，提供了请求解析、响应写入、JWT 注入等功能。本文档详细介绍了包中各个组件的作用和使用方法。

## 核心组件

### 1. CustomResponseWriter

#### 作用
`CustomResponseWriter` 是一个 HTTP 响应写入的中间层拦截器，主要功能包括：
- 代理写入操作到底层的 `http.ResponseWriter`
- 设置 `Wrote` 状态标记，防止重复写入响应
- 提供拦截和监控能力

#### 核心字段
```go
type CustomResponseWriter struct {
    http.ResponseWriter
    Wrote bool // 标记是否已写入响应体
}
```

#### Write 方法
位于 `http.go` 第 52-60 行，该方法：
- 将数据写入底层的 `http.ResponseWriter`
- 在成功写入后将 `Wrote` 字段设置为 `true`
- **注意**：这不是一个直接的响应函数，而是底层的写入代理

### 2. 防重复写入机制

#### 设计目的
防止以下情况发生：
1. **HTTP 协议层面错误**：多次调用 `WriteHeader()` 导致 panic
2. **业务逻辑重复响应**：Logic 层和 Handler 层都尝试写入响应
3. **中间件冲突**：中间件提前响应后 Handler 仍尝试写入
4. **异常处理重复**：Panic Recovery 中可能发生的重复写入

#### 工作原理
```go
// 典型的使用模式
cw := &CustomResponseWriter{ResponseWriter: w}
// ... 业务逻辑处理 ...
if !cw.Wrote {
    // 只有在未写入时才发送响应
    httpx.JsonBaseResponseCtx(ctx, cw, data)
}
```

#### 错误示例
如果没有防重复写入机制，可能出现：
```
http: multiple response.WriteHeader calls
```

### 3. 响应函数对比

#### OkJson vs JsonBaseResponseCtx vs Write 方法

| 函数 | 类型 | 作用 | 位置 |
|------|------|------|------|
| `Write` 方法 | 底层代理 | 实际写入数据到响应体 | `CustomResponseWriter` |
| `OkJson` | 响应函数 | 格式化 JSON 并调用 Write | `base.go` |
| `JsonBaseResponseCtx` | 高级响应函数 | 包装数据后调用 OkJson | `base.go` |

**调用链**：`JsonBaseResponseCtx` → `OkJson` → `CustomResponseWriter.Write`

### 4. 文件传输

#### SendFile vs SendFileCopy

**SendFile** (推荐用于文件下载)：
- 使用 `http.ServeContent`
- 支持 HTTP 缓存 (ETag, Last-Modified)
- 支持断点续传 (Range requests)
- 自动设置 Content-Type
- 内存优化

**SendFileCopy** (适用于流式数据)：
- 使用 `io.Copy`
- 简单的数据复制
- 需要手动设置响应头
- 不支持缓存和断点续传

### 5. 请求解析 (parase.go)

#### Parse 函数
综合解析函数，支持：
- 路径参数绑定
- 表单数据解析
- 请求头提取
- JSON 请求体解析
- 数据验证

#### ParseJsonBody 函数
专门用于 JSON 请求体解析和验证。

#### 全局验证器
```go
var validator atomic.Value // 线程安全的全局验证器
```

**特点**：
- 支持动态配置
- 并发安全
- 结构体级别验证优先于全局验证器

### 6. JWT 注入 (inject.go)

#### StartWithInjectJwt
- 用于开发/测试环境
- 自动注入指定的 JWT 令牌
- 条件注入：仅在 `inject=true` 且无 Authorization 头时生效

#### StartWithInjectUserJwt
- 自动生成并注入指定用户的 JWT 令牌
- 简化开发和测试流程

## HTTP 响应头设置机制

### Header().Set() vs 实际写入

```go
// 仅设置响应头值，未实际写入
w.Header().Set("Content-Type", "application/json")

// 实际写入发生在以下时机：
// 1. 首次调用 Write()
// 2. 调用 WriteHeader()
// 3. 调用 http.ServeContent()
```

**重要**：一旦响应体开始写入，响应头就无法再修改。

## 最佳实践

### 1. 使用 CustomResponseWriter
```go
cw := &CustomResponseWriter{ResponseWriter: w}
// 业务逻辑处理
if !cw.Wrote {
    httpx.JsonBaseResponseCtx(ctx, cw, response)
}
```

### 2. 选择合适的响应函数
- 简单 JSON 响应：使用 `OkJson`
- 标准化响应：使用 `JsonBaseResponseCtx`
- 文件下载：使用 `SendFile`
- 流式数据：使用 `SendFileCopy`

### 3. 请求解析
```go
var req RequestStruct
if err := httpx.Parse(r, &req); err != nil {
    // 处理解析错误
}
```

### 4. 开发环境 JWT 注入
```go
// 注入固定令牌
httpx.StartWithInjectJwt(handler, true, "your-jwt-token")

// 注入用户令牌
httpx.StartWithInjectUserJwt(handler, true, userID, jwtSecret)
```

## 注意事项

1. **防重复写入**：始终检查 `CustomResponseWriter.Wrote` 状态
2. **响应头设置**：在写入响应体前完成所有响应头设置
3. **错误处理**：合理处理解析和验证错误
4. **性能考虑**：文件传输优先使用 `SendFile`
5. **安全性**：生产环境禁用 JWT 注入功能

## 总结

`httpx` 包通过 `CustomResponseWriter` 提供了强大的响应写入控制，防止了常见的重复写入错误。配合完善的请求解析、文件传输和开发工具，为 HTTP 服务开发提供了完整的解决方案。

关键设计理念：**防御性编程** - 通过状态跟踪和条件检查，确保程序的健壮性和可维护性。