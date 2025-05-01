package pkg

import "log"

// Logger 定义日志记录器接口
type Logger interface {
	Debugf(format string, a ...any)
	Infof(format string, a ...any)
	Warnf(format string, a ...any)
	Errorf(format string, a ...any)
}

// LogLevel 定义日志级别类型
type LogLevel uint32

const (
	LogLevelDebug = LogLevel(0)
	LogLevelInfo  = LogLevel(1)
	LogLevelWarn  = LogLevel(2)
	LogLevelError = LogLevel(3)
)

var DefaultLogger Logger = &defaultLogger{
	logLevel: LogLevelInfo,
}

var DebugLogger Logger = &defaultLogger{
	logLevel: LogLevelDebug,
}

// defaultLogger 默认日志记录器实现
// logLevel: 当前日志级别
type defaultLogger struct {
	logLevel LogLevel
}

// Debugf 记录调试级别日志
// format: 格式化字符串
// a: 格式化参数
func (l *defaultLogger) Debugf(format string, a ...any) {
	if l.logLevel > LogLevelDebug {
		return
	}
	log.Printf("[Debug] "+format+"\n", a...)
}

// Infof 记录信息级别日志
// format: 格式化字符串
// a: 格式化参数
func (l *defaultLogger) Infof(format string, a ...any) {
	if l.logLevel > LogLevelInfo {
		return
	}
	log.Printf("[Info] "+format+"\n", a...)
}

// Warnf 记录警告级别日志
// format: 格式化字符串
// a: 格式化参数
func (l *defaultLogger) Warnf(format string, a ...any) {
	if l.logLevel > LogLevelWarn {
		return
	}
	log.Printf("[Warn] "+format+"\n", a...)
}

// Errorf 记录错误级别日志
// format: 格式化字符串
// a: 格式化参数
func (l *defaultLogger) Errorf(format string, a ...any) {
	if l.logLevel > LogLevelError {
		return
	}
	log.Printf("[Error] "+format+"\n", a...)
}
