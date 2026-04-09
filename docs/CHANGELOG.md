# 版本日志

> 版本日志辅助工具： https://github.com/git-chglog/git-chglog

## [v0.3.0] - 2026-04-10
### 💥 Breaking Changes (破坏性更新)
- **核心重构**: 彻底移除 `internal/global` 全局变量状态，改为实例化依赖注入 (Dependency Injection) 的 `GotaxyServer`。
- **架构升级**: 从单客户端限制重构为支持多租户/多节点同时接入的 `SessionManager` 会话管理器。
- **目录重构**: 废除并重组大量遗留目录，如将客户端核心从 `tunnel/clientCore` 完全独立至 `internal/client`。

### ✨ Features (新特性)
- **事件驱动架构**: 引入全新的 `internal/eventbus` 内部事件总线 (Pub/Sub 机制)，实现控制流与数据流的完美解耦。
- **插件化报警**: 新增 `internal/alert` 报警路由插件。现通过监听 `EventClientDisconnected` 事件，在客户端心跳超时或掉线时自动触发无阻塞邮件报警。

### ♻️ Refactoring (重构)
- **心跳机制**: 废弃难以维护的 `heart.HeartbeatRing` 环形队列机制，重构为基于事件派发的轻量级保活检测。
- **数据面分离**: 拆分 Tunnel 控制面调度与 Proxy 数据转发逻辑。
- **流量统计优化**: 流量计算不再侵入 Proxy 主逻辑，改为定时抛出 `EventTrafficReport` 异步事件交由外层累加，极大提升转发吞吐。
- **CLI 终端适配**: 终端命令全面改为调用 `GotaxyServer` 的标准 API，杜绝直接操作 DB。

