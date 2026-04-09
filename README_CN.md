# Gotaxy

<img align="right" width="280px" src="docs/images/logo2.png" alt="Gotaxy Logo">

[English](README.md) | 简体中文

✈️ **Gotaxy** 是一款基于 Go 语言开发的高性能、事件驱动的内网穿透工具。它不仅帮助开发者将内网服务安全、便捷地暴露到公网，更在架构上支持了多租户接入、Web 控制面板与无缝的报警集成。

**_"Go beyond NAT, with style."_**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache-blue.svg)](LICENSE)
[![SQLite](https://img.shields.io/badge/SQLite-1.38-blue?logo=sqlite)](https://pkg.go.dev/modernc.org/sqlite#section-readme)
[![smux](https://img.shields.io/badge/xtaci%2Fsmux-1.5.34-brightgreen)](https://github.com/xtaci/smux)
[![readline](https://img.shields.io/badge/chzyer%2Freadline-1.5.1-orange)](https://github.com/chzyer/readline)
[![Stars](https://img.shields.io/github/stars/JustGopher/Gotaxy?style=social)](https://github.com/JustGopher/Gotaxy/stargazers)

## 🌟 核心特性
- **高度安全 (mTLS)**：内置自签名 CA 证书颁发机制，从“身份验证”到“数据加密”构筑完整安全链路。
- **事件驱动架构 (EDA)**：基于内存 EventBus 实现控制面与数据面的深度解耦，极大提升了网络转发吞吐。
- **双端管理 (Web & CLI)**：
  - **Web 控制面板**：内置基于 Alpine.js + Tailwind CSS 的现代化面板，一键管控配置、证书与映射规则。
  - **CLI 交互终端**：支持命令行多级指令、高亮输出及 Tab 键智能自动补全。
- **插件化报警**：支持心跳超时掉线自动触发邮件报警（扩展层机制）。
- **轻量且纯粹**：所有资源（包括 Web 模板）均打包入单一二进制文件，无任何外部依赖，开箱即用。

---

## 🚀 快速开始

### 1. 启动服务端与 Web 面板

建议直接通过源码运行，或者下载最新的编译后程序。

```bash
# 启动服务端 (同时会启动 Web 面板与 CLI 终端)
go run cmd/server/server.go
```

启动后，您可以选择通过 **浏览器** 或 **CLI 终端** 来进行配置：

👉 **选项 A：使用 Web 控制面板 (推荐)**
1. 打开浏览器访问 `http://localhost:8080/`
2. 在面板中修改**公网 IP** 与**监听端口**
3. 点击 **mTLS 证书管理**，生成 CA 并签发服务端与客户端证书
4. 添加一条映射规则（例如：映射内网 3306 端口）
5. 点击左上角 **启动服务**

👉 **选项 B：使用交互式 CLI 终端**
在终端中依次输入以下命令（支持 Tab 键补全）：
```bash
config set ip <你的公网IP>
config set port 9000
cert ca              # 生成 CA 根证书
cert gen 365         # 签发 365 天有效期的 TLS 证书
mapping add web 8080 127.0.0.1:3000  # 添加映射规则
mapping start web    # 开启该规则
service start        # 启动核心穿透服务
```

### 2. 客户端连接

将服务端生成的 `certs` 目录中的 `ca.crt`, `client.crt`, `client.key` 拷贝到客户端对应位置。

启动客户端建立加密隧道：
```bash
go run cmd/client/client.go -h <服务端IP> -p <服务端监听端口>
```

客户端参数说明：
- `-h` : 服务端的公网 IP (默认 127.0.0.1)
- `-p` : 服务端的控制监听端口 (默认 9000)
- `-ca` : CA 根证书路径 (默认 certs/ca.crt)
- `-crt` : 客户端证书路径 (默认 certs/client.crt)
- `-key` : 客户端私钥路径 (默认 certs/client.key)

---

## 💻 CLI 终端完整命令参考

```text
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
  mapping add <name> <pub_port> <target>              添加新规则
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
```

---

## 📚 架构设计与文档

深入了解 Gotaxy 的架构设计和重构历程，请参阅：
- [架构设计文档 (DESIGN.md)](docs/DESIGN.md)
- [版本更新日志 (CHANGELOG.md)](docs/CHANGELOG.md)
- [需求分析文档 (REQUIREMENTS.md)](docs/REQUIREMENTS.md)

---

## 🤝 提交贡献

欢迎提交 Issue 和 Pull Request！

- 贡献代码前请查阅 [CONTRIBUTING.md](docs/CONTRIBUTING.md)。
- 提交规范请阅读 [COMMIT_CONVENTION.md](docs/COMMIT_CONVENTION.md)，我们遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范。

<h3 align="left">贡献墙</h3>

<a href="https://github.com/JustGopher/Gotaxy/graphs/contributors">
<img src="https://contri.buzz/api/wall?repo=JustGopher/Gotaxy&onlyAvatars=true" alt="Contributors' Wall for JustGopher/Gotaxy" />
</a>
