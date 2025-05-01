package protocol

// CancelledNotification 表示请求取消通知
// [重要] 该通知由服务器发送给客户端，表示某个请求已被取消
// RequestID: 被取消请求的唯一标识符
// Reason: 取消原因(可选)，用于调试目的
type CancelledNotification struct {
	RequestID RequestID `json:"requestId"`
	Reason    string    `json:"reason,omitempty"`
}

// NewCancelledNotification 创建新的取消通知
// [注意] 该方法由服务器内部调用，客户端不应直接调用
// requestID: 被取消请求的ID
// reason: 取消原因描述(可选)
func NewCancelledNotification(requestID RequestID, reason string) *CancelledNotification {
	return &CancelledNotification{
		RequestID: requestID,
		Reason:    reason,
	}
}
