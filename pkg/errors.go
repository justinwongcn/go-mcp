package pkg

import (
	"errors"
	"fmt"
)

// 预定义错误集合
var (
	ErrClientNotSupport          = errors.New("this feature client not support")
	ErrServerNotSupport          = errors.New("this feature server not support")
	ErrRequestInvalid            = errors.New("request invalid")
	ErrLackResponseChan          = errors.New("lack response chan")
	ErrDuplicateResponseReceived = errors.New("duplicate response received")
	ErrMethodNotSupport          = errors.New("method not support")
	ErrJSONUnmarshal             = errors.New("json unmarshal error")
	ErrSessionHasNotInitialized  = errors.New("the session has not been initialized")
	ErrLackSession               = errors.New("lack session")
	ErrSessionClosed             = errors.New("session closed")
	ErrSendEOF                   = errors.New("send EOF")
)

// ResponseError 定义标准错误响应结构
// Code: 错误码
// Message: 错误消息
// Data: 附加错误数据
type ResponseError struct {
	Code    int
	Message string
	Data    interface{}
}

// NewResponseError 创建新的错误响应
// code: 错误码
// message: 错误消息
// data: 附加数据
// 返回: ResponseError指针
func NewResponseError(code int, message string, data interface{}) *ResponseError {
	return &ResponseError{Code: code, Message: message, Data: data}
}

// Error 实现error接口，格式化错误信息
// 返回: 格式化后的错误字符串
func (e *ResponseError) Error() string {
	return fmt.Sprintf("code=%d message=%s data=%+v", e.Code, e.Message, e.Data)
}
