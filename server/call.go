package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/server/session"
)

// Ping 处理Ping请求
// 参数说明：
//   - ctx: 上下文，包含会话ID等信息
//   - request: Ping请求参数
//
// 返回值：
//   - *protocol.PingResult: Ping响应结果
//   - error: 错误信息
//
// 功能流程：
//  1. 从上下文中获取会话ID
//  2. 调用客户端方法发送Ping请求
//  3. 解析并返回响应结果
//
// [注意] 需确保会话存在且有效
func (server *Server) Ping(ctx context.Context, request *protocol.PingRequest) (*protocol.PingResult, error) {
	sessionID, err := getSessionIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	response, err := server.callClient(ctx, sessionID, protocol.Ping, request)
	if err != nil {
		return nil, err
	}

	var result protocol.PingResult
	if err = pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

// Sampling 处理采样消息创建请求
// 参数说明：
//   - ctx: 上下文，包含会话ID等信息
//   - request: 创建消息请求参数
//
// 返回值：
//   - *protocol.CreateMessageResult: 创建结果
//   - error: 错误信息
//
// 前置条件：
//  1. 会话必须存在且有效
//  2. 客户端必须支持Sampling功能
//
// 典型用例：
//   - 客户端请求创建新的采样消息时调用
func (server *Server) Sampling(ctx context.Context, request *protocol.CreateMessageRequest) (*protocol.CreateMessageResult, error) {
	sessionID, err := getSessionIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s, ok := server.sessionManager.GetSession(sessionID)
	if !ok {
		return nil, pkg.ErrLackSession
	}

	if s.GetClientCapabilities() == nil || s.GetClientCapabilities().Sampling == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := server.callClient(ctx, sessionID, protocol.SamplingCreateMessage, request)
	if err != nil {
		return nil, err
	}

	var result protocol.CreateMessageResult
	if err = pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

func (server *Server) sendNotification4ToolListChanges(ctx context.Context) error {
	if server.capabilities.Tools == nil || !server.capabilities.Tools.ListChanged {
		return pkg.ErrServerNotSupport
	}

	var errList []error
	server.sessionManager.RangeSessions(func(sessionID string, _ *session.State) bool {
		if err := server.sendMsgWithNotification(ctx, sessionID, protocol.NotificationToolsListChanged, protocol.NewToolListChangedNotification()); err != nil {
			errList = append(errList, fmt.Errorf("sessionID=%s, err: %w", sessionID, err))
		}
		return true
	})
	return pkg.JoinErrors(errList)
}

func (server *Server) sendNotification4PromptListChanges(ctx context.Context) error {
	if server.capabilities.Prompts == nil || !server.capabilities.Prompts.ListChanged {
		return pkg.ErrServerNotSupport
	}

	var errList []error
	server.sessionManager.RangeSessions(func(sessionID string, _ *session.State) bool {
		if err := server.sendMsgWithNotification(ctx, sessionID, protocol.NotificationPromptsListChanged, protocol.NewPromptListChangedNotification()); err != nil {
			errList = append(errList, fmt.Errorf("sessionID=%s, err: %w", sessionID, err))
		}
		return true
	})
	return pkg.JoinErrors(errList)
}

func (server *Server) sendNotification4ResourceListChanges(ctx context.Context) error {
	if server.capabilities.Resources == nil || !server.capabilities.Resources.ListChanged {
		return pkg.ErrServerNotSupport
	}

	var errList []error
	server.sessionManager.RangeSessions(func(sessionID string, _ *session.State) bool {
		if err := server.sendMsgWithNotification(ctx, sessionID, protocol.NotificationResourcesListChanged,
			protocol.NewResourceListChangedNotification()); err != nil {
			errList = append(errList, fmt.Errorf("sessionID=%s, err: %w", sessionID, err))
		}
		return true
	})
	return pkg.JoinErrors(errList)
}

func (server *Server) SendNotification4ResourcesUpdated(ctx context.Context, notify *protocol.ResourceUpdatedNotification) error {
	if server.capabilities.Resources == nil || !server.capabilities.Resources.Subscribe {
		return pkg.ErrServerNotSupport
	}

	var errList []error
	server.sessionManager.RangeSessions(func(sessionID string, s *session.State) bool {
		if _, ok := s.GetSubscribedResources().Get(notify.URI); !ok {
			return true
		}

		if err := server.sendMsgWithNotification(ctx, sessionID, protocol.NotificationResourcesUpdated, notify); err != nil {
			errList = append(errList, fmt.Errorf("sessionID=%s, err: %w", sessionID, err))
		}
		return true
	})
	return pkg.JoinErrors(errList)
}

// callClient 客户端调用核心方法
// 参数说明：
//   - ctx: 上下文，用于超时控制
//   - sessionID: 目标会话ID
//   - method: 调用的RPC方法
//   - params: 请求参数
//
// 返回值：
//   - json.RawMessage: 原始响应数据
//   - error: 错误信息
//
// 核心流程：
//  1. 验证会话有效性
//  2. 生成唯一请求ID
//  3. 创建响应通道并注册
//  4. 发送请求消息
//  5. 等待响应或超时
//
// [重要] 线程安全：
//   - 使用会话状态中的并发安全映射表管理请求
//
// 性能提示：
//   - 响应通道缓冲区大小为1，避免阻塞
func (server *Server) callClient(ctx context.Context, sessionID string, method protocol.Method, params protocol.ServerRequest) (json.RawMessage, error) {
	session, ok := server.sessionManager.GetSession(sessionID)
	if !ok {
		return nil, fmt.Errorf("callClient: %w", pkg.ErrLackSession)
	}

	requestID := strconv.FormatInt(session.IncRequestID(), 10)
	respChan := make(chan *protocol.JSONRPCResponse, 1)
	session.GetReqID2respChan().Set(requestID, respChan)
	defer session.GetReqID2respChan().Remove(requestID)

	if err := server.sendMsgWithRequest(ctx, sessionID, requestID, method, params); err != nil {
		return nil, fmt.Errorf("callClient: %w", err)
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
