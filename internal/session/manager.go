package session

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/xtaci/smux"
)

// Mapping 单个映射关系
type Mapping struct {
	Name       string             `json:"name"`       // 规则名称
	PublicPort string             `json:"publicPort"` // 公网监听端口
	TargetAddr string             `json:"targetAddr"` // 映射目标地址
	Ctx        context.Context    `json:"-"`          // 上下文，用于关闭连接 (忽略 JSON 序列化)
	CtxCancel  context.CancelFunc `json:"-"`          // 上下文取消函数 (忽略 JSON 序列化)
	Traffic    int64              `json:"traffic"`    // 流量统计
	RateLimit  int64              `json:"rateLimit"`  // 流量限速
	Status     string             `json:"status"`     // 连接状态
	Enable     bool               `json:"enable"`     // 是否启用
}

// ClientSession 包装了单个客户端的连接会话
type ClientSession struct {
	ID      string
	Session *smux.Session
}

// Manager 统一的会话与映射管理器
type Manager struct {
	mu         sync.RWMutex
	table      map[string]*Mapping
	sessions   map[string]*ClientSession // 多客户端支持 (key: ClientID)
	defaultID  string                    // 兼容当前设计的默认 ClientID
	totalConns int64
}

// NewManager 初始化管理器
func NewManager() *Manager {
	return &Manager{
		table:    make(map[string]*Mapping),
		sessions: make(map[string]*ClientSession),
	}
}

// ---- Mapping 管理部分 ----

func (m *Manager) SetMapping(name, port, target string, enable bool, traffic int64, rateLimit int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalConns++
	m.table[name] = &Mapping{
		Name:       name,
		PublicPort: port,
		TargetAddr: target,
		Status:     "inactive",
		Enable:     enable,
		Traffic:    traffic,
		RateLimit:  rateLimit,
	}
}

func (m *Manager) GetAllMappings() []*Mapping {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*Mapping, 0, len(m.table))
	for _, mapping := range m.table {
		list = append(list, mapping)
	}
	return list
}

func (m *Manager) GetMapping(name string) *Mapping {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.table[name]
}

func (m *Manager) DeleteMapping(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.table, name)
	m.totalConns--
}

func (m *Manager) UpdateEnable(name string, enable bool) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if mapping, ok := m.table[name]; ok {
		mapping.Enable = enable
		return true
	}
	return false
}

func (m *Manager) UpdateStatus(name string, status string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if mapping, ok := m.table[name]; ok {
		mapping.Status = status
		return true
	}
	return false
}

func (m *Manager) UpdateRateLimit(name string, rateLimit int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if mapping, ok := m.table[name]; ok {
		mapping.RateLimit = rateLimit
	}
}

func (m *Manager) UpdateMapping(name string, port string, add string, limit int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if mapping, ok := m.table[name]; ok {
		mapping.PublicPort = port
		mapping.TargetAddr = add
		mapping.RateLimit = limit
	}
}

func (m *Manager) CloseMapping(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	mapping, ok := m.table[name]
	if !ok {
		return errors.New("规则不存在，请检查name是否正确")
	}
	if mapping.Status == "active" && mapping.CtxCancel != nil {
		mapping.CtxCancel()
	}
	return nil
}

func (m *Manager) AddTraffic(name string, traffic int64) {
	m.mu.RLock()
	mapping, ok := m.table[name]
	m.mu.RUnlock()

	if ok {
		atomic.AddInt64(&mapping.Traffic, traffic)
	}
}

// GetTraffic 获取特定映射的安全流量值
func (m *Manager) GetTraffic(name string) int64 {
	m.mu.RLock()
	mapping, ok := m.table[name]
	m.mu.RUnlock()

	if ok {
		return atomic.LoadInt64(&mapping.Traffic)
	}
	return 0
}

// ---- Session 管理部分 ----

func (m *Manager) SetSession(clientID string, session *smux.Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[clientID] = &ClientSession{
		ID:      clientID,
		Session: session,
	}
	// 记录最近连接的客户端，用于单节点兼容
	m.defaultID = clientID
}

func (m *Manager) GetSession(clientID string) *smux.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if sess, ok := m.sessions[clientID]; ok {
		return sess.Session
	}
	return nil
}

func (m *Manager) GetDefaultSession() *smux.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if sess, ok := m.sessions[m.defaultID]; ok {
		return sess.Session
	}
	return nil
}

func (m *Manager) RemoveSession(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, clientID)
	if m.defaultID == clientID {
		m.defaultID = ""
	}
}
