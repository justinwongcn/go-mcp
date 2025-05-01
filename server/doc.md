# server 模块文档

## 概述
server模块实现MCP协议的服务端核心逻辑，提供注册、会话管理和请求处理能力。

## 文件结构

### server.go
- 核心功能：
  - 服务端主结构定义 (`Server`)
  - 服务配置选项 (`Option`)
  - 服务启动/停止方法 (`Start`, `Stop`)

### handle.go
- 核心功能：
  - 请求处理逻辑 (`handleRequestWithInitialize`)
  - 会话管理机制
  - 协议版本兼容性验证

### call.go
- 核心功能：
  - 客户端调用方法 (`Ping`, `Sampling`)
  - 会话ID处理 (`getSessionIDFromCtx`)
  - 响应结果解析

### session/
- 核心功能：
  - 会话状态管理 (`state.go`)
  - 会话管理器 (`manager.go`)

## 使用示例
```go
// 创建服务端实例
srv := server.NewServer(
    transport,
    server.WithCapabilities(capabilities),
    server.WithServerInfo(serverInfo),
)

// 启动服务
if err := srv.Start(); err != nil {
    log.Fatal(err)
}

// 处理请求
result, err := srv.Ping(ctx, request)
```

## 注意事项
1. 服务配置选项需在NewServer时传入
2. 会话ID通过transport.SessionIDForReturnKey上下文键处理
3. 协议版本必须兼容