package core

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	"github/JustGopher/Gotaxy/internal/alert"
	"github/JustGopher/Gotaxy/internal/config"
	"github/JustGopher/Gotaxy/internal/eventbus"
	"github/JustGopher/Gotaxy/internal/session"
	"github/JustGopher/Gotaxy/internal/storage/models"
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

	server := &GotaxyServer{
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

	// 启动定时持久化流量任务
	go server.startTrafficPersistence()

	return server
}

// startTrafficPersistence 每 30 秒将内存中的流量统计持久化到 SQLite 数据库
func (s *GotaxyServer) startTrafficPersistence() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 1. 更新总流量
		totalStr := strconv.FormatInt(s.Config.TotalTraffic, 10)
		err := models.UpdateCfg(s.DB, "total_traffic", totalStr)
		if err != nil {
			s.ErrorLog.Printf("持久化总流量失败: %v", err)
		}

		// 2. 更新每个映射规则的流量
		mappings := s.Sessions.GetAllMappings()
		for _, m := range mappings {
			if m.Traffic > 0 {
				err := models.UpdateTra(s.DB, m.Name, m.Traffic)
				if err != nil {
					s.ErrorLog.Printf("持久化映射 [%s] 流量失败: %v", m.Name, err)
				}
			}
		}
	}
}

// SetRunStatus 控制面启动逻辑的入口可以在这里或 tunnel 模块
func (s *GotaxyServer) SetRunStatus(status bool) {
	s.IsRun = status
}
