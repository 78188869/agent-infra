# Local Persistent Logging Design

> **Issue**: #33 - 本地持久化日志
> **Date**: 2026-03-28
> **Status**: Approved

---

## Background

本地开发环境下，结构化日志仅输出到 stdout，进程结束后日志丢失。调试任务执行流程（状态变更、工具调用、错误信息）时无法回溯历史。需要在无外部日志服务的情况下，将日志持久化到本地文件。

## Scope

- 业务执行日志（状态变更、工具调用、LLM 输入输出、错误等 EventType 相关）
- HTTP 请求/响应日志
- 不影响生产环境的 SLS 日志行为

## Design Decisions

### 1. 统一日志库：slog

选择 Go 1.21 标准库 `log/slog`，无外部依赖，与现有代码一致。

### 2. 自定义 slog Handler + 文件输出

实现 `slog.Handler` 接口的 `MultiOutputHandler`，支持同时输出到 stdout 和本地 JSONL 文件。

- 与现有 slog 代码完全兼容，无需改动业务代码
- JSONL 格式，可用 grep/jq 直接查询
- 本地/生产切换只需改配置

### 3. 按类型分文件

- `logs/business-YYYY-MM-DD.jsonl` — 业务执行日志
- `logs/http-YYYY-MM-DD.jsonl` — HTTP 请求/响应日志

通过 record 的 group 前缀（`business.*` / `http.*`）分发到对应文件。

### 4. 环境切换

通过 `AppConfig.IsLocal()` 判断环境：
- **本地开发**：`config.local.yaml` 配置 `outputs: both`，同时写 stdout + file
- **生产环境**：`config.yaml` 不配置 `outputs` 和 `file`，默认 stdout，不写本地文件

## Configuration

### LogConfig 扩展

```go
type LogFileConfig struct {
    Dir         string `yaml:"dir"`
    MaxSizeMB   int    `yaml:"max_size_mb"`
    MaxBackups  int    `yaml:"max_backups"`
    MaxAgeDays  int    `yaml:"max_age_days"`
}

type LogConfig struct {
    Level   string        `yaml:"level"`
    Format  string        `yaml:"format"`
    Outputs string        `yaml:"outputs"`   // stdout | file | both
    File    LogFileConfig `yaml:"file"`
}
```

### config.local.yaml 新增

```yaml
log:
  level: debug
  format: text
  outputs: both
  file:
    dir: logs
    max_size_mb: 100
    max_backups: 7
    max_age_days: 30
```

### config.yaml（生产）

保持不变，无 `outputs` 和 `file` 配置，默认 `stdout`。

## Architecture

### New Files

| File | Purpose |
|------|---------|
| `internal/monitoring/file_handler.go` | 自定义 slog Handler：文件输出 + 按类型分文件 |
| `internal/monitoring/file_handler_test.go` | 单元测试 |
| `internal/monitoring/logger.go` | Logger 初始化入口 |
| `internal/monitoring/logger_test.go` | 单元测试 |

### Data Flow

```
业务代码调用 slog.Info/Error/...
       |
       v
MultiOutputHandler (file_handler.go)
       |
       v
  解析 record group 前缀
       |
  +---------+---------+
  | business.*  http.*       |
  v            v             |
  business.jsonl  http.jsonl |
                             |
  同时输出到 stdout (可选) <--+
```

### Core Components

**MultiOutputHandler**：

```go
type MultiOutputHandler struct {
    stdoutHandler slog.Handler
    fileWriters   map[string]*FileWriter
    mu            sync.Mutex
}

type FileWriter struct {
    path string
    file *os.File
    date string    // 用于按日切换
}
```

- `Handle()` 解析 record group 前缀，分发到对应 FileWriter
- 每个 FileWriter 内部维护日期，跨日自动切换新文件
- 启动时扫描目录，删除超过 max_age_days 的旧文件

**Logger 初始化**：

```go
func NewLogger(cfg *config.AppConfig) *slog.Logger
```

在 `cmd/control-plane/main.go` 中调用，替换现有 `slog.SetDefault()`。

### HTTP Middleware

改造 `internal/api/middleware/logger.go`：
- 替换 `fmt.Sprintf` 为 `slog.Info`
- 使用 `slog.With("component", "http")` 标记来源

### MonitoringService Integration

- 业务日志通过 `slog.With("component", "business")` 标记
- `RecordLogEntry` 等方法增加 slog 调用
- SLS 配置为空时自动跳过

## File Rotation

- MVP 阶段：按日期创建文件，文件名带日期后缀
- 启动时清理超过 `max_age_days` 的旧文件
- 不做按大小切割

## Acceptance Criteria Mapping

| Criteria | Design Coverage |
|----------|----------------|
| 关键事件持久化到本地 | MultiOutputHandler 写 JSONL 文件 |
| 进程重启后仍可查看 | 文件持久化，非内存 |
| grep/jq 可查询 | JSONL 格式，每行一条 JSON |
| 统一管理不分散 | 通过 slog Handler 统一，按 component 分文件 |
| 不影响生产环境 | IsLocal() 判断 + 配置驱动 |

## Documentation Updates

- `docs/knowledge/monitoring.md` — 新增本地持久化日志章节
- `docs/knowledge/quick-reference.md` — 日志配置项速查

## Other Changes

- `.gitignore` 新增 `logs/` 排除日志目录
