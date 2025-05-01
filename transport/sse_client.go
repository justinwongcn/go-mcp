// Package transport 实现基于SSE的客户端传输逻辑
// 模块功能：处理客户端与服务器之间的SSE连接和消息传输
// 项目定位：go-mcp项目的核心传输组件
// 版本历史：
// - 2023-10-01 初始版本 (ThinkInAI)
// 依赖说明：
// - net/http: HTTP客户端实现
// - github.com/ThinkInAIXYZ/go-mcp/pkg: 基础工具包
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
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

// SSEClientTransportOption 定义SSE客户端传输的配置选项类型
// 用于通过函数式选项模式配置传输参数
type SSEClientTransportOption func(*sseClientTransport)

// WithSSEClientOptionReceiveTimeout 设置SSE接收超时时间
// 输入参数：
// - timeout: 超时时间
// 返回值：
// - SSEClientTransportOption: 配置函数
// [注意] 超时时间必须大于0
func WithSSEClientOptionReceiveTimeout(timeout time.Duration) SSEClientTransportOption {
	return func(t *sseClientTransport) {
		t.receiveTimeout = timeout
	}
}

// WithSSEClientOptionHTTPClient 设置自定义HTTP客户端
// 输入参数：
// - client: 自定义HTTP客户端实例
// 返回值：
// - SSEClientTransportOption: 配置函数
// [注意] 客户端必须支持长连接
func WithSSEClientOptionHTTPClient(client *http.Client) SSEClientTransportOption {
	return func(t *sseClientTransport) {
		t.client = client
	}
}

// WithSSEClientOptionLogger 设置自定义日志记录器
// 输入参数：
// - log: 日志记录器实例
// 返回值：
// - SSEClientTransportOption: 配置函数
// [重要] 日志记录器必须实现pkg.Logger接口
func WithSSEClientOptionLogger(log pkg.Logger) SSEClientTransportOption {
	return func(t *sseClientTransport) {
		t.logger = log
	}
}

// sseClientTransport 实现基于SSE的客户端传输
// 结构体字段说明：
// - ctx: 上下文控制传输生命周期
// - cancel: 取消函数
// - serverURL: 服务器URL
// - endpointChan: 端点通知通道
// - messageEndpoint: 消息端点URL
// - receiver: 消息接收器
// - logger: 日志记录器
// - receiveTimeout: 接收超时时间
// - client: HTTP客户端
// - sseConnectClose: SSE连接关闭信号
// [注意] 所有字段访问需要同步控制
type sseClientTransport struct {
	ctx    context.Context
	cancel context.CancelFunc

	serverURL *url.URL

	endpointChan    chan struct{}
	messageEndpoint *url.URL
	receiver        clientReceiver

	// options
	logger         pkg.Logger
	receiveTimeout time.Duration
	client         *http.Client

	sseConnectClose chan struct{}
}

// NewSSEClientTransport 创建新的SSE客户端传输实例
// 输入参数：
// - serverURL: 服务器URL
// - opts: 配置选项
// 返回值：
// - ClientTransport: 传输接口实例
// - error: 创建错误
// 功能说明：
// 1. 解析服务器URL
// 2. 应用配置选项
// 3. 初始化传输结构体
// [重要] 服务器URL必须有效
func NewSSEClientTransport(serverURL string, opts ...SSEClientTransportOption) (ClientTransport, error) {
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server URL: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	x := &sseClientTransport{
		ctx:             ctx,
		cancel:          cancel,
		serverURL:       parsedURL,
		endpointChan:    make(chan struct{}, 1),
		messageEndpoint: nil,
		receiver:        nil,
		logger:          pkg.DefaultLogger,
		receiveTimeout:  time.Second * 30,
		client:          http.DefaultClient,
		sseConnectClose: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(x)
	}

	return x, nil
}

// Start 启动SSE客户端传输
// 返回值：
// - error: 启动错误
// 功能说明：
// 1. 建立SSE连接
// 2. 启动消息接收协程
// [注意] 只能调用一次
func (t *sseClientTransport) Start() error {
	errChan := make(chan error, 1)
	go func() {
		defer pkg.Recover()

		req, err := http.NewRequestWithContext(t.ctx, http.MethodGet, t.serverURL.String(), nil)
		if err != nil {
			errChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Connection", "keep-alive")

		resp, err := t.client.Do(req) //nolint:bodyclose
		if err != nil {
			errChan <- fmt.Errorf("failed to connect to SSE stream: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errChan <- fmt.Errorf("unexpected status code: %d, status: %s", resp.StatusCode, resp.Status)
			return
		}

		t.readSSE(resp.Body)

		close(t.sseConnectClose)
	}()

	// Wait for the endpoint to be received
	select {
	case <-t.endpointChan:
	// Endpoint received, proceed
	case err := <-errChan:
		return fmt.Errorf("error in SSE stream: %w", err)
	case <-time.After(10 * time.Second): // Add a timeout
		return fmt.Errorf("timeout waiting for endpoint")
	}

	return nil
}

// readSSE continuously reads the SSE stream and processes events.
// It runs until the connection is closed or an error occurs.
// readSSE 读取SSE流数据
// 输入参数：
// - reader: 读取器
// 功能说明：
// 1. 解析SSE事件格式
// 2. 分发到对应处理器
// [注意] 需要处理连接中断
func (t *sseClientTransport) readSSE(reader io.ReadCloser) {
	defer func() {
		_ = reader.Close()
	}()

	br := bufio.NewReader(reader)
	var event, data string

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Process any pending event before exit
				if event != "" && data != "" {
					t.handleSSEEvent(event, data)
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

		// Remove only newline markers
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			// Empty line means end of event
			if event != "" && data != "" {
				t.handleSSEEvent(event, data)
				event = ""
				data = ""
			}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
}

// handleSSEEvent processes SSE events based on their type.
// Handles 'endpoint' events for connection setup and 'message' events for JSON-RPC communication.
// handleSSEEvent 处理SSE事件
// 输入参数：
// - event: 事件类型
// - data: 事件数据
// 功能说明：
// 1. 处理endpoint事件建立消息通道
// 2. 处理message事件转发消息
// [重要] 事件类型必须有效
func (t *sseClientTransport) handleSSEEvent(event, data string) {
	switch event {
	case "endpoint":
		endpoint, err := t.serverURL.Parse(data)
		if err != nil {
			t.logger.Errorf("Error parsing endpoint URL: %v", err)
			return
		}
		t.logger.Debugf("Received endpoint: %s", endpoint.String())
		t.messageEndpoint = endpoint
		close(t.endpointChan)
	case "message":
		ctx, cancel := context.WithTimeout(t.ctx, t.receiveTimeout)
		defer cancel()
		if err := t.receiver.Receive(ctx, []byte(data)); err != nil {
			t.logger.Errorf("Error receive message: %v", err)
			return
		}
	}
}

// Send 发送消息到服务器
// 输入参数：
// - ctx: 上下文
// - msg: 消息内容
// 返回值：
// - error: 发送错误
// 功能说明：
// 1. 构造HTTP请求
// 2. 发送消息
// [注意] 需要有效的消息端点
func (t *sseClientTransport) Send(ctx context.Context, msg Message) error {
	t.logger.Debugf("Sending message: %s to %s", msg, t.messageEndpoint.String())

	var (
		err  error
		req  *http.Request
		resp *http.Response
	)

	req, err = http.NewRequestWithContext(ctx, http.MethodPost, t.messageEndpoint.String(), bytes.NewReader(msg))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if resp, err = t.client.Do(req); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unexpected status code: %d, status: %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// SetReceiver 设置消息接收器
// 输入参数：
// - receiver: 接收器实例
// 功能说明：
// 1. 设置消息处理回调
// [重要] 接收器必须实现clientReceiver接口
func (t *sseClientTransport) SetReceiver(receiver clientReceiver) {
	t.receiver = receiver
}

// Close 关闭SSE客户端传输
// 返回值：
// - error: 关闭错误
// 功能说明：
// 1. 取消上下文
// 2. 等待接收协程退出
// 3. 关闭连接
// [注意] 确保资源释放
func (t *sseClientTransport) Close() error {
	t.cancel()

	<-t.sseConnectClose

	return nil
}
