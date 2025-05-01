package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

// ListPromptsRequest 表示列出可用提示词的请求
type ListPromptsRequest struct{}

// ListPromptsResult 表示列出提示词的响应
// Prompts: 提示词列表
// NextCursor: 下一页游标(可选)
type ListPromptsResult struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

// Prompt 提示词结构
// Name: 提示词名称
// Description: 提示词描述(可选)
// Arguments: 提示词参数列表(可选)
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument 提示词参数
// Name: 参数名称
// Description: 参数描述(可选)
// Required: 是否必需(可选)
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// GetPromptRequest 表示获取特定提示词的请求
// Name: 提示词名称
// Arguments: 参数键值对(可选)
type GetPromptRequest struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// GetPromptResult 表示获取提示词的响应
// Messages: 消息列表
// Description: 描述信息(可选)
type GetPromptResult struct {
	Messages    []PromptMessage `json:"messages"`
	Description string          `json:"description,omitempty"`
}

// PromptMessage 提示词消息
// Role: 消息角色
// Content: 消息内容
type PromptMessage struct {
	Role    Role    `json:"role"`
	Content Content `json:"content"`
}

// UnmarshalJSON 实现json.Unmarshaler接口
// [重要] 该方法用于处理不同类型的消息内容
func (m *PromptMessage) UnmarshalJSON(data []byte) error {
	type Alias PromptMessage
	aux := &struct {
		Content json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	if err := pkg.JSONUnmarshal(data, &aux); err != nil {
		return err
	}

	// 尝试解析为文本内容
	var textContent *TextContent
	if err := pkg.JSONUnmarshal(aux.Content, &textContent); err == nil {
		m.Content = textContent
		return nil
	}

	// 尝试解析为图片内容
	var imageContent *ImageContent
	if err := pkg.JSONUnmarshal(aux.Content, &imageContent); err == nil {
		m.Content = imageContent
		return nil
	}

	// 尝试解析为音频内容
	var audioContent *AudioContent
	if err := pkg.JSONUnmarshal(aux.Content, &audioContent); err == nil {
		m.Content = audioContent
		return nil
	}

	// 尝试解析为嵌入式资源
	var embeddedResource *EmbeddedResource
	if err := pkg.JSONUnmarshal(aux.Content, &embeddedResource); err == nil {
		m.Content = embeddedResource
		return nil
	}

	return fmt.Errorf("unknown content type")
}

// PromptListChangedNotification 表示提示词列表变更通知
type PromptListChangedNotification struct {
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

// NewListPromptsRequest 创建新的列出提示词请求
func NewListPromptsRequest() *ListPromptsRequest {
	return &ListPromptsRequest{}
}

// NewListPromptsResult 创建新的列出提示词响应
// prompts: 提示词列表
// nextCursor: 下一页游标(可选)
func NewListPromptsResult(prompts []Prompt, nextCursor string) *ListPromptsResult {
	return &ListPromptsResult{
		Prompts:    prompts,
		NextCursor: nextCursor,
	}
}

// NewGetPromptRequest 创建新的获取提示词请求
// name: 提示词名称
// arguments: 参数键值对(可选)
func NewGetPromptRequest(name string, arguments map[string]string) *GetPromptRequest {
	return &GetPromptRequest{
		Name:      name,
		Arguments: arguments,
	}
}

// NewGetPromptResult 创建新的获取提示词响应
// messages: 消息列表
// description: 描述信息(可选)
func NewGetPromptResult(messages []PromptMessage, description string) *GetPromptResult {
	return &GetPromptResult{
		Messages:    messages,
		Description: description,
	}
}

// NewPromptListChangedNotification 创建新的提示词列表变更通知
func NewPromptListChangedNotification() *PromptListChangedNotification {
	return &PromptListChangedNotification{}
}
