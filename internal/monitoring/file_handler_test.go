package monitoring

import (
	"io"
	"log/slog"
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

	fw.mu.Lock()
	fw.date = "2000-01-01"
	fw.mu.Unlock()

	line1 := `{"msg":"old"}` + "\n"
	fw.write([]byte(line1))

	fw.mu.Lock()
	fw.date = ""
	fw.mu.Unlock()

	line2 := `{"msg":"new"}` + "\n"
	fw.write([]byte(line2))

	oldPath := filepath.Join(dir, "test-2000-01-01.jsonl")
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		t.Error("old date file should exist")
	}

	today := time.Now().Format("2006-01-02")
	todayPath := filepath.Join(dir, "test-"+today+".jsonl")
	data, _ := os.ReadFile(todayPath)
	if !strings.Contains(string(data), "new") {
		t.Error("today file should contain new content")
	}
}

func TestCleanupOldFiles(t *testing.T) {
	dir := t.TempDir()

	old := []string{
		filepath.Join(dir, "test-2000-01-01.jsonl"),
		filepath.Join(dir, "test-2000-01-02.jsonl"),
	}
	for _, f := range old {
		os.WriteFile(f, []byte("old\n"), 0644)
	}

	today := filepath.Join(dir, "test-"+time.Now().Format("2006-01-02")+".jsonl")
	os.WriteFile(today, []byte("today\n"), 0644)

	cleanupOldFiles(dir, "test", 1)

	for _, f := range old {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("old file %q should be removed", f)
		}
	}
	if _, err := os.Stat(today); os.IsNotExist(err) {
		t.Error("today file should remain")
	}
}

func TestMultiOutputHandler_RoutesByComponent(t *testing.T) {
	dir := t.TempDir()
	h, err := NewMultiOutputHandler(io.Discard, dir, []string{"business", "http"})
	if err != nil {
		t.Fatalf("NewMultiOutputHandler: %v", err)
	}
	defer h.Close()

	logger := slog.New(h)
	logger.Info("biz event", "component", "business", "task_id", "t1")
	logger.Info("http request", "component", "http", "method", "GET")
	logger.Info("unknown component", "component", "other")

	h.mu.Lock()
	for _, fw := range h.files {
		fw.close()
	}
	h.mu.Unlock()

	date := time.Now().Format("2006-01-02")

	bizData, _ := os.ReadFile(filepath.Join(dir, "business-"+date+".jsonl"))
	if !strings.Contains(string(bizData), "biz event") {
		t.Error("business file should contain biz event")
	}

	httpData, _ := os.ReadFile(filepath.Join(dir, "http-"+date+".jsonl"))
	if !strings.Contains(string(httpData), "http request") {
		t.Error("http file should contain http request")
	}
}
