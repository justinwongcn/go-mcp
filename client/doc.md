# Client 模块文档

## 概述
Client模块是与服务端通信的核心组件，负责处理JSON-RPC协议的消息收发、请求响应和通知处理。

## 文件结构

### call.go
- 核心功能：
  - 初始化客户端 (`initialization`)
  - 心跳检测 (`Ping`)
  - 提示词管理 (`ListPrompts`, `GetPrompt`)
  - 资源管理 (`ListResources`, `ReadResource`, `ListResourceTemplates`)

### send.go
- 核心功能：
  - 发送JSON-RPC请求 (`sendMsgWithRequest`)
  - 发送响应 (`sendMsgWithResponse`)
  - 发送通知 (`sendMsgWithNotification`)
  - 发送错误 (`sendMsgWithError`)

### receive.go
- 核心功能：
  - 消息分发处理 (`receive`)
  - 请求处理 (`receiveRequest`)
  - 通知处理 (`receiveNotify`)
  - 响应处理 (`receiveResponse`)

### client.go
- 客户端核心结构定义

### handle.go
- 请求处理器实现

### notify_handler.go
- 通知处理器实现

## 使用示例
```go
// 初始化客户端
client := NewClient(...)
result, err := client.initialization(ctx, request)

// 发送心跳检测
pingResult, err := client.Ping(ctx, &protocol.PingRequest{...})

// 获取提示词列表
prompts, err := client.ListPrompts(ctx)
```

## 注意事项
1. 所有方法都需要传入context.Context参数以支持超时和取消
2. 错误处理遵循Go的error模式
3. 线程安全设计，支持并发调用