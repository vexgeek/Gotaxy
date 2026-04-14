package web

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github/JustGopher/Gotaxy/internal/core"
	"github/JustGopher/Gotaxy/internal/storage/models"
	"github/JustGopher/Gotaxy/internal/tunnel"
	"github/JustGopher/Gotaxy/internal/tunnel/proxy"
	"github/JustGopher/Gotaxy/pkg/tlsgen"
	"github/JustGopher/Gotaxy/pkg/utils"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*
var tmplFS embed.FS

type WebServer struct {
	server *core.GotaxyServer
}

func NewWebServer(server *core.GotaxyServer) *WebServer {
	return &WebServer{server: server}
}

func (w *WebServer) Start(port string) {
	gin.SetMode(gin.ReleaseMode)

	// 使用 New() 而不是 Default() 来避免默认的 Logger 中间件
	r := gin.New()

	// 自定义 Logger 中间件，静默掉所有状态码 < 400（即成功）的响应日志，只打印异常/错误请求
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		if param.StatusCode < 400 {
			return ""
		}

		// 对于异常或错误状态，使用默认格式打印详细信息
		return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %#v\n%s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
			param.ErrorMessage,
		)
	}))
	r.Use(gin.Recovery())

	// 加载嵌入的模板
	templ := template.Must(template.ParseFS(tmplFS, "templates/*.html"))
	r.SetHTMLTemplate(templ)

	// 注册路由
	w.registerRoutes(r)

	fmt.Printf("🌐 Web 面板已启动！请访问: http://localhost:%s\n", port)
	w.server.InfoLog.Printf("Web 面板监听在 :%s", port)

	err := r.Run(":" + port)
	if err != nil {
		w.server.ErrorLog.Println("Web 启动失败: ", err)
	}
}

func (w *WebServer) registerRoutes(r *gin.Engine) {
	// 页面渲染
	r.GET("/", w.indexHandler)

	// API 接口
	api := r.Group("/api")
	{
		api.GET("/status", w.statusHandler)
		api.POST("/start", w.startServiceHandler)
		api.POST("/stop", w.stopServiceHandler)
		api.GET("/config", w.getConfigHandler)
		api.POST("/config/update", w.updateConfigHandler)

		api.POST("/certs/ca", w.generateCAHandler)
		api.POST("/certs/server-client", w.generateCertsHandler)

		api.GET("/mappings", w.getMappingsHandler)
		api.POST("/mappings/add", w.addMappingHandler)
		api.POST("/mappings/delete", w.deleteMappingHandler)
		api.POST("/mappings/update", w.updateMappingHandler)
		api.POST("/mappings/toggle", w.toggleMappingHandler)
	}
}

// ---- Handlers ----

func (w *WebServer) indexHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Gotaxy 控制面板",
	})
}

func (w *WebServer) statusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"isRunning": w.server.IsRun,
	})
}

func (w *WebServer) startServiceHandler(c *gin.Context) {
	if w.server.IsRun {
		c.JSON(http.StatusBadRequest, gin.H{"error": "服务已在运行中"})
		return
	}

	if !tlsgen.CheckServerCertExist("certs") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "证书缺失，请先在【mTLS 证书管理】中生成证书"})
		return
	}

	// 重新生成 Context
	w.server.Ctx, w.server.Cancel = context.WithCancel(context.Background())

	tunnelCtrl := tunnel.NewController(w.server)
	go tunnelCtrl.Start()
	w.server.SetRunStatus(true)
	c.JSON(http.StatusOK, gin.H{"message": "服务启动成功"})
}

func (w *WebServer) stopServiceHandler(c *gin.Context) {
	if !w.server.IsRun {
		c.JSON(http.StatusBadRequest, gin.H{"error": "服务未运行"})
		return
	}

	if w.server.Cancel != nil {
		w.server.Cancel()
	}
	w.server.SetRunStatus(false)
	c.JSON(http.StatusOK, gin.H{"message": "服务已停止"})
}

func (w *WebServer) getConfigHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"serverIP":     w.server.Config.ServerIP,
		"listenPort":   w.server.Config.ListenPort,
		"email":        w.server.Config.Email,
		"totalTraffic": w.server.Config.TotalTraffic,
	})
}

func (w *WebServer) getMappingsHandler(c *gin.Context) {
	mappings := w.server.Sessions.GetAllMappings()
	c.JSON(http.StatusOK, mappings)
}

func (w *WebServer) toggleMappingHandler(c *gin.Context) {
	var req struct {
		Name   string `json:"name"`
		Enable bool   `json:"enable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if !w.server.IsRun && req.Enable {
		w.server.Sessions.UpdateEnable(req.Name, true)
		c.JSON(http.StatusOK, gin.H{"message": "状态已更新(服务未启动)"})
		return
	}

	if req.Enable {
		ok := w.server.Sessions.UpdateEnable(req.Name, true)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
			return
		}
		mapping := w.server.Sessions.GetMapping(req.Name)
		mapping.Ctx, mapping.CtxCancel = context.WithCancel(context.Background())
		go proxy.StartPublicListener(w.server.Ctx, mapping, w.server)
		c.JSON(http.StatusOK, gin.H{"message": "规则已开启"})
	} else {
		err := w.server.Sessions.CloseMapping(req.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		w.server.Sessions.UpdateEnable(req.Name, false)
		c.JSON(http.StatusOK, gin.H{"message": "规则已关闭"})
	}
}

func (w *WebServer) updateConfigHandler(c *gin.Context) {
	var req struct {
		ServerIP   string `json:"serverIP"`
		ListenPort string `json:"listenPort"`
		Email      string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败"})
		return
	}

	if req.ServerIP != "" && !utils.IsValidateIP(req.ServerIP) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "IP格式错误"})
		return
	}
	if req.Email != "" && !utils.IsValidateEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email格式错误"})
		return
	}
	if req.ListenPort != "" {
		port, err := strconv.Atoi(req.ListenPort)
		if err != nil || port <= 0 || port > 65535 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "端口格式错误"})
			return
		}
	}

	if req.ServerIP != "" {
		w.server.Config.ServerIP = req.ServerIP
		_ = models.UpdateCfg(w.server.DB, "server_ip", req.ServerIP)
	}
	if req.ListenPort != "" {
		w.server.Config.ListenPort = req.ListenPort
		_ = models.UpdateCfg(w.server.DB, "listen_port", req.ListenPort)
	}
	if req.Email != "" {
		w.server.Config.Email = req.Email
		_ = models.UpdateCfg(w.server.DB, "email", req.Email)
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置更新成功"})
}

func (w *WebServer) addMappingHandler(c *gin.Context) {
	var req struct {
		Name       string `json:"name"`
		PublicPort string `json:"publicPort"`
		TargetAddr string `json:"targetAddr"`
		RateLimit  int64  `json:"rateLimit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败"})
		return
	}

	if req.Name == "" || req.PublicPort == "" || req.TargetAddr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写完整参数"})
		return
	}
	if !utils.IsValidateAddr(req.TargetAddr) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "目标地址格式不正确"})
		return
	}

	if w.server.Sessions.GetMapping(req.Name) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则名已存在"})
		return
	}

	limit := req.RateLimit
	if limit == 0 {
		limit = 1024 * 1024 * 2 // 默认 2M
	}

	err := models.InsertMpg(w.server.DB, models.Mapping{
		Name:       req.Name,
		PublicPort: req.PublicPort,
		TargetAddr: req.TargetAddr,
		Enable:     false,
		Traffic:    0,
		RateLimit:  limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库保存失败"})
		return
	}

	w.server.Sessions.SetMapping(req.Name, req.PublicPort, req.TargetAddr, false, 0, limit)
	c.JSON(http.StatusOK, gin.H{"message": "规则添加成功"})
}

func (w *WebServer) deleteMappingHandler(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	mapping := w.server.Sessions.GetMapping(req.Name)
	if mapping == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}
	if mapping.Status == "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前映射正在运行中，请关闭后重试"})
		return
	}

	w.server.Sessions.DeleteMapping(req.Name)
	_ = models.DeleteMapByName(w.server.DB, req.Name)

	c.JSON(http.StatusOK, gin.H{"message": "规则已删除"})
}

func (w *WebServer) updateMappingHandler(c *gin.Context) {
	var req struct {
		Name       string `json:"name"`
		PublicPort string `json:"publicPort"`
		TargetAddr string `json:"targetAddr"`
		RateLimit  int64  `json:"rateLimit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	mapping := w.server.Sessions.GetMapping(req.Name)
	if mapping == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}
	if mapping.Status == "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前映射正在运行中，请关闭后重试"})
		return
	}

	if req.TargetAddr != "" && !utils.IsValidateAddr(req.TargetAddr) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "目标地址格式不正确"})
		return
	}

	_, err := models.UpdateMap(w.server.DB, req.Name, req.PublicPort, req.TargetAddr, mapping.Enable, req.RateLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库更新失败"})
		return
	}

	w.server.Sessions.UpdateMapping(req.Name, req.PublicPort, req.TargetAddr, req.RateLimit)
	c.JSON(http.StatusOK, gin.H{"message": "规则更新成功"})
}

func (w *WebServer) generateCAHandler(c *gin.Context) {
	var req struct {
		Year      int  `json:"year"`
		Overwrite bool `json:"overwrite"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.Year <= 0 {
		req.Year = 10 // 默认 10 年
	}

	err := tlsgen.GenerateCA("certs", req.Year, req.Overwrite)
	if err != nil {
		w.server.ErrorLog.Println("生成 CA 证书失败:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 CA 证书失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "CA 证书生成成功"})
}

func (w *WebServer) generateCertsHandler(c *gin.Context) {
	var req struct {
		Days int `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.Days <= 0 {
		req.Days = 365 // 默认 1 年
	}

	serverIP := w.server.Config.ServerIP
	if serverIP == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置 ServerIP"})
		return
	}

	err := tlsgen.GenerateServerAndClientCerts(serverIP, "certs", req.Days, "certs/ca.crt", "certs/ca.key")
	if err != nil {
		w.server.ErrorLog.Println("生成服务端/客户端证书失败:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成证书失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "服务端与客户端证书生成成功"})
}
