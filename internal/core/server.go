package core

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github/JustGopher/Gotaxy/internal/alert"
	"github/JustGopher/Gotaxy/internal/config"
	"github/JustGopher/Gotaxy/internal/eventbus"
	"github/JustGopher/Gotaxy/internal/session"
)

// GotaxyServer 核心服务端结构，取代原本的 global 状态
type GotaxyServer struct {
	Ctx      context.Context
	Cancel   context.CancelFunc
	EventBus *eventbus.EventBus
	Config   *config.Config
	Sessions *session.Manager
	DB       *sql.DB
	Alert    *alert.Router
	InfoLog  *log.Logger
	ErrorLog *log.Logger
	IsRun    bool
}

// NewServer 初始化核心服务器依赖
func NewServer(db *sql.DB, infoLog, errorLog *log.Logger) *GotaxyServer {
	if infoLog == nil {
		infoLog = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	}
	if errorLog == nil {
		errorLog = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	}

	ctx, cancel := context.WithCancel(context.Background())
	bus := eventbus.NewEventBus()
	cfg := &config.Config{}
	sessMgr := session.NewManager()

	// 初始化报警路由
	alertRouter := alert.NewRouter(bus, func() string {
		return cfg.Email
	})
	alertRouter.Start()

	// 监听流量上报事件，更新内存中的流量统计
	bus.Subscribe(eventbus.EventTrafficReport, func(e eventbus.Event) {
		payload, ok := e.Payload.(eventbus.TrafficReportPayload)
		if ok {
			sessMgr.AddTraffic(payload.MappingName, payload.Bytes)
			cfg.TotalTraffic += payload.Bytes
		}
	})

	return &GotaxyServer{
		Ctx:      ctx,
		Cancel:   cancel,
		EventBus: bus,
		Config:   cfg,
		Sessions: sessMgr,
		DB:       db,
		Alert:    alertRouter,
		InfoLog:  infoLog,
		ErrorLog: errorLog,
	}
}

// Start 控制面启动逻辑的入口可以在这里或 tunnel 模块
func (s *GotaxyServer) SetRunStatus(status bool) {
	s.IsRun = status
}
