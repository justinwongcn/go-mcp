package pkg

import (
	"context"
	"time"
)

// CancelShieldContext 提供取消保护的上下文包装
// Context: 底层上下文对象
type CancelShieldContext struct {
	context.Context
}

// NewCancelShieldContext 创建新的取消保护上下文
// ctx: 原始上下文
// 返回: 包装后的上下文对象
func NewCancelShieldContext(ctx context.Context) context.Context {
	return CancelShieldContext{Context: ctx}
}

// Deadline 实现上下文接口，始终返回无超时
// 返回: 零值time.Time和false
func (v CancelShieldContext) Deadline() (deadline time.Time, ok bool) {
	return
}

// Done 实现上下文接口，始终返回nil
// 返回: nil通道
func (v CancelShieldContext) Done() <-chan struct{} {
	return nil
}

// Err 实现上下文接口，始终返回nil
// 返回: nil错误
func (v CancelShieldContext) Err() error {
	return nil
}
