// Package server 实现MCP协议的服务端核心逻辑
// 模块功能：提供MCP协议服务端的注册、会话管理和请求处理能力
// 项目定位：go-mcp项目的核心服务端组件
// 版本历史：
// - 2023-10-01 初始版本 (ThinkInAI)
// - 2023-11-15 增加资源模板支持 (ThinkInAI)
// 依赖说明：
// - github.com/ThinkInAIXYZ/go-mcp/pkg: 基础工具包
// - github.com/ThinkInAIXYZ/go-mcp/protocol: MCP协议定义
// - github.com/ThinkInAIXYZ/go-mcp/transport: 传输层抽象
package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/server/session"
	"github.com/ThinkInAIXYZ/go-mcp/transport"
)

// Option 服务配置函数类型
// [注意] 所有Option函数都应在NewServer时传入，运行时修改无效

type Option func(*Server)

// WithCapabilities 设置服务端能力配置
// 输入：protocol.ServerCapabilities 结构体
// 典型用例：
//
//	server.NewServer(transport, WithCapabilities(capabilities))
func WithCapabilities(capabilities protocol.ServerCapabilities) Option {
	return func(s *Server) {
		s.capabilities = &capabilities
	}
}

func WithServerInfo(serverInfo protocol.Implementation) Option {
	return func(s *Server) {
		s.serverInfo = &serverInfo
	}
}

func WithInstructions(instructions string) Option {
	return func(s *Server) {
		s.instructions = instructions
	}
}

func WithSessionMaxIdleTime(maxIdleTime time.Duration) Option {
	return func(s *Server) {
		s.sessionManager.SetMaxIdleTime(maxIdleTime)
	}
}

func WithLogger(logger pkg.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}

// Server MCP协议服务端核心结构
// [重要] 所有字段都应在NewServer中初始化，避免并发问题
// 设计决策：
// - 使用SyncMap替代原生map，解决并发安全问题
// - 采用WaitGroup跟踪处理中的请求，确保优雅关闭
// 安全要求：
// - 所有导出方法都应考虑并发安全
// - 敏感操作需校验session状态
type Server struct {
	transport transport.ServerTransport // 底层传输层实现

	// 功能组件注册表
	tools             pkg.SyncMap[*toolEntry]             // 工具注册表
	prompts           pkg.SyncMap[*promptEntry]           // 提示词注册表
	resources         pkg.SyncMap[*resourceEntry]         // 资源注册表
	resourceTemplates pkg.SyncMap[*resourceTemplateEntry] // 资源模板注册表

	sessionManager *session.Manager // 会话管理器

	// 状态控制
	inShutdown   *pkg.AtomicBool // true表示服务正在关闭
	inFlyRequest sync.WaitGroup  // 跟踪处理中的请求

	// 服务元信息
	capabilities *protocol.ServerCapabilities // 服务能力声明
	serverInfo   *protocol.Implementation     // 服务实现信息
	instructions string                       // 服务使用说明

	logger pkg.Logger // 日志记录器
}

// NewServer 创建并初始化MCP服务端实例
// 输入参数：
// - t: 传输层实现，支持SSE/Stdio/HTTP等协议
// - opts: 可选配置项，用于定制服务端行为
// 返回值：
// - *Server: 服务端实例
// - error: 初始化错误
// [注意] 必须提供有效的transport实现
// 典型用例：
//
//	transport := sse.NewServerTransport()
//	server, err := server.NewServer(transport)
func NewServer(t transport.ServerTransport, opts ...Option) (*Server, error) {
	server := &Server{
		transport: t,
		capabilities: &protocol.ServerCapabilities{
			Prompts:   &protocol.PromptsCapability{ListChanged: true},
			Resources: &protocol.ResourcesCapability{ListChanged: true, Subscribe: true},
			Tools:     &protocol.ToolsCapability{ListChanged: true},
		},
		inShutdown: pkg.NewAtomicBool(),
		serverInfo: &protocol.Implementation{},
		logger:     pkg.DefaultLogger,
	}

	t.SetReceiver(transport.ServerReceiverF(server.receive))

	server.sessionManager = session.NewManager(server.sessionDetection)

	for _, opt := range opts {
		opt(server)
	}

	server.sessionManager.SetLogger(server.logger)

	t.SetSessionManager(server.sessionManager)

	return server, nil
}

// Run 启动MCP服务端
// 功能说明：
// 1. 启动会话心跳检测协程
// 2. 启动底层传输层
// 副作用：
// - 会创建后台goroutine进行会话管理
// 性能提示：
// - 每个会话会创建独立的goroutine
// 典型用例：
//
//	if err := server.Run(); err != nil {
//	  log.Fatal(err)
//	}
func (server *Server) Run() error {
	go func() {
		defer pkg.Recover()

		server.sessionManager.StartHeartbeatAndCleanInvalidSessions()
	}()

	if err := server.transport.Run(); err != nil {
		return fmt.Errorf("init mcp server transpor run fail: %w", err)
	}
	return nil
}

type toolEntry struct {
	tool    *protocol.Tool
	handler ToolHandlerFunc
}

type ToolHandlerFunc func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)

// RegisterTool 注册工具到服务端
// 输入参数：
// - tool: 工具定义，包含名称、描述等元信息
// - toolHandler: 工具处理函数，实现具体业务逻辑
// 业务映射：
// 对应MCP协议中的ToolsCall方法
// 典型用例：
//
//	server.RegisterTool(&protocol.Tool{Name:"example"}, func(ctx,req){...})
//
// [注意] 注册后会自动通知已连接客户端
func (server *Server) RegisterTool(tool *protocol.Tool, toolHandler ToolHandlerFunc) {
	server.tools.Store(tool.Name, &toolEntry{tool: tool, handler: toolHandler})
	if !server.sessionManager.IsEmpty() {
		if err := server.sendNotification4ToolListChanges(context.Background()); err != nil {
			server.logger.Warnf("send notification toll list changes fail: %v", err)
			return
		}
	}
}

func (server *Server) UnregisterTool(name string) {
	server.tools.Delete(name)
	if !server.sessionManager.IsEmpty() {
		if err := server.sendNotification4ToolListChanges(context.Background()); err != nil {
			server.logger.Warnf("send notification toll list changes fail: %v", err)
			return
		}
	}
}

type promptEntry struct {
	prompt  *protocol.Prompt
	handler PromptHandlerFunc
}

type PromptHandlerFunc func(context.Context, *protocol.GetPromptRequest) (*protocol.GetPromptResult, error)

// RegisterPrompt 注册提示词到服务端
// 输入参数：
// - prompt: 提示词定义，包含名称、内容模板等
// - promptHandler: 提示词处理函数，实现动态内容生成
// 业务映射：
// 对应MCP协议中的PromptsGet方法
// 典型用例：
//
//	server.RegisterPrompt(&protocol.Prompt{Name:"welcome"}, func(ctx,req){...})
//
// [注意] 注册后会自动通知已连接客户端
func (server *Server) RegisterPrompt(prompt *protocol.Prompt, promptHandler PromptHandlerFunc) {
	server.prompts.Store(prompt.Name, &promptEntry{prompt: prompt, handler: promptHandler})
	if !server.sessionManager.IsEmpty() {
		if err := server.sendNotification4PromptListChanges(context.Background()); err != nil {
			server.logger.Warnf("send notification prompt list changes fail: %v", err)
			return
		}
	}
}

func (server *Server) UnregisterPrompt(name string) {
	server.prompts.Delete(name)
	if !server.sessionManager.IsEmpty() {
		if err := server.sendNotification4PromptListChanges(context.Background()); err != nil {
			server.logger.Warnf("send notification prompt list changes fail: %v", err)
			return
		}
	}
}

type resourceEntry struct {
	resource *protocol.Resource
	handler  ResourceHandlerFunc
}

type ResourceHandlerFunc func(context.Context, *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error)

// RegisterResource 注册资源到服务端
// 输入参数：
// - resource: 资源定义，包含URI、类型等元信息
// - resourceHandler: 资源处理函数，实现资源读取逻辑
// 业务映射：
// 对应MCP协议中的ResourcesRead方法
// 典型用例：
//
//	server.RegisterResource(&protocol.Resource{URI:"file:///data"}, func(ctx,req){...})
//
// [注意] 注册后会自动通知已连接客户端
func (server *Server) RegisterResource(resource *protocol.Resource, resourceHandler ResourceHandlerFunc) {
	server.resources.Store(resource.URI, &resourceEntry{resource: resource, handler: resourceHandler})
	if !server.sessionManager.IsEmpty() {
		if err := server.sendNotification4ResourceListChanges(context.Background()); err != nil {
			server.logger.Warnf("send notification resource list changes fail: %v", err)
			return
		}
	}
}

func (server *Server) UnregisterResource(uri string) {
	server.resources.Delete(uri)
	if !server.sessionManager.IsEmpty() {
		if err := server.sendNotification4ResourceListChanges(context.Background()); err != nil {
			server.logger.Warnf("send notification resource list changes fail: %v", err)
			return
		}
	}
}

type resourceTemplateEntry struct {
	resourceTemplate *protocol.ResourceTemplate
	handler          ResourceHandlerFunc
}

func (server *Server) RegisterResourceTemplate(resource *protocol.ResourceTemplate, resourceHandler ResourceHandlerFunc) error {
	if err := resource.ParseURITemplate(); err != nil {
		return err
	}
	server.resourceTemplates.Store(resource.URITemplate, &resourceTemplateEntry{resourceTemplate: resource, handler: resourceHandler})
	if !server.sessionManager.IsEmpty() {
		if err := server.sendNotification4ResourceListChanges(context.Background()); err != nil {
			server.logger.Warnf("send notification resource list changes fail: %v", err)
			return nil
		}
	}
	return nil
}

func (server *Server) UnregisterResourceTemplate(uriTemplate string) {
	server.resourceTemplates.Delete(uriTemplate)
	if !server.sessionManager.IsEmpty() {
		if err := server.sendNotification4ResourceListChanges(context.Background()); err != nil {
			server.logger.Warnf("send notification resource list changes fail: %v", err)
			return
		}
	}
}

// Shutdown 优雅关闭服务端
// 输入参数：
// - userCtx: 用户上下文，用于控制关闭超时
// 实现原理：
// 1. 设置关闭标志阻止新请求
// 2. 等待处理中的请求完成
// 3. 停止会话心跳
// 4. 关闭传输层
// [重要] 必须确保所有资源已释放
// 典型用例：
//
//	ctx, cancel := context.WithTimeout(10*time.Second)
//	defer cancel()
//	server.Shutdown(ctx)
func (server *Server) Shutdown(userCtx context.Context) error {
	server.inShutdown.Store(true)

	serverCtx, cancel := context.WithCancel(userCtx)
	defer cancel()

	go func() {
		defer pkg.Recover()

		server.inFlyRequest.Wait()
		cancel()
	}()

	server.sessionManager.StopHeartbeat()

	return server.transport.Shutdown(userCtx, serverCtx)
}

func (server *Server) sessionDetection(ctx context.Context, sessionID string) error {
	if server.inShutdown.Load() {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if _, err := server.Ping(setSessionIDToCtx(ctx, sessionID), protocol.NewPingRequest()); err != nil {
		return err
	}
	return nil
}
