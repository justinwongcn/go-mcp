// Package transport 定义MCP协议传输层接口
// [模块功能] 提供客户端和服务端通信的抽象接口
// [项目定位] 属于go-mcp核心传输层，定义标准通信规范
// [版本历史]
// v1.0.0 2023-05-15 初始版本 定义基础接口
// v1.1.0 2023-06-20 增加上下文支持
// [依赖说明]
// - github.com/ThinkInAIXYZ/go-mcp/pkg >= v1.2.0
package transport

import (
	"context"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

/*
* Transport是底层传输层的抽象
* GO-MCP需要能够在服务端和客户端之间传输JSON-RPC消息
* [设计决策] 采用接口实现多种传输方式
* [典型实现]
* - StdioTransport 标准输入输出
* - HTTPTransport HTTP传输
* - WebSocketTransport WebSocket传输
 */

// Message 定义基础消息接口
type Message []byte

// String 实现字符串转换
// [性能提示] 频繁调用可能影响性能
func (msg Message) String() string {
	return pkg.B2S(msg)
}

// ClientTransport 客户端传输接口
// [重要] 线程安全设计，支持并发调用
type ClientTransport interface {
	// Start 初始化传输连接
	// [注意] 非幂等操作，重复调用可能导致错误
	Start() error

	// Send 发送消息
	// 输入:
	// - ctx 上下文控制
	// - msg 消息内容
	// 输出: 错误信息
	Send(ctx context.Context, msg Message) error

	// SetReceiver 设置消息接收处理器
	// [副作用] 会替换现有处理器
	SetReceiver(receiver clientReceiver)

	// Close 终止传输连接
	// [安全要求] 需确保资源正确释放
	Close() error
}

// clientReceiver 客户端消息接收接口
// [业务映射] 处理来自服务端的消息
type clientReceiver interface {
	// Receive 接收消息回调
	// [注意] 需处理并发调用
	Receive(ctx context.Context, msg []byte) error
}

// ClientReceiverF 客户端接收函数类型
// [设计决策] 提供函数到接口的适配
type ClientReceiverF func(ctx context.Context, msg []byte) error

// Receive 实现clientReceiver接口
func (f ClientReceiverF) Receive(ctx context.Context, msg []byte) error {
	return f(ctx, msg)
}

type ServerTransport interface {
	// Run starts listening for requests, this is synchronous, and cannot return before Shutdown is called
	Run() error

	// Send transmits a message
	Send(ctx context.Context, sessionID string, msg Message) error

	// SetReceiver sets the handler for messages from the peer
	SetReceiver(serverReceiver)

	SetSessionManager(manager sessionManager)

	// Shutdown gracefully closes, the internal implementation needs to stop receiving messages first,
	// then wait for serverCtx to be canceled, while using userCtx to control timeout.
	// userCtx is used to control the timeout of the server shutdown.
	// serverCtx is used to coordinate the internal cleanup sequence:
	// 1. turn off message listen
	// 2. Wait for serverCtx to be done (indicating server shutdown is complete)
	// 3. Cancel the transport's context to stop all ongoing operations
	// 4. Wait for all in-flight sends to complete
	// 5. Close all session
	Shutdown(userCtx context.Context, serverCtx context.Context) error
}

type serverReceiver interface {
	Receive(ctx context.Context, sessionID string, msg []byte) (<-chan []byte, error)
}

type ServerReceiverF func(ctx context.Context, sessionID string, msg []byte) (<-chan []byte, error)

func (f ServerReceiverF) Receive(ctx context.Context, sessionID string, msg []byte) (<-chan []byte, error) {
	return f(ctx, sessionID, msg)
}

type sessionManager interface {
	CreateSession() string
	OpenMessageQueueForSend(sessionID string) error
	EnqueueMessageForSend(ctx context.Context, sessionID string, message []byte) error
	DequeueMessageForSend(ctx context.Context, sessionID string) ([]byte, error)
	CloseSession(sessionID string)
	CloseAllSessions()
}
