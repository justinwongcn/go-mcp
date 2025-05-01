package protocol

// ListRootsRequest 表示列出根目录的请求
type ListRootsRequest struct{}

// ListRootsResult 表示列出根目录的响应
// Roots: 根目录列表
type ListRootsResult struct {
	Roots []Root `json:"roots"`
}

// Root 表示服务器可以操作的根目录或文件
// Name: 根目录名称(可选)
// URI: 根目录URI
type Root struct {
	Name string `json:"name,omitempty"`
	URI  string `json:"uri"`
}

// RootsListChangedNotification 表示根目录列表变更通知
type RootsListChangedNotification struct {
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

// NewListRootsRequest 创建新的列出根目录请求
func NewListRootsRequest() *ListRootsRequest {
	return &ListRootsRequest{}
}

// NewListRootsResult 创建新的列出根目录响应
// roots: 根目录列表
func NewListRootsResult(roots []Root) *ListRootsResult {
	return &ListRootsResult{
		Roots: roots,
	}
}

// NewRootsListChangedNotification 创建新的根目录列表变更通知
func NewRootsListChangedNotification() *RootsListChangedNotification {
	return &RootsListChangedNotification{}
}
