package shell

import (
	"context"
	"fmt"
	"strconv"

	"github/JustGopher/Gotaxy/internal/storage/models"
	"github/JustGopher/Gotaxy/internal/tunnel"
	"github/JustGopher/Gotaxy/internal/tunnel/proxy"
	"github/JustGopher/Gotaxy/pkg/tlsgen"
	"github/JustGopher/Gotaxy/pkg/utils"
)

var shellInstance *Shell

func RegisterCMD(sh *Shell) {
	shellInstance = sh
	sh.Register("service", handleService)
	sh.Register("config", handleConfig)
	sh.Register("mapping", handleMapping)
	sh.Register("cert", handleCert)
	sh.Register("clear", func(args []string) {
		print("\033[H\033[2J") // Clear terminal
	})
}

// ---- Service 相关命令 ----
func handleService(args []string) {
	if len(args) == 0 {
		fmt.Println("用法: service [start|stop|status]")
		return
	}
	switch args[0] {
	case "start":
		server := shellInstance.server
		if server.IsRun {
			fmt.Println("❌ 服务已在运行中")
			return
		}
		if !tlsgen.CheckServerCertExist("certs") {
			fmt.Println("❌ 证书缺失，请先使用 `cert gen` 生成证书")
			return
		}
		tunnelCtrl := tunnel.NewController(server)
		go tunnelCtrl.Start()
		server.SetRunStatus(true)
		fmt.Println("✅ 服务启动成功")
	case "stop":
		server := shellInstance.server
		if !server.IsRun {
			fmt.Println("❌ 服务未运行")
			return
		}
		if server.Cancel != nil {
			server.Cancel()
		}
		server.SetRunStatus(false)
		fmt.Println("✅ 服务已停止")
	case "status":
		if shellInstance.server.IsRun {
			fmt.Println("🌟 服务状态: 运行中 (Active)")
		} else {
			fmt.Println("💤 服务状态: 已停止 (Inactive)")
		}
	default:
		fmt.Printf("未知子命令: %s\n", args[0])
	}
}

// ---- Config 相关命令 ----
func handleConfig(args []string) {
	if len(args) == 0 {
		fmt.Println("用法: config [show|set]")
		return
	}
	server := shellInstance.server
	switch args[0] {
	case "show":
		fmt.Println("=== 当前核心配置 ===")
		fmt.Printf("  🌐 Server IP   : %s\n", server.Config.ServerIP)
		fmt.Printf("  🔌 Listen Port : %s\n", server.Config.ListenPort)
		fmt.Printf("  📧 Alert Email : %s\n", server.Config.Email)
		fmt.Printf("  📊 Total Traffic: %d Bytes\n", server.Config.TotalTraffic)
		fmt.Println("====================")
	case "set":
		if len(args) < 3 {
			fmt.Println("用法: config set [ip|port|email] <value>")
			return
		}
		key, value := args[1], args[2]
		switch key {
		case "ip":
			if !utils.IsValidateIP(value) {
				fmt.Println("❌ IP地址格式不正确")
				return
			}
			server.Config.ServerIP = value
			_ = models.UpdateCfg(server.DB, "server_ip", value)
			fmt.Println("✅ IP 更新成功")
		case "port":
			port, err := strconv.Atoi(value)
			if err != nil || port <= 0 || port > 65535 {
				fmt.Println("❌ 端口必须是1-65535之间的数字")
				return
			}
			server.Config.ListenPort = value
			_ = models.UpdateCfg(server.DB, "listen_port", value)
			fmt.Println("✅ 控制端口更新成功")
		case "email":
			if !utils.IsValidateEmail(value) {
				fmt.Println("❌ Email格式不正确")
				return
			}
			server.Config.Email = value
			_ = models.UpdateCfg(server.DB, "email", value)
			fmt.Println("✅ 报警邮箱更新成功")
		default:
			fmt.Printf("未知配置项: %s\n", key)
		}
	default:
		fmt.Printf("未知子命令: %s\n", args[0])
	}
}

// ---- Mapping 相关命令 ----
func handleMapping(args []string) {
	if len(args) == 0 {
		fmt.Println("用法: mapping [list|add|del|upd|start|stop]")
		return
	}
	server := shellInstance.server
	switch args[0] {
	case "list":
		mpg := server.Sessions.GetAllMappings()
		if len(mpg) == 0 {
			fmt.Println("📝 当前暂无任何映射规则")
			return
		}
		fmt.Printf("%-15s %-10s %-20s %-10s %-15s %s\n", "NAME", "PORT", "TARGET", "STATUS", "RATE_LIMIT", "TRAFFIC")
		fmt.Println(stringsRepeat("-", 85))
		for _, v := range mpg {
			statusStr := "❌已停止"
			if v.Enable {
				statusStr = "✅运行中"
			}
			fmt.Printf("%-15s %-10s %-20s %-10s %-15d %d\n", v.Name, v.PublicPort, v.TargetAddr, statusStr, v.RateLimit, v.Traffic)
		}
	case "start":
		if len(args) < 2 {
			fmt.Println("用法: mapping start <name>")
			return
		}
		name := args[1]
		if !server.IsRun {
			ok := server.Sessions.UpdateEnable(name, true)
			if !ok {
				fmt.Printf("❌ 规则 '%s' 不存在\n", name)
				return
			}
			mapping := server.Sessions.GetMapping(name)
			_, _ = models.UpdateMap(server.DB, mapping.Name, mapping.PublicPort, mapping.TargetAddr, true, mapping.RateLimit)
			fmt.Printf("✅ 规则 '%s' 状态已更新为开启 (待服务启动后生效)\n", name)
		} else {
			ok := server.Sessions.UpdateEnable(name, true)
			if !ok {
				fmt.Printf("❌ 规则 '%s' 不存在\n", name)
				return
			}
			mapping := server.Sessions.GetMapping(name)
			mapping.Ctx, mapping.CtxCancel = context.WithCancel(context.Background())
			go proxy.StartPublicListener(server.Ctx, mapping, server)
			fmt.Printf("✅ 规则 '%s' 已启动监听\n", name)
		}
	case "stop":
		if len(args) < 2 {
			fmt.Println("用法: mapping stop <name>")
			return
		}
		name := args[1]
		err := server.Sessions.CloseMapping(name)
		if err != nil {
			fmt.Println("❌ " + err.Error())
			return
		}
		server.Sessions.UpdateEnable(name, false)
		fmt.Printf("✅ 规则 '%s' 已停止\n", name)
	case "add":
		if len(args) < 4 {
			fmt.Println("用法: mapping add <name> <public_port> <target_addr>")
			return
		}
		name, portStr, target := args[1], args[2], args[3]
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port > 65535 {
			fmt.Println("❌ 公网端口格式错误")
			return
		}
		if !utils.IsValidateAddr(target) {
			fmt.Println("❌ 目标地址格式不正确 (例: 127.0.0.1:3306)")
			return
		}
		if server.Sessions.GetMapping(name) != nil {
			fmt.Println("❌ 规则名已存在")
			return
		}
		limit := int64(1024 * 1024 * 2) // 默认 2M
		err = models.InsertMpg(server.DB, models.Mapping{
			Name:       name,
			PublicPort: portStr,
			TargetAddr: target,
			Enable:     false,
			Traffic:    0,
			RateLimit:  limit,
		})
		if err != nil {
			fmt.Println("❌ 数据库保存失败:", err)
			return
		}
		server.Sessions.SetMapping(name, portStr, target, false, 0, limit)
		fmt.Println("✅ 规则添加成功")
	case "del":
		if len(args) < 2 {
			fmt.Println("用法: mapping del <name>")
			return
		}
		name := args[1]
		mpg := server.Sessions.GetMapping(name)
		if mpg == nil {
			fmt.Println("❌ 规则不存在")
			return
		}
		if mpg.Status == "active" {
			fmt.Println("❌ 当前映射正在运行中，无法删除，请先执行 `mapping stop`")
			return
		}
		server.Sessions.DeleteMapping(name)
		_ = models.DeleteMapByName(server.DB, name)
		fmt.Println("✅ 规则已删除")
	case "upd":
		if len(args) < 5 {
			fmt.Println("用法: mapping upd <name> <port> <target_addr> <rate_limit>")
			return
		}
		name, portStr, target, limitStr := args[1], args[2], args[3], args[4]
		mpg := server.Sessions.GetMapping(name)
		if mpg == nil {
			fmt.Println("❌ 规则不存在")
			return
		}
		if mpg.Status == "active" {
			fmt.Println("❌ 当前映射正在运行中，无法修改，请先执行 `mapping stop`")
			return
		}
		if !utils.IsValidateAddr(target) {
			fmt.Println("❌ 目标地址格式不正确")
			return
		}
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			fmt.Println("❌ 限速参数必须是数字(字节/秒)")
			return
		}
		_, err = models.UpdateMap(server.DB, name, portStr, target, mpg.Enable, int64(limit))
		if err != nil {
			fmt.Println("❌ 更新映射数据失败")
			return
		}
		server.Sessions.UpdateMapping(name, portStr, target, int64(limit))
		fmt.Println("✅ 规则更新成功")
	default:
		fmt.Printf("未知子命令: %s\n", args[0])
	}
}

// ---- Cert 相关命令 ----
func handleCert(args []string) {
	if len(args) == 0 {
		fmt.Println("用法: cert [ca|gen]")
		return
	}
	server := shellInstance.server
	switch args[0] {
	case "ca":
		year := 10
		overwrite := false
		if len(args) > 1 {
			if y, err := strconv.Atoi(args[1]); err == nil && y > 0 {
				year = y
			} else if args[1] == "-overwrite" {
				overwrite = true
			}
		}
		if len(args) > 2 && args[2] == "-overwrite" {
			overwrite = true
		}

		if overwrite {
			fmt.Print("⚠️ 确定要重新生成 CA 证书吗？原有证书将全部失效 (y/n): ")
			line, _ := shellInstance.Rl.Readline()
			if line != "y" {
				fmt.Println("操作已取消")
				return
			}
		}

		err := tlsgen.GenerateCA("certs", year, overwrite)
		if err != nil {
			fmt.Println("❌ 生成 CA 证书失败:", err)
			return
		}
		fmt.Println("✅ CA 证书生成成功")
	case "gen":
		days := 365
		if len(args) > 1 {
			if d, err := strconv.Atoi(args[1]); err == nil && d > 0 {
				days = d
			}
		}
		if server.Config.ServerIP == "" {
			fmt.Println("❌ 缺少 Server IP 配置，请先执行 `config set ip <ip>`")
			return
		}
		err := tlsgen.GenerateServerAndClientCerts(server.Config.ServerIP, "certs", days, "certs/ca.crt", "certs/ca.key")
		if err != nil {
			fmt.Println("❌ 签发证书失败:", err)
			return
		}
		fmt.Println("✅ 服务端与客户端证书签发成功")
	default:
		fmt.Printf("未知子命令: %s\n", args[0])
	}
}

func stringsRepeat(s string, count int) string {
	res := ""
	for i := 0; i < count; i++ {
		res += s
	}
	return res
}
