// Package server 实现MCP协议的服务端核心逻辑
// 模块功能：处理服务端消息发送，包括请求和通知
// 项目定位：go-mcp项目的核心通信发送组件
// 版本历史：
// - 2023-10-01 初始版本 (ThinkInAI)
// 依赖说明：
// - github.com/ThinkInAIXYZ/go-mcp/protocol: MCP协议定义
package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// sendMsgWithRequest 发送请求消息到客户端
// 输入参数：
// - ctx: 上下文
// - sessionID: 会话ID
// - requestID: 请求ID
// - method: 方法名
// - params: 请求参数
// 返回值：
// - error: 发送错误
// 功能说明：
// 1. 构造JSON-RPC请求
// 2. 序列化消息
// 3. 通过传输层发送
// [注意] requestID不能为空
func (server *Server) sendMsgWithRequest(ctx context.Context, sessionID string, requestID protocol.RequestID,
	method protocol.Method, params protocol.ServerRequest,
) error { //nolint:whitespace
	if requestID == nil {
		return fmt.Errorf("requestID can't is nil")
	}

	req := protocol.NewJSONRPCRequest(requestID, method, params)

	message, err := json.Marshal(req)
	if err != nil {
		return err
	}

	if err := server.transport.Send(ctx, sessionID, message); err != nil {
		return fmt.Errorf("sendRequest: transport send: %w", err)
	}
	return nil
}

// sendMsgWithNotification 发送通知消息到客户端
// 输入参数：
// - ctx: 上下文
// - sessionID: 会话ID
// - method: 方法名
// - params: 通知参数
// 返回值：
// - error: 发送错误
// 功能说明：
// 1. 构造JSON-RPC通知
// 2. 序列化消息
// 3. 通过传输层发送
// [注意] 通知不需要响应
func (server *Server) sendMsgWithNotification(ctx context.Context, sessionID string, method protocol.Method, params protocol.ServerNotify) error {
	notify := protocol.NewJSONRPCNotification(method, params)

	message, err := json.Marshal(notify)
	if err != nil {
		return err
	}

	if err := server.transport.Send(ctx, sessionID, message); err != nil {
		return fmt.Errorf("sendNotification: transport send: %w", err)
	}
	return nil
}
