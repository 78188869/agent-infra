package monitoring

import (
	"context"
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

// fileWriter writes log lines to a date-stamped JSONL file.
// File naming convention: <dir>/<name>-YYYY-MM-DD.jsonl
type fileWriter struct {
	mu      sync.Mutex
	dir     string
	name    string
	date    string
	file    *os.File
	curPath string // tracks the actual path of the currently open file
}

// newFileWriter creates a new fileWriter, creating the directory if needed.
func newFileWriter(dir, name string) (*fileWriter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	fw := &fileWriter{dir: dir, name: name}
	if err := fw.openToday(); err != nil {
		return nil, fmt.Errorf("openToday: %w", err)
	}
	return fw, nil
}

// today returns today's date as a formatted string.
func today() string {
	return time.Now().Format("2006-01-02")
}

// openToday opens the file for today's date if needed.
// It only opens a new file when the date has changed or no file is open.
func (fw *fileWriter) openToday() error {
	now := today()
	targetPath := filepath.Join(fw.dir, fw.name+"-"+now+".jsonl")
	// If the correct file is already open, nothing to do
	if fw.curPath == targetPath && fw.file != nil {
		return nil
	}
	// Date changed or file not open: rotate
	if fw.file != nil {
		fw.file.Close()
	}
	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open %s: %w", targetPath, err)
	}
	fw.file = f
	fw.date = now
	fw.curPath = targetPath
	return nil
}

// openForDate opens (or returns the existing) file for the given date string.
func (fw *fileWriter) openForDate(date string) error {
	targetPath := filepath.Join(fw.dir, fw.name+"-"+date+".jsonl")
	// If the correct file is already open, nothing to do
	if fw.curPath == targetPath && fw.file != nil {
		return nil
	}
	if fw.file != nil {
		fw.file.Close()
	}
	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open %s: %w", targetPath, err)
	}
	fw.file = f
	fw.date = date
	fw.curPath = targetPath
	return nil
}

// write appends data to the current file, rotating if the date has changed.
func (fw *fileWriter) write(data []byte) (int, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// If a date is set and differs from today, write to that date's file
	if fw.date != "" && fw.date != today() {
		if err := fw.openForDate(fw.date); err != nil {
			return 0, fmt.Errorf("openForDate: %w", err)
		}
	} else if err := fw.openToday(); err != nil {
		return 0, fmt.Errorf("openToday: %w", err)
	}
	return fw.file.Write(data)
}

// close closes the underlying file handle.
func (fw *fileWriter) close() {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if fw.file != nil {
		fw.file.Close()
		fw.file = nil
	}
}

// cleanupOldFiles removes files matching <dir>/<name>-*.jsonl that are older than maxAgeDays.
func cleanupOldFiles(dir, name string, maxAgeDays int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	prefix := name + "-"
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fn := entry.Name()
		if !strings.HasPrefix(fn, prefix) || !strings.HasSuffix(fn, ".jsonl") {
			continue
		}
		// Extract date from filename: name-YYYY-MM-DD.jsonl
		dateStr := strings.TrimPrefix(fn, prefix)
		dateStr = strings.TrimSuffix(dateStr, ".jsonl")
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if fileDate.Before(cutoff) {
			os.Remove(filepath.Join(dir, fn))
		}
	}
}

// MultiOutputHandler implements slog.Handler and routes log records to both
// stdout and per-component JSONL files based on the "component" attribute.
type MultiOutputHandler struct {
	stdout io.Writer
	files  map[string]*fileWriter
	mu     sync.Mutex
}

// NewMultiOutputHandler creates a MultiOutputHandler that writes to stdout
// and creates a fileWriter for each category in the specified directory.
func NewMultiOutputHandler(stdout io.Writer, dir string, categories []string) (*MultiOutputHandler, error) {
	h := &MultiOutputHandler{
		stdout: stdout,
		files:  make(map[string]*fileWriter),
	}
	for _, cat := range categories {
		fw, err := newFileWriter(dir, cat)
		if err != nil {
			// Close already opened files on error
			for _, f := range h.files {
				f.close()
			}
			return nil, fmt.Errorf("newFileWriter(%s): %w", cat, err)
		}
		h.files[cat] = fw
	}
	return h, nil
}

// Enabled returns true for all log levels.
func (h *MultiOutputHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// Handle processes a slog.Record by writing JSON to stdout and routing to
// the appropriate file based on the "component" attribute.
func (h *MultiOutputHandler) Handle(_ context.Context, r slog.Record) error {
	// Build a JSON object from the record
	entry := make(map[string]interface{})
	entry["level"] = r.Level.String()
	entry["msg"] = r.Message
	entry["time"] = r.Time.Format(time.RFC3339Nano)

	var component string
	r.Attrs(func(a slog.Attr) bool {
		entry[a.Key] = a.Value.Any()
		if a.Key == "component" {
			component = a.Value.String()
		}
		return true
	})

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	line := append(data, '\n')

	// Write to stdout (best effort)
	_, _ = h.stdout.Write(line)

	// Route to file by component, supporting prefix matching (e.g. "business.xxx" -> "business")
	if component != "" {
		h.mu.Lock()
		defer h.mu.Unlock()
		for key, fw := range h.files {
			if component == key || strings.HasPrefix(component, key+".") {
				_, _ = fw.write(line)
				break
			}
		}
	}

	return nil
}

// multiOutputChild is a child handler that carries extra attributes
// from logger.With() calls. It delegates core logic to the parent MultiOutputHandler.
type multiOutputChild struct {
	parent *MultiOutputHandler
	attrs  []slog.Attr
}

// Enabled delegates to parent.
func (c *multiOutputChild) Enabled(ctx context.Context, level slog.Level) bool {
	return c.parent.Enabled(ctx, level)
}

// Handle processes a slog.Record by merging the child's pre-bound attrs
// and delegating to the parent's stdout + file routing logic.
func (c *multiOutputChild) Handle(ctx context.Context, r slog.Record) error {
	// Build a combined record with both child attrs and record attrs
	entry := make(map[string]interface{})
	entry["level"] = r.Level.String()
	entry["msg"] = r.Message
	entry["time"] = r.Time.Format(time.RFC3339Nano)

	var component string

	// First add pre-bound attrs
	for _, a := range c.attrs {
		entry[a.Key] = a.Value.Any()
		if a.Key == "component" {
			component = a.Value.String()
		}
	}

	// Then add record-level attrs (may override pre-bound ones)
	r.Attrs(func(a slog.Attr) bool {
		entry[a.Key] = a.Value.Any()
		if a.Key == "component" {
			component = a.Value.String()
		}
		return true
	})

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	line := append(data, '\n')

	_, _ = c.parent.stdout.Write(line)

	if component != "" {
		c.parent.mu.Lock()
		defer c.parent.mu.Unlock()
		for key, fw := range c.parent.files {
			if component == key || strings.HasPrefix(component, key+".") {
				_, _ = fw.write(line)
				break
			}
		}
	}

	return nil
}

// WithAttrs returns a new child handler with additional pre-bound attributes.
func (c *multiOutputChild) WithAttrs(attrs []slog.Attr) slog.Handler {
	merged := make([]slog.Attr, len(c.attrs), len(c.attrs)+len(attrs))
	copy(merged, c.attrs)
	merged = append(merged, attrs...)
	return &multiOutputChild{parent: c.parent, attrs: merged}
}

// WithGroup returns the child itself (groups not supported).
func (c *multiOutputChild) WithGroup(name string) slog.Handler {
	return c
}

// WithAttrs returns a child handler that carries the given attributes.
func (h *MultiOutputHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &multiOutputChild{parent: h, attrs: attrs}
}

// WithGroup returns the handler itself (groups not supported).
func (h *MultiOutputHandler) WithGroup(name string) slog.Handler {
	return h
}

// Close closes all underlying file writers.
func (h *MultiOutputHandler) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, fw := range h.files {
		fw.close()
	}
}
