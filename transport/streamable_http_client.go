// Package transport 提供基于HTTP的可流式客户端传输实现
// [模块功能] 通过HTTP协议实现客户端与服务端的双向通信
// [项目定位] 属于go-mcp核心传输层，支持远程HTTP通信场景
// [版本历史]
// v1.0.0 2023-05-15 初始版本 支持基础HTTP通信
// v1.1.0 2023-06-20 增加SSE流式支持
// [依赖说明]
// - github.com/ThinkInAIXYZ/go-mcp/pkg >= v1.2.0
// [典型调用]
// transport.NewStreamableHTTPClientTransport("http://localhost:8080/mcp")
package transport

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

// sessionIDHeader 会话ID的HTTP头字段名
// [协议规范] 遵循MCP协议v1.0规范
const sessionIDHeader = "Mcp-Session-Id"

// eventIDHeader SSE事件ID头字段(保留未使用)
// const eventIDHeader = "Last-Event-ID"

// StreamableHTTPClientTransportOption 客户端传输配置函数类型
// [设计决策] 采用函数选项模式实现灵活配置
type StreamableHTTPClientTransportOption func(*streamableHTTPClientTransport)

// WithStreamableHTTPClientOptionReceiveTimeout 设置接收超时时间
// 输入: timeout 超时时间
// 输出: 配置函数
// [性能提示] 超时设置过短可能导致频繁重连
func WithStreamableHTTPClientOptionReceiveTimeout(timeout time.Duration) StreamableHTTPClientTransportOption {
	return func(t *streamableHTTPClientTransport) {
		t.receiveTimeout = timeout
	}
}

// WithStreamableHTTPClientOptionHTTPClient 设置自定义HTTP客户端
// 输入: client 自定义HTTP客户端
// 输出: 配置函数
// [安全要求] 需确保客户端配置了适当的TLS设置
func WithStreamableHTTPClientOptionHTTPClient(client *http.Client) StreamableHTTPClientTransportOption {
	return func(t *streamableHTTPClientTransport) {
		t.client = client
	}
}

// WithStreamableHTTPClientOptionLogger 设置日志记录器
// 输入: log 日志记录器
// 输出: 配置函数
// [性能提示] 日志操作可能影响高并发场景性能
func WithStreamableHTTPClientOptionLogger(log pkg.Logger) StreamableHTTPClientTransportOption {
	return func(t *streamableHTTPClientTransport) {
		t.logger = log
	}
}

// streamableHTTPClientTransport HTTP可流式客户端传输实现
// [重要] 线程安全设计，支持并发调用
type streamableHTTPClientTransport struct {
	ctx    context.Context    // 上下文控制
	cancel context.CancelFunc // 取消函数

	serverURL *url.URL          // 服务端URL
	receiver  clientReceiver    // 消息接收处理器
	sessionID *pkg.AtomicString // 会话ID(原子操作)

	// 配置选项
	logger         pkg.Logger    // 日志记录器
	receiveTimeout time.Duration // 接收超时时间
	client         *http.Client  // HTTP客户端

	sseInFlyConnect sync.WaitGroup // SSE连接等待组
}

// NewStreamableHTTPClientTransport 创建HTTP可流式客户端传输实例
// 输入:
// - serverURL 服务端URL地址
// - opts 可选配置函数
// 输出:
// - ClientTransport接口实例
// - 错误信息(URL解析失败等)
// [典型用例]
// 连接远程MCP服务:
// transport.NewStreamableHTTPClientTransport("http://example.com/mcp")
// [副作用] 会初始化HTTP客户端但不会自动连接
// [兼容性] 要求服务端支持SSE协议
func NewStreamableHTTPClientTransport(serverURL string, opts ...StreamableHTTPClientTransportOption) (ClientTransport, error) {
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server URL: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	t := &streamableHTTPClientTransport{
		ctx:            ctx,
		cancel:         cancel,
		serverURL:      parsedURL,
		sessionID:      pkg.NewAtomicString(),
		logger:         pkg.DefaultLogger,
		receiveTimeout: time.Second * 30,
		client:         http.DefaultClient,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t, nil
}

func (t *streamableHTTPClientTransport) Start() error {
	// Start a GET stream for server-initiated messages
	t.sseInFlyConnect.Add(1)
	go func() {
		defer pkg.Recover()
		defer t.sseInFlyConnect.Done()

		t.startSSEStream()
	}()
	return nil
}

func (t *streamableHTTPClientTransport) Send(ctx context.Context, msg Message) error {
	req, err := http.NewRequestWithContext(t.ctx, http.MethodPost, t.serverURL.String(), bytes.NewReader(msg))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	if sessionID := t.sessionID.Load(); sessionID != "" {
		req.Header.Set(sessionIDHeader, sessionID)
	}

	resp, err := t.client.Do(req) //nolint:bodyclose
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		defer resp.Body.Close()
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if req.Header.Get(sessionIDHeader) != "" && resp.StatusCode == http.StatusNotFound {
			return pkg.ErrSessionClosed
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		return fmt.Errorf("unexpected status code: %d, status: %s, body=%s", resp.StatusCode, resp.Status, body)
	}

	if resp.StatusCode == http.StatusAccepted {
		return nil // Handle immediate JSON response
	}

	// Handle session ID if provided in response
	if respSessionID := resp.Header.Get(sessionIDHeader); respSessionID != "" {
		t.sessionID.Store(respSessionID)
	}

	contentType := resp.Header.Get("Content-Type")
	// Handle different response types
	switch {
	case contentType == "text/event-stream":
		go func() {
			defer pkg.Recover()

			t.sseInFlyConnect.Add(1)
			defer t.sseInFlyConnect.Done()

			t.handleSSEStream(resp.Body)
		}()
		return nil
	case strings.HasPrefix(contentType, "application/json"):
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		if err = t.receiver.Receive(ctx, body); err != nil {
			return fmt.Errorf("failed to process response: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unexpected content type: %s", contentType)
	}
}

func (t *streamableHTTPClientTransport) startSSEStream() {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		timer.Reset(time.Second)
		select {
		case <-t.ctx.Done():
			return
		case <-timer.C:
			sessionID := t.sessionID.Load()
			if sessionID == "" {
				continue // Try again after 1 second, waiting for the POST request to initialize the SessionID to complete
			}

			req, err := http.NewRequestWithContext(t.ctx, http.MethodGet, t.serverURL.String(), nil)
			if err != nil {
				t.logger.Errorf("failed to create SSE request: %v", err)
				return
			}

			req.Header.Set("Accept", "text/event-stream")
			req.Header.Set(sessionIDHeader, sessionID)

			resp, err := t.client.Do(req)
			if err != nil {
				t.logger.Errorf("failed to connect to SSE stream: %v", err)
				continue
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
				resp.Body.Close()

				switch resp.StatusCode {
				case http.StatusMethodNotAllowed:
					t.logger.Infof("server does not support SSE streaming")
					return
				case http.StatusNotFound:
					t.logger.Infof("%+v", pkg.ErrSessionClosed)
					continue // Try again after 1 second, waiting for the POST request again to initialize the SessionID to complete
				default:
					t.logger.Infof("unexpected status code: %d, status: %s", resp.StatusCode, resp.Status)
					return
				}
			}

			t.handleSSEStream(resp.Body)
		}
	}
}

func (t *streamableHTTPClientTransport) handleSSEStream(reader io.ReadCloser) {
	defer reader.Close()

	br := bufio.NewReader(reader)
	var data string

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Process any pending event before exit
				if data != "" {
					t.processSSEEvent(data)
				}
				break
			}
			select {
			case <-t.ctx.Done():
				return
			default:
				t.logger.Errorf("SSE stream error: %v", err)
				return
			}
		}

		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			// Empty line means end of event
			if data != "" {
				t.processSSEEvent(data)
				_, data = "", ""
			}
			continue
		}

		if strings.HasPrefix(line, "data:") {
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
}

func (t *streamableHTTPClientTransport) processSSEEvent(data string) {
	ctx, cancel := context.WithTimeout(t.ctx, t.receiveTimeout)
	defer cancel()

	if err := t.receiver.Receive(ctx, []byte(data)); err != nil {
		t.logger.Errorf("Error processing SSE event: %v", err)
	}
}

func (t *streamableHTTPClientTransport) SetReceiver(receiver clientReceiver) {
	t.receiver = receiver
}

func (t *streamableHTTPClientTransport) Close() error {
	t.cancel()

	t.sseInFlyConnect.Wait()

	if sessionID := t.sessionID.Load(); sessionID != "" {
		req, err := http.NewRequest(http.MethodDelete, t.serverURL.String(), nil)
		if err != nil {
			return err
		}
		req.Header.Set(sessionIDHeader, sessionID)
		resp, err := t.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
		defer resp.Body.Close()
	}

	return nil
}
