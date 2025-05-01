package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/transport"
)

// Option 定义客户端配置选项类型
// 用于通过函数式选项模式配置Client实例
type Option func(*Client)

func WithNotifyHandler(handler NotifyHandler) Option {
	return func(s *Client) {
		s.notifyHandler = handler
	}
}

func WithSamplingHandler(handler SamplingHandler) Option {
	return func(s *Client) {
		s.samplingHandler = handler
	}
}

func WithClientInfo(info protocol.Implementation) Option {
	return func(s *Client) {
		s.clientInfo = &info
	}
}

func WithInitTimeout(timeout time.Duration) Option {
	return func(s *Client) {
		s.initTimeout = timeout
	}
}

func WithLogger(logger pkg.Logger) Option {
	return func(s *Client) {
		s.logger = logger
	}
}

// Client 定义MCP客户端核心结构
// transport: 底层传输层实现
// reqID2respChan: 请求ID到响应通道的映射
// samplingHandler: 采样消息处理器
// notifyHandler: 通知处理器
// requestID: 当前请求ID计数器
// ready: 标识客户端是否已初始化完成
// initializationMu: 初始化互斥锁
// clientInfo: 客户端实现信息
// clientCapabilities: 客户端能力声明
// serverCapabilities: 服务端能力声明
// serverInfo: 服务端实现信息
// serverInstructions: 服务端指令
// initTimeout: 初始化超时时间
// closed: 关闭信号通道
// logger: 日志记录器
type Client struct {
	transport transport.ClientTransport

	reqID2respChan cmap.ConcurrentMap[string, chan *protocol.JSONRPCResponse]

	samplingHandler SamplingHandler

	notifyHandler NotifyHandler

	requestID int64

	ready            *pkg.AtomicBool
	initializationMu sync.Mutex

	clientInfo         *protocol.Implementation
	clientCapabilities *protocol.ClientCapabilities

	serverCapabilities *protocol.ServerCapabilities
	serverInfo         *protocol.Implementation
	serverInstructions string

	initTimeout time.Duration

	closed chan struct{}

	logger pkg.Logger
}

// NewClient 创建新的MCP客户端
// t: 传输层实现
// opts: 配置选项
// 返回: 客户端实例和错误信息
// 1. 初始化客户端结构体
// 2. 设置传输层接收器
// 3. 应用配置选项
// 4. 设置默认通知处理器
// 5. 启动传输层
// 6. 执行初始化流程
// 7. 启动会话检测协程
func NewClient(t transport.ClientTransport, opts ...Option) (*Client, error) {
	client := &Client{
		transport:          t,
		reqID2respChan:     cmap.New[chan *protocol.JSONRPCResponse](),
		ready:              pkg.NewAtomicBool(),
		clientInfo:         &protocol.Implementation{},
		clientCapabilities: &protocol.ClientCapabilities{},
		initTimeout:        time.Second * 30,
		closed:             make(chan struct{}),
		logger:             pkg.DefaultLogger,
	}
	t.SetReceiver(transport.ClientReceiverF(client.receive))

	for _, opt := range opts {
		opt(client)
	}

	if client.notifyHandler == nil {
		h := NewBaseNotifyHandler()
		h.Logger = client.logger
		client.notifyHandler = h
	}

	if client.samplingHandler != nil {
		client.clientCapabilities.Sampling = struct{}{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), client.initTimeout)
	defer cancel()

	if err := client.transport.Start(); err != nil {
		return nil, fmt.Errorf("init mcp client transpor start fail: %w", err)
	}

	if _, err := client.initialization(ctx, protocol.NewInitializeRequest(*client.clientInfo, *client.clientCapabilities)); err != nil {
		return nil, err
	}

	go func() {
		defer pkg.Recover()

		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-client.closed:
				return
			case <-ticker.C:
				client.sessionDetection()
			}
		}
	}()

	return client, nil
}

// GetServerCapabilities 获取服务端能力声明
// 返回: 服务端能力结构体
func (client *Client) GetServerCapabilities() protocol.ServerCapabilities {
	return *client.serverCapabilities
}

// GetServerInfo 获取服务端实现信息
// 返回: 服务端实现信息结构体
func (client *Client) GetServerInfo() protocol.Implementation {
	return *client.serverInfo
}

// GetServerInstructions 获取服务端指令
// 返回: 服务端指令字符串
func (client *Client) GetServerInstructions() string {
	return client.serverInstructions
}

// Close 关闭客户端连接
// 返回: 错误信息
// 1. 发送关闭信号
// 2. 关闭底层传输层
func (client *Client) Close() error {
	close(client.closed)

	return client.transport.Close()
}

// sessionDetection 会话检测协程
// 1. 定期发送ping请求检测连接状态
// 2. 记录连接异常日志
func (client *Client) sessionDetection() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx, protocol.NewPingRequest()); err != nil {
		client.logger.Warnf("mcp client ping server fail: %v", err)
	}
}
