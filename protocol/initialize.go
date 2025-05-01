package protocol

// InitializeRequest 表示客户端到服务器的初始化请求
// [重要] 这是连接建立后第一个必须发送的请求
// ClientInfo: 客户端实现信息
// Capabilities: 客户端能力描述
// ProtocolVersion: 协议版本号
type InitializeRequest struct {
	ClientInfo      Implementation     `json:"clientInfo"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ProtocolVersion string             `json:"protocolVersion"`
}

// InitializeResult 表示服务器对初始化请求的响应
// ServerInfo: 服务器实现信息
// Capabilities: 服务器能力描述
// ProtocolVersion: 协议版本号
// Instructions: 初始化指令(可选)
type InitializeResult struct {
	ServerInfo      Implementation     `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ProtocolVersion string             `json:"protocolVersion"`
	Instructions    string             `json:"instructions,omitempty"`
}

// Implementation 描述MCP实现的名称和版本
// Name: 实现名称
// Version: 实现版本号
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities 客户端能力描述
// Sampling: 采样能力配置(可选)
type ClientCapabilities struct {
	// Experimental map[string]interface{} `json:"experimental,omitempty"`
	// Roots        *RootsCapability       `json:"roots,omitempty"`
	Sampling interface{} `json:"sampling,omitempty"`
}

// RootsCapability 根目录能力配置
// ListChanged: 是否支持根目录列表变更通知
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerCapabilities 服务器能力描述
// Prompts: 提示词能力配置(可选)
// Resources: 资源能力配置(可选)
// Tools: 工具能力配置(可选)
type ServerCapabilities struct {
	// Experimental map[string]interface{} `json:"experimental,omitempty"`
	// Logging      interface{}            `json:"logging,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Tools     *ToolsCapability     `json:"tools,omitempty"`
}

// PromptsCapability 提示词能力配置
// ListChanged: 是否支持提示词列表变更通知
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability 资源能力配置
// ListChanged: 是否支持资源列表变更通知
// Subscribe: 是否支持资源订阅
type ResourcesCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
	Subscribe   bool `json:"subscribe,omitempty"`
}

// ToolsCapability 工具能力配置
// ListChanged: 是否支持工具列表变更通知
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializedNotification 表示初始化完成通知
// [注意] 该通知由客户端在初始化完成后发送给服务器
// Meta: 元数据(可选)
type InitializedNotification struct {
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

// NewInitializeRequest 创建新的初始化请求
// clientInfo: 客户端实现信息
// capabilities: 客户端能力描述
func NewInitializeRequest(clientInfo Implementation, capabilities ClientCapabilities) *InitializeRequest {
	return &InitializeRequest{
		ClientInfo:      clientInfo,
		Capabilities:    capabilities,
		ProtocolVersion: Version,
	}
}

// NewInitializeResult 创建新的初始化响应
// serverInfo: 服务器实现信息
// capabilities: 服务器能力描述
// instructions: 初始化指令(可选)
func NewInitializeResult(serverInfo Implementation, capabilities ServerCapabilities, instructions string) *InitializeResult {
	return &InitializeResult{
		ServerInfo:      serverInfo,
		Capabilities:    capabilities,
		ProtocolVersion: Version,
		Instructions:    instructions,
	}
}

// NewInitializedNotification 创建新的初始化完成通知
func NewInitializedNotification() *InitializedNotification {
	return &InitializedNotification{}
}
