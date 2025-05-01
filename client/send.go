package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// sendMsgWithRequest 发送请求消息
// ctx: 上下文
// requestID: 请求ID
// method: 请求方法
// params: 请求参数
// 返回: 错误信息
// 1. 创建JSON-RPC请求对象
// 2. 序列化并发送消息
func (client *Client) sendMsgWithRequest(ctx context.Context, requestID protocol.RequestID, method protocol.Method, params protocol.ClientRequest) error {
	if requestID == nil {
		return fmt.Errorf("requestID can't is nil")
	}

	req := protocol.NewJSONRPCRequest(requestID, method, params)

	message, err := json.Marshal(req)
	if err != nil {
		return err
	}

	if err = client.transport.Send(ctx, message); err != nil {
		if !errors.Is(err, pkg.ErrSessionClosed) {
			return fmt.Errorf("sendRequest: transport send: %w", err)
		}
		if err = client.againInitialization(ctx); err != nil {
			return err
		}
	}
	return nil
}

// sendMsgWithResponse 发送响应消息
// ctx: 上下文
// requestID: 请求ID
// result: 响应结果
// 返回: 错误信息
// 1. 创建JSON-RPC响应对象
// 2. 序列化并发送消息
func (client *Client) sendMsgWithResponse(ctx context.Context, requestID protocol.RequestID, result protocol.ClientResponse) error {
	if requestID == nil {
		return fmt.Errorf("requestID can't is nil")
	}

	resp := protocol.NewJSONRPCSuccessResponse(requestID, result)

	message, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	if err = client.transport.Send(ctx, message); err != nil {
		return fmt.Errorf("sendResponse: transport send: %w", err)
	}
	return nil
}

// sendMsgWithNotification 发送通知消息
// ctx: 上下文
// method: 通知方法
// params: 通知参数
// 返回: 错误信息
// 1. 创建JSON-RPC通知对象
// 2. 序列化并发送消息
func (client *Client) sendMsgWithNotification(ctx context.Context, method protocol.Method, params protocol.ClientNotify) error {
	notify := protocol.NewJSONRPCNotification(method, params)

	message, err := json.Marshal(notify)
	if err != nil {
		return err
	}

	if err = client.transport.Send(ctx, message); err != nil {
		return fmt.Errorf("sendNotification: transport send: %w", err)
	}
	return nil
}

// sendMsgWithError 发送错误响应
// ctx: 上下文
// requestID: 请求ID
// code: 错误码
// msg: 错误消息
// 返回: 错误信息
// 1. 创建JSON-RPC错误响应对象
// 2. 序列化并发送消息
func (client *Client) sendMsgWithError(ctx context.Context, requestID protocol.RequestID, code int, msg string) error {
	if requestID == nil {
		return fmt.Errorf("requestID can't is nil")
	}

	resp := protocol.NewJSONRPCErrorResponse(requestID, code, msg)

	message, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	if err = client.transport.Send(ctx, message); err != nil {
		return fmt.Errorf("sendResponse: transport send: %w", err)
	}
	return nil
}

func (client *Client) againInitialization(ctx context.Context) error {
	client.ready.Store(false)

	client.initializationMu.Lock()
	defer client.initializationMu.Unlock()

	if client.ready.Load() {
		return nil
	}

	if _, err := client.initialization(ctx, protocol.NewInitializeRequest(*client.clientInfo, *client.clientCapabilities)); err != nil {
		return err
	}
	client.ready.Store(true)
	return nil
}
