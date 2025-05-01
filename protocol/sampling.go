package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

// CreateMessageRequest 表示通过采样创建消息的请求
// Messages: 消息列表
// MaxTokens: 最大token数
// Temperature: 采样温度(可选)
// StopSequences: 停止序列(可选)
// SystemPrompt: 系统提示(可选)
// ModelPreferences: 模型偏好(可选)
// IncludeContext: 包含上下文(可选)
// Metadata: 元数据(可选)
type CreateMessageRequest struct {
	Messages         []SamplingMessage      `json:"messages"`
	MaxTokens        int                    `json:"maxTokens"`
	Temperature      float64                `json:"temperature,omitempty"`
	StopSequences    []string               `json:"stopSequences,omitempty"`
	SystemPrompt     string                 `json:"systemPrompt,omitempty"`
	ModelPreferences *ModelPreferences      `json:"modelPreferences,omitempty"`
	IncludeContext   string                 `json:"includeContext,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// SamplingMessage 采样消息
// Role: 消息角色
// Content: 消息内容
type SamplingMessage struct {
	Role    Role    `json:"role"`
	Content Content `json:"content"`
}

// UnmarshalJSON 实现json.Unmarshaler接口
// [重要] 该方法用于处理不同类型的消息内容
func (r *SamplingMessage) UnmarshalJSON(data []byte) error {
	type Alias SamplingMessage
	aux := &struct {
		Content json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := pkg.JSONUnmarshal(data, &aux); err != nil {
		return err
	}

	// 尝试解析为文本内容
	var textContent *TextContent
	if err := pkg.JSONUnmarshal(aux.Content, &textContent); err == nil {
		r.Content = textContent
		return nil
	}

	// 尝试解析为图片内容
	var imageContent *ImageContent
	if err := pkg.JSONUnmarshal(aux.Content, &imageContent); err == nil {
		r.Content = imageContent
		return nil
	}

	// 尝试解析为音频内容
	var audioContent *AudioContent
	if err := pkg.JSONUnmarshal(aux.Content, &audioContent); err == nil {
		r.Content = audioContent
		return nil
	}

	return fmt.Errorf("unknown content type, content=%s", aux.Content)
}

// CreateMessageResult 表示创建消息请求的响应
// Content: 消息内容
// Role: 消息角色
// Model: 使用的模型
// StopReason: 停止原因(可选)
type CreateMessageResult struct {
	Content    Content `json:"content"`
	Role       Role    `json:"role"`
	Model      string  `json:"model"`
	StopReason string  `json:"stopReason,omitempty"`
}

// UnmarshalJSON 实现json.Unmarshaler接口
// [重要] 该方法用于处理不同类型的消息内容
func (r *CreateMessageResult) UnmarshalJSON(data []byte) error {
	type Alias CreateMessageResult
	aux := &struct {
		Content json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := pkg.JSONUnmarshal(data, &aux); err != nil {
		return err
	}

	// 尝试解析为文本内容
	var textContent *TextContent
	if err := pkg.JSONUnmarshal(aux.Content, &textContent); err == nil {
		r.Content = textContent
		return nil
	}

	// 尝试解析为图片内容
	var imageContent *ImageContent
	if err := pkg.JSONUnmarshal(aux.Content, &imageContent); err == nil {
		r.Content = imageContent
		return nil
	}

	// 尝试解析为音频内容
	var audioContent *AudioContent
	if err := pkg.JSONUnmarshal(aux.Content, &audioContent); err == nil {
		r.Content = audioContent
		return nil
	}

	return fmt.Errorf("unknown content type, content=%s", aux.Content)
}

// NewCreateMessageRequest 创建新的创建消息请求
// messages: 消息列表
// maxTokens: 最大token数
// opts: 可选参数
func NewCreateMessageRequest(messages []SamplingMessage, maxTokens int, opts ...CreateMessageOption) *CreateMessageRequest {
	req := &CreateMessageRequest{
		Messages:  messages,
		MaxTokens: maxTokens,
	}

	for _, opt := range opts {
		opt(req)
	}

	return req
}

// NewCreateMessageResult 创建新的创建消息响应
// content: 消息内容
// role: 消息角色
// model: 使用的模型
// stopReason: 停止原因(可选)
func NewCreateMessageResult(content Content, role Role, model string, stopReason string) *CreateMessageResult {
	return &CreateMessageResult{
		Content:    content,
		Role:       role,
		Model:      model,
		StopReason: stopReason,
	}
}

// CreateMessageOption 表示创建消息的可选参数
type CreateMessageOption func(*CreateMessageRequest)

// WithTemperature 设置采样温度
// temp: 温度值
func WithTemperature(temp float64) CreateMessageOption {
	return func(r *CreateMessageRequest) {
		r.Temperature = temp
	}
}

// WithStopSequences 设置停止序列
// sequences: 停止序列列表
func WithStopSequences(sequences []string) CreateMessageOption {
	return func(r *CreateMessageRequest) {
		r.StopSequences = sequences
	}
}

// WithSystemPrompt 设置系统提示
// prompt: 提示内容
func WithSystemPrompt(prompt string) CreateMessageOption {
	return func(r *CreateMessageRequest) {
		r.SystemPrompt = prompt
	}
}

// WithModelPreferences 设置模型偏好
// prefs: 模型偏好配置
func WithModelPreferences(prefs *ModelPreferences) CreateMessageOption {
	return func(r *CreateMessageRequest) {
		r.ModelPreferences = prefs
	}
}

// WithIncludeContext 设置包含上下文
// ctx: 上下文内容
func WithIncludeContext(ctx string) CreateMessageOption {
	return func(r *CreateMessageRequest) {
		r.IncludeContext = ctx
	}
}

// WithMetadata 设置元数据
// metadata: 元数据键值对
func WithMetadata(metadata map[string]interface{}) CreateMessageOption {
	return func(r *CreateMessageRequest) {
		r.Metadata = metadata
	}
}
