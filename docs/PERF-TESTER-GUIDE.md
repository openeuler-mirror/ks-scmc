# ks-scmc 性能分析功能测试使用文档（测试同学）

本文档用于验证最新提交引入的性能分析能力（`perf`），面向测试同学快速执行与验收。

## 1. 本次功能点（测什么）

最新提交新增了以下能力：

- 可选开启 `pprof` HTTP 服务（默认监听 `127.0.0.1:6060`）
- 新增两个排查端点：
  - `/debug/leak/heap`：落盘 heap profile（`.pprof`）
  - `/debug/leak/goroutine`：落盘 goroutine 栈（`.txt`）
- 支持发送 `SIGUSR1` 触发 heap dump 落盘
- 支持定时输出内存统计日志（可配置）

代码入口：

- `cmd/ks-scmc-server/main.go`：服务启动时调用 `common.StartPerfTools(mode)`
- `common/perf.go`：核心实现
- `scripts/etc/server.toml`：新增 `[perf]` 配置段

---

