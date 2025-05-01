// Package transport 提供标准输入输出(stdio)的客户端传输实现
// [模块功能] 通过子进程的标准输入输出流实现进程间通信
// [项目定位] 属于go-mcp核心传输层，支持本地进程间通信场景
// [版本历史]
// v1.0.0 2023-05-15 初始版本 支持基础stdio通信
// v1.1.0 2023-06-20 增加环境变量配置选项
// [依赖说明]
// - github.com/ThinkInAIXYZ/go-mcp/pkg >= v1.2.0
// [典型调用]
// transport.NewStdioClientTransport("python", []string{"script.py"})
package transport

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

// StdioClientTransportOption 客户端传输配置函数类型
// [设计决策] 采用函数选项模式实现灵活配置
// [安全要求] 环境变量配置需注意敏感信息泄露风险
type StdioClientTransportOption func(*stdioClientTransport)

// WithStdioClientOptionLogger 配置日志记录器
// 输入: pkg.Logger接口实现
// 输出: 配置函数
// [性能提示] 日志操作可能影响高并发场景性能
func WithStdioClientOptionLogger(log pkg.Logger) StdioClientTransportOption {
	return func(t *stdioClientTransport) {
		t.logger = log
	}
}

// WithStdioClientOptionEnv 配置子进程环境变量
// 输入: 环境变量键值对(格式: "KEY=VALUE")
// 输出: 配置函数
// [注意] 会覆盖父进程同名环境变量
func WithStdioClientOptionEnv(env ...string) StdioClientTransportOption {
	return func(t *stdioClientTransport) {
		t.cmd.Env = append(t.cmd.Env, env...)
	}
}

const mcpMessageDelimiter = '\n'

// stdioClientTransport 标准输入输出客户端传输实现
// [重要] 非线程安全，并发调用需外部同步
// [调试技巧] 可通过设置详细日志级别跟踪消息流
type stdioClientTransport struct {
	cmd      *exec.Cmd      // 子进程命令对象
	receiver clientReceiver // 消息接收处理器
	reader   io.Reader      // 标准输出读取器
	writer   io.WriteCloser // 标准输入写入器

	logger pkg.Logger // 日志记录器

	cancel          context.CancelFunc // 上下文取消函数
	receiveShutDone chan struct{}      // 接收协程关闭信号
}

// NewStdioClientTransport 创建标准输入输出客户端传输实例
// 输入:
// - command: 子进程命令路径
// - args: 子进程命令行参数
// - opts: 可选配置函数
// 输出:
// - ClientTransport接口实例
// - 错误信息(管道创建失败等)
// [典型用例]
// 调用Python脚本处理数据:
// transport.NewStdioClientTransport("python", []string{"process.py"})
// [副作用] 会启动子进程但不会自动运行
// [兼容性] 要求子进程支持行缓冲模式通信
func NewStdioClientTransport(command string, args []string, opts ...StdioClientTransportOption) (ClientTransport, error) {
	cmd := exec.Command(command, args...)

	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	t := &stdioClientTransport{
		cmd:             cmd,
		reader:          stdout,
		writer:          stdin,
		logger:          pkg.DefaultLogger,
		receiveShutDone: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(t)
	}
	return t, nil
}

func (t *stdioClientTransport) Start() error {
	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	innerCtx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel

	go func() {
		defer pkg.Recover()

		t.startReceive(innerCtx)
		close(t.receiveShutDone)
	}()

	return nil
}

func (t *stdioClientTransport) Send(_ context.Context, msg Message) error {
	_, err := t.writer.Write(append(msg, mcpMessageDelimiter))
	return err
}

func (t *stdioClientTransport) SetReceiver(receiver clientReceiver) {
	t.receiver = receiver
}

func (t *stdioClientTransport) Close() error {
	t.cancel()

	if err := t.writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	if err := t.cmd.Wait(); err != nil {
		return err
	}

	<-t.receiveShutDone

	return nil
}

func (t *stdioClientTransport) startReceive(ctx context.Context) {
	s := bufio.NewReader(t.reader)

	for {
		line, err := s.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.ErrClosedPipe) || // This error occurs during unit tests, suppressing it here
				errors.Is(err, io.EOF) {
				return
			}
			t.logger.Errorf("client receive unexpected error reading input: %v", err)
			return
		}

		line = bytes.TrimRight(line, "\n")
		// filter empty messages
		// filter space messages and \t messages
		if len(bytes.TrimFunc(line, func(r rune) bool { return r == ' ' || r == '\t' })) == 0 {
			t.logger.Debugf("skipping empty message")
			continue
		}

		select {
		case <-ctx.Done():
			return
		default:
			if err = t.receiver.Receive(ctx, line); err != nil {
				t.logger.Errorf("receiver failed: %v", err)
			}
		}
	}
}
