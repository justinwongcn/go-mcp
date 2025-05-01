package pkg

import "sync/atomic"

// AtomicBool 提供线程安全的布尔值操作
// b: 底层atomic.Value存储布尔值
type AtomicBool struct {
	b atomic.Value
}

// NewAtomicBool 创建新的AtomicBool实例
// 返回: 初始值为false的AtomicBool指针
func NewAtomicBool() *AtomicBool {
	b := &AtomicBool{}
	b.b.Store(false)
	return b
}

// Store 原子地存储布尔值
// value: 要存储的布尔值
func (b *AtomicBool) Store(value bool) {
	b.b.Store(value)
}

// Load 原子地加载布尔值
// 返回: 当前存储的布尔值
func (b *AtomicBool) Load() bool {
	return b.b.Load().(bool)
}

// AtomicString 提供线程安全的字符串操作
// b: 底层atomic.Value存储字符串
type AtomicString struct {
	b atomic.Value
}

// NewAtomicString 创建新的AtomicString实例
// 返回: 初始值为空字符串的AtomicString指针
func NewAtomicString() *AtomicString {
	b := &AtomicString{}
	b.b.Store("")
	return b
}

// Store 原子地存储字符串
// value: 要存储的字符串
func (b *AtomicString) Store(value string) {
	b.b.Store(value)
}

// Load 原子地加载字符串
// 返回: 当前存储的字符串
func (b *AtomicString) Load() string {
	return b.b.Load().(string)
}
