package pkg

import (
	"errors"
	"log"
	"runtime/debug"
	"strings"
	"unsafe"
)

// Recover 捕获panic并记录调用栈
func Recover() {
	if r := recover(); r != nil {
		log.Printf("panic: %v\nstack: %s", r, debug.Stack())
	}
}

// RecoverWithFunc 捕获panic并调用自定义处理函数
// f: 自定义处理函数
func RecoverWithFunc(f func(r any)) {
	if r := recover(); r != nil {
		f(r)
		log.Printf("panic: %v\nstack: %s", r, debug.Stack())
	}
}

// B2S 字节切片转字符串(零拷贝)
// b: 字节切片
// 返回: 转换后的字符串
func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// JoinErrors 合并多个错误为一个错误
// errs: 错误切片
// 返回: 合并后的错误
func JoinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	messages := make([]string, len(errs))
	for i, err := range errs {
		messages[i] = err.Error()
	}
	return errors.New(strings.Join(messages, "; "))
}
