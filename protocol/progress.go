package protocol

// ProgressNotification 表示长请求的进度通知
// [重要] 该通知用于向客户端报告长时间运行操作的进度
// ProgressToken: 进度令牌，用于关联通知和原始请求
// Progress: 当前进度值
// Total: 总进度值(可选)
type ProgressNotification struct {
	ProgressToken ProgressToken `json:"progressToken"`
	Progress      float64       `json:"progress"`
	Total         float64       `json:"total,omitempty"`
}

// ProgressToken 进度令牌接口
// [注意] 可以是字符串或整数类型，用于唯一标识进度
// 实现者应确保令牌在请求生命周期内保持唯一
type ProgressToken interface{} // can be string or integer

// NewProgressNotification 创建新的进度通知
// token: 进度令牌
// progress: 当前进度值
// total: 总进度值(可选)
func NewProgressNotification(token ProgressToken, progress float64, total float64) *ProgressNotification {
	return &ProgressNotification{
		ProgressToken: token,
		Progress:      progress,
		Total:         total,
	}
}
