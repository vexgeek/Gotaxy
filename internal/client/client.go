package client

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/xtaci/smux"
)

// Start 启动客户端
func Start(serverAddr, certFile, keyFile, caFile string) {
	log.Printf("🔌 正在准备连接服务端 [%s]...", serverAddr)
	log.Printf("🔐 正在加载 mTLS 证书 (ca: %s, crt: %s, key: %s)...", caFile, certFile, keyFile)

	tlsCfg, err := LoadClientTLSConfig(certFile, keyFile, caFile)
	if err != nil {
		log.Fatalf("❌ 加载 TLS 配置失败: %v", err)
	}

	log.Printf("🔗 正在建立 TLS 连接...")
	conn, err := tls.Dial("tcp", serverAddr, tlsCfg)
	if err != nil {
		log.Fatalf("❌ 连接服务端失败: %v\n(请检查网络、IP端口以及证书是否匹配)", err)
	}
	log.Println("✅ 已通过 TLS 安全连接到服务端")

	session, err := smux.Client(conn, nil)
	if err != nil {
		log.Fatalf("❌ 创建多路复用会话失败: %v", err)
	}
	log.Println("✅ 隧道会话 (smux) 创建成功，等待转发请求...")

	for {
		stream, err := session.AcceptStream()
		if err != nil {
			log.Println("⚠️ 接受数据流 (stream) 失败:", err)
			return
		}
		go handleStream(stream)
	}
}

// LoadClientTLSConfig 客户端 TLS 配置（支持 mTLS）
func LoadClientTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	// 客户端证书（client.crt + client.key）
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("加载客户端证书失败: %w", err)
	}

	// 加载 CA 根证书
	caCertPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("读取 CA 根证书失败: %w", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("解析 CA 根证书失败")
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caPool,
		InsecureSkipVerify: false,
	}, nil
}

// handleStream 处理每个 stream
func handleStream(stream *smux.Stream) {
	reader := bufio.NewReader(stream)
	header, err := reader.ReadString('\n')
	if err != nil {
		log.Println("⚠️ 读取指令头失败:", err)
		_ = stream.Close()
		return
	}
	switch strings.TrimSpace(header) {
	case "HEARTBEAT":
		payload, _ := reader.ReadString('\n')
		if strings.TrimSpace(payload) == "PING" {
			_, _ = stream.Write([]byte("PONG"))
		}
		_ = stream.Close()
	case "DIRECT":
		target, err := reader.ReadString('\n')
		if err != nil {
			log.Println("⚠️ 读取目标地址失败:", err)
			_ = stream.Close()
			return
		}
		target = strings.TrimSpace(target)
		log.Printf("⚡ 收到转发请求，目标本地服务: %s", target)
		handleForward(target, stream)
	default:
		log.Println("⚠️ 收到未知指令类型:", header)
		_ = stream.Close()
	}
}

func handleForward(target string, stream *smux.Stream) {
	localConn, err := net.Dial("tcp", target)
	if err != nil {
		log.Printf("❌ 连接本地服务 [%s] 失败: %v", target, err)
		_ = stream.Close()
		return
	}

	log.Printf("🔄 开始数据转发: 服务端 (%s) <=> 本地 (%s)", stream.RemoteAddr(), target)
	go proxy(stream, localConn)
	go proxy(localConn, stream)
}

// proxy 数据双向转发
func proxy(dst, src net.Conn) {
	defer func(dst net.Conn) {
		_ = dst.Close()
	}(dst)
	defer func(src net.Conn) {
		_ = src.Close()
	}(src)
	_, _ = io.Copy(dst, src)
}

// HelloServe 演示用的测试服务
func HelloServe() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		timeNow := time.Now().Format(time.UnixDate)
		timeNow = "现在时间: " + timeNow + "\n你好 Gotaxy\n" + "现在时间: " + time.Now().Format(time.UnixDate)
		fmt.Println(timeNow)
		_, _ = w.Write([]byte(timeNow))
	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
