package protocol

// CompleteRequest 表示完成选项请求
// [重要] 该请求由客户端发送给服务器，用于获取可能的完成选项
// Argument: 包含参数名称和值的结构体
// Ref: 引用类型，可以是PromptReference或ResourceReference
type CompleteRequest struct {
	Argument struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"argument"`
	Ref any `json:"ref"` // Can be PromptReference or ResourceReference
}

// PromptReference 提示词引用类型
// Type: 引用类型标识
// Name: 提示词名称
type PromptReference struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// ResourceReference 资源引用类型
// Type: 引用类型标识
// URI: 资源统一标识符
type ResourceReference struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
}

// CompleteResult 表示完成请求的响应
// Completion: 包含完成选项的结构体
type CompleteResult struct {
	Completion Complete `json:"completion"`
}

// Complete 完成选项结构
// Values: 可能的完成值列表
// HasMore: 是否还有更多结果(可选)
// Total: 总结果数(可选)
type Complete struct {
	Values  []string `json:"values"`
	HasMore bool     `json:"hasMore,omitempty"`
	Total   int      `json:"total,omitempty"`
}

// NewCompleteRequest 创建新的完成请求
// argName: 参数名称
// argValue: 参数值
// ref: 引用对象(PromptReference或ResourceReference)
func NewCompleteRequest(argName string, argValue string, ref interface{}) *CompleteRequest {
	return &CompleteRequest{
		Argument: struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}{
			Name:  argName,
			Value: argValue,
		},
		Ref: ref,
	}
}

// NewCompleteResult 创建新的完成响应
// values: 完成值列表
// hasMore: 是否还有更多结果(可选)
// total: 总结果数(可选)
func NewCompleteResult(values []string, hasMore bool, total int) *CompleteResult {
	return &CompleteResult{
		Completion: struct {
			Values  []string `json:"values"`
			HasMore bool     `json:"hasMore,omitempty"`
			Total   int      `json:"total,omitempty"`
		}{
			Values:  values,
			HasMore: hasMore,
			Total:   total,
		},
	}
}
