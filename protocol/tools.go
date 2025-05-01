package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

// ListToolsRequest 表示列出可用工具的请求
// [协议规范] 遵循JSON-RPC 2.0规范
// 典型调用:
// ```go
// req := protocol.NewListToolsRequest()
// ```

type ListToolsRequest struct{}

// ListToolsResult 表示列出工具请求的响应
// [重要] 包含工具列表和分页游标
// 字段说明:
// - Tools: 工具列表
// - NextCursor: 分页游标(可选)

type ListToolsResult struct {
	Tools      []*Tool `json:"tools"`
	NextCursor string  `json:"nextCursor,omitempty"`
}

// Tool 定义客户端可调用的工具
// [设计决策] 使用JSON Schema定义输入参数
// 字段说明:
// - Name: 工具唯一标识
// - Description: 工具描述(可选)
// - InputSchema: 输入参数schema

type Tool struct {
	// Name is the unique identifier of the tool
	Name string `json:"name"`

	// Description is a human-readable description of the tool
	Description string `json:"description,omitempty"`

	// InputSchema defines the expected parameters for the tool using JSON Schema
	InputSchema InputSchema `json:"inputSchema"`

	RawInputSchema json.RawMessage `json:"-"`
}

// MarshalJSON 实现Tool的JSON序列化
// [注意] 处理RawInputSchema和InputSchema的冲突
func (t *Tool) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{}, 3)

	m["name"] = t.Name
	if t.Description != "" {
		m["description"] = t.Description
	}

	// Determine which schema to use
	if t.RawInputSchema != nil {
		if t.InputSchema.Type != "" {
			return nil, fmt.Errorf("inputSchema field conflict")
		}
		m["inputSchema"] = t.RawInputSchema
	} else {
		// Use the structured InputSchema
		m["inputSchema"] = t.InputSchema
	}

	return json.Marshal(m)
}

type InputSchemaType string

// InputSchemaType 定义输入schema类型
// [协议规范] 目前仅支持object类型
const Object InputSchemaType = "object"

// InputSchema 定义工具的输入参数schema
// [重要] 用于参数验证和文档生成
type InputSchema struct {
	Type       InputSchemaType      `json:"type"`
	Properties map[string]*Property `json:"properties,omitempty"`
	Required   []string             `json:"required,omitempty"`
}

// CallToolRequest 表示调用特定工具的请求
// [安全要求] 必须验证工具名和参数
// 字段说明:
// - Name: 工具名
// - Arguments: 参数键值对
// - RawArguments: 原始参数JSON(可选)
type CallToolRequest struct {
	Name         string                 `json:"name"`
	Arguments    map[string]interface{} `json:"arguments,omitempty"`
	RawArguments json.RawMessage        `json:"-"`
}

// UnmarshalJSON 实现CallToolRequest的反序列化
// [注意] 处理Arguments和RawArguments的转换
func (r *CallToolRequest) UnmarshalJSON(data []byte) error {
	type alias CallToolRequest
	temp := &struct {
		Arguments json.RawMessage `json:"arguments,omitempty"`
		*alias
	}{
		alias: (*alias)(r),
	}

	if err := pkg.JSONUnmarshal(data, temp); err != nil {
		return err
	}

	r.RawArguments = temp.Arguments

	if len(r.RawArguments) != 0 {
		if err := pkg.JSONUnmarshal(r.RawArguments, &r.Arguments); err != nil {
			return err
		}
	}

	return nil
}

// MarshalJSON 实现CallToolRequest的序列化
// [性能提示] 优先使用RawArguments避免重复序列化
func (r *CallToolRequest) MarshalJSON() ([]byte, error) {
	type alias CallToolRequest
	temp := &struct {
		Arguments json.RawMessage `json:"arguments,omitempty"`
		*alias
	}{
		alias: (*alias)(r),
	}

	if len(r.RawArguments) > 0 {
		temp.Arguments = r.RawArguments
	} else if r.Arguments != nil {
		var err error
		temp.Arguments, err = json.Marshal(r.Arguments)
		if err != nil {
			return nil, err
		}
	}

	return json.Marshal(temp)
}

// CallToolResult 表示工具调用的响应
// [重要] 支持多种内容类型(文本/图片/音频)
// 字段说明:
// - Content: 内容列表
// - IsError: 是否为错误结果(可选)
type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// UnmarshalJSON 实现CallToolResult的反序列化
// [算法说明] 尝试多种内容类型的反序列化
func (r *CallToolResult) UnmarshalJSON(data []byte) error {
	type Alias CallToolResult
	aux := &struct {
		Content []json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := pkg.JSONUnmarshal(data, &aux); err != nil {
		return err
	}

	r.Content = make([]Content, len(aux.Content))
	for i, content := range aux.Content {
		// Try to unmarshal content as TextContent first
		var textContent *TextContent
		if err := pkg.JSONUnmarshal(content, &textContent); err == nil {
			r.Content[i] = textContent
			continue
		}

		// Try to unmarshal content as ImageContent
		var imageContent *ImageContent
		if err := pkg.JSONUnmarshal(content, &imageContent); err == nil {
			r.Content[i] = imageContent
			continue
		}

		// Try to unmarshal content as AudioContent
		var audioContent *AudioContent
		if err := pkg.JSONUnmarshal(content, &audioContent); err == nil {
			r.Content[i] = audioContent
			continue
		}

		// Try to unmarshal content as embeddedResource
		var embeddedResource *EmbeddedResource
		if err := pkg.JSONUnmarshal(content, &embeddedResource); err == nil {
			r.Content[i] = embeddedResource
			return nil
		}

		return fmt.Errorf("unknown content type at index %d", i)
	}

	return nil
}

// ToolListChangedNotification 表示工具列表变更通知
// [协议规范] 使用_meta字段传递扩展信息
type ToolListChangedNotification struct {
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

// NewTool 创建工具实例
// [设计决策] 从结构体生成输入schema
// 参数说明:
// - name: 工具名
// - description: 工具描述
// - inputReqStruct: 输入参数结构体
func NewTool(name string, description string, inputReqStruct interface{}) (*Tool, error) {
	schema, err := generateSchemaFromReqStruct(inputReqStruct)
	if err != nil {
		return nil, err
	}

	return &Tool{
		Name:        name,
		Description: description,
		InputSchema: *schema,
	}, nil
}

// NewToolWithRawSchema 使用原始schema创建工具
// [临时方案] 用于无法从结构体生成schema的情况
func NewToolWithRawSchema(name, description string, schema json.RawMessage) *Tool {
	return &Tool{
		Name:           name,
		Description:    description,
		RawInputSchema: schema,
	}
}

// NewListToolsRequest 创建列出工具请求
// [典型用例] 客户端初始化时获取可用工具列表
func NewListToolsRequest() *ListToolsRequest {
	return &ListToolsRequest{}
}

// NewListToolsResult 创建列出工具响应
// [性能提示] 避免大工具列表一次性返回
func NewListToolsResult(tools []*Tool, nextCursor string) *ListToolsResult {
	return &ListToolsResult{
		Tools:      tools,
		NextCursor: nextCursor,
	}
}

// NewCallToolRequest 创建调用工具请求
// [安全要求] 必须验证工具名存在
func NewCallToolRequest(name string, arguments map[string]interface{}) *CallToolRequest {
	return &CallToolRequest{
		Name:      name,
		Arguments: arguments,
	}
}

// NewCallToolRequestWithRawArguments 使用原始参数创建调用请求
// [临时方案] 用于复杂参数结构
func NewCallToolRequestWithRawArguments(name string, rawArguments json.RawMessage) *CallToolRequest {
	return &CallToolRequest{
		Name:         name,
		RawArguments: rawArguments,
	}
}

// NewCallToolResult 创建工具调用响应
// [协议规范] 支持多内容类型返回
func NewCallToolResult(content []Content, isError bool) *CallToolResult {
	return &CallToolResult{
		Content: content,
		IsError: isError,
	}
}

// NewToolListChangedNotification 创建工具列表变更通知
// [设计决策] 使用轻量级通知机制
func NewToolListChangedNotification() *ToolListChangedNotification {
	return &ToolListChangedNotification{}
}
