package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// initialization 执行客户端初始化流程
// ctx: 上下文
// request: 初始化请求
// 返回: 初始化结果和错误信息
// 1. 设置协议版本
// 2. 调用服务端初始化方法
// 3. 检查协议版本是否支持
// 4. 发送初始化完成通知
// 5. 更新客户端和服务端信息
// 6. 标记客户端为就绪状态
func (client *Client) initialization(ctx context.Context, request *protocol.InitializeRequest) (*protocol.InitializeResult, error) {
	request.ProtocolVersion = protocol.Version

	response, err := client.callServer(ctx, protocol.Initialize, request)
	if err != nil {
		return nil, err
	}
	var result protocol.InitializeResult
	if err = pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if _, ok := protocol.SupportedVersion[result.ProtocolVersion]; !ok {
		return nil, fmt.Errorf("protocol version not supported, supported lastest version is %v", protocol.Version)
	}

	if err = client.sendNotification4Initialized(ctx); err != nil {
		return nil, fmt.Errorf("failed to send InitializedNotification: %w", err)
	}

	client.clientInfo = &request.ClientInfo
	client.clientCapabilities = &request.Capabilities

	client.serverInfo = &result.ServerInfo
	client.serverCapabilities = &result.Capabilities
	client.serverInstructions = result.Instructions

	client.ready.Store(true)
	return &result, nil
}

// Ping 发送ping请求检测服务端是否存活
// ctx: 上下文，用于控制请求超时和取消
// request: ping请求参数
// 返回: ping结果和错误信息
// 1. 调用服务端ping方法
// 2. 解析响应数据
// 3. 返回ping结果
// 注意: 此方法用于心跳检测和连接保活
func (client *Client) Ping(ctx context.Context, request *protocol.PingRequest) (*protocol.PingResult, error) {
	response, err := client.callServer(ctx, protocol.Ping, request)
	if err != nil {
		return nil, err
	}

	var result protocol.PingResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

// ListPrompts 获取提示词列表
// ctx: 上下文，用于控制请求超时和取消
// 返回: 提示词列表结果和错误信息
// 1. 检查服务端是否支持提示词功能
// 2. 调用服务端获取提示词列表方法
// 3. 解析响应数据
// 4. 返回提示词列表
// 注意: 返回的提示词列表包含名称和描述信息
func (client *Client) ListPrompts(ctx context.Context) (*protocol.ListPromptsResult, error) {
	if client.serverCapabilities.Prompts == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.PromptsList, protocol.NewListPromptsRequest())
	if err != nil {
		return nil, err
	}

	var result protocol.ListPromptsResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

// GetPrompt 获取指定提示词的详细信息
// ctx: 上下文，用于控制请求超时和取消
// request: 获取提示词请求参数，包含提示词名称和可选参数
// 返回: 提示词详细信息和错误信息
// 1. 检查服务端是否支持提示词功能
// 2. 调用服务端获取提示词方法
// 3. 解析响应数据
// 4. 返回提示词详细信息
// 注意: 返回的提示词信息包含内容、角色和元数据
func (client *Client) GetPrompt(ctx context.Context, request *protocol.GetPromptRequest) (*protocol.GetPromptResult, error) {
	if client.serverCapabilities.Prompts == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.PromptsGet, request)
	if err != nil {
		return nil, err
	}

	var result protocol.GetPromptResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// ListResources 获取资源列表
// ctx: 上下文，用于控制请求超时和取消
// 返回: 资源列表结果和错误信息
// 1. 检查服务端是否支持资源功能
// 2. 调用服务端获取资源列表方法
// 3. 解析响应数据
// 4. 返回资源列表
// 注意: 返回的资源列表包含资源ID和基本信息
func (client *Client) ListResources(ctx context.Context) (*protocol.ListResourcesResult, error) {
	if client.serverCapabilities.Resources == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ResourcesList, protocol.NewListResourcesRequest())
	if err != nil {
		return nil, err
	}

	var result protocol.ListResourcesResult
	if err = pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, err
}

// ListResourceTemplates 获取资源模板列表
// ctx: 上下文，用于控制请求超时和取消
// 返回: 资源模板列表结果和错误信息
// 1. 检查服务端是否支持资源功能
// 2. 调用服务端获取资源模板列表方法
// 3. 解析响应数据
// 4. 返回资源模板列表
// 注意: 返回的模板列表包含模板ID和配置信息
func (client *Client) ListResourceTemplates(ctx context.Context) (*protocol.ListResourceTemplatesResult, error) {
	if client.serverCapabilities.Resources == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ResourceListTemplates, protocol.NewListResourceTemplatesRequest())
	if err != nil {
		return nil, err
	}

	var result protocol.ListResourceTemplatesResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

// ReadResource 读取指定资源的详细信息
// ctx: 上下文，用于控制请求超时和取消
// request: 读取资源请求参数，包含资源ID和可选参数
// 返回: 资源详细信息和错误信息
// 1. 检查服务端是否支持资源功能
// 2. 调用服务端读取资源方法
// 3. 解析响应数据
// 4. 返回资源详细信息
// 注意: 返回的资源信息包含内容和元数据
func (client *Client) ReadResource(ctx context.Context, request *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
	if client.serverCapabilities.Resources == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ResourcesRead, request)
	if err != nil {
		return nil, err
	}

	var result protocol.ReadResourceResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

// SubscribeResourceChange 订阅资源变更通知
// ctx: 上下文，用于控制请求超时和取消
// request: 订阅请求参数，包含资源ID和回调信息
// 返回: 订阅结果和错误信息
// 1. 检查服务端是否支持资源订阅功能
// 2. 调用服务端订阅资源变更方法
// 3. 解析响应数据
// 4. 返回订阅结果
// 注意: 订阅成功后服务端会在资源变更时推送通知
func (client *Client) SubscribeResourceChange(ctx context.Context, request *protocol.SubscribeRequest) (*protocol.SubscribeResult, error) {
	if client.serverCapabilities.Resources == nil || !client.serverCapabilities.Resources.Subscribe {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ResourcesSubscribe, request)
	if err != nil {
		return nil, err
	}

	var result protocol.SubscribeResult
	if len(response) > 0 {
		if err = pkg.JSONUnmarshal(response, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}
	return &result, nil
}

// UnSubscribeResourceChange 取消订阅资源变更通知
// ctx: 上下文，用于控制请求超时和取消
// request: 取消订阅请求参数，包含订阅ID
// 返回: 取消订阅结果和错误信息
// 1. 检查服务端是否支持资源订阅功能
// 2. 调用服务端取消订阅方法
// 3. 解析响应数据
// 4. 返回取消订阅结果
// 注意: 取消订阅后将不再接收该资源的变更通知
func (client *Client) UnSubscribeResourceChange(ctx context.Context, request *protocol.UnsubscribeRequest) (*protocol.UnsubscribeResult, error) {
	if client.serverCapabilities.Resources == nil || !client.serverCapabilities.Resources.Subscribe {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ResourcesUnsubscribe, request)
	if err != nil {
		return nil, err
	}

	var result protocol.UnsubscribeResult
	if len(response) > 0 {
		if err = pkg.JSONUnmarshal(response, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}
	return &result, nil
}

// ListTools 获取工具列表
// ctx: 上下文，用于控制请求超时和取消
// 返回: 工具列表结果和错误信息
// 1. 检查服务端是否支持工具功能
// 2. 调用服务端获取工具列表方法
// 3. 解析响应数据
// 4. 返回工具列表
// 注意: 返回的工具列表包含工具ID和基本信息
func (client *Client) ListTools(ctx context.Context) (*protocol.ListToolsResult, error) {
	if client.serverCapabilities.Tools == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ToolsList, protocol.NewListToolsRequest())
	if err != nil {
		return nil, err
	}

	var result protocol.ListToolsResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

// CallTool 调用指定工具
// ctx: 上下文，用于控制请求超时和取消
// request: 调用工具请求参数，包含工具ID和输入参数
// 返回: 工具调用结果和错误信息
// 1. 检查服务端是否支持工具功能
// 2. 调用服务端工具调用方法
// 3. 解析响应数据
// 4. 返回工具调用结果
// 注意: 工具执行结果可能包含输出数据和状态信息
func (client *Client) CallTool(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	if client.serverCapabilities.Tools == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ToolsCall, request)
	if err != nil {
		return nil, err
	}

	var result protocol.CallToolResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

func (client *Client) sendNotification4Initialized(ctx context.Context) error {
	return client.sendMsgWithNotification(ctx, protocol.NotificationInitialized, protocol.NewInitializedNotification())
}

// Responsible for request and response assembly
// callServer 调用服务端方法
// ctx: 上下文
// method: 方法名
// params: 请求参数
// 返回: 原始响应数据和错误信息
// 1. 检查客户端是否就绪(除初始化和ping方法)
// 2. 生成请求ID并创建响应通道
// 3. 发送请求消息
// 4. 等待响应或超时
// 5. 处理错误响应
func (client *Client) callServer(ctx context.Context, method protocol.Method, params protocol.ClientRequest) (json.RawMessage, error) {
	if !client.ready.Load() && (method != protocol.Initialize && method != protocol.Ping) {
		return nil, errors.New("callServer: client not ready")
	}

	requestID := strconv.FormatInt(atomic.AddInt64(&client.requestID, 1), 10)
	respChan := make(chan *protocol.JSONRPCResponse, 1)
	client.reqID2respChan.Set(requestID, respChan)
	defer client.reqID2respChan.Remove(requestID)

	if err := client.sendMsgWithRequest(ctx, requestID, method, params); err != nil {
		return nil, fmt.Errorf("callServer: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-respChan:
		if err := response.Error; err != nil {
			return nil, pkg.NewResponseError(err.Code, err.Message, err.Data)
		}
		return response.RawResult, nil
	}
}
