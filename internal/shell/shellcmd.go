package shell

import (
	"context"
	"fmt"
	"log"
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
	sh.Register("gen-ca", generateCA)
	sh.Register("gen-certs", generateCerts)
	sh.Register("start", start)
	sh.Register("stop", stop)
	sh.Register("show-config", showConfig)
	sh.Register("show-mapping", showMapping)
	sh.Register("set-ip", setIP)
	sh.Register("set-port", setPort)
	sh.Register("set-email", setEmail)
	sh.Register("add-mapping", AddMapping)
	sh.Register("del-mapping", DelMapping)
	sh.Register("upd-mapping", UpdMapping)
	sh.Register("open-mapping", OpenMapping)
	sh.Register("close-mapping", CloseMapping)
	sh.Register("heart", Heart)
}

func OpenMapping(args []string) {
	if len(args) != 1 {
		fmt.Printf("无效的参数 '%s'，正确格式为：open-mapping [映射名称]\n", args)
		return
	}
	name := args[0]
	server := shellInstance.server

	if !server.IsRun {
		ok := server.Sessions.UpdateEnable(name, true)
		if !ok {
			fmt.Printf("规则 '%s' 不存在\n", name)
			return
		}
		mapping := server.Sessions.GetMapping(name)
		updateMap, err := models.UpdateMap(server.DB, mapping.Name, mapping.PublicPort, mapping.TargetAddr, mapping.Enable, mapping.RateLimit)
		if err != nil {
			server.ErrorLog.Println("OpenMapping() 修改规则失败", err)
			return
		}
		server.ErrorLog.Printf("OpenMapping() 修改 '%s' 成功", updateMap.Name)
	} else {
		ok := server.Sessions.UpdateEnable(name, true)
		if !ok {
			fmt.Printf("规则 '%s' 不存在\n", name)
			return
		}
		mapping := server.Sessions.GetMapping(name)
		mapping.Ctx, mapping.CtxCancel = context.WithCancel(context.Background())
		go proxy.StartPublicListener(server.Ctx, mapping, server)
	}
}

func CloseMapping(args []string) {
	if len(args) != 1 {
		fmt.Printf("无效的参数 '%s'，正确格式为：close-mapping [映射名称]\n", args)
		return
	}
	name := args[0]
	server := shellInstance.server

	err := server.Sessions.CloseMapping(name)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	server.Sessions.UpdateEnable(name, false)
	fmt.Printf("关闭 '%s' 成功\n", name)
}

func start(args []string) {
	server := shellInstance.server
	if server.IsRun {
		fmt.Println("服务已启动")
		return
	}
	if !tlsgen.CheckServerCertExist("certs") {
		fmt.Println("证书缺失，请先生成证书")
		return
	}

	// 注意这里不再重新生成全局 ctx
	tunnelCtrl := tunnel.NewController(server)
	go tunnelCtrl.Start()
	server.SetRunStatus(true)
}

func stop(args []string) {
	server := shellInstance.server
	if !server.IsRun {
		fmt.Println("服务未启动")
		return
	}
	if server.Cancel != nil {
		server.Cancel()
	}
	server.SetRunStatus(false)
}

func generateCA(args []string) {
	year := 10
	overwrite := false
	length := len(args)
	server := shellInstance.server

	if length > 2 {
		fmt.Printf("无效的参数 '%s'，正确格式为：gen-ca [有效期] [-overwrite]\n", args)
		return
	}
	if length == 1 {
		input := args[0]
		if input == "-overwrite" {
			overwrite = true
		} else if d, err := strconv.Atoi(input); err == nil {
			if d <= 0 {
				fmt.Printf("无效的有效期参数 '%s'，请传入正整数，例如：gen-ca 10\n", input)
				return
			}
			year = d
		} else {
			fmt.Printf("无效的参数 '%s'，正确格式为：gen-ca [正整数] [-overwrite]\n", input)
			return
		}
	}
	if length == 2 {
		if d, err := strconv.Atoi(args[0]); err == nil {
			if d <= 0 {
				fmt.Printf("无效的参数 '%s'，正确格式为：gen-ca [正整数] [-overwrite]\n", args[0])
				return
			}
			year = d
		} else {
			fmt.Printf("无效的参数 '%s'，正确格式为：gen-ca [正整数] [-overwrite]\n", args[0])
			return
		}
		if args[1] != "-overwrite" {
			fmt.Printf("无效的参数 '%s'，正确格式为：gen-ca [正整数] [-overwrite]\n", args[1])
			return
		} else {
			overwrite = true
		}
	}
	if overwrite {
		for {
			fmt.Printf("确定要重新生成 CA 证书吗？(y/n) \n")
			readline, err := shellInstance.Rl.Readline()
			if err != nil {
				server.ErrorLog.Printf("generateCA() shellcmd.Rl.Readline() 读取输入失败: %v", err)
				fmt.Println("读取输入失败:", err)
				return
			}
			if readline == "n" {
				fmt.Println("已取消重新生成 CA 证书")
				return
			} else if readline == "y" {
				break
			} else {
				fmt.Println("无效的输入，请输入 'y' 或 'n'")
				continue
			}
		}
	}
	err := tlsgen.GenerateCA("certs", year, overwrite)
	if err != nil {
		server.ErrorLog.Printf("generateCA() 生成 CA 证书失败: %v", err)
		log.Println("generateCA() 生成 CA 证书失败:", err)
		return
	}
}

func generateCerts(args []string) {
	day := 365
	length := len(args)
	server := shellInstance.server

	if length > 1 {
		fmt.Printf("无效的参数 '%s'，正确格式为：gen-certs [正整数]\n", args)
		return
	}
	if len(args) == 1 {
		d, err := strconv.Atoi(args[0])
		if err != nil || d <= 0 {
			fmt.Printf("无效的参数 '%s'，正确格式为：gen-certs [正整数]\n", args[0])
			return
		}
		day = d
	}

	err := tlsgen.GenerateServerAndClientCerts(server.Config.ServerIP, "certs", day, "certs/ca.crt", "certs/ca.key")
	if err != nil {
		server.ErrorLog.Printf("generateCerts() 生成证书失败: %v", err)
		log.Println("generateCerts() 生成证书失败:", err)
		return
	}
}

func showConfig(args []string) {
	server := shellInstance.server
	fmt.Println(" IP         ：", server.Config.ServerIP)
	fmt.Println(" ListenPort ：", server.Config.ListenPort)
	fmt.Println(" Email      ：", server.Config.Email)
}

func showMapping(args []string) {
	server := shellInstance.server
	mpg := server.Sessions.GetAllMappings()

	fmt.Println("Name\tPublicPort\tTargetAddr\t\tStatus\t\tEnable\t\tTraffic\t\tRateLimit")

	for _, v := range mpg {
		fmt.Println(v.Name, "\t", v.PublicPort, "\t\t", v.TargetAddr, "\t", v.Status, "\t", v.Enable, "\t\t", v.Traffic, "\t\t", v.RateLimit)
	}
}

func setIP(args []string) {
	length := len(args)
	server := shellInstance.server

	if length == 0 {
		fmt.Println("参数不能为空，正确格式为：set-ip <ip>")
		return
	}
	if length != 1 {
		fmt.Printf("无效的参数 '%s'，正确格式为：set-ip <ip>\n", args)
		return
	}
	ip := args[0]
	if ip == "" {
		fmt.Println("IP地址不能为空")
		return
	}
	if !utils.IsValidateIP(ip) {
		fmt.Println("IP地址格式不正确")
		return
	}
	server.Config.ServerIP = ip
	err := models.UpdateCfg(server.DB, "server_ip", ip)
	if err != nil {
		server.ErrorLog.Printf("setIP() 更新配置数据失败: %v", err)
		fmt.Println("更新配置数据失败:", err)
		return
	}
}

func setPort(args []string) {
	length := len(args)
	server := shellInstance.server

	if length != 1 {
		fmt.Printf("无效的参数 '%s'，正确格式为：set-port <port>\n", args)
		return
	}

	port, err := strconv.Atoi(args[0])
	if err != nil || port <= 0 || port > 65535 {
		fmt.Printf("无效的参数 '%s'，参数必须是1-65535范围内的数字！\n", args)
		return
	}

	server.Config.ListenPort = args[0]
	err = models.UpdateCfg(server.DB, "listen_port", args[0])
	if err != nil {
		return
	}
}

func setEmail(args []string) {
	length := len(args)
	server := shellInstance.server

	if length != 1 {
		fmt.Printf("无效的参数 '%s'，正确格式为：set-email <email>\n", args)
		return
	}

	if args[0] == "" {
		fmt.Println("Email地址不能为空")
		return
	}

	if !utils.IsValidateEmail(args[0]) {
		fmt.Println("Email地址格式不正确")
		return
	}

	server.Config.Email = args[0]
	err := models.UpdateCfg(server.DB, "email", args[0])
	if err != nil {
		server.ErrorLog.Printf("setEmail() 更新配置数据失败: %v", err)
		fmt.Println("更新配置数据失败:", err)
		return
	}
}

func AddMapping(args []string) {
	length := len(args)
	server := shellInstance.server

	if length != 3 {
		fmt.Printf("无效的参数 '%s'，正确格式为：add-mapping <name> <public_port> <target_addr>\n", args)
		return
	}

	if args[0] == "" || args[1] == "" || args[2] == "" {
		fmt.Println("参数缺失!，正确格式为：add-mapping <name> <public_port> <target_addr>")
		return
	}

	port, err := strconv.Atoi(args[1])
	if err != nil || port <= 0 || port > 65535 {
		fmt.Printf("无效的参数 '%s'，参数必须是1-65535范围内的数字！\n", args)
		return
	}

	if !utils.IsValidateAddr(args[2]) {
		fmt.Println("目标地址格式不正确")
		return
	}

	err = models.InsertMpg(server.DB, models.Mapping{
		Name:       args[0],
		PublicPort: args[1],
		TargetAddr: args[2],
		Enable:     false,
		Traffic:    0,
		RateLimit:  1024 * 1024 * 2,
	})
	if err != nil {
		server.ErrorLog.Printf("addMapping() 插入映射数据失败: %v", err)
		fmt.Println("插入映射数据失败:", err)
		return
	}
	server.Sessions.SetMapping(args[0], args[1], args[2], false, 0, 2048)
}

func DelMapping(args []string) {
	if len(args) != 1 {
		fmt.Printf("无效的参数 '%s'，正确格式为：del-mapping <name>\n", args)
	}

	if args[0] == "" {
		fmt.Println("参数缺失!，正确格式为：del-mapping <name>")
		return
	}
	server := shellInstance.server
	mpg := server.Sessions.GetMapping(args[0])
	if mpg == nil {
		fmt.Println("映射不存在，请检查name是否正确")
		return
	}

	if mpg.Status == "active" {
		fmt.Println("当前映射正在运行中，无法删除，请关闭后重试")
		return
	}

	server.Sessions.DeleteMapping(mpg.Name)

	err := models.DeleteMapByName(server.DB, args[0])
	if err != nil {
		server.ErrorLog.Printf("delMapping() 删除映射数据失败: %v", err)
		fmt.Println("删除映射数据失败:", err)
		return
	}
}

func UpdMapping(args []string) {
	if len(args) != 4 {
		fmt.Printf("无效的参数 '%s'，正确格式为：upd-mapping <name> <port> <addr> <rate_limit>\n", args)
		return
	}

	if args[0] == "" {
		fmt.Println(" name 不能为空！")
		return
	}

	server := shellInstance.server

	if server.Sessions.GetMapping(args[0]).Status == "active" {
		fmt.Println("当前映射正在运行中，无法更新，请关闭后重试")
		return
	}

	port, err := strconv.Atoi(args[1])
	if err != nil || port <= 0 || port > 65535 {
		fmt.Printf("无效的参数 '%s'，参数必须是1-65535范围内的数字！\n", args)
		return
	}

	if !utils.IsValidateAddr(args[2]) {
		fmt.Println("目标地址格式不正确")
		return
	}

	retaLimit, err := strconv.Atoi(args[3])
	if err != nil {
		fmt.Printf("无效的参数 '%s'，参数必须是数字！\n", args)
		return
	}

	enable := server.Sessions.GetMapping(args[0]).Enable

	_, err = models.UpdateMap(server.DB, args[0], args[1], args[2], enable, int64(retaLimit))
	if err != nil {
		server.ErrorLog.Printf("updMapping() 更新映射数据失败: %v", err)
		fmt.Println("更新映射数据失败:", err)
		return
	}
	server.Sessions.UpdateMapping(args[0], args[1], args[2], int64(retaLimit))
}

func Heart(args []string) {
	fmt.Println("心跳机制已重构为基于事件总线的异步报警模式，日志详见错误日志。")
}
