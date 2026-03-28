package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// pullImageIfMissing pulls a Docker image if it's not available locally.
// This ensures integration tests work offline after the first pull.
func pullImageIfMissing(image string) error {
	// Check if image exists locally
	cmd := exec.Command("docker", "image", "inspect", image)
	if err := cmd.Run(); err == nil {
		return nil // image already exists
	}
	// Pull with retry
	cmd = exec.Command("docker", "pull", image)
	return cmd.Run()
}

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

func TestComposeManager_Up_Down(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping docker compose integration test in short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}

	// Pull test image locally to avoid Docker Hub network issues
	testImage := "alpine:3.19"
	if err := pullImageIfMissing(testImage); err != nil {
		t.Skipf("skipping: failed to pull test image %s: %v", testImage, err)
	}

	tmpDir := t.TempDir()
	cfg := DefaultDockerConfig()
	cfg.ComposeDir = tmpDir
	cfg.CLIRunnerImage = testImage
	cfg.WrapperImage = testImage

	cm, _ := NewComposeManager(cfg)
	taskID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"

	// Write a simple compose that exits immediately for testing
	taskDir := filepath.Join(tmpDir, "task-"+taskID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatalf("failed to create task dir: %v", err)
	}
	err := os.WriteFile(filepath.Join(taskDir, "docker-compose.yml"), []byte(`services:
  cli-runner:
    image: `+testImage+`
    command: ["echo", "hello"]
  wrapper:
    image: `+testImage+`
    command: ["sleep", "30"]
    ports:
      - "9090"
    depends_on:
      - cli-runner
`), 0644)
	if err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	ctx := context.Background()

	// Up
	if err := cm.Up(ctx, taskID); err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	// Ensure cleanup
	defer cm.Down(context.Background(), taskID)

	// GetStatus
	statuses, err := cm.GetStatus(ctx, taskID)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if len(statuses) == 0 {
		t.Error("GetStatus() returned empty statuses")
	}

	// Down
	if err := cm.Down(ctx, taskID); err != nil {
		t.Fatalf("Down() error = %v", err)
	}
}

func TestComposeManager_GetStatus_NotFound(t *testing.T) {
	cfg := DefaultDockerConfig()
	cfg.ComposeDir = t.TempDir()

	cm, _ := NewComposeManager(cfg)
	taskID := "nonexistent-task-id"

	// GetStatus on non-existent task should error
	_, err := cm.GetStatus(context.Background(), taskID)
	if err == nil {
		t.Error("expected error for non-existent task")
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"line1\nline2\nline3", 3},
		{"line1\n\nline3", 2},
		{"", 0},
		{"single", 1},
	}
	for _, tt := range tests {
		got := splitLines(tt.input)
		if len(got) != tt.want {
			t.Errorf("splitLines(%q) returned %d lines, want %d", tt.input, len(got), tt.want)
		}
	}
}

func TestComposeManager_GenerateSingleContainerConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultDockerConfig()
	cfg.ComposeDir = tmpDir

	cm, err := NewComposeManager(cfg)
	if err != nil {
		t.Fatalf("NewComposeManager() error = %v", err)
	}

	taskID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	data := &SingleContainerTemplateData{
		TaskID:          taskID,
		WrapperImage:    "agent-infra/wrapper:latest",
		WorkspaceDir:    "./workspace",
		ControlPlaneURL: "http://localhost:8080",
		AnthropicAPIKey: "sk-test-key",
		TaskPrompt:      "Fix the bug in auth module",
		MaxTimeout:      "3600",
		GitRepo:         "https://github.com/example/repo.git",
		GitBranch:       "main",
		ClaudeMdContent: "Follow project conventions",
		AllowedTools:    "Bash,Read,Write",
	}

	err = cm.GenerateSingleContainerConfig(context.Background(), data)
	if err != nil {
		t.Fatalf("GenerateSingleContainerConfig() error = %v", err)
	}

	// Verify compose file was created.
	composeFile := filepath.Join(tmpDir, "task-"+taskID, "docker-compose.yml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		t.Fatal("compose file was not created")
	}

	contentBytes, err := os.ReadFile(composeFile)
	if err != nil {
		t.Fatalf("failed to read compose file: %v", err)
	}
	content := string(contentBytes)

	// Verify single service (no cli-runner).
	if containsSubstring(content, "cli-runner") {
		t.Error("single-container compose should not contain cli-runner service")
	}
	if !containsSubstring(content, "wrapper:") {
		t.Error("compose file missing wrapper service")
	}
	// Verify environment variables.
	if !containsSubstring(content, "TASK_ID="+taskID) {
		t.Error("compose file missing TASK_ID")
	}
	if !containsSubstring(content, "CONTROL_PLANE_URL=http://localhost:8080") {
		t.Error("compose file missing CONTROL_PLANE_URL")
	}
	if !containsSubstring(content, "ANTHROPIC_API_KEY=sk-test-key") {
		t.Error("compose file missing ANTHROPIC_API_KEY")
	}
	if !containsSubstring(content, "TASK_PROMPT=Fix the bug in auth module") {
		t.Error("compose file missing TASK_PROMPT")
	}
	if !containsSubstring(content, "MAX_TIMEOUT=3600") {
		t.Error("compose file missing MAX_TIMEOUT")
	}
	if !containsSubstring(content, "GIT_REPO=https://github.com/example/repo.git") {
		t.Error("compose file missing GIT_REPO")
	}
	if !containsSubstring(content, "GIT_BRANCH=main") {
		t.Error("compose file missing GIT_BRANCH")
	}
	if !containsSubstring(content, "CLAUDE_MD_CONTENT=Follow project conventions") {
		t.Error("compose file missing CLAUDE_MD_CONTENT")
	}
	if !containsSubstring(content, "ALLOWED_TOOLS=Bash,Read,Write") {
		t.Error("compose file missing ALLOWED_TOOLS")
	}
	// Verify image.
	if !containsSubstring(content, "agent-infra/wrapper:latest") {
		t.Error("compose file missing wrapper image")
	}
	// Verify volume mount.
	if !containsSubstring(content, "./workspace/"+taskID+":/workspace") {
		t.Error("compose file missing workspace volume mount")
	}
	// Verify port.
	if !containsSubstring(content, "9090") {
		t.Error("compose file missing port 9090")
	}
}

func TestComposeManager_GenerateSingleContainerConfig_NilData(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultDockerConfig()
	cfg.ComposeDir = tmpDir

	cm, err := NewComposeManager(cfg)
	if err != nil {
		t.Fatalf("NewComposeManager() error = %v", err)
	}

	err = cm.GenerateSingleContainerConfig(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil template data")
	}
}
