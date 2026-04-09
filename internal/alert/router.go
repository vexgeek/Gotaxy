package alert

import (
	"fmt"
	"log"

	"github/JustGopher/Gotaxy/internal/eventbus"
	"github/JustGopher/Gotaxy/pkg/email"
)

// Router 报警路由模块
type Router struct {
	bus           *eventbus.EventBus
	targetEmailFn func() string // 动态获取目标邮箱地址的方法
}

// NewRouter 初始化报警路由
func NewRouter(bus *eventbus.EventBus, targetEmailFn func() string) *Router {
	return &Router{
		bus:           bus,
		targetEmailFn: targetEmailFn,
	}
}

// Start 启动报警路由订阅
func (r *Router) Start() {
	r.bus.Subscribe(eventbus.EventClientDisconnected, r.handleClientDisconnected)
}

// 处理客户端断开连接事件
func (r *Router) handleClientDisconnected(e eventbus.Event) {
	payload, ok := e.Payload.(eventbus.ClientDisconnectedPayload)
	if !ok {
		return
	}

	targetEmail := r.targetEmailFn()
	if targetEmail == "" {
		log.Println("[AlertRouter] 未配置接收邮箱，跳过报警发送")
		return
	}

	subject := "【Gotaxy 报警】客户端掉线通知"
	body := fmt.Sprintf("<h2>Gotaxy 客户端掉线通知</h2><p><b>客户端 ID:</b> %s</p><p><b>断开时间:</b> %v</p><p><b>断开原因:</b> %s</p>",
		payload.ClientID, e.Timestamp.Format("2006-01-02 15:04:05"), payload.Reason)

	err := email.SendEmail(targetEmail, subject, body)
	if err != nil {
		log.Printf("[AlertRouter] 发送邮件失败: %v\n", err)
	} else {
		log.Printf("[AlertRouter] 成功向 %s 发送报警邮件\n", targetEmail)
	}
}
