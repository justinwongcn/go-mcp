package protocol

// LoggingLevel 表示日志消息的严重级别
// [重要] 日志级别从高到低依次为：
// Emergency > Alert > Critical > Error > Warning > Notice > Info > Debug
type LoggingLevel string

const (
	LogEmergency LoggingLevel = "emergency" // 系统不可用
	LogAlert     LoggingLevel = "alert"     // 必须立即采取行动
	LogCritical  LoggingLevel = "critical"  // 严重错误
	LogError     LoggingLevel = "error"     // 一般错误
	LogWarning   LoggingLevel = "warning"   // 警告
	LogNotice    LoggingLevel = "notice"    // 需要注意
	LogInfo      LoggingLevel = "info"      // 一般信息
	LogDebug     LoggingLevel = "debug"     // 调试信息
)

// SetLoggingLevelRequest 表示设置日志级别的请求
// Level: 要设置的日志级别
type SetLoggingLevelRequest struct {
	Level LoggingLevel `json:"level"`
}

// SetLoggingLevelResult 表示设置日志级别的响应
// Success: 是否设置成功
type SetLoggingLevelResult struct {
	Success bool `json:"success"`
}

// LogMessageNotification 表示日志消息通知
// Level: 日志级别
// Message: 日志消息内容
// Meta: 元数据(可选)
type LogMessageNotification struct {
	Level   LoggingLevel           `json:"level"`
	Message string                 `json:"message"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// NewSetLoggingLevelRequest 创建新的设置日志级别请求
// level: 要设置的日志级别
func NewSetLoggingLevelRequest(level LoggingLevel) *SetLoggingLevelRequest {
	return &SetLoggingLevelRequest{
		Level: level,
	}
}

// NewSetLoggingLevelResult 创建新的设置日志级别响应
// success: 是否设置成功
func NewSetLoggingLevelResult(success bool) *SetLoggingLevelResult {
	return &SetLoggingLevelResult{
		Success: success,
	}
}

// NewLogMessageNotification 创建新的日志消息通知
// level: 日志级别
// message: 日志消息内容
// meta: 元数据(可选)
func NewLogMessageNotification(level LoggingLevel, message string, meta map[string]interface{}) *LogMessageNotification {
	return &LogMessageNotification{
		Level:   level,
		Message: message,
		Meta:    meta,
	}
}
