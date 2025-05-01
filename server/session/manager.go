package session

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

// Manager 会话管理器核心结构
// [重要] 线程安全设计：所有会话操作通过SyncMap保证并发安全
// 模块功能：管理会话生命周期，包括创建/关闭/心跳检测/消息队列操作
// 项目定位：server核心组件，负责维护所有活跃会话状态
type Manager struct {
	activeSessions pkg.SyncMap[*State]   // 活跃会话映射表
	closedSessions pkg.SyncMap[struct{}] // 已关闭会话记录（防重复关闭）

	stopHeartbeat chan struct{} // 心跳检测停止信号

	logger pkg.Logger // 日志记录器

	detection   func(ctx context.Context, sessionID string) error // 会话健康检测函数
	maxIdleTime time.Duration                                     // 会话最大空闲时间（0表示不限制）
}

// NewManager 创建会话管理器实例
// 参数说明：
//   - detection: 会话健康检测回调函数，返回nil表示会话健康
//
// 设计决策：
//   - 使用默认日志器，可通过SetLogger()替换
//   - 心跳检测通道初始化为无缓冲，确保及时停止
func NewManager(detection func(ctx context.Context, sessionID string) error) *Manager {
	return &Manager{
		detection:     detection,
		stopHeartbeat: make(chan struct{}),
		logger:        pkg.DefaultLogger,
	}
}

func (m *Manager) SetMaxIdleTime(d time.Duration) {
	m.maxIdleTime = d
}

func (m *Manager) SetLogger(logger pkg.Logger) {
	m.logger = logger
}

// CreateSession 创建新会话
// 返回值：
//   - string: 生成的唯一会话ID
//
// 算法说明：
//   - 使用UUID v4生成唯一会话标识
//   - 初始化会话状态结构体
//
// [注意] 并发安全：通过SyncMap.Store保证线程安全
func (m *Manager) CreateSession() string {
	sessionID := uuid.NewString()
	state := NewState()
	m.activeSessions.Store(sessionID, state)
	return sessionID
}

// IsActiveSession 检查会话是否活跃
// 参数说明：
//   - sessionID: 要检查的会话ID
//
// 返回值：
//   - bool: true表示会话存在且活跃
//
// 性能提示：
//   - O(1)时间复杂度，基于并发安全哈希表查找
func (m *Manager) IsActiveSession(sessionID string) bool {
	_, has := m.activeSessions.Load(sessionID)
	return has
}

func (m *Manager) IsClosedSession(sessionID string) bool {
	_, has := m.closedSessions.Load(sessionID)
	return has
}

// GetSession 获取会话状态
// 参数说明：
//   - sessionID: 要获取的会话ID
//
// 返回值：
//   - *State: 会话状态对象指针
//   - bool: true表示获取成功
//
// [注意] 空会话ID会直接返回false
// 典型用例：
//   - 在消息收发前验证会话有效性
func (m *Manager) GetSession(sessionID string) (*State, bool) {
	if sessionID == "" {
		return nil, false
	}
	state, has := m.activeSessions.Load(sessionID)
	if !has {
		return nil, false
	}
	return state, true
}

func (m *Manager) OpenMessageQueueForSend(sessionID string) error {
	state, has := m.GetSession(sessionID)
	if !has {
		return pkg.ErrLackSession
	}
	state.openMessageQueueForSend()
	return nil
}

func (m *Manager) EnqueueMessageForSend(ctx context.Context, sessionID string, message []byte) error {
	state, has := m.GetSession(sessionID)
	if !has {
		return pkg.ErrLackSession
	}
	return state.enqueueMessage(ctx, message)
}

func (m *Manager) DequeueMessageForSend(ctx context.Context, sessionID string) ([]byte, error) {
	state, has := m.GetSession(sessionID)
	if !has {
		return nil, pkg.ErrLackSession
	}
	return state.dequeueMessage(ctx)
}

func (m *Manager) UpdateSessionLastActiveAt(sessionID string) {
	state, ok := m.activeSessions.Load(sessionID)
	if !ok {
		return
	}
	state.updateLastActiveAt()
}

func (m *Manager) CloseSession(sessionID string) {
	state, ok := m.activeSessions.LoadAndDelete(sessionID)
	if !ok {
		return
	}
	state.Close()
	m.closedSessions.Store(sessionID, struct{}{})
}

func (m *Manager) CloseAllSessions() {
	m.activeSessions.Range(func(sessionID string, _ *State) bool {
		// Here we load the session again to prevent concurrency conflicts with CloseSession, which may cause repeated close chan
		m.CloseSession(sessionID)
		return true
	})
}

// StartHeartbeatAndCleanInvalidSessions 启动心跳检测和会话清理
// 功能说明：
//   - 每分钟检查一次所有会话状态
//   - 清理条件：
//     1. 会话超过最大空闲时间(maxIdleTime)
//     2. 健康检测连续失败3次
//
// 设计决策：
//   - 使用time.Ticker实现定时任务
//   - 通过stopHeartbeat通道实现优雅停止
//
// [重要] 并发安全：
//   - 使用Range方法保证遍历时的线程安全
//   - 日志记录会话关闭原因便于问题排查
func (m *Manager) StartHeartbeatAndCleanInvalidSessions() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopHeartbeat:
			return
		case <-ticker.C:
			now := time.Now()
			m.activeSessions.Range(func(sessionID string, state *State) bool {
				if m.maxIdleTime != 0 && now.Sub(state.lastActiveAt) > m.maxIdleTime {
					m.logger.Infof("session expire, session id: %v", sessionID)
					m.CloseSession(sessionID)
					return true
				}

				var err error
				for i := 0; i < 3; i++ {
					if err = m.detection(context.Background(), sessionID); err == nil {
						return true
					}
				}
				m.logger.Infof("session detection fail, session id: %v, fail reason: %+v", sessionID, err)
				m.CloseSession(sessionID)
				return true
			})
		}
	}
}

func (m *Manager) StopHeartbeat() {
	close(m.stopHeartbeat)
}

func (m *Manager) RangeSessions(f func(sessionID string, state *State) bool) {
	m.activeSessions.Range(f)
}

func (m *Manager) IsEmpty() bool {
	isEmpty := true
	m.activeSessions.Range(func(string, *State) bool {
		isEmpty = false
		return false
	})
	return isEmpty
}
