package main

import (
	"fmt"
	"os"
	"strings"

	"github/JustGopher/Gotaxy/internal/client"
)

func main() {
	// 预设默认值
	flag := map[string]string{
		"-h":   "127.0.0.1",
		"-p":   "9000",
		"-ca":  "certs/ca.crt",
		"-crt": "certs/client.crt",
		"-key": "certs/client.key",
	}

	for i, arg := range os.Args {
		switch arg {
		case "--help", "-help":
			showHelp()
			os.Exit(0)
		case "-h", "-p", "-ca", "-crt", "-key":
			if i+1 >= len(os.Args) {
				fmt.Println("错误：参数后面缺少值")
				os.Exit(1)
			}
			nextArg := os.Args[i+1]
			if strings.HasPrefix(nextArg, "-") {
				fmt.Println("错误：参数后面缺少值，不能是另一个参数")
				os.Exit(1)
			}
			flag[arg] = nextArg
		}
	}

	serverAddr := flag["-h"] + ":" + flag["-p"]
	client.Start(serverAddr, flag["-crt"], flag["-key"], flag["-ca"])
}

// 显示帮助信息
func showHelp() {
	fmt.Println(`Usage:
  go run cmd/client/client.go -h [host] -p <port> [-ca <ca-cert-path>] [-crt <client-cert-path>] [-key <private-key-path>]

Options:
  -h [host]     
        The hostname or IP address of the server (default "127.0.0.1")
  -p <port>
        The port number to connect to (default 9000)
  -ca <ca-cert-path>
        Path to the CA certificate file (default "certs/ca.crt")
  -crt <client-cert-path>
        Path to the client certificate file (default "certs/client.crt")
  -key <private-key-path>
        Path to the client private key file (default "certs/client.key")`)
}
