package protocol

import (
	"encoding/json"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

// JSON-RPC协议版本号
const jsonrpcVersion = "2.0"

// Standard JSON-RPC error codes
const (
	ParseError     = -32700 // 无效的JSON格式
	InvalidRequest = -32600 // 发送的JSON不是有效的请求对象
	MethodNotFound = -32601 // 方法不存在/不可用
	InvalidParams  = -32602 // 无效的方法参数
	InternalError  = -32603 // 内部JSON-RPC错误

	// 可以定义自己的错误代码，范围在-32000 以上。
)

// RequestID 定义请求ID的类型，可以是字符串或数值
type RequestID interface{}

// JSONRPCRequest 定义JSON-RPC请求结构
// JSONRPC: 协议版本号
// ID: 请求标识符
// Method: 调用的方法名
// Params: 方法参数
// RawParams: 原始参数数据(不参与JSON序列化)
type JSONRPCRequest struct {
	JSONRPC   string          `json:"jsonrpc"`
	ID        RequestID       `json:"id"`
	Method    Method          `json:"method"`
	Params    interface{}     `json:"params,omitempty"`
	RawParams json.RawMessage `json:"-"`
}

// UnmarshalJSON 自定义JSON反序列化方法
// 用于处理Params字段的特殊解析需求
// 1. 首先解析原始JSON数据到临时结构体
// 2. 保存原始参数数据到RawParams
// 3. 如果RawParams不为空，则进一步解析到Params字段
func (r *JSONRPCRequest) UnmarshalJSON(data []byte) error {
	type alias JSONRPCRequest
	temp := &struct {
		Params json.RawMessage `json:"params,omitempty"`
		*alias
	}{
		alias: (*alias)(r),
	}

	if err := pkg.JSONUnmarshal(data, temp); err != nil {
		return err
	}

	r.RawParams = temp.Params

	if len(r.RawParams) != 0 {
		if err := pkg.JSONUnmarshal(r.RawParams, &r.Params); err != nil {
			return err
		}
	}

	return nil
}

// IsValid 检查请求是否符合JSON-RPC 2.0规范
// 有效的请求必须满足:
// 1. JSONRPC字段等于当前协议版本号
// 2. Method字段不为空
// 3. ID字段不为nil
func (r *JSONRPCRequest) IsValid() bool {
	return r.JSONRPC == jsonrpcVersion && r.Method != "" && r.ID != nil
}

// JSONRPCResponse represents a response to a request.
type JSONRPCResponse struct {
	JSONRPC   string          `json:"jsonrpc"`
	ID        RequestID       `json:"id"`
	Result    interface{}     `json:"result,omitempty"`
	RawResult json.RawMessage `json:"-"`
	Error     *responseErr    `json:"error,omitempty"`
}

type responseErr struct {
	// The error type that occurred.
	Code int `json:"code"`
	// A short description of the error. The message SHOULD be limited
	// to a concise single sentence.
	Message string `json:"message"`
	// Additional information about the error. The value of this member
	// is defined by the sender (e.g. detailed error information, nested errors etc.).
	Data interface{} `json:"data,omitempty"`
}

func (r *JSONRPCResponse) UnmarshalJSON(data []byte) error {
	type alias JSONRPCResponse
	temp := &struct {
		Result json.RawMessage `json:"result,omitempty"`
		*alias
	}{
		alias: (*alias)(r),
	}

	if err := pkg.JSONUnmarshal(data, temp); err != nil {
		return err
	}

	r.RawResult = temp.Result

	if len(r.RawResult) != 0 {
		if err := pkg.JSONUnmarshal(r.RawResult, &r.Result); err != nil {
			return err
		}
	}

	return nil
}

type JSONRPCNotification struct {
	JSONRPC   string          `json:"jsonrpc"`
	Method    Method          `json:"method"`
	Params    interface{}     `json:"params,omitempty"`
	RawParams json.RawMessage `json:"-"`
}

func (r *JSONRPCNotification) UnmarshalJSON(data []byte) error {
	type alias JSONRPCNotification
	temp := &struct {
		Params json.RawMessage `json:"params,omitempty"`
		*alias
	}{
		alias: (*alias)(r),
	}

	if err := pkg.JSONUnmarshal(data, temp); err != nil {
		return err
	}

	r.RawParams = temp.Params

	if len(r.RawParams) != 0 {
		if err := pkg.JSONUnmarshal(r.RawParams, &r.Params); err != nil {
			return err
		}
	}

	return nil
}

// NewJSONRPCRequest creates a new JSON-RPC request
func NewJSONRPCRequest(id RequestID, method Method, params interface{}) *JSONRPCRequest {
	return &JSONRPCRequest{
		JSONRPC: jsonrpcVersion,
		ID:      id,
		Method:  method,
		Params:  params,
	}
}

// NewJSONRPCSuccessResponse creates a new JSON-RPC response
func NewJSONRPCSuccessResponse(id RequestID, result interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: jsonrpcVersion,
		ID:      id,
		Result:  result,
	}
}

// NewJSONRPCErrorResponse NewError creates a new JSON-RPC error response
func NewJSONRPCErrorResponse(id RequestID, code int, message string) *JSONRPCResponse {
	err := &JSONRPCResponse{
		JSONRPC: jsonrpcVersion,
		ID:      id,
		Error: &responseErr{
			Code:    code,
			Message: message,
		},
	}
	return err
}

// NewJSONRPCNotification creates a new JSON-RPC notification
func NewJSONRPCNotification(method Method, params interface{}) *JSONRPCNotification {
	return &JSONRPCNotification{
		JSONRPC: jsonrpcVersion,
		Method:  method,
		Params:  params,
	}
}
