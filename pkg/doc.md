# pkg 模块文档

## 概述
pkg模块提供了一系列基础工具和实用功能，包括原子操作、上下文包装、错误处理、日志记录等通用组件。

## 文件结构

### atomic.go
- 核心功能：
  - 线程安全的布尔值操作 (`AtomicBool`)
  - 线程安全的字符串操作 (`AtomicString`)

### context.go
- 核心功能：
  - 取消保护的上下文包装 (`CancelShieldContext`)

### errors.go
- 核心功能：
  - 预定义错误集合 (`ErrClientNotSupport`, `ErrServerNotSupport`等)
  - 标准错误响应结构 (`ResponseError`)

### helper.go
- 核心功能：
  - panic捕获与调用栈记录 (`Recover`, `RecoverWithFunc`)
  - 字节切片与字符串转换 (`B2S`)
  - 错误合并 (`JoinErrors`)

### json.go
- 核心功能：
  - JSON解析与错误格式化 (`JSONUnmarshal`)

### log.go
- 核心功能：
  - 日志记录器接口 (`Logger`)
  - 日志级别定义 (`LogLevel`)
  - 默认日志实现 (`defaultLogger`)

### sync_map.go
- 核心功能：
  - 类型安全的同步映射 (`SyncMap`)

## 使用示例
```go
// 使用AtomicBool
b := pkg.NewAtomicBool()
b.Store(true)
val := b.Load()

// 使用CancelShieldContext
ctx := pkg.NewCancelShieldContext(context.Background())

// 使用SyncMap
m := &pkg.SyncMap[int]{}
m.Store("key", 123)
val, ok := m.Load("key")
```

## 注意事项
1. 原子操作类型(AtomicBool/AtomicString)是线程安全的
2. CancelShieldContext会屏蔽原始上下文的取消信号
3. SyncMap是类型安全的，但需要指定具体类型参数
4. 默认日志级别为Info，可通过设置DefaultLogger修改