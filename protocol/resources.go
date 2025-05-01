package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/yosida95/uritemplate/v3"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

// ListResourcesRequest 客户端发送的请求，用于获取服务器拥有的资源列表
type ListResourcesRequest struct{}

// ListResourcesResult 服务器对客户端resources/list请求的响应
// Resources: 资源列表
// NextCursor: 不透明的分页令牌，表示最后返回结果后的分页位置(可选)
// [注意] 如果存在NextCursor，表示可能还有更多结果可用
type ListResourcesResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

// ListResourceTemplatesRequest 表示列出资源模板的请求
type ListResourceTemplatesRequest struct{}

// ListResourceTemplatesResult 表示列出资源模板的响应
// ResourceTemplates: 资源模板列表
// NextCursor: 下一页游标(可选)
type ListResourceTemplatesResult struct {
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
	NextCursor        string             `json:"nextCursor,omitempty"`
}

// ReadResourceRequest 表示读取特定资源的请求
// URI: 资源URI
// Arguments: 参数键值对(内部使用)
type ReadResourceRequest struct {
	URI       string                 `json:"uri"`
	Arguments map[string]interface{} `json:"-"`
}

// ReadResourceResult 服务器对客户端resources/read请求的响应
// Contents: 资源内容列表
type ReadResourceResult struct {
	Contents []ResourceContents `json:"contents"`
}

// UnmarshalJSON 实现json.Unmarshaler接口
// [重要] 该方法用于处理不同类型的资源内容
func (r *ReadResourceResult) UnmarshalJSON(data []byte) error {
	type Alias ReadResourceResult
	aux := &struct {
		Contents []json.RawMessage `json:"contents"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := pkg.JSONUnmarshal(data, &aux); err != nil {
		return err
	}

	r.Contents = make([]ResourceContents, len(aux.Contents))
	for i, content := range aux.Contents {
		// 尝试解析为文本资源内容
		var textContent TextResourceContents
		if err := pkg.JSONUnmarshal(content, &textContent); err == nil {
			r.Contents[i] = textContent
			continue
		}

		// 尝试解析为二进制资源内容
		var blobContent BlobResourceContents
		if err := pkg.JSONUnmarshal(content, &blobContent); err == nil {
			r.Contents[i] = blobContent
			continue
		}

		return fmt.Errorf("unknown content type at index %d", i)
	}

	return nil
}

// Resource 服务器能够读取的已知资源
// Name: 资源的人类可读名称，可用于填充UI元素
// URI: 资源URI
// Description: 资源描述(可选)，可用于改进LLM对可用资源的理解
// MimeType: 资源的MIME类型(可选)
// Size: 资源大小(可选)
type Resource struct {
	Annotated
	Name        string `json:"name"`
	URI         string `json:"uri"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

type ResourceTemplate struct {
	Annotated
	Name              string                `json:"name"`
	URITemplate       string                `json:"uriTemplate"`
	URITemplateParsed *uritemplate.Template `json:"-"`
	Description       string                `json:"description,omitempty"`
	MimeType          string                `json:"mimeType,omitempty"`
}

func (t *ResourceTemplate) UnmarshalJSON(data []byte) error {
	type Alias ResourceTemplate
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Parse the URI template after unmarshaling
	if t.URITemplate != "" {
		template, err := uritemplate.New(t.URITemplate)
		if err != nil {
			return err
		}
		t.URITemplateParsed = template
	}
	return nil
}

func (t *ResourceTemplate) ParseURITemplate() error {
	template, err := uritemplate.New(t.URITemplate)
	if err != nil {
		return err
	}
	t.URITemplateParsed = template
	return nil
}

func (t *ResourceTemplate) GetURITemplate() *uritemplate.Template {
	return t.URITemplateParsed
}

// Annotated represents base objects that include optional annotations
type Annotated struct {
	Annotations *Annotations `json:"annotations,omitempty"`
}

// Annotations represents optional annotations for objects
type Annotations struct {
	Audience []Role  `json:"audience,omitempty"`
	Priority float64 `json:"priority,omitempty"`
}

// ModelHint represents hints to use for model selection
type ModelHint struct {
	Name string `json:"name,omitempty"`
}

// ModelPreferences represents the server's preferences for model selection
type ModelPreferences struct {
	CostPriority         float64     `json:"costPriority,omitempty"`
	IntelligencePriority float64     `json:"intelligencePriority,omitempty"`
	SpeedPriority        float64     `json:"speedPriority,omitempty"`
	Hints                []ModelHint `json:"hints,omitempty"`
}

// Content interfaces and types
type Content interface {
	GetType() string
}

type TextContent struct {
	Annotated
	Type string `json:"type"`
	Text string `json:"text"`
}

func (t *TextContent) GetType() string {
	return "text"
}

type ImageContent struct {
	Annotated
	Type     string `json:"type"`
	Data     []byte `json:"data"`
	MimeType string `json:"mimeType"`
}

func (i *ImageContent) GetType() string {
	return "image"
}

type AudioContent struct {
	Annotated
	Type     string `json:"type"`
	Data     []byte `json:"data"`
	MimeType string `json:"mimeType"`
}

func (i *AudioContent) GetType() string {
	return "audio"
}

// EmbeddedResource represents the contents of a resource, embedded into a prompt or tool call result.
// It is up to the client how best to render embedded resources for the benefit of the LLM and/or the user.
type EmbeddedResource struct {
	Type        string           `json:"type"` // Must be "resource"
	Resource    ResourceContents `json:"resource"`
	Annotations *Annotations     `json:"annotations,omitempty"`
}

// NewEmbeddedResource creates a new EmbeddedResource
func NewEmbeddedResource(resource ResourceContents, annotations *Annotations) *EmbeddedResource {
	return &EmbeddedResource{
		Type:        "resource",
		Resource:    resource,
		Annotations: annotations,
	}
}

func (i *EmbeddedResource) GetType() string {
	return "resource"
}

type ResourceContents interface {
	GetURI() string
	GetMimeType() string
}

type TextResourceContents struct {
	URI      string `json:"uri"`
	Text     string `json:"text"`
	MimeType string `json:"mimeType,omitempty"`
}

func (t TextResourceContents) GetURI() string {
	return t.URI
}

func (t TextResourceContents) GetMimeType() string {
	return t.MimeType
}

type BlobResourceContents struct {
	URI      string `json:"uri"`
	Blob     []byte `json:"blob"`
	MimeType string `json:"mimeType,omitempty"`
}

func (b BlobResourceContents) GetURI() string {
	return b.URI
}

func (b BlobResourceContents) GetMimeType() string {
	return b.MimeType
}

// SubscribeRequest represents a request to subscribe to resource updates
type SubscribeRequest struct {
	URI string `json:"uri"`
}

// UnsubscribeRequest represents a request to unsubscribe from resource updates
type UnsubscribeRequest struct {
	URI string `json:"uri"`
}

type SubscribeResult struct{}

type UnsubscribeResult struct{}

// ResourceListChangedNotification represents a notification that the resource list has changed
type ResourceListChangedNotification struct {
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

// ResourceUpdatedNotification represents a notification that a resource has been updated
type ResourceUpdatedNotification struct {
	URI string `json:"uri"`
}

// NewListResourcesRequest creates a new list resources request
func NewListResourcesRequest() *ListResourcesRequest {
	return &ListResourcesRequest{}
}

// NewListResourcesResult creates a new list resources response
func NewListResourcesResult(resources []Resource, nextCursor string) *ListResourcesResult {
	return &ListResourcesResult{
		Resources:  resources,
		NextCursor: nextCursor,
	}
}

// NewListResourceTemplatesRequest creates a new list resource templates request
func NewListResourceTemplatesRequest() *ListResourceTemplatesRequest {
	return &ListResourceTemplatesRequest{}
}

// NewListResourceTemplatesResult creates a new list resource templates response
func NewListResourceTemplatesResult(templates []ResourceTemplate, nextCursor string) *ListResourceTemplatesResult {
	return &ListResourceTemplatesResult{
		ResourceTemplates: templates,
		NextCursor:        nextCursor,
	}
}

// NewReadResourceRequest creates a new read resource request
func NewReadResourceRequest(uri string) *ReadResourceRequest {
	return &ReadResourceRequest{URI: uri}
}

// NewReadResourceResult creates a new read resource response
func NewReadResourceResult(contents []ResourceContents) *ReadResourceResult {
	return &ReadResourceResult{
		Contents: contents,
	}
}

// NewSubscribeRequest creates a new subscribe request
func NewSubscribeRequest(uri string) *SubscribeRequest {
	return &SubscribeRequest{URI: uri}
}

// NewUnsubscribeRequest creates a new unsubscribe request
func NewUnsubscribeRequest(uri string) *UnsubscribeRequest {
	return &UnsubscribeRequest{URI: uri}
}

func NewSubscribeResult() *SubscribeResult {
	return &SubscribeResult{}
}

func NewUnsubscribeResult() *UnsubscribeResult {
	return &UnsubscribeResult{}
}

// NewResourceListChangedNotification creates a new resource list changed notification
func NewResourceListChangedNotification() *ResourceListChangedNotification {
	return &ResourceListChangedNotification{}
}

// NewResourceUpdatedNotification creates a new resource updated notification
func NewResourceUpdatedNotification(uri string) *ResourceUpdatedNotification {
	return &ResourceUpdatedNotification{URI: uri}
}
