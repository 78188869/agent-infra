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

	bizLogger := logger.With("component", "business")
	bizLogger.Info("test business log", "task_id", "task-123")

	httpLogger := logger.With("component", "http")
	httpLogger.Info("test http log", "method", "GET", "path", "/api/tasks")

	if mh, ok := logger.Handler().(*MultiOutputHandler); ok {
		mh.Close()
	}

	files, _ := filepath.Glob(filepath.Join(dir, "business-*.jsonl"))
	if len(files) == 0 {
		t.Fatal("business log file should exist")
	}
	data, _ := os.ReadFile(files[0])
	if !strings.Contains(string(data), "test business log") {
		t.Errorf("business file should contain log message, got: %s", string(data))
	}

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
