package session

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

var ErrQueueNotOpened = errors.New("queue has not been opened")

// State 会话状态核心结构
// [重要] 线程安全设计：
//   - 消息队列操作使用RWMutex保护
//   - 其他字段通过原子操作或并发安全容器保证
//
// 模块功能：维护单个会话的所有状态信息
type State struct {
	lastActiveAt time.Time // 最后活跃时间戳

	mu       sync.RWMutex // 消息队列操作锁
	sendChan chan []byte  // 消息发送通道(有缓冲)

	requestID int64 // 自增请求ID

	reqID2respChan cmap.ConcurrentMap[string, chan *protocol.JSONRPCResponse] // 请求ID到响应通道的映射

	// 客户端初始化信息缓存
	clientInfo         *protocol.Implementation     // 客户端实现信息
	clientCapabilities *protocol.ClientCapabilities // 客户端能力声明

	// 订阅资源集合
	subscribedResources cmap.ConcurrentMap[string, struct{}] // 资源URI集合

	receivedInitRequest *pkg.AtomicBool // 是否收到初始化请求
	ready               *pkg.AtomicBool // 会话是否就绪
	closed              *pkg.AtomicBool // 会话是否已关闭
}

func NewState() *State {
	return &State{
		lastActiveAt:        time.Now(),
		reqID2respChan:      cmap.New[chan *protocol.JSONRPCResponse](),
		subscribedResources: cmap.New[struct{}](),
		receivedInitRequest: pkg.NewAtomicBool(),
		ready:               pkg.NewAtomicBool(),
		closed:              pkg.NewAtomicBool(),
	}
}

// SetClientInfo 设置客户端信息
// 参数说明：
//   - ClientInfo: 客户端实现详情
//   - ClientCapabilities: 客户端支持的能力
//
// 典型用例：
//   - 在初始化请求处理时调用
//
// [注意] 非线程安全，应在会话初始化阶段调用
func (s *State) SetClientInfo(ClientInfo *protocol.Implementation, ClientCapabilities *protocol.ClientCapabilities) {
	s.clientInfo = ClientInfo
	s.clientCapabilities = ClientCapabilities
}

func (s *State) GetClientCapabilities() *protocol.ClientCapabilities {
	return s.clientCapabilities
}

func (s *State) SetReceivedInitRequest() {
	s.receivedInitRequest.Store(true)
}

func (s *State) GetReceivedInitRequest() bool {
	return s.receivedInitRequest.Load()
}

func (s *State) SetReady() {
	s.ready.Store(true)
}

func (s *State) GetReady() bool {
	return s.ready.Load()
}

func (s *State) IncRequestID() int64 {
	return atomic.AddInt64(&s.requestID, 1)
}

func (s *State) GetReqID2respChan() cmap.ConcurrentMap[string, chan *protocol.JSONRPCResponse] {
	return s.reqID2respChan
}

func (s *State) GetSubscribedResources() cmap.ConcurrentMap[string, struct{}] {
	return s.subscribedResources
}

func (s *State) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed.Store(true)

	if s.sendChan != nil {
		close(s.sendChan)
	}
}

func (s *State) updateLastActiveAt() {
	s.lastActiveAt = time.Now()
}

func (s *State) openMessageQueueForSend() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sendChan == nil {
		s.sendChan = make(chan []byte, 64)
	}
}

// enqueueMessage 消息入队
// 参数说明：
//   - ctx: 上下文，用于超时控制
//   - message: 要发送的原始消息
//
// 返回值：
//   - error: 发送失败原因
//
// 设计决策：
//   - 使用读锁保护通道操作
//   - 优先检查会话状态避免无效操作
//
// 性能提示：
//   - 通道操作可能阻塞，需结合上下文超时控制
func (s *State) enqueueMessage(ctx context.Context, message []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed.Load() {
		return errors.New("session already closed")
	}

	if s.sendChan == nil {
		return ErrQueueNotOpened
	}

	select {
	case s.sendChan <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *State) dequeueMessage(ctx context.Context) ([]byte, error) {
	s.mu.RLock()
	if s.sendChan == nil {
		s.mu.RUnlock()
		return nil, ErrQueueNotOpened
	}
	s.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg, ok := <-s.sendChan:
		if msg == nil && !ok {
			// There are no new messages and the chan has been closed, indicating that the request may need to be terminated.
			return nil, pkg.ErrSendEOF
		}
		return msg, nil
	}
}
