# Local Persistent Logging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在本地开发环境下将业务日志和 HTTP 请求日志持久化到本地 JSONL 文件，进程重启后可回溯。

**Architecture:** 实现自定义 slog.Handler（MultiOutputHandler），根据 record 的 component 分组写入不同类型文件（business / http）。通过 AppConfig.IsLocal() 判断环境，本地开发写文件，生产环境仅 stdout。

**Tech Stack:** Go 1.21 + log/slog + JSONL 文件格式

**Spec:** `docs/superpowers/specs/2026-03-28-local-persistent-logging-design.md`

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `internal/config/config.go` | 扩展 LogConfig 结构体，新增 Outputs、LogFileConfig |
| Modify | `configs/config.yaml` | 新增 log.outputs、log.file 配置模板 |
| Modify | `configs/config.local.yaml` | 新增本地日志配置 |
| Create | `internal/monitoring/file_handler.go` | FileWriter + MultiOutputHandler 实现 |
| Create | `internal/monitoring/file_handler_test.go` | 文件 handler 单元测试 |
| Create | `internal/monitoring/logger.go` | NewLogger 初始化入口 |
| Create | `internal/monitoring/logger_test.go` | Logger 初始化测试 |
| Modify | `internal/api/middleware/logger.go` | 替换 fmt.Sprintf 为 slog |
| Modify | `internal/service/monitoring_service.go` | 业务日志增加 slog 调用 |
| Modify | `cmd/control-plane/main.go` | 调用 NewLogger 替换默认日志 |
| Modify | `docs/knowledge/monitoring.md` | 新增本地持久化日志章节 |
| Modify | `docs/knowledge/quick-reference.md` | 新增日志配置速查 |

---

### Task 1: 扩展 LogConfig 结构体

**Files:**
- Modify: `internal/config/config.go:40-44`

- [ ] **Step 1: 在 LogConfig 后面新增 LogFileConfig 结构体，扩展 LogConfig 字段**

```go
// LogFileConfig holds local file logging configuration.
type LogFileConfig struct {
	Dir        string `yaml:"dir"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level   string        `yaml:"level"`
	Format  string        `yaml:"format"`
	Outputs string        `yaml:"outputs"` // stdout | file | both
	File    LogFileConfig `yaml:"file"`
}
```

- [ ] **Step 2: 在 ApplyDefaults 中为本地环境设置 outputs 默认值**

在 `if c.IsLocal()` 分支的 `Log.Format` 设置后面添加：

```go
if c.Log.Outputs == "" {
    c.Log.Outputs = "both"
}
if c.Log.File.Dir == "" {
    c.Log.File.Dir = "logs"
}
if c.Log.File.MaxAgeDays == 0 {
    c.Log.File.MaxAgeDays = 30
}
```

在 else 分支的 `Log.Format` 设置后面添加：

```go
if c.Log.Outputs == "" {
    c.Log.Outputs = "stdout"
}
```

- [ ] **Step 3: 运行测试验证编译通过**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-33 && go build ./internal/config/...`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): extend LogConfig with file output settings"
```

---

### Task 2: 更新配置文件

**Files:**
- Modify: `configs/config.yaml:30-32`
- Modify: `configs/config.local.yaml:18-20`

- [ ] **Step 1: 更新 config.yaml 的 log 块**

将生产模板的 log 块替换为：

```yaml
log:
  level: ${LOG_LEVEL:info}
  format: json
  outputs: ${LOG_OUTPUTS:stdout}
  file:
    dir: ${LOG_DIR:}
    max_size_mb: ${LOG_MAX_SIZE:100}
    max_backups: ${LOG_MAX_BACKUPS:7}
    max_age_days: ${LOG_MAX_AGE:30}
```

- [ ] **Step 2: 更新 config.local.yaml 的 log 块**

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

- [ ] **Step 3: 验证配置加载**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-33 && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add configs/config.yaml configs/config.local.yaml
git commit -m "feat(config): add log file output settings to config files"
```

---

### Task 3: 实现 FileWriter

**Files:**
- Create: `internal/monitoring/file_handler.go`
- Create: `internal/monitoring/file_handler_test.go`

- [ ] **Step 1: 写 FileWriter 的测试**

创建 `internal/monitoring/file_handler_test.go`：

```go
package monitoring

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileWriter_WritesJSONL(t *testing.T) {
	dir := t.TempDir()
	fw, err := newFileWriter(dir, "test")
	if err != nil {
		t.Fatalf("newFileWriter: %v", err)
	}
	defer fw.close()

	line := `{"level":"info","msg":"hello"}` + "\n"
	if _, err := fw.write([]byte(line)); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read back and verify
	date := time.Now().Format("2006-01-02")
	path := filepath.Join(dir, "test-"+date+".jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != line {
		t.Errorf("got %q, want %q", string(data), line)
	}
}

func TestFileWriter_DailyRotation(t *testing.T) {
	dir := t.TempDir()
	fw, err := newFileWriter(dir, "test")
	if err != nil {
		t.Fatalf("newFileWriter: %v", err)
	}
	defer fw.close()

	// Simulate date change
	fw.mu.Lock()
	fw.date = "2000-01-01"
	fw.mu.Unlock()

	line1 := `{"msg":"old"}` + "\n"
	fw.write([]byte(line1))

	// Reset to today — should open new file
	fw.mu.Lock()
	fw.date = ""
	fw.mu.Unlock()

	line2 := `{"msg":"new"}` + "\n"
	fw.write([]byte(line2))

	// Old file should exist
	oldPath := filepath.Join(dir, "test-2000-01-01.jsonl")
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		t.Error("old date file should exist")
	}

	// Today file should have new content
	today := time.Now().Format("2006-01-02")
	todayPath := filepath.Join(dir, "test-"+today+".jsonl")
	data, _ := os.ReadFile(todayPath)
	if !strings.Contains(string(data), "new") {
		t.Error("today file should contain new content")
	}
}

func TestCleanupOldFiles(t *testing.T) {
	dir := t.TempDir()

	// Create old files
	old := []string{
		filepath.Join(dir, "test-2000-01-01.jsonl"),
		filepath.Join(dir, "test-2000-01-02.jsonl"),
	}
	for _, f := range old {
		os.WriteFile(f, []byte("old\n"), 0644)
	}

	// Create today file
	today := filepath.Join(dir, "test-"+time.Now().Format("2006-01-02")+".jsonl")
	os.WriteFile(today, []byte("today\n"), 0644)

	cleanupOldFiles(dir, "test", 1) // maxAgeDays=1

	// Old files should be removed
	for _, f := range old {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("old file %q should be removed", f)
		}
	}
	// Today file should remain
	if _, err := os.Stat(today); os.IsNotExist(err) {
		t.Error("today file should remain")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-33 && go test ./internal/monitoring/ -run "TestFileWriter|TestCleanup" -v`
Expected: FAIL (file_handler.go not yet created)

- [ ] **Step 3: 实现 FileWriter**

创建 `internal/monitoring/file_handler.go`：

```go
package monitoring

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileWriter writes log lines to a dated JSONL file.
type FileWriter struct {
	mu   sync.Mutex
	dir  string
	name string // e.g. "business", "http"
	date string // current date "2006-01-02"
	file *os.File
}

// newFileWriter creates a FileWriter that writes to dir/name-YYYY-MM-DD.jsonl.
func newFileWriter(dir, name string) (*FileWriter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir %s: %w", dir, err)
	}
	fw := &FileWriter{dir: dir, name: name}
	if err := fw.openToday(); err != nil {
		return nil, err
	}
	return fw, nil
}

func (fw *FileWriter) openToday() error {
	today := time.Now().Format("2006-01-02")
	if fw.date == today && fw.file != nil {
		return nil
	}
	if fw.file != nil {
		fw.file.Close()
	}
	path := filepath.Join(fw.dir, fw.name+"-"+today+".jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", path, err)
	}
	fw.file = f
	fw.date = today
	return nil
}

// write appends data to the current day's file, rotating if needed.
func (fw *FileWriter) write(data []byte) (int, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if err := fw.openToday(); err != nil {
		return 0, err
	}
	return fw.file.Write(data)
}

// close closes the underlying file.
func (fw *FileWriter) close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if fw.file != nil {
		err := fw.file.Close()
		fw.file = nil
		return err
	}
	return nil
}

// cleanupOldFiles removes log files older than maxAgeDays.
func cleanupOldFiles(dir, name string, maxAgeDays int) {
	if maxAgeDays <= 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	prefix := name + "-"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// MultiOutputHandler is a slog.Handler that writes to stdout and/or JSONL files.
type MultiOutputHandler struct {
	stdout io.Writer
	files  map[string]*FileWriter // keyed by component: "business", "http"
	mu     sync.Mutex
}

// NewMultiOutputHandler creates a handler that routes logs to stdout and categorized files.
func NewMultiOutputHandler(stdout io.Writer, dir string, categories []string) (*MultiOutputHandler, error) {
	h := &MultiOutputHandler{
		stdout: stdout,
		files:  make(map[string]*FileWriter),
	}
	for _, cat := range categories {
		fw, err := newFileWriter(dir, cat)
		if err != nil {
			// Close already opened writers
			for _, w := range h.files {
				w.close()
			}
			return nil, fmt.Errorf("create writer for %s: %w", cat, err)
		}
		h.files[cat] = fw
	}
	return h, nil
}

// Enabled implements slog.Handler.
func (h *MultiOutputHandler) Enabled(level slog.Level) bool { return true }

// Handle implements slog.Handler. It writes the record to stdout and routes to file by component.
func (h *MultiOutputHandler) Handle(r slog.Record) error {
	// Build JSON line
	entry := map[string]interface{}{
		"time":  r.Time.Format(time.RFC3339Nano),
		"level": r.Level.String(),
		"msg":   r.Message,
	}
	r.Attrs(func(a slog.Attr) bool {
		entry[a.Key] = a.Value.Any()
		return true
	})

	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	line = append(line, '\n')

	// Write to stdout
	if h.stdout != nil {
		h.stdout.Write(line)
	}

	// Route to file by component
	component := ""
	if c, ok := entry["component"].(string); ok {
		component = c
	}
	// Support grouped components: "business.xxx" -> "business"
	if idx := strings.Index(component, "."); idx > 0 {
		component = component[:idx]
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if fw, ok := h.files[component]; ok {
		fw.write(line)
	}

	return nil
}

// WithAttrs implements slog.Handler.
func (h *MultiOutputHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup implements slog.Handler.
func (h *MultiOutputHandler) WithGroup(name string) slog.Handler {
	return h
}

// Close closes all file writers.
func (h *MultiOutputHandler) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, fw := range h.files {
		fw.close()
	}
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-33 && go test ./internal/monitoring/ -run "TestFileWriter|TestCleanup" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/monitoring/file_handler.go internal/monitoring/file_handler_test.go
git commit -m "feat(monitoring): implement FileWriter and MultiOutputHandler"
```

---

### Task 4: 实现 Logger 初始化

**Files:**
- Create: `internal/monitoring/logger.go`
- Create: `internal/monitoring/logger_test.go`

- [ ] **Step 1: 写 NewLogger 的测试**

创建 `internal/monitoring/logger_test.go`：

```go
package monitoring

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/example/agent-infra/internal/config"
)

func TestNewLogger_StdoutOnly(t *testing.T) {
	cfg := &config.AppConfig{
		Env: "production",
		Log: config.LogConfig{
			Level:   "info",
			Format:  "json",
			Outputs: "stdout",
		},
	}
	logger := NewLogger(cfg)
	if logger == nil {
		t.Fatal("logger should not be nil")
	}
}

func TestNewLogger_WithFile(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.AppConfig{
		Env: "local",
		Log: config.LogConfig{
			Level:   "debug",
			Format:  "text",
			Outputs: "both",
			File: config.LogFileConfig{
				Dir:        dir,
				MaxAgeDays: 30,
			},
		},
	}

	logger := NewLogger(cfg)
	if logger == nil {
		t.Fatal("logger should not be nil")
	}

	// Log a business event
	bizLogger := logger.With("component", "business")
	bizLogger.Info("test business log", "task_id", "task-123")

	// Log an http event
	httpLogger := logger.With("component", "http")
	httpLogger.Info("test http log", "method", "GET", "path", "/api/tasks")

	// Close file handler to flush
	if mh, ok := logger.Handler().(*MultiOutputHandler); ok {
		mh.Close()
	}

	// Verify business file
	files, _ := filepath.Glob(filepath.Join(dir, "business-*.jsonl"))
	if len(files) == 0 {
		t.Fatal("business log file should exist")
	}
	data, _ := os.ReadFile(files[0])
	if !strings.Contains(string(data), "test business log") {
		t.Errorf("business file should contain log message, got: %s", string(data))
	}

	// Verify http file
	files, _ = filepath.Glob(filepath.Join(dir, "http-*.jsonl"))
	if len(files) == 0 {
		t.Fatal("http log file should exist")
	}
	data, _ = os.ReadFile(files[0])
	if !strings.Contains(string(data), "test http log") {
		t.Errorf("http file should contain log message, got: %s", string(data))
	}
}

func TestNewLogger_FileOnly(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.AppConfig{
		Env: "local",
		Log: config.LogConfig{
			Level:   "info",
			Format:  "json",
			Outputs: "file",
			File: config.LogFileConfig{
				Dir:        dir,
				MaxAgeDays: 30,
			},
		},
	}

	logger := NewLogger(cfg)
	bizLogger := logger.With("component", "business")
	bizLogger.Info("file-only test")

	if mh, ok := logger.Handler().(*MultiOutputHandler); ok {
		mh.Close()
	}

	files, _ := filepath.Glob(filepath.Join(dir, "business-*.jsonl"))
	if len(files) == 0 {
		t.Fatal("business log file should exist even with file-only output")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-33 && go test ./internal/monitoring/ -run "TestNewLogger" -v`
Expected: FAIL (logger.go not yet created)

- [ ] **Step 3: 实现 NewLogger**

创建 `internal/monitoring/logger.go`：

```go
package monitoring

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/example/agent-infra/internal/config"
)

// NewLogger creates a slog.Logger based on application config.
// - stdout: only outputs to os.Stdout
// - file: only outputs to categorized JSONL files
// - both: outputs to both stdout and files
func NewLogger(cfg *config.AppConfig) *slog.Logger {
	outputs := cfg.Log.Outputs
	if outputs == "" {
		outputs = "stdout"
	}

	// Determine stdout writer
	var stdout io.Writer
	switch outputs {
	case "file":
		stdout = io.Discard
	default: // "stdout" or "both"
		stdout = os.Stdout
	}

	// If no file output needed, use standard handler
	if outputs == "stdout" {
		var handler slog.Handler
		if cfg.Log.Format == "text" {
			handler = slog.NewTextHandler(stdout, &slog.HandlerOptions{
				Level: parseLevel(cfg.Log.Level),
			})
		} else {
			handler = slog.NewJSONHandler(stdout, &slog.HandlerOptions{
				Level: parseLevel(cfg.Log.Level),
			})
		}
		return slog.New(handler)
	}

	// File output needed — create MultiOutputHandler
	dir := cfg.Log.File.Dir
	if dir == "" {
		dir = "logs"
	}

	// Cleanup old log files on startup
	cleanupOldFiles(dir, "business", cfg.Log.File.MaxAgeDays)
	cleanupOldFiles(dir, "http", cfg.Log.File.MaxAgeDays)

	handler, err := NewMultiOutputHandler(stdout, dir, []string{"business", "http"})
	if err != nil {
		// Fallback to stdout-only if file creation fails
		slog.Warn("failed to create file handler, falling back to stdout", "error", err)
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	return slog.New(handler)
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Format is handled by MultiOutputHandler which always outputs JSONL to files.
// The format config field (json/text) only affects stdout output in non-file mode.
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-33 && go test ./internal/monitoring/ -run "TestNewLogger" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/monitoring/logger.go internal/monitoring/logger_test.go
git commit -m "feat(monitoring): add NewLogger with file output support"
```

---

### Task 5: 改造 HTTP 中间件

**Files:**
- Modify: `internal/api/middleware/logger.go`

- [ ] **Step 1: 替换 middleware/logger.go 为 slog 实现**

```go
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a gin middleware for structured request logging via slog.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		slog.Info("http request",
			"component", "http",
			"method", method,
			"path", path,
			"status", status,
			"latency", latency.String(),
			"client_ip", c.ClientIP(),
		)
	}
}
```

- [ ] **Step 2: 验证编译通过**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-33 && go build ./internal/api/...`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/api/middleware/logger.go
git commit -m "refactor(middleware): replace fmt logger with slog for HTTP requests"
```

---

### Task 6: 业务日志增加 slog 调用

**Files:**
- Modify: `internal/service/monitoring_service.go`

- [ ] **Step 1: 在 RecordTaskStatusChange 中增加 slog 调用**

在 `s.hub.Broadcast` 调用后、`return s.sls.RecordEvent` 之前添加：

```go
slog.Info("task status changed",
    "component", "business",
    "task_id", taskID,
    "tenant_id", tenantID,
    "old_status", oldStatus,
    "new_status", newStatus,
)
```

需要增加 import `"log/slog"`。

- [ ] **Step 2: 在 RecordLogEntry 中增加 slog 调用**

在 `return s.sls.RecordEvent` 之前添加：

```go
slog.Info("log entry recorded",
    "component", "business",
    "task_id", taskID,
    "tenant_id", tenantID,
    "event_type", string(eventType),
    "event_name", eventName,
)
```

- [ ] **Step 3: 在 BroadcastTaskCompletion 中增加 slog 调用**

在 `s.hub.Broadcast` 之后添加：

```go
slog.Info("task completed",
    "component", "business",
    "task_id", taskID,
    "tenant_id", tenantID,
)
```

- [ ] **Step 4: 验证编译通过**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-33 && go build ./internal/service/...`
Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
git add internal/service/monitoring_service.go
git commit -m "feat(monitoring): add slog calls to MonitoringService for file logging"
```

---

### Task 7: 集成到 main.go

**Files:**
- Modify: `cmd/control-plane/main.go`

- [ ] **Step 1: 在配置加载后初始化 logger**

在 `env := cfg.GetEnvironment()` 行之后添加：

```go
// Initialize structured logger (file output in local env, stdout in production)
logger := monitoring.NewLogger(cfg)
slog.SetDefault(logger)
```

需要增加 import `"log/slog"`。

- [ ] **Step 2: 将 `log.Printf` 替换为 `slog.Info`**

替换 `log.Printf("Starting control-plane in %q environment", env)` 为：
```go
slog.Info("starting control-plane", "env", env)
```

替换 `log.Printf("Starting control-plane server on %s (env=%s)", addr, env)` 为：
```go
slog.Info("starting server", "addr", addr, "env", env)
```

替换 `log.Fatalf("Failed to start server: %v", err)` 为：
```go
slog.Error("failed to start server", "error", err)
os.Exit(1)
```

需要增加 import `"os"`，移除不再使用的 `"log"` import。

- [ ] **Step 3: 验证编译通过**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-33 && go build ./cmd/control-plane/...`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add cmd/control-plane/main.go
git commit -m "feat(main): integrate NewLogger for environment-aware log output"
```

---

### Task 8: 全量测试验证

- [ ] **Step 1: 运行全部测试**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-33 && make test`
Expected: ALL PASS

- [ ] **Step 2: 运行 lint**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-33 && make lint`
Expected: PASS

- [ ] **Step 3: 运行覆盖率检查**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-33 && go test -cover ./internal/monitoring/...`
Expected: coverage > 80% for monitoring package

- [ ] **Step 4: 手动冒烟测试 — 启动服务，验证日志文件生成**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-33 && APP_ENV=local go run ./cmd/control-plane/ -config configs/config.local.yaml &`
然后访问 HTTP 端点触发请求日志，检查 `logs/` 目录下是否生成 `http-*.jsonl` 文件。

- [ ] **Step 5: 验证 grep/jq 可查询**

Run: `cat logs/http-*.jsonl | jq '.method'`
Expected: 输出 HTTP method 字段值

---

### Task 9: 更新文档

**Files:**
- Modify: `docs/knowledge/monitoring.md`
- Modify: `docs/knowledge/quick-reference.md`

- [ ] **Step 1: 在 monitoring.md 末尾（Change History 之前）新增章节**

```markdown
## 4.5 Local Persistent Logging

本地开发环境下（`APP_ENV=local`），日志持久化到本地 JSONL 文件。

**配置项**（`config.local.yaml`）：

```yaml
log:
  outputs: both        # stdout | file | both
  file:
    dir: logs          # 日志目录
    max_size_mb: 100
    max_backups: 7
    max_age_days: 30
```

**文件命名**：
- `logs/business-YYYY-MM-DD.jsonl` — 业务执行日志（状态变更、工具调用、错误）
- `logs/http-YYYY-MM-DD.jsonl` — HTTP 请求/响应日志

**环境切换**：
- 本地开发：`outputs: both`，同时输出到 stdout + 文件
- 生产环境：`outputs: stdout`（默认），仅输出到终端，SLS 负责持久化

**查询示例**：

```bash
# 查询所有业务错误
cat logs/business-*.jsonl | jq 'select(.level=="ERROR")'

# 查询指定任务的日志
grep "task-123" logs/business-*.jsonl

# 查询 HTTP 4xx/5xx 请求
cat logs/http-*.jsonl | jq 'select(.status >= 400)'
```
```

并在 Change History 表中新增一行：
```
| 2026-03-28 | v1.1 | #33 | - | §9 | 新增本地持久化日志支持 |
```

- [ ] **Step 2: 在 quick-reference.md 末尾新增日志配置速查**

```markdown
---

## Log Configuration

| Config | Values | Default (prod) | Default (local) |
|--------|--------|----------------|-----------------|
| `log.outputs` | stdout / file / both | stdout | both |
| `log.file.dir` | path | - | logs |
| `log.file.max_age_days` | int | 30 | 30 |
| `log.file.max_backups` | int | 7 | 7 |

**Log Files**:

| File | Content |
|------|---------|
| `logs/business-YYYY-MM-DD.jsonl` | 业务执行日志 |
| `logs/http-YYYY-MM-DD.jsonl` | HTTP 请求日志 |
```

- [ ] **Step 3: Commit**

```bash
git add docs/knowledge/monitoring.md docs/knowledge/quick-reference.md
git commit -m "docs(knowledge): add local persistent logging docs"
```

---

### Task 10: 提交推送并创建 PR

- [ ] **Step 1: 推送分支**

```bash
git push -u origin feature/issue-33
```

- [ ] **Step 2: 创建 PR**

```bash
gh pr create --base main --title "feat(monitoring): local persistent logging for issue-33" --body "$(cat <<'EOF'
## Summary
- 实现自定义 slog.Handler（MultiOutputHandler），按类型分文件写入 JSONL
- 本地开发环境（APP_ENV=local）自动持久化业务日志和 HTTP 请求日志
- 生产环境不受影响，仅 stdout 输出

## Test Plan
- [x] FileWriter 单元测试（写入、日切、清理）
- [x] NewLogger 集成测试（stdout-only、file、both 模式）
- [x] 全量 `make test` + `make lint` 通过
- [ ] 手动冒烟：本地启动验证 logs/ 目录生成 JSONL 文件
- [ ] `jq` 查询验证日志内容可解析

Closes #33

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
