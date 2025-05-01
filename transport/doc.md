# transport 模块文档

## 概述
transport模块定义MCP协议的传输层接口，提供客户端和服务端通信的抽象实现。

## 文件结构

### transport.go
- 核心功能：
  - 基础消息接口定义 (`Message`)
  - 客户端传输接口 (`ClientTransport`)
  - 服务端传输接口 (`ServerTransport`)

### sse_client.go
- 核心功能：
  - SSE客户端传输实现 (`sseClientTransport`)
  - 接收超时配置 (`WithSSEClientOptionReceiveTimeout`)
  - 自定义HTTP客户端配置 (`WithSSEClientOptionHTTPClient`)

### stdio_client.go
- 核心功能：
  - 标准输入输出传输实现 (`stdioClientTransport`)
  - 日志记录器配置 (`WithStdioClientOptionLogger`)
  - 环境变量配置 (`WithStdioClientOptionEnv`)

### streamable_http_*.go
- 核心功能：
  - 可流式HTTP传输实现
  - 支持分块传输编码

## 使用示例
```go
// 创建SSE客户端传输
client := transport.NewSSEClientTransport(
    "http://example.com/sse",
    transport.WithSSEClientOptionReceiveTimeout(30*time.Second),
)

// 启动传输连接
if err := client.Start(); err != nil {
    log.Fatal(err)
}

// 发送消息
if err := client.Send(ctx, []byte("message")); err != nil {
    log.Fatal(err)
}
```

## 注意事项
1. 客户端传输接口设计为线程安全
2. Start方法非幂等，重复调用可能导致错误
3. SSE传输需服务端支持长连接