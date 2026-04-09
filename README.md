# Gotaxy

<img align="right" width="280px" src="docs/images/logo2.png" alt="Gotaxy Logo">

English | [简体中文](README_CN.md)

✈️ **Gotaxy** is a high-performance, event-driven reverse proxy and intranet penetration tool written in Go. It not only allows developers to securely and conveniently expose internal services to the public internet, but its architecture also supports multi-tenant connections, an elegant Web Control Panel, and seamless alerting integration.

**_"Go beyond NAT, with style."_**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache-blue.svg)](LICENSE)
[![SQLite](https://img.shields.io/badge/SQLite-1.38-blue?logo=sqlite)](https://pkg.go.dev/modernc.org/sqlite#section-readme)
[![smux](https://img.shields.io/badge/xtaci%2Fsmux-1.5.34-brightgreen)](https://github.com/xtaci/smux)
[![readline](https://img.shields.io/badge/chzyer%2Freadline-1.5.1-orange)](https://github.com/chzyer/readline)
[![Stars](https://img.shields.io/github/stars/JustGopher/Gotaxy?style=social)](https://github.com/JustGopher/Gotaxy/stargazers)

## 🌟 Key Features
- **Highly Secure (mTLS)**: Built-in self-signed CA certificate issuance mechanism, building a complete security chain from "authentication" to "data encryption".
- **Event-Driven Architecture (EDA)**: Deep decoupling of the Control Plane and Data Plane based on an in-memory EventBus, significantly boosting network forwarding throughput.
- **Dual Management Interfaces (Web & CLI)**:
  - **Web Control Panel**: A modern built-in dashboard powered by Alpine.js + Tailwind CSS, offering one-click control over configurations, certificates, and port mapping rules.
  - **Interactive CLI**: Supports multi-level commands, highlighted outputs, and intelligent Tab auto-completion.
- **Pluggable Alerting**: Supports automatic email alerts triggered by client heartbeat timeouts (Extension Layer mechanism).
- **Lightweight & Pure**: All resources (including Web templates) are bundled into a single binary. No external dependencies, completely out-of-the-box.

---

## 🚀 Quick Start

### 1. Start the Server and Web Panel

It is recommended to run directly from the source code or download the latest compiled release.

```bash
# Start the server (which automatically spins up the Web Panel and CLI terminal)
go run cmd/server/server.go
```

Once started, you can configure Gotaxy via the **Browser** or the **CLI Terminal**:

👉 **Option A: Using the Web Control Panel (Recommended)**
1. Open your browser and navigate to `http://localhost:8080/`.
2. Update the **Public IP** and **Listen Port** in the configuration panel.
3. Click on **mTLS Certificate Management** to generate a CA and issue server/client certificates.
4. Add a mapping rule (e.g., exposing an internal 3306 port).
5. Click **Start Service** in the top left corner.

👉 **Option B: Using the Interactive CLI Terminal**
Enter the following commands sequentially in your terminal (Tab completion supported):
```bash
config set ip <your_public_ip>
config set port 9000
cert ca              # Generate root CA certificate
cert gen 365         # Issue TLS certificates valid for 365 days
mapping add web 8080 127.0.0.1:3000  # Add a mapping rule
mapping start web    # Enable the rule
service start        # Start the core proxy service
```

### 2. Client Connection

Copy the `ca.crt`, `client.crt`, and `client.key` from the server's `certs` directory to the corresponding location on your client machine.

Start the client to establish an encrypted tunnel:
```bash
go run cmd/client/client.go -h <server_ip> -p <server_listen_port>
```

Client arguments:
- `-h` : The public IP of the server (Default: 127.0.0.1)
- `-p` : The control listen port of the server (Default: 9000)
- `-ca` : Path to the CA root certificate (Default: certs/ca.crt)
- `-crt` : Path to the client certificate (Default: certs/client.crt)
- `-key` : Path to the client private key (Default: certs/client.key)

---

## 💻 CLI Terminal Command Reference

```text
================= Gotaxy Interactive CLI Help =================

🛠️  Service Control
  service start               Start the proxy server
  service stop                Stop the running service
  service status              Check the current service status

⚙️  Core Configuration (Config)
  config show                 Display all current configurations
  config set ip <ip>          Set the server's public IP (Required)
  config set port <port>      Set the control listen port (e.g., 9000)
  config set email <email>    Set the alert email address

🔌 Mapping Rules
  mapping list                                        List all mapping rules
  mapping add <name> <pub_port> <target>              Add a new rule
  mapping del <name>                                  Delete a rule
  mapping upd <name> <pub_port> <target> <rate_limit> Update a rule (including rate limit)
  mapping start <name>                                Start a specific rule
  mapping stop <name>                                 Stop a specific rule

🔐 Certificate Management (Cert)
  cert ca [year] [-overwrite] Generate a root CA certificate (Default: 10 years)
  cert gen [days]             Issue server & client mTLS certificates (Default: 365 days)

💻 Terminal Operations
  clear                       Clear the screen
  mode [vi|emacs]             Switch command-line editing mode
  help                        Display this help message
  exit                        Stop all services and exit the terminal safely
===============================================================
```

---

## 📚 Architecture & Documentation

To dive deeper into Gotaxy's architectural design and refactoring journey, please refer to:
- [Architecture Design (DESIGN.md)](docs/DESIGN.md)
- [Changelog (CHANGELOG.md)](docs/CHANGELOG.md)
- [Requirements (REQUIREMENTS.md)](docs/REQUIREMENTS.md)

---

## 🤝 Contributing

Issues and Pull Requests are highly welcome!

- Please review [CONTRIBUTING.md](docs/CONTRIBUTING.md) before contributing code.
- Read [COMMIT_CONVENTION.md](docs/COMMIT_CONVENTION.md) for commit message guidelines. We strictly follow the [Conventional Commits](https://www.conventionalcommits.org/) specification.

<h3 align="left">Contributors Wall</h3>

<a href="https://github.com/JustGopher/Gotaxy/graphs/contributors">
<img src="https://contri.buzz/api/wall?repo=JustGopher/Gotaxy&onlyAvatars=true" alt="Contributors' Wall for JustGopher/Gotaxy" />
</a>
