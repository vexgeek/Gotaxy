package tunnel

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"os"
	"time"

	"github/JustGopher/Gotaxy/internal/core"
	"github/JustGopher/Gotaxy/internal/eventbus"
	"github/JustGopher/Gotaxy/internal/tunnel/proxy"

	"github.com/xtaci/smux"
)

// Controller 隧道调度器
type Controller struct {
	server *core.GotaxyServer
}

// NewController 实例化调度器
func NewController(server *core.GotaxyServer) *Controller {
	return &Controller{
		server: server,
	}
}

// Start 启动隧道控制服务
func (c *Controller) Start() {
	// 开启穿透端口监听
	allMappings := c.server.Sessions.GetAllMappings()
	for _, mapping := range allMappings {
		if mapping.Enable {
			// 让 mapping.Ctx 从 server.Ctx 派生，这样 server.Ctx 被取消时，所有的 mapping.Ctx 都会被自动取消
			mapping.Ctx, mapping.CtxCancel = context.WithCancel(c.server.Ctx)
			go proxy.StartPublicListener(c.server.Ctx, mapping, c.server)
		}
	}

	// 开启控制端口监听
	go c.waitControlConn()

	<-c.server.Ctx.Done()
	fmt.Println("收到退出信号，停止中...")

	// 主动关闭当前会话
	if session := c.server.Sessions.GetDefaultSession(); session != nil {
		_ = session.Close()
	}
}

// 不断接受控制连接
func (c *Controller) waitControlConn() {
	tlsCfg, err := c.LoadServerTLSConfig("certs/server.crt", "certs/server.key", "certs/ca.crt")
	if err != nil {
		c.server.ErrorLog.Println("waitControlConn() 加载 TLS 配置失败: ", err)
		panic("加载 TLS 配置失败: " + err.Error())
	}

	listener, err := tls.Listen("tcp", ":"+c.server.Config.ListenPort, tlsCfg)
	if err != nil {
		c.server.ErrorLog.Println("waitControlConn() 监听失败: ", err)
		panic("监听失败: " + err.Error())
	}
	fmt.Printf("控制端口监听 %s 端口中...\n", c.server.Config.ListenPort)

	go func() {
		<-c.server.Ctx.Done()
		fmt.Println("关闭控制连接监听")
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-c.server.Ctx.Done():
				return // 正常退出
			default:
				fmt.Println("控制连接接入失败:", err)
				c.server.ErrorLog.Println("控制连接接入失败:", err)
				continue
			}
		}

		session, err := smux.Server(conn, nil)
		if err != nil {
			fmt.Println("创建会话失败:", err)
			c.server.ErrorLog.Println("创建会话失败:", err)
			_ = conn.Close()
			continue
		}

		// 为客户端生成一个标识，这里简单使用 RemoteAddr
		clientID := conn.RemoteAddr().String()
		c.server.InfoLog.Println("会话建立成功, ClientID:", clientID)

		c.server.Sessions.SetSession(clientID, session)
		c.server.EventBus.Publish(eventbus.EventClientConnected, eventbus.ClientConnectedPayload{
			ClientID:   clientID,
			RemoteAddr: conn.RemoteAddr().String(),
		})

		go c.startHeartbeat(clientID, session, 5*time.Second)
	}
}

// LoadServerTLSConfig 加载服务端 TLS 配置（含双向认证）
func (c *Controller) LoadServerTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("加载服务端证书失败: %w", err)
	}

	caCertPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("读取 CA 文件失败: %w", err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("解析 CA 证书失败")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// startHeartbeat 启动心跳检测
func (c *Controller) startHeartbeat(clientID string, session *smux.Session, interval time.Duration) {
	failCount := 0
	for {
		time.Sleep(2 * time.Second)

		stream, err := session.OpenStream()
		if err != nil {
			c.server.ErrorLog.Println("heartbeat: OpenStream失败:", err)
			failCount++
		} else {
			_ = stream.SetDeadline(time.Now().Add(5 * time.Second))
			_, err = stream.Write([]byte("HEARTBEAT\nPING\n"))
			if err != nil {
				c.server.ErrorLog.Println("heartbeat: 写入失败:", err)
				failCount++
				_ = stream.Close()
			} else {
				buffer := make([]byte, 4)
				_, err = io.ReadFull(stream, buffer)
				if err != nil || string(buffer) != "PONG" {
					c.server.ErrorLog.Println("heartbeat: 读失败或未收到PONG:", err)
					failCount++
				} else {
					// 成功
					failCount = 0
				}
				_ = stream.Close()
			}
		}

		if failCount >= 3 {
			c.server.ErrorLog.Printf("客户端 %s 心跳连续失败 3 次，断开连接\n", clientID)
			c.server.Sessions.RemoveSession(clientID)
			_ = session.Close()

			// 触发断开事件
			c.server.EventBus.Publish(eventbus.EventClientDisconnected, eventbus.ClientDisconnectedPayload{
				ClientID: clientID,
				Reason:   "心跳超时",
			})
			return
		}

		time.Sleep(interval)
	}
}
