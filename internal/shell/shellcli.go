package shell

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github/JustGopher/Gotaxy/internal/core"

	"github.com/chzyer/readline"
)

type Shell struct {
	Rl       *readline.Instance
	server   *core.GotaxyServer
	commands map[string]func(args []string)
}

func New(server *core.GotaxyServer) *Shell {
	return &Shell{
		server:   server,
		commands: make(map[string]func(args []string)),
	}
}

func (s *Shell) Register(cmd string, handler func(args []string)) {
	s.commands[cmd] = handler
}

func (s *Shell) Run() {
	completer := s.buildCompleter()
	rl, err := readline.NewEx(&readline.Config{
		Prompt:              "\033[31m»\033[0m ",
		HistoryFile:         "/tmp/readline.tmp",
		AutoComplete:        completer,
		InterruptPrompt:     "^C",
		EOFPrompt:           "exit",
		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	})
	if err != nil {
		panic(err)
	}
	defer func(rl *readline.Instance) {
		err := rl.Close()
		if err != nil {
			return
		}
	}(rl)
	s.Rl = rl
	rl.CaptureExitSignal()
	log.SetOutput(rl.Stderr())

	setPasswordCfg := rl.GenPasswordConfig()
	setPasswordCfg.SetListener(func(line []rune, pos int, key rune) (newLine []rune, newPos int, ok bool) {
		rl.SetPrompt(fmt.Sprintf("Enter password(%v): ", len(line)))
		rl.Refresh()
		return nil, 0, false
	})

	for {
		line, err := rl.Readline()
		if errors.Is(err, readline.ErrInterrupt) {
			if len(line) == 0 {
				break
			}
			continue
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		isExit := false

		// 固定命令
		switch {
		case strings.HasPrefix(line, "mode "):
			switch line[5:] {
			case "vi":
				rl.SetVimMode(true)
			case "emacs":
				rl.SetVimMode(false)
			default:
				fmt.Println("invalid mode:", line[5:])
			}
			continue
		case line == "mode":
			if rl.IsVimMode() {
				fmt.Println("current mode: vim")
			} else {
				fmt.Println("current mode: emacs")
			}
			continue
		case line == "help":
			s.printHelpDoc()
			continue
		case line == "exit":
			if s.server.Cancel != nil {
				s.server.Cancel()
			}
			time.Sleep(time.Second)
			isExit = true
		}
		if isExit {
			break
		}
		// 自定义命令分发
		parts := strings.Fields(line)
		cmd, args := parts[0], parts[1:]

		if handler, ok := s.commands[cmd]; ok {
			handler(args)
		} else {
			log.Println("Unknown command:", strconv.Quote(line))
		}
	}
}

func (s *Shell) buildCompleter() *readline.PrefixCompleter {
	items := []readline.PrefixCompleterInterface{
		readline.PcItem("mode", readline.PcItem("vi"), readline.PcItem("emacs")),
		readline.PcItem("exit"),
		readline.PcItem("help"),
	}
	for name := range s.commands {
		items = append(items, readline.PcItem(name))
	}
	return readline.NewPrefixCompleter(items...)
}

// printHelpDoc 打印命令帮助文档
// nolint:funlen
func (s *Shell) printHelpDoc() {
	helpDoc := `命令帮助文档:
  gen-ca [time(year)] [-overwrite]
	有效期: 可选参数，指定CA证书的有效期，默认为10年
	-overwrite: 可选参数，强制覆盖已存在的证书
	示例: gen-ca 5 -overwrite  (生成有效期为5年的CA证书并覆盖已有证书)

  gen-certs [time(day)]
	有效期: 可选参数，指定证书的有效期(天)，默认为365天
	示例: gen-certs 30  (生成有效期为30天的证书)

  start 
	功能: 启动服务器，会检查证书是否存在

  stop 
	功能: 停止运行中的服务器

  show-config 
	功能: 显示当前服务器IP、监听端口和邮箱配置

  show-mapping 
	功能: 显示所有配置的端口映射及其状态

  set-ip <ip>
	功能: 设置服务端IP地址
	示例: set-ip 192.168.1.100

  set-port <port>
	功能: 设置服务端监听端口，范围为1-65535
	示例: set-port 9000

  set-email <email>
	功能: 设置服务端邮箱地址，用于接收通知
	示例: set-email admin@example.com

  add-mapping <name> <public_port> <target_addr>
	功能: 添加一个新的端口映射配置
	示例: add-mapping web 8080 127.0.0.1:3000

  del-mapping <name>
	功能: 删除指定名称的端口映射
	示例: del-mapping web

  upd-mapping <name> <public_port> <target_addr> <rate>
	功能: 更新指定名称的端口映射配置
	示例: upd-mapping web 8080 127.0.0.1:3000 2,097,152(2MB)

  open-mapping <name>
	功能: 打开指定名称的端口映射	
	示例: open-mapping web

  close-mapping <name>
	功能: 关闭指定名称的端口映射	
	示例: close-mapping web

  heart
	功能: 查看当前链接状态

  mode [vi|emacs]
	功能: 设置命令行编辑模式
	示例: mode vi  (切换到vi模式)

  help 
	功能: 显示此帮助信息

  exit 
	功能: 停止服务并退出命令行界面`

	fmt.Println(helpDoc)
}

func filterInput(r rune) (rune, bool) {
	switch r {
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}
