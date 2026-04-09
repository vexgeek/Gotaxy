package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github/JustGopher/Gotaxy/internal/core"
	"github/JustGopher/Gotaxy/internal/eventbus"
	"github/JustGopher/Gotaxy/internal/session"

	"golang.org/x/time/rate"
)

// StartPublicListener 持续监听公网端口流量，建立 stream 连接
// status 只在此函数中更新
func StartPublicListener(ctx context.Context, mapping *session.Mapping, server *core.GotaxyServer) {
	pubPort := mapping.PublicPort
	target := mapping.TargetAddr
	listener, err := net.Listen("tcp", ":"+pubPort)
	if err != nil {
		if strings.Contains(err.Error(), "address already in use") {
			fmt.Printf("端口 %s 已被占用\n", pubPort)
			return
		}
		fmt.Printf("监听端口 %s 失败: %v\n", pubPort, err)
		return
	}
	log.Printf("监听端口 %s 映射到客户端 %s\n", pubPort, target)

	mapping.Status = "active"
	defer func() {
		mapping.Status = "inactive"
	}()

	// 初始化限流器，每秒 rateLimit 字节
	rateLimit := int(mapping.RateLimit)
	limiter := rate.NewLimiter(rate.Limit(rateLimit), rateLimit)

	go func() {
		select {
		case <-ctx.Done():
			server.InfoLog.Println("关闭端口监听 :", pubPort)
			fmt.Printf("关闭端口监听 :%s\n", pubPort)
			_ = listener.Close()
			return
		case <-mapping.Ctx.Done():
			server.InfoLog.Println("关闭端口监听 :", pubPort)
			fmt.Printf("关闭端口监听 :%s\n", pubPort)
			_ = listener.Close()
			return
		}
	}()

	for {
		if !mapping.Enable {
			_ = listener.Close()
			return
		}
		publicConn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-mapping.Ctx.Done():
				return
			default:
				server.ErrorLog.Println("listener.Accept() 连接失败:", err)
				fmt.Printf("连接失败: %v", err)
				continue
			}
		}

		sess := server.Sessions.GetDefaultSession()
		if sess == nil {
			fmt.Println("无有效客户端连接，关闭连接")
			_ = publicConn.Close()
			continue
		}

		stream, err := sess.OpenStream()
		if err != nil {
			server.ErrorLog.Println("session.OpenStream() smux stream创建失败: ", err)
			fmt.Printf("smux stream 创建失败: %v", err)
			_ = publicConn.Close()
			continue
		}

		_, err = stream.Write([]byte("DIRECT\n" + target + "\n"))
		if err != nil {
			server.ErrorLog.Println("写入目标地址失败:", err)
			_ = publicConn.Close()
			_ = stream.Close()
			continue
		}

		fmt.Printf("建立转发: 端口 %s <=> 客户端本地 %s\n", pubPort, target)
		go rateLimitedProxy(mapping.Ctx, server.Ctx, publicConn, stream, limiter, nil, server)
		go rateLimitedProxy(mapping.Ctx, server.Ctx, stream, publicConn, limiter, mapping, server)
	}
}

// rateLimitedProxy 使用 rate.Limiter 限制速率的代理
func rateLimitedProxy(mappingCtx context.Context, serverCtx context.Context, dst, src net.Conn, limiter *rate.Limiter, mapping *session.Mapping, server *core.GotaxyServer) {
	defer func(dst net.Conn) {
		_ = dst.Close()
	}(dst)
	defer func(src net.Conn) {
		_ = src.Close()
	}(src)

	buf := make([]byte, 1024*512) // 缓存大小，512kb

	for {
		select {
		case <-mappingCtx.Done():
			return
		case <-serverCtx.Done():
			return
		default:
		}

		n, err := src.Read(buf)
		if err != nil {
			if err != io.EOF {
				server.ErrorLog.Println("数据读取失败:", err)
			}
			break
		}

		err = limiter.WaitN(mappingCtx, n)
		if err != nil {
			server.ErrorLog.Println("限流失败:", err)
			break
		}

		_, err = dst.Write(buf[:n])
		if err != nil {
			server.ErrorLog.Println("数据写入失败:", err)
			break
		}

		if mapping != nil {
			// 发布流量统计事件
			server.EventBus.Publish(eventbus.EventTrafficReport, eventbus.TrafficReportPayload{
				MappingName: mapping.Name,
				Bytes:       int64(n),
			})
		}
	}
}
