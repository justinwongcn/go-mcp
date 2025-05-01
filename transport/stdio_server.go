// Package transport 提供标准输入输出(stdio)的服务端传输实现
// [模块功能] 通过标准输入输出流实现进程间通信服务
// [项目定位] 属于go-mcp核心传输层，支持本地进程间通信场景
// [版本历史]
// v1.0.0 2023-05-15 初始版本 支持基础stdio通信
// v1.1.0 2023-06-20 增加会话管理功能
// [依赖说明]
// - github.com/ThinkInAIXYZ/go-mcp/pkg >= v1.2.0
// [典型调用]
// transport.NewStdioServerTransport()
package transport

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

// StdioServerTransportOption 服务端传输配置函数类型
// [设计决策] 采用函数选项模式实现灵活配置
type StdioServerTransportOption func(*stdioServerTransport)

// WithStdioServerOptionLogger 配置日志记录器
// 输入: pkg.Logger接口实现
// 输出: 配置函数
// [性能提示] 日志操作可能影响高并发场景性能
func WithStdioServerOptionLogger(log pkg.Logger) StdioServerTransportOption {
	return func(t *stdioServerTransport) {
		t.logger = log
	}
}

// stdioServerTransport 标准输入输出服务端传输实现
// [重要] 非线程安全，并发调用需外部同步
// [调试技巧] 可通过设置详细日志级别跟踪消息流
type stdioServerTransport struct {
	receiver serverReceiver // 消息接收处理器
	reader   io.ReadCloser  // 标准输入读取器
	writer   io.Writer      // 标准输出写入器

	sessionManager sessionManager // 会话管理器
	sessionID      string         // 当前会话ID

	logger pkg.Logger // 日志记录器

	cancel          context.CancelFunc // 上下文取消函数
	receiveShutDone chan struct{}      // 接收协程关闭信号
}

// NewStdioServerTransport 创建标准输入输出服务端传输实例
// 输入: 可选配置函数
// 输出: ServerTransport接口实例
// [典型用例]
// 作为命令行工具的后端服务:
// transport.NewStdioServerTransport()
// [副作用] 会占用标准输入输出流
// [兼容性] 要求客户端支持行缓冲模式通信
func NewStdioServerTransport(opts ...StdioServerTransportOption) ServerTransport {
	t := &stdioServerTransport{
		reader: os.Stdin,
		writer: os.Stdout,
		logger: pkg.DefaultLogger,

		receiveShutDone: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *stdioServerTransport) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel

	t.sessionID = t.sessionManager.CreateSession()

	t.startReceive(ctx)

	close(t.receiveShutDone)
	return nil
}

func (t *stdioServerTransport) Send(_ context.Context, _ string, msg Message) error {
	if _, err := t.writer.Write(append(msg, mcpMessageDelimiter)); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}
	return nil
}

func (t *stdioServerTransport) SetReceiver(receiver serverReceiver) {
	t.receiver = receiver
}

func (t *stdioServerTransport) SetSessionManager(m sessionManager) {
	t.sessionManager = m
}

func (t *stdioServerTransport) Shutdown(userCtx context.Context, serverCtx context.Context) error {
	t.cancel()

	if err := t.reader.Close(); err != nil {
		return err
	}

	select {
	case <-t.receiveShutDone:
		return nil
	case <-serverCtx.Done():
		return nil
	case <-userCtx.Done():
		return userCtx.Err()
	}
}

func (t *stdioServerTransport) startReceive(ctx context.Context) {
	s := bufio.NewReader(t.reader)

	for {
		line, err := s.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.ErrClosedPipe) || // This error occurs during unit tests, suppressing it here
				errors.Is(err, io.EOF) {
				return
			}
			t.logger.Errorf("client receive unexpected error reading input: %v", err)
		}
		line = bytes.TrimRight(line, "\n")

		select {
		case <-ctx.Done():
			return
		default:
			t.receive(ctx, line)
		}
	}
}

func (t *stdioServerTransport) receive(ctx context.Context, line []byte) {
	outputMsgCh, err := t.receiver.Receive(ctx, t.sessionID, line)
	if err != nil {
		t.logger.Errorf("receiver failed: %v", err)
		return
	}

	if outputMsgCh == nil {
		return
	}

	go func() {
		defer pkg.Recover()

		msg := <-outputMsgCh
		if len(msg) == 0 {
			t.logger.Errorf("handle request fail")
			return
		}
		if err := t.Send(context.Background(), t.sessionID, msg); err != nil {
			t.logger.Errorf("Failed to send message: %v", err)
		}
	}()
}
