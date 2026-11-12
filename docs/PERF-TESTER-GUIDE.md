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

## 2. 测试前准备

### 2.1 环境要求

- 测试机可启动 `ks-scmc-server`（systemd 方式）
- 可访问配置文件 `/etc/ks-scmc/server.toml`
- 有权限查看服务日志：`journalctl -u ks-scmc-server`
- 有权限发送进程信号：`kill -USR1 <pid>`

### 2.2 必要配置

编辑测试机上的 `/etc/ks-scmc/server.toml`，确认存在并启用 `[perf]`：

```toml
[perf]
enable = true
host = "127.0.0.1"
port = 6060

block_profile_rate = 0
mutex_profile_fraction = 0
memstats_log_interval_sec = 10

heap_dump_dir = "/tmp/ks-scmc-pprof"
```

说明：

- `enable = true` 才会启动 perf 功能
- `heap_dump_dir` 不能为空，否则 `/debug/leak/*` 与 `SIGUSR1` 会报错
- `memstats_log_interval_sec = 0` 表示关闭定时内存日志

---

## 3. 快速验证（建议先跑这一段）

### 步骤 1：重启服务

```bash
systemctl restart ks-scmc-server
systemctl status ks-scmc-server --no-pager
```

### 步骤 2：确认 perf 已启用

```bash
journalctl -u ks-scmc-server -n 50 --no-pager | grep -i "perf enabled"
```

期望：出现类似 `perf enabled: mode=<agent|controller> pprof_listen=127.0.0.1:6060`

### 步骤 3：验证 pprof 基础端点

```bash
curl -s http://127.0.0.1:6060/debug/pprof/ | head -20
curl -s http://127.0.0.1:6060/debug/pprof/cmdline
```

期望：有正常响应，不是连接失败/404。

### 步骤 4：验证 heap 落盘端点

```bash
mkdir -p /tmp/ks-scmc-pprof
curl -s http://127.0.0.1:6060/debug/leak/heap
ls -la /tmp/ks-scmc-pprof/
```

期望：

- 返回 JSON，包含 `"profile":"heap"`
- 目录下新增 `heap-*.pprof`

### 步骤 5：验证 goroutine 落盘端点

```bash
curl -s http://127.0.0.1:6060/debug/leak/goroutine
ls -la /tmp/ks-scmc-pprof/
```

期望：

- 返回 JSON，包含 `"profile":"goroutine"`
- 目录下新增 `goroutine-*.txt`

---

## 4. 信号触发验证（SIGUSR1）

```bash
PID=$(pgrep -f ks-scmc-server | head -1)
echo "$PID"

kill -USR1 "$PID"
sleep 2

journalctl -u ks-scmc-server -n 20 --no-pager | grep -i "heap dump"
ls -la /tmp/ks-scmc-pprof/
```

期望：

- 日志出现 `heap dump saved`
- 目录中新增一份新的 `heap-*.pprof`

---

## 5. 验收标准（通过/失败判定）

通过标准（全部满足）：

- 服务启动后能看到 `perf enabled` 日志
- `/debug/pprof/` 可访问
- `/debug/leak/heap` 返回 JSON 且产出 `.pprof` 文件
- `/debug/leak/goroutine` 返回 JSON 且产出 `.txt` 文件
- `SIGUSR1` 触发后日志与文件都符合预期

失败标准（任一命中）：

- 端口不通、404、连接被拒绝
- JSON 返回带错误信息（例如 `perf.heap_dump_dir is empty`）
- 信号触发后无日志、无文件
- 服务重启失败或异常退出

---

## 6. 常见问题与处理

### 6.1 访问 6060 失败

排查：

```bash
ss -lntp | grep 6060
journalctl -u ks-scmc-server -n 100 --no-pager
```

处理：

- 检查 `[perf] enable` 是否为 `true`
- 检查 `host/port` 是否配置正确且未被占用
- 重启服务后重新验证

### 6.2 `/debug/leak/heap` 返回错误

常见原因：`heap_dump_dir` 未配置或不可写。

处理：

```bash
mkdir -p /tmp/ks-scmc-pprof
chmod 755 /tmp/ks-scmc-pprof
```

并确认配置为：

