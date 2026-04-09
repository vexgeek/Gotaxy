package main

import (
	"github/JustGopher/Gotaxy/internal/core"
	"github/JustGopher/Gotaxy/internal/inits"
	"github/JustGopher/Gotaxy/internal/shell"
	"github/JustGopher/Gotaxy/internal/web"
)

func main() {
	// 1. 初始化日志
	infoLog, errorLog := inits.LogInit()

	// 2. 初始化数据库
	db := inits.DBInit(errorLog)
	defer db.Close()

	// 3. 实例化并组装核心服务器
	server := core.NewServer(db, infoLog, errorLog)
	server.Config.ConfigLoad(db, server.Sessions)
	server.InfoLog.Println("Gotaxy 启动成功，架构版本: v0.3.0 (事件驱动架构)")

	// 4. 启动 Web 面板服务 (后台协程运行)
	webServer := web.NewWebServer(server)
	go webServer.Start("8080")

	// 5. 启动交互式 CLI 终端
	sh := shell.New(server)
	shell.RegisterCMD(sh)
	sh.Run()
}
