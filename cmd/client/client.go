package main

import (
	"flag"
	"fmt"
	"os"

	"github/JustGopher/Gotaxy/internal/client"
)

func main() {
	var (
		host     = flag.String("h", "127.0.0.1", "服务端的公网 IP 或域名")
		port     = flag.String("p", "9000", "服务端控制监听端口")
		caFile   = flag.String("ca", "certs/ca.crt", "CA 根证书路径")
		certFile = flag.String("crt", "certs/client.crt", "客户端证书路径")
		keyFile  = flag.String("key", "certs/client.key", "客户端私钥路径")
	)

	// 自定义帮助信息
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "🚀 Gotaxy Client - 安全的内网穿透客户端\n\n")
		fmt.Fprintf(os.Stderr, "用法:\n")
		fmt.Fprintf(os.Stderr, "  go run cmd/client/client.go -h <host> -p <port> [options]\n\n")
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// 打印欢迎信息
	fmt.Println("========================================")
	fmt.Println("🚀 Gotaxy Client - 启动中...")
	fmt.Println("========================================")

	serverAddr := fmt.Sprintf("%s:%s", *host, *port)
	client.Start(serverAddr, *certFile, *keyFile, *caFile)
}
