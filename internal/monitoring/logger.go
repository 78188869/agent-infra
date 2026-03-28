package monitoring

import (
	"io"
	"log/slog"
	"os"

	"github.com/example/agent-infra/internal/config"
)

// NewLogger creates a slog.Logger based on application config.
//   - stdout: only outputs to os.Stdout
//   - file: only outputs to categorized JSONL files
//   - both: outputs to both stdout and files
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
	default:
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
		slog.Warn("failed to create file handler, falling back to stdout", "error", err)
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	return slog.New(handler)
}

// parseLevel converts a log level string to slog.Level.
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
