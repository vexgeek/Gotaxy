package eventbus

import (
	"sync"
	"time"
)

// EventType 定义核心事件类型
type EventType string

const (
	EventClientConnected    EventType = "CLIENT_CONNECTED"
	EventClientDisconnected EventType = "CLIENT_DISCONNECTED"
	EventTrafficReport      EventType = "TRAFFIC_REPORT"
	EventTunnelError        EventType = "TUNNEL_ERROR"
)

// Event 事件结构体
type Event struct {
	Type      EventType
	Timestamp time.Time
	Payload   interface{}
}

// Handler 事件处理器
type Handler func(Event)

// EventBus 内部事件总线
type EventBus struct {
	handlers map[EventType][]Handler
	mu       sync.RWMutex
}

// NewEventBus 创建新的事件总线
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]Handler),
	}
}

// Subscribe 订阅事件
func (b *EventBus) Subscribe(eventType EventType, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Publish 异步发布事件
func (b *EventBus) Publish(eventType EventType, payload interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Payload:   payload,
	}

	if handlers, ok := b.handlers[eventType]; ok {
		for _, handler := range handlers {
			// 开启 Goroutine 避免阻塞核心调度线程
			go handler(event)
		}
	}
}
