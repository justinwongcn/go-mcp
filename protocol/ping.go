package protocol

// PingRequest 表示心跳检测请求
// [注意] 该请求用于检测连接是否存活
type PingRequest struct{}

// PingResult 表示心跳检测响应
// [注意] 该响应仅表示服务器已收到并处理了心跳请求
type PingResult struct{}

// NewPingRequest 创建新的心跳检测请求
func NewPingRequest() *PingRequest {
	return &PingRequest{}
}

// NewPingResult 创建新的心跳检测响应
func NewPingResult() *PingResult {
	return &PingResult{}
}
