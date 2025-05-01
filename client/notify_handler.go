package client

import (
	"context"
	"encoding/json"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// SamplingHandler 定义采样消息处理器接口
// CreateMessage: 创建采样消息
type SamplingHandler interface {
	CreateMessage(ctx context.Context, request *protocol.CreateMessageRequest) (*protocol.CreateMessageResult, error)
}

// NotifyHandler
// When implementing a custom NotifyHandler, you can combine it with BaseNotifyHandler to implement it on demand without implementing extra methods.
// NotifyHandler 定义通知处理器接口
// 实现自定义通知处理器时，可以结合BaseNotifyHandler按需实现
// ToolsListChanged: 处理工具列表变更通知
// PromptListChanged: 处理提示词列表变更通知
// ResourceListChanged: 处理资源列表变更通知
// ResourcesUpdated: 处理资源更新通知
type NotifyHandler interface {
	ToolsListChanged(ctx context.Context, request *protocol.ToolListChangedNotification) error
	PromptListChanged(ctx context.Context, request *protocol.PromptListChangedNotification) error
	ResourceListChanged(ctx context.Context, request *protocol.ResourceListChangedNotification) error
	ResourcesUpdated(ctx context.Context, request *protocol.ResourceUpdatedNotification) error
}

// BaseNotifyHandler 提供通知处理器的默认实现
// Logger: 日志记录器实例
type BaseNotifyHandler struct {
	Logger pkg.Logger
}

// NewBaseNotifyHandler 创建基础通知处理器实例
// 返回: 基础通知处理器指针
func NewBaseNotifyHandler() *BaseNotifyHandler {
	return &BaseNotifyHandler{pkg.DefaultLogger}
}

// ToolsListChanged 处理工具列表变更通知
// ctx: 上下文
// request: 工具列表变更通知参数
// 返回: 错误信息
// 1. 调用默认通知处理器
func (handler *BaseNotifyHandler) ToolsListChanged(_ context.Context, request *protocol.ToolListChangedNotification) error {
	return handler.defaultNotifyHandler(protocol.NotificationToolsListChanged, request)
}

// PromptListChanged 处理提示词列表变更通知
// ctx: 上下文
// request: 提示词列表变更通知参数
// 返回: 错误信息
// 1. 调用默认通知处理器
func (handler *BaseNotifyHandler) PromptListChanged(_ context.Context, request *protocol.PromptListChangedNotification) error {
	return handler.defaultNotifyHandler(protocol.NotificationPromptsListChanged, request)
}

// ResourceListChanged 处理资源列表变更通知
// ctx: 上下文
// request: 资源列表变更通知参数
// 返回: 错误信息
// 1. 调用默认通知处理器
func (handler *BaseNotifyHandler) ResourceListChanged(_ context.Context, request *protocol.ResourceListChangedNotification) error {
	return handler.defaultNotifyHandler(protocol.NotificationResourcesListChanged, request)
}

// ResourcesUpdated 处理资源更新通知
// ctx: 上下文
// request: 资源更新通知参数
// 返回: 错误信息
// 1. 调用默认通知处理器
func (handler *BaseNotifyHandler) ResourcesUpdated(_ context.Context, request *protocol.ResourceUpdatedNotification) error {
	return handler.defaultNotifyHandler(protocol.NotificationResourcesUpdated, request)
}

func (handler *BaseNotifyHandler) defaultNotifyHandler(method protocol.Method, notify interface{}) error {
	b, err := json.Marshal(notify)
	if err != nil {
		return err
	}
	handler.Logger.Infof("receive notify: method=%s, notify=%s", method, b)
	return nil
}
