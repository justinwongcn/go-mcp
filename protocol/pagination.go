package protocol

// PaginatedRequest 表示支持分页的请求
// [注意] 该请求用于需要分页处理的场景
// Cursor: 分页游标(可选)，为空表示第一页
type PaginatedRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// PaginatedResult 表示支持分页的响应
// NextCursor: 下一页游标(可选)，为空表示没有更多数据
type PaginatedResult struct {
	NextCursor string `json:"nextCursor,omitempty"`
}
