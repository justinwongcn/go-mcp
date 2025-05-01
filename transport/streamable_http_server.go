// Package transport 提供基于HTTP的可流式服务端传输实现
// [模块功能] 通过HTTP协议实现服务端与客户端的双向通信
// [项目定位] 属于go-mcp核心传输层，支持远程HTTP通信场景
// [版本历史]
// v1.0.0 2023-05-15 初始版本 支持基础HTTP通信
// v1.1.0 2023-06-20 增加SSE流式支持
// [依赖说明]
// - github.com/ThinkInAIXYZ/go-mcp/pkg >= v1.2.0
// [典型调用]
// transport.NewStreamableHTTPServerTransport(":8080")
package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

type StateMode string

// 定义状态模式常量
// [协议规范] 遵循MCP协议v1.0规范
const (
	Stateful  StateMode = "stateful"  // 有状态模式
	Stateless StateMode = "stateless" // 无状态模式
)

type SessionIDForReturnKey struct{}

type SessionIDForReturn struct {
	SessionID string
}

// StreamableHTTPServerTransportOption 服务端传输配置函数类型
// [设计决策] 采用函数选项模式实现灵活配置
type StreamableHTTPServerTransportOption func(*streamableHTTPServerTransport)

// WithStreamableHTTPServerTransportOptionLogger 设置日志记录器
// 输入: logger 日志记录器
// 输出: 配置函数
// [性能提示] 日志操作可能影响高并发场景性能
func WithStreamableHTTPServerTransportOptionLogger(logger pkg.Logger) StreamableHTTPServerTransportOption {
	return func(t *streamableHTTPServerTransport) {
		t.logger = logger
	}
}

func WithStreamableHTTPServerTransportOptionEndpoint(endpoint string) StreamableHTTPServerTransportOption {
	return func(t *streamableHTTPServerTransport) {
		t.mcpEndpoint = endpoint
	}
}

func WithStreamableHTTPServerTransportOptionStateMode(mode StateMode) StreamableHTTPServerTransportOption {
	return func(t *streamableHTTPServerTransport) {
		t.stateMode = mode
	}
}

type StreamableHTTPServerTransportAndHandlerOption func(*streamableHTTPServerTransport)

func WithStreamableHTTPServerTransportAndHandlerOptionLogger(logger pkg.Logger) StreamableHTTPServerTransportAndHandlerOption {
	return func(t *streamableHTTPServerTransport) {
		t.logger = logger
	}
}

func WithStreamableHTTPServerTransportAndHandlerOptionStateMode(mode StateMode) StreamableHTTPServerTransportAndHandlerOption {
	return func(t *streamableHTTPServerTransport) {
		t.stateMode = mode
	}
}

// streamableHTTPServerTransport HTTP可流式服务端传输实现
// [重要] 线程安全设计，支持并发调用
type streamableHTTPServerTransport struct {
	// ctx 控制服务器生命周期的上下文
	ctx    context.Context
	cancel context.CancelFunc

	httpSvr *http.Server // HTTP服务器实例

	stateMode StateMode // 状态模式

	inFlySend sync.WaitGroup // 进行中的发送操作计数器

	receiver serverReceiver // 消息接收处理器

	sessionManager sessionManager // 会话管理器

	// 配置选项
	logger      pkg.Logger // 日志记录器
	mcpEndpoint string     // MCP端点路径
}

type StreamableHTTPHandler struct {
	transport *streamableHTTPServerTransport
}

// HandleMCP handles incoming MCP requests
func (h *StreamableHTTPHandler) HandleMCP() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.transport.handleMCPEndpoint(w, r)
	})
}

// NewStreamableHTTPServerTransportAndHandler returns transport without starting the HTTP server,
// and returns a Handler for users to start their own HTTP server externally
// eg:
// transport, handler, _ := NewStreamableHTTPServerTransportAndHandler()
// http.Handle("/mcp", handler.HandleMCP())
// http.ListenAndServe(":8080", nil)
func NewStreamableHTTPServerTransportAndHandler(
	opts ...StreamableHTTPServerTransportAndHandlerOption,
) (ServerTransport, *StreamableHTTPHandler, error) { //nolint:whitespace

	ctx, cancel := context.WithCancel(context.Background())

	t := &streamableHTTPServerTransport{
		ctx:       ctx,
		cancel:    cancel,
		stateMode: Stateless,
		logger:    pkg.DefaultLogger,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t, &StreamableHTTPHandler{transport: t}, nil
}

// NewStreamableHTTPServerTransport 创建HTTP可流式服务端传输实例
// 输入:
// - addr 服务监听地址
// - opts 可选配置函数
// 输出: ServerTransport接口实例
// [典型用例]
// 启动MCP服务:
// transport.NewStreamableHTTPServerTransport(":8080")
// [副作用] 会初始化HTTP服务器但不会自动启动
// [安全要求] 需确保配置了适当的TLS设置
func NewStreamableHTTPServerTransport(addr string, opts ...StreamableHTTPServerTransportOption) ServerTransport {
	ctx, cancel := context.WithCancel(context.Background())

	t := &streamableHTTPServerTransport{
		ctx:         ctx,
		cancel:      cancel,
		stateMode:   Stateless,
		logger:      pkg.DefaultLogger,
		mcpEndpoint: "/mcp", // 默认MCP端点
	}

	for _, opt := range opts {
		opt(t)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(t.mcpEndpoint, t.handleMCPEndpoint)

	t.httpSvr = &http.Server{
		Addr:        addr,
		Handler:     mux,
		IdleTimeout: time.Minute,
	}

	return t
}

func (t *streamableHTTPServerTransport) Run() error {
	if t.httpSvr == nil {
		<-t.ctx.Done()
		return nil
	}

	fmt.Printf("starting mcp server at http://%s%s\n", t.httpSvr.Addr, t.mcpEndpoint)

	if err := t.httpSvr.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	return nil
}

func (t *streamableHTTPServerTransport) Send(ctx context.Context, sessionID string, msg Message) error {
	t.inFlySend.Add(1)
	defer t.inFlySend.Done()

	select {
	case <-t.ctx.Done():
		return t.ctx.Err()
	default:
		return t.sessionManager.EnqueueMessageForSend(ctx, sessionID, msg)
	}
}

func (t *streamableHTTPServerTransport) SetReceiver(receiver serverReceiver) {
	t.receiver = receiver
}

func (t *streamableHTTPServerTransport) SetSessionManager(manager sessionManager) {
	t.sessionManager = manager
}

func (t *streamableHTTPServerTransport) handleMCPEndpoint(w http.ResponseWriter, r *http.Request) {
	defer pkg.RecoverWithFunc(func(_ any) {
		t.writeError(w, http.StatusInternalServerError, "Internal server error")
	})

	switch r.Method {
	case http.MethodPost:
		t.handlePost(w, r)
	case http.MethodGet:
		t.handleGet(w, r)
	case http.MethodDelete:
		t.handleDelete(w, r)
	default:
		t.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (t *streamableHTTPServerTransport) handlePost(w http.ResponseWriter, r *http.Request) {
	// Validate Accept header
	accept := r.Header.Get("Accept")
	if accept == "" {
		t.writeError(w, http.StatusBadRequest, "Missing Accept header")
		return
	}

	// Read and process the message
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		t.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// Disconnection SHOULD NOT be interpreted as the client canceling its request.
	// To cancel, the client SHOULD explicitly send an MCP CancelledNotification.
	ctx := pkg.NewCancelShieldContext(r.Context())

	// For InitializeRequest HTTP response
	if t.stateMode == Stateful {
		ctx = context.WithValue(ctx, SessionIDForReturnKey{}, &SessionIDForReturn{})
	}

	outputMsgCh, err := t.receiver.Receive(ctx, r.Header.Get(sessionIDHeader), bs)
	if err != nil {
		if errors.Is(err, pkg.ErrSessionClosed) {
			t.writeError(w, http.StatusNotFound, fmt.Sprintf("Failed to receive: %v", err))
			return
		}
		t.writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to receive: %v", err))
		return
	}

	if outputMsgCh == nil { // reply response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		return
	}

	msg := <-outputMsgCh
	if len(msg) == 0 {
		t.writeError(w, http.StatusInternalServerError, "handle request fail")
		return
	}

	if t.stateMode == Stateful {
		if sid := ctx.Value(SessionIDForReturnKey{}).(*SessionIDForReturn); sid.SessionID != "" { // in server.handleRequestWithInitialize assign
			w.Header().Set(sessionIDHeader, sid.SessionID)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err = w.Write(msg); err != nil {
		t.logger.Errorf("streamableHTTPServerTransport post write: %+v", err)
		return
	}
}

func (t *streamableHTTPServerTransport) handleGet(w http.ResponseWriter, r *http.Request) {
	defer pkg.RecoverWithFunc(func(_ any) {
		t.writeError(w, http.StatusInternalServerError, "Internal server error")
	})

	if t.stateMode == Stateless {
		t.writeError(w, http.StatusMethodNotAllowed, "server is stateless, not support sse connection")
		return
	}

	if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		t.writeError(w, http.StatusBadRequest, "Must accept text/event-stream")
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create flush-supporting writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		t.writeError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}
	sessionID := r.Header.Get(sessionIDHeader)
	if sessionID == "" {
		t.writeError(w, http.StatusBadRequest, "Missing Session ID")
		flusher.Flush()
		return
	}
	if err := t.sessionManager.OpenMessageQueueForSend(sessionID); err != nil {
		t.writeError(w, http.StatusBadRequest, err.Error())
		flusher.Flush()
		return
	}
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for {
		msg, err := t.sessionManager.DequeueMessageForSend(r.Context(), sessionID)
		if err != nil {
			if errors.Is(err, pkg.ErrSendEOF) {
				return
			}
			t.logger.Debugf("sse connect dequeueMessage err: %+v, sessionID=%s", err.Error(), sessionID)
			return
		}

		t.logger.Debugf("Sending message: %s", string(msg))

		if _, err = fmt.Fprintf(w, "data: %s\n\n", msg); err != nil {
			t.logger.Errorf("Failed to write message: %v", err)
			continue
		}
		flusher.Flush()
	}
}

func (t *streamableHTTPServerTransport) handleDelete(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		t.writeError(w, http.StatusBadRequest, "Missing session ID")
		return
	}

	t.sessionManager.CloseSession(sessionID)
	w.WriteHeader(http.StatusOK)
}

func (t *streamableHTTPServerTransport) writeError(w http.ResponseWriter, code int, message string) {
	if code == http.StatusMethodNotAllowed {
		t.logger.Infof("streamableHTTPServerTransport response: code: %d, message: %s", code, message)
	} else {
		t.logger.Errorf("streamableHTTPServerTransport Error: code: %d, message: %s", code, message)
	}

	resp := protocol.NewJSONRPCErrorResponse(nil, protocol.InternalError, message)
	bytes, err := json.Marshal(resp)
	if err != nil {
		t.logger.Errorf("streamableHTTPServerTransport writeError json.Marshal: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write(bytes); err != nil {
		t.logger.Errorf("streamableHTTPServerTransport writeError Write: %v", err)
	}
}

func (t *streamableHTTPServerTransport) Shutdown(userCtx context.Context, serverCtx context.Context) error {
	shutdownFunc := func() {
		<-serverCtx.Done()

		t.cancel()

		t.inFlySend.Wait()

		t.sessionManager.CloseAllSessions()
	}

	if t.httpSvr == nil {
		shutdownFunc()
		return nil
	}

	t.httpSvr.RegisterOnShutdown(shutdownFunc)

	if err := t.httpSvr.Shutdown(userCtx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	return nil
}
