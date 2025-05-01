package server

import (
	"context"
	"errors"
)

// sessionIDKey 上下文会话ID键类型
// 用途：作为context.WithValue的key类型，用于在上下文中存储会话ID
// [注意] 使用空结构体作为key类型是最佳实践，避免内存分配
type sessionIDKey struct{}

// setSessionIDToCtx 设置会话ID到上下文
// 参数说明：
//   - ctx: 原始上下文
//   - sessionID: 要设置的会话ID
//
// 返回值：
//   - context.Context: 包含会话ID的新上下文
//
// 设计决策：
//   - 使用私有key类型防止冲突
func setSessionIDToCtx(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, sessionID)
}

// getSessionIDFromCtx 从上下文中获取会话ID
// 参数说明：
//   - ctx: 要查询的上下文
//
// 返回值：
//   - string: 会话ID
//   - error: 错误信息（未找到会话ID时）
//
// [重要] 类型安全：
//   - 对获取的值进行类型断言确保是string类型
func getSessionIDFromCtx(ctx context.Context) (string, error) {
	sessionID := ctx.Value(sessionIDKey{})
	if sessionID == nil {
		return "", errors.New("no session id found")
	}
	return sessionID.(string), nil
}
