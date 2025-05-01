# protocol 模块文档

## 概述
protocol模块定义了MCP协议的JSON-RPC接口规范，包括请求/响应结构、错误处理、日志记录、分页等通用功能。

## 文件结构

### cancellation.go
- 核心功能：
  - 请求取消通知结构 (`CancelledNotification`)
  - 创建取消通知的方法 (`NewCancelledNotification`)

### completion.go
- 核心功能：
  - 完成选项请求结构 (`CompleteRequest`)
  - 提示词/资源引用类型 (`PromptReference`, `ResourceReference`)
  - 完成结果结构 (`CompleteResult`, `Complete`)

### initialize.go
- 核心功能：
  - 初始化请求/响应结构 (`InitializeRequest`, `InitializeResult`)
  - 实现描述结构 (`Implementation`)
  - 客户端/服务器能力描述 (`ClientCapabilities`, `ServerCapabilities`)

### jsonrpc.go
- 核心功能：
  - JSON-RPC协议版本和错误码定义
  - 请求/响应基础结构 (`JSONRPCRequest`, `JSONRPCResponse`)
  - 自定义JSON序列化/反序列化方法

### logging.go
- 核心功能：
  - 日志级别定义 (`LoggingLevel`)
  - 设置日志级别请求/响应 (`SetLoggingLevelRequest`, `SetLoggingLevelResult`)
  - 日志消息通知 (`LogMessageNotification`)

### pagination.go
- 核心功能：
  - 分页请求/响应结构 (`PaginatedRequest`, `PaginatedResult`)

## 使用示例
```go
// 创建初始化请求
req := &protocol.InitializeRequest{
    ClientInfo: protocol.Implementation{
        Name: "example-client",
        Version: "1.0.0",
    },
    ProtocolVersion: "2.0",
}

// 创建日志消息通知
notification := &protocol.LogMessageNotification{
    Level: protocol.LogInfo,
    Message: "Processing request",
}

// 使用分页请求
pageReq := &protocol.PaginatedRequest{
    Cursor: "next-page-token",
}
```

## 注意事项
1. JSON-RPC请求必须包含有效的协议版本号(2.0)
2. 所有请求ID必须是字符串或数值类型
3. 日志级别从高到低依次为: Emergency > Alert > Critical > Error > Warning > Notice > Info > Debug
4. 分页请求中的Cursor为空表示获取第一页数据