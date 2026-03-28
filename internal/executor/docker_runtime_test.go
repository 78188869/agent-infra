package executor

import (
	"context"
	"os/exec"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/google/uuid"
)

func TestDockerRuntime_Create_InvalidTask(t *testing.T) {
	cfg := DefaultDockerConfig()
	cfg.ComposeDir = t.TempDir()

	rt, err := NewDockerRuntime(cfg)
	if err != nil {
		t.Fatalf("NewDockerRuntime() error = %v", err)
	}

	_, err = rt.Create(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil task")
	}
}

func TestDockerRuntime_GetStatus_NotFound(t *testing.T) {
	cfg := DefaultDockerConfig()
	cfg.ComposeDir = t.TempDir()

	rt, _ := NewDockerRuntime(cfg)

	taskID := uuid.New().String()
	_, err := rt.GetStatus(context.Background(), taskID)
	if err == nil {
		t.Error("expected error for non-existent task")
	}
}

func TestDockerRuntime_Delete_NoExist(t *testing.T) {
	cfg := DefaultDockerConfig()
	cfg.ComposeDir = t.TempDir()

	rt, _ := NewDockerRuntime(cfg)

	taskID := uuid.New().String()
	// Delete returns an error because the compose file does not exist,
	// which is the expected behavior for ComposeManager.Down.
	err := rt.Delete(context.Background(), taskID)
	if err == nil {
		t.Error("expected error when deleting non-existent task compose")
	}
}

func TestDockerRuntime_GetAddress_NoCompose(t *testing.T) {
	cfg := DefaultDockerConfig()
	cfg.ComposeDir = t.TempDir()

	rt, _ := NewDockerRuntime(cfg)

	taskID := uuid.New().String()
	_, err := rt.GetAddress(context.Background(), taskID)
	if err == nil {
		t.Error("expected error for task with no running compose")
	}
}

func TestMapDockerStateToPhase(t *testing.T) {
	tests := []struct {
		dockerState string
		want        string
	}{
		{"running", "Running"},
		{"exited", "Succeeded"},
		{"paused", "Paused"},
		{"dead", "Failed"},
		{"removing", "Failed"},
		{"created", "Pending"},
		{"restarting", "Pending"},
		{"unknown-state", "Unknown"},
	}
	for _, tt := range tests {
		got := mapDockerStateToPhase(tt.dockerState)
		if got != tt.want {
			t.Errorf("mapDockerStateToPhase(%q) = %q, want %q", tt.dockerState, got, tt.want)
		}
	}
}

func TestDockerRuntime_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping docker integration test in short mode")
	}

	// Check docker is available
	_, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker not available")
	}

	cfg := DefaultDockerConfig()
	cfg.ComposeDir = t.TempDir()
	cfg.CLIRunnerImage = "alpine:3.19"
	cfg.WrapperImage = "alpine:3.19"

	rt, err := NewDockerRuntime(cfg)
	if err != nil {
		t.Fatalf("NewDockerRuntime() error = %v", err)
	}

	taskID := uuid.New()
	task := &model.Task{}
	task.ID = taskID
	task.Status = model.TaskStatusScheduled

	ctx := context.Background()

	// Create
	info, err := rt.Create(ctx, task)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer rt.Delete(context.Background(), taskID.String())

	if info.Name == "" {
		t.Error("Create() returned empty name")
	}
	if info.Namespace != "docker" {
		t.Errorf("expected Namespace 'docker', got %s", info.Namespace)
	}

	// GetStatus
	status, err := rt.GetStatus(ctx, taskID.String())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if status.Phase == "" {
		t.Error("GetStatus() returned empty phase")
	}

	// Delete
	err = rt.Delete(ctx, taskID.String())
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}
