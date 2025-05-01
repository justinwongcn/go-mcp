package client

import (
	"context"
	"encoding/json"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// handleRequestWithPing 处理ping请求
// 返回: ping结果和错误信息
// 1. 创建并返回默认ping结果
func (client *Client) handleRequestWithPing() (*protocol.PingResult, error) {
	return protocol.NewPingResult(), nil
}

// handleRequestWithCreateMessagesSampling 处理采样消息创建请求
// ctx: 上下文
// rawParams: 原始参数数据
// 返回: 创建消息结果和错误信息
// 1. 检查客户端是否支持采样功能
// 2. 解析请求参数
// 3. 调用采样处理器创建消息
func (client *Client) handleRequestWithCreateMessagesSampling(ctx context.Context, rawParams json.RawMessage) (*protocol.CreateMessageResult, error) {
	if client.clientCapabilities.Sampling == nil {
		return nil, pkg.ErrClientNotSupport
	}

	var request *protocol.CreateMessageRequest
	if err := pkg.JSONUnmarshal(rawParams, &request); err != nil {
		return nil, err
	}

	return client.samplingHandler.CreateMessage(ctx, request)
}

// handleNotifyWithToolsListChanged 处理工具列表变更通知
// ctx: 上下文
// rawParams: 原始通知参数
// 返回: 错误信息
// 1. 解析通知参数
// 2. 调用通知处理器
func (client *Client) handleNotifyWithToolsListChanged(ctx context.Context, rawParams json.RawMessage) error {
	notify := &protocol.ToolListChangedNotification{}
	if len(rawParams) > 0 {
		if err := pkg.JSONUnmarshal(rawParams, notify); err != nil {
			return err
		}
	}
	return client.notifyHandler.ToolsListChanged(ctx, notify)
}

// handleNotifyWithPromptsListChanged 处理提示词列表变更通知
// ctx: 上下文
// rawParams: 原始通知参数
// 返回: 错误信息
// 1. 解析通知参数
// 2. 调用通知处理器
func (client *Client) handleNotifyWithPromptsListChanged(ctx context.Context, rawParams json.RawMessage) error {
	notify := &protocol.PromptListChangedNotification{}
	if len(rawParams) > 0 {
		if err := pkg.JSONUnmarshal(rawParams, notify); err != nil {
			return err
		}
	}
	return client.notifyHandler.PromptListChanged(ctx, notify)
}

// handleNotifyWithResourcesListChanged 处理资源列表变更通知
// ctx: 上下文
// rawParams: 原始通知参数
// 返回: 错误信息
// 1. 解析通知参数
// 2. 调用通知处理器
func (client *Client) handleNotifyWithResourcesListChanged(ctx context.Context, rawParams json.RawMessage) error {
	notify := &protocol.ResourceListChangedNotification{}
	if len(rawParams) > 0 {
		if err := pkg.JSONUnmarshal(rawParams, notify); err != nil {
			return err
		}
	}
	return client.notifyHandler.ResourceListChanged(ctx, notify)
}

// handleNotifyWithResourcesUpdated 处理资源更新通知
// ctx: 上下文
// rawParams: 原始通知参数
// 返回: 错误信息
// 1. 解析通知参数
// 2. 调用通知处理器
func (client *Client) handleNotifyWithResourcesUpdated(ctx context.Context, rawParams json.RawMessage) error {
	notify := &protocol.ResourceUpdatedNotification{}
	if len(rawParams) > 0 {
		if err := pkg.JSONUnmarshal(rawParams, notify); err != nil {
			return err
		}
	}
	return client.notifyHandler.ResourcesUpdated(ctx, notify)
}
