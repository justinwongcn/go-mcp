package pkg

import "sync"

// SyncMap 提供类型安全的同步映射
// m: 底层sync.Map实例
type SyncMap[V any] struct {
	m sync.Map
}

// Delete 删除指定键
// key: 要删除的键
func (m *SyncMap[V]) Delete(key string) {
	m.m.Delete(key)
}

// Load 加载指定键的值
// key: 要加载的键
// 返回: 值和是否存在标志
func (m *SyncMap[V]) Load(key string) (value V, ok bool) {
	v, ok := m.m.Load(key)
	if !ok {
		return value, ok
	}
	return v.(V), ok
}

// LoadAndDelete 加载并删除指定键
// key: 要操作的键
// 返回: 值和是否加载标志
func (m *SyncMap[V]) LoadAndDelete(key string) (value V, loaded bool) {
	v, loaded := m.m.LoadAndDelete(key)
	if !loaded {
		return value, loaded
	}
	return v.(V), loaded
}

// LoadOrStore 加载或存储指定键的值
// key: 要操作的键
// value: 要存储的值
// 返回: 实际值和是否加载标志
func (m *SyncMap[V]) LoadOrStore(key string, value V) (actual V, loaded bool) {
	a, loaded := m.m.LoadOrStore(key, value)
	return a.(V), loaded
}

// Range 遍历映射中的所有键值对
// f: 遍历函数
func (m *SyncMap[V]) Range(f func(key string, value V) bool) {
	m.m.Range(func(key, value any) bool { return f(key.(string), value.(V)) })
}

// Store 存储键值对
// key: 键
// value: 值
func (m *SyncMap[V]) Store(key string, value V) {
	m.m.Store(key, value)
}
