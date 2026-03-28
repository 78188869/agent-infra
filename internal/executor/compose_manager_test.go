package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDockerConfigDefaults(t *testing.T) {
	cfg := DefaultDockerConfig()

	if cfg.WorkspaceDir != "./workspace" {
		t.Errorf("expected WorkspaceDir './workspace', got %s", cfg.WorkspaceDir)
	}
	if cfg.CLIRunnerImage != "agent-infra/cli-runner:latest" {
		t.Errorf("expected CLIRunnerImage 'agent-infra/cli-runner:latest', got %s", cfg.CLIRunnerImage)
	}
	if cfg.WrapperImage != "agent-infra/wrapper:latest" {
		t.Errorf("expected WrapperImage 'agent-infra/wrapper:latest', got %s", cfg.WrapperImage)
	}
	if cfg.WrapperPort != 9090 {
		t.Errorf("expected WrapperPort 9090, got %d", cfg.WrapperPort)
	}
	if cfg.ComposeDir != "/tmp/agent-infra/compose" {
		t.Errorf("expected ComposeDir '/tmp/agent-infra/compose', got %s", cfg.ComposeDir)
	}
}

func TestComposeManager_GenerateConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultDockerConfig()
	cfg.ComposeDir = tmpDir

	cm, err := NewComposeManager(cfg)
	if err != nil {
		t.Fatalf("NewComposeManager() error = %v", err)
	}

	taskID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	err = cm.GenerateConfig(context.Background(), taskID, map[string]string{
		"GIT_REPO_URL": "https://github.com/example/repo.git",
		"TASK_PROMPT":  "Fix the bug",
	})
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	// Verify compose file was created
	composeFile := filepath.Join(tmpDir, "task-"+taskID, "docker-compose.yml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		t.Fatal("compose file was not created")
	}

	// Verify content contains key elements
	data, err := os.ReadFile(composeFile)
	if err != nil {
		t.Fatalf("failed to read compose file: %v", err)
	}
	content := string(data)

	// Check services
	if !containsSubstring(content, "cli-runner") {
		t.Error("compose file missing cli-runner service")
	}
	if !containsSubstring(content, "wrapper") {
		t.Error("compose file missing wrapper service")
	}
	// Check task ID in volume path
	if !containsSubstring(content, taskID) {
		t.Error("compose file missing task ID in volume path")
	}
	// Check images
	if !containsSubstring(content, "agent-infra/cli-runner:latest") {
		t.Error("compose file missing cli-runner image")
	}
	if !containsSubstring(content, "agent-infra/wrapper:latest") {
		t.Error("compose file missing wrapper image")
	}
	// Check env vars
	if !containsSubstring(content, "GIT_REPO_URL=https://github.com/example/repo.git") {
		t.Error("compose file missing GIT_REPO_URL env var")
	}
	if !containsSubstring(content, "TASK_PROMPT=Fix the bug") {
		t.Error("compose file missing TASK_PROMPT env var")
	}
}

func TestComposeManager_TaskDir(t *testing.T) {
	cfg := &DockerConfig{ComposeDir: "/tmp/test"}
	cm, _ := NewComposeManager(cfg)

	expected := "/tmp/test/task-abc-123"
	got := cm.TaskDir("abc-123")
	if got != expected {
		t.Errorf("TaskDir() = %q, want %q", got, expected)
	}
}

func TestComposeManager_NilConfig(t *testing.T) {
	cm, err := NewComposeManager(nil)
	if err != nil {
		t.Fatalf("NewComposeManager(nil) error = %v", err)
	}
	if cm.config.WorkspaceDir != "./workspace" {
		t.Error("nil config should use defaults")
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
