// Package server 实现MCP协议的服务端核心逻辑
// 模块功能：处理客户端请求和响应，包括请求分发、会话管理和错误处理
// 项目定位：go-mcp项目的核心通信处理组件
// 版本历史：
// - 2023-10-01 初始版本 (ThinkInAI)
// - 2023-11-15 增加会话状态校验 (ThinkInAI)
// 依赖说明：
// - github.com/tidwall/gjson: JSON快速解析
// - github.com/ThinkInAIXYZ/go-mcp/pkg: 基础工具包
// - github.com/ThinkInAIXYZ/go-mcp/protocol: MCP协议定义
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// receive 处理客户端发送的消息
// 输入参数：
// - ctx: 上下文
// - sessionID: 会话ID
// - msg: 原始消息字节
// 返回值：
// - <-chan []byte: 响应消息通道
// - error: 处理错误
// 功能说明：
// 1. 校验会话状态
// 2. 区分通知、请求和响应
// 3. 分发到对应处理方法
// [注意] 该方法会创建goroutine处理耗时操作
func (server *Server) receive(ctx context.Context, sessionID string, msg []byte) (<-chan []byte, error) {
	if sessionID != "" && !server.sessionManager.IsActiveSession(sessionID) {
		if server.sessionManager.IsClosedSession(sessionID) {
			return nil, pkg.ErrSessionClosed
		}
		return nil, pkg.ErrLackSession
	}

	if !gjson.GetBytes(msg, "id").Exists() {
		notify := &protocol.JSONRPCNotification{}
		if err := pkg.JSONUnmarshal(msg, &notify); err != nil {
			return nil, err
		}
		if err := server.receiveNotify(sessionID, notify); err != nil {
			notify.RawParams = nil // simplified log
			server.logger.Errorf("receive notify:%+v error: %s", notify, err.Error())
			return nil, err
		}
		return nil, nil
	}

	// case request or response
	if !gjson.GetBytes(msg, "method").Exists() {
		resp := &protocol.JSONRPCResponse{}
		if err := pkg.JSONUnmarshal(msg, &resp); err != nil {
			return nil, err
		}

		if err := server.receiveResponse(sessionID, resp); err != nil {
			resp.RawResult = nil // simplified log
			server.logger.Errorf("receive response:%+v error: %s", resp, err.Error())
			return nil, err
		}
		return nil, nil
	}

	req := &protocol.JSONRPCRequest{}
	if err := pkg.JSONUnmarshal(msg, &req); err != nil {
		return nil, err
	}
	if !req.IsValid() {
		return nil, pkg.ErrRequestInvalid
	}

	if sessionID != "" && req.Method != protocol.Initialize && req.Method != protocol.Ping {
		if s, ok := server.sessionManager.GetSession(sessionID); !ok {
			return nil, pkg.ErrLackSession
		} else if !s.GetReady() {
			return nil, pkg.ErrSessionHasNotInitialized
		}
	}

	server.inFlyRequest.Add(1)

	if server.inShutdown.Load() {
		server.inFlyRequest.Done()
		return nil, errors.New("server already shutdown")
	}

	ch := make(chan []byte, 1)
	go func(ctx context.Context) {
		defer pkg.Recover()
		defer server.inFlyRequest.Done()
		defer close(ch)

		resp := server.receiveRequest(ctx, sessionID, req)
		message, err := json.Marshal(resp)
		if err != nil {
			server.logger.Errorf("receive json marshal response:%+v error: %s", resp, err.Error())
			return
		}
		ch <- message
	}(pkg.NewCancelShieldContext(ctx))
	return ch, nil
}

// receiveRequest 处理客户端请求
// 输入参数：
// - ctx: 上下文
// - sessionID: 会话ID
// - request: JSON-RPC请求
// 返回值：
// - *protocol.JSONRPCResponse: JSON-RPC响应
// 功能说明：
// 1. 更新会话活跃时间
// 2. 根据方法名分发到对应处理器
// 3. 统一错误处理
// [重要] 所有请求方法必须在此注册
func (server *Server) receiveRequest(ctx context.Context, sessionID string, request *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	ctx = setSessionIDToCtx(ctx, sessionID)

	if request.Method != protocol.Ping {
		server.sessionManager.UpdateSessionLastActiveAt(sessionID)
	}

	var (
		result protocol.ServerResponse
		err    error
	)

	switch request.Method {
	case protocol.Ping:
		result, err = server.handleRequestWithPing()
	case protocol.Initialize:
		result, err = server.handleRequestWithInitialize(ctx, sessionID, request.RawParams)
	case protocol.PromptsList:
		result, err = server.handleRequestWithListPrompts(request.RawParams)
	case protocol.PromptsGet:
		result, err = server.handleRequestWithGetPrompt(ctx, request.RawParams)
	case protocol.ResourcesList:
		result, err = server.handleRequestWithListResources(request.RawParams)
	case protocol.ResourceListTemplates:
		result, err = server.handleRequestWithListResourceTemplates(request.RawParams)
	case protocol.ResourcesRead:
		result, err = server.handleRequestWithReadResource(ctx, request.RawParams)
	case protocol.ResourcesSubscribe:
		result, err = server.handleRequestWithSubscribeResourceChange(sessionID, request.RawParams)
	case protocol.ResourcesUnsubscribe:
		result, err = server.handleRequestWithUnSubscribeResourceChange(sessionID, request.RawParams)
	case protocol.ToolsList:
		result, err = server.handleRequestWithListTools(request.RawParams)
	case protocol.ToolsCall:
		result, err = server.handleRequestWithCallTool(ctx, request.RawParams)
	default:
		err = fmt.Errorf("%w: method=%s", pkg.ErrMethodNotSupport, request.Method)
	}

	if err != nil {
		var code int
		switch {
		case errors.Is(err, pkg.ErrMethodNotSupport):
			code = protocol.MethodNotFound
		case errors.Is(err, pkg.ErrRequestInvalid):
			code = protocol.InvalidRequest
		case errors.Is(err, pkg.ErrJSONUnmarshal):
			code = protocol.ParseError
		default:
			code = protocol.InternalError
		}
		return protocol.NewJSONRPCErrorResponse(request.ID, code, err.Error())
	}
	return protocol.NewJSONRPCSuccessResponse(request.ID, result)
}

// receiveNotify 处理客户端通知
// 输入参数：
// - sessionID: 会话ID
// - notify: JSON-RPC通知
// 返回值：
// - error: 处理错误
// 功能说明：
// 1. 校验会话状态
// 2. 根据通知类型分发处理
// [注意] 通知不期待响应
func (server *Server) receiveNotify(sessionID string, notify *protocol.JSONRPCNotification) error {
	if sessionID != "" {
		if s, ok := server.sessionManager.GetSession(sessionID); !ok {
			return pkg.ErrLackSession
		} else if notify.Method != protocol.NotificationInitialized && !s.GetReady() {
			return pkg.ErrSessionHasNotInitialized
		}
	}

	switch notify.Method {
	case protocol.NotificationInitialized:
		return server.handleNotifyWithInitialized(sessionID, notify.RawParams)
	default:
		return fmt.Errorf("%w: method=%s", pkg.ErrMethodNotSupport, notify.Method)
	}
}

// receiveResponse 处理客户端响应
// 输入参数：
// - sessionID: 会话ID
// - response: JSON-RPC响应
// 返回值：
// - error: 处理错误
// 功能说明：
// 1. 查找对应的请求通道
// 2. 将响应发送到通道
// [重要] 必须确保请求-响应匹配
func (server *Server) receiveResponse(sessionID string, response *protocol.JSONRPCResponse) error {
	s, ok := server.sessionManager.GetSession(sessionID)
	if !ok {
		return pkg.ErrLackSession
	}

	respChan, ok := s.GetReqID2respChan().Get(fmt.Sprint(response.ID))
	if !ok {
		return fmt.Errorf("%w: sessionID=%+v, requestID=%+v", pkg.ErrLackResponseChan, sessionID, response.ID)
	}

	select {
	case respChan <- response:
	default:
		return fmt.Errorf("%w: sessionID=%+v, response=%+v", pkg.ErrDuplicateResponseReceived, sessionID, response)
	}
	return nil
}
