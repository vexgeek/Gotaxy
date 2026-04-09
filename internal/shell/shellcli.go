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
	return readline.NewPrefixCompleter(
		readline.PcItem("service",
			readline.PcItem("start"),
			readline.PcItem("stop"),
			readline.PcItem("status"),
		),
		readline.PcItem("config",
			readline.PcItem("show"),
			readline.PcItem("set",
				readline.PcItem("ip"),
				readline.PcItem("port"),
				readline.PcItem("email"),
			),
		),
		readline.PcItem("mapping",
			readline.PcItem("list"),
			readline.PcItem("add"),
			readline.PcItem("del"),
			readline.PcItem("upd"),
			readline.PcItem("start"),
			readline.PcItem("stop"),
		),
		readline.PcItem("cert",
			readline.PcItem("ca"),
			readline.PcItem("gen"),
		),
		readline.PcItem("mode",
			readline.PcItem("vi"),
			readline.PcItem("emacs"),
		),
		readline.PcItem("help"),
		readline.PcItem("exit"),
		readline.PcItem("clear"),
	)
}

// printHelpDoc 打印命令帮助文档
func (s *Shell) printHelpDoc() {
	helpDoc := `
================= Gotaxy 交互式终端帮助文档 =================

🛠️  服务控制 (Service)
  service start               启动内网穿透服务端
  service stop                停止运行中的服务
  service status              查看当前服务运行状态

⚙️  核心配置 (Config)
  config show                 显示当前所有配置信息
  config set ip <ip>          设置服务端公网IP (必须)
  config set port <port>      设置服务端控制监听端口 (如: 9000)
  config set email <email>    设置报警邮箱

🔌 映射规则 (Mapping)
  mapping list                                        列出所有映射规则
  mapping add <name> <pub_port> <target>              添加新规则 (例: mapping add web 8080 127.0.0.1:3000)
  mapping del <name>                                  删除映射规则
  mapping upd <name> <pub_port> <target> <rate_limit> 更新映射规则 (包含限速)
  mapping start <name>                                启动指定的映射规则
  mapping stop <name>                                 停止指定的映射规则

🔐 证书管理 (Cert)
  cert ca [year] [-overwrite] 生成 CA 根证书 (默认 10 年)
  cert gen [days]             基于 CA 签发服务端与客户端 mTLS 证书 (默认 365 天)

💻 终端操作
  clear                       清屏
  mode [vi|emacs]             切换命令行编辑模式
  help                        显示本帮助信息
  exit                        停止所有服务并安全退出终端

=============================================================
`
	fmt.Println(helpDoc)
}

func filterInput(r rune) (rune, bool) {
	switch r {
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}
