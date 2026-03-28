package executor

import (
	"context"
	"testing"

	"github.com/example/agent-infra/internal/model"
)

func TestDefaultJobConfig(t *testing.T) {
	cfg := DefaultJobConfig()

	if cfg.NamePrefix != "sandbox-" {
		t.Errorf("expected NamePrefix 'sandbox-', got %s", cfg.NamePrefix)
	}
	if cfg.Namespace != "sandbox" {
		t.Errorf("expected Namespace 'sandbox', got %s", cfg.Namespace)
	}
	if cfg.CLIRunnerImage != "agent-infra/cli-runner:latest" {
		t.Errorf("unexpected CLIRunnerImage: %s", cfg.CLIRunnerImage)
	}
	if cfg.WrapperImage != "agent-infra/wrapper:latest" {
		t.Errorf("unexpected WrapperImage: %s", cfg.WrapperImage)
	}
	if cfg.WrapperPort != 9090 {
		t.Errorf("expected WrapperPort 9090, got %d", cfg.WrapperPort)
	}
	if cfg.TTLSecondsAfterFinished != 3600 {
		t.Errorf("expected TTLSecondsAfterFinished 3600, got %d", cfg.TTLSecondsAfterFinished)
	}
	if cfg.DefaultTimeoutSeconds != 3600 {
		t.Errorf("expected DefaultTimeoutSeconds 3600, got %d", cfg.DefaultTimeoutSeconds)
	}
}

func TestJobInfo(t *testing.T) {
	info := &JobInfo{
		Name:      "test-job",
		Namespace: "default",
		PodName:   "test-pod",
		Status: JobStatus{
			Phase: "Running",
		},
		CreatedAt: 1234567890,
	}

	if info.Name != "test-job" {
		t.Errorf("expected Name 'test-job', got %s", info.Name)
	}
	if info.Namespace != "default" {
		t.Errorf("expected Namespace 'default', got %s", info.Namespace)
	}
	if info.Status.Phase != "Running" {
		t.Errorf("expected Status.Phase 'Running', got %s", info.Status.Phase)
	}
}

func TestJobStatus(t *testing.T) {
	startTime := int64(1234567890)
	completionTime := int64(1234567900)
	exitCode := int32(0)

	status := &JobStatus{
		Phase:          "Succeeded",
		Message:        "Job completed successfully",
		StartTime:      &startTime,
		CompletionTime: &completionTime,
		ExitCode:       &exitCode,
	}

	if status.Phase != "Succeeded" {
		t.Errorf("expected Phase 'Succeeded', got %s", status.Phase)
	}
	if *status.StartTime != startTime {
		t.Errorf("expected StartTime %d, got %d", startTime, *status.StartTime)
	}
	if *status.ExitCode != exitCode {
		t.Errorf("expected ExitCode %d, got %d", exitCode, *status.ExitCode)
	}
}

func TestExecutorConfig(t *testing.T) {
	cfg := &ExecutorConfig{
		WrapperPort: 9090,
		UpdateTaskStatus: func(ctx context.Context, taskID string, status string, message string) error {
			return nil
		},
	}

	if cfg.WrapperPort != 9090 {
		t.Errorf("expected WrapperPort 9090, got %d", cfg.WrapperPort)
	}
	if cfg.UpdateTaskStatus == nil {
		t.Error("UpdateTaskStatus should not be nil")
	}
}

func TestTaskStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant TaskStatus
		expected string
	}{
		{"Pending", TaskStatusPending, model.TaskStatusPending},
		{"Scheduled", TaskStatusScheduled, model.TaskStatusScheduled},
		{"Running", TaskStatusRunning, model.TaskStatusRunning},
		{"Paused", TaskStatusPaused, model.TaskStatusPaused},
		{"WaitingApproval", TaskStatusWaitingApproval, model.TaskStatusWaitingApproval},
		{"Retrying", TaskStatusRetrying, model.TaskStatusRetrying},
		{"Succeeded", TaskStatusSucceeded, model.TaskStatusSucceeded},
		{"Failed", TaskStatusFailed, model.TaskStatusFailed},
		{"Cancelled", TaskStatusCancelled, model.TaskStatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s: expected %s, got %s", tt.name, tt.expected, tt.constant)
			}
		})
	}
}

func TestJobConfigDefaults(t *testing.T) {
	cfg := &JobConfig{}

	// Test that zero values are valid
	if cfg.NamePrefix != "" {
		t.Errorf("expected empty NamePrefix, got %s", cfg.NamePrefix)
	}

	// Test default creation
	defaultCfg := DefaultJobConfig()
	if defaultCfg.Labels == nil {
		t.Error("Labels should be initialized")
	}
	if defaultCfg.Annotations == nil {
		t.Error("Annotations should be initialized")
	}
}
