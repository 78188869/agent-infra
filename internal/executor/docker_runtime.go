package executor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/example/agent-infra/internal/model"
)

// DockerRuntime implements ContainerRuntime using Docker Compose.
type DockerRuntime struct {
	compose *ComposeManager
	logger  *slog.Logger
}

// NewDockerRuntime creates a new DockerRuntime instance.
func NewDockerRuntime(cfg *DockerConfig) (*DockerRuntime, error) {
	cm, err := NewComposeManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create compose manager: %w", err)
	}
	return &DockerRuntime{
		compose: cm,
		logger:  slog.Default().With("component", "docker_runtime"),
	}, nil
}

// Create generates compose config and starts containers for the task.
func (r *DockerRuntime) Create(ctx context.Context, task *model.Task) (*RuntimeInfo, error) {
	if task == nil {
		return nil, ErrInvalidJobConfig
	}

	taskID := task.ID.String()

	envVars := map[string]string{
		"GIT_REPO_URL": "",
		"TASK_PROMPT":  "",
	}
	if err := r.compose.GenerateConfig(ctx, taskID, envVars); err != nil {
		return nil, fmt.Errorf("failed to generate compose config: %w", err)
	}

	if err := r.compose.Up(ctx, taskID); err != nil {
		return nil, fmt.Errorf("failed to start containers: %w", err)
	}

	now := time.Now().Unix()
	return &RuntimeInfo{
		Name:      "task-" + taskID,
		Namespace: "docker",
		Status: RuntimeStatus{
			Phase:     "Running",
			StartTime: &now,
		},
		CreatedAt: now,
	}, nil
}

// GetStatus returns the aggregated status of containers for the task.
func (r *DockerRuntime) GetStatus(ctx context.Context, taskID string) (*RuntimeStatus, error) {
	statuses, err := r.compose.GetStatus(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container status: %w", err)
	}

	if len(statuses) == 0 {
		return nil, ErrJobNotFound
	}

	phase := "Running"
	for _, s := range statuses {
		p := mapDockerStateToPhase(s.State)
		if p == "Failed" {
			phase = "Failed"
			break
		}
		if p != "Running" && p != "Succeeded" {
			phase = p
		}
	}

	return &RuntimeStatus{
		Phase: phase,
	}, nil
}

// Delete stops and removes containers for the task.
func (r *DockerRuntime) Delete(ctx context.Context, taskID string) error {
	return r.compose.Down(ctx, taskID)
}

// GetAddress returns the wrapper HTTP address for the task.
func (r *DockerRuntime) GetAddress(ctx context.Context, taskID string) (string, error) {
	port, err := r.compose.GetServicePort(ctx, taskID, "wrapper", 9090)
	if err != nil {
		return "", fmt.Errorf("failed to get wrapper address: %w", err)
	}
	return fmt.Sprintf("http://localhost:%d", port), nil
}

// mapDockerStateToPhase converts a Docker container state to a runtime phase.
func mapDockerStateToPhase(state string) string {
	switch state {
	case "running":
		return "Running"
	case "exited":
		return "Succeeded"
	case "paused":
		return "Paused"
	case "dead", "removing":
		return "Failed"
	case "created", "restarting":
		return "Pending"
	default:
		return "Unknown"
	}
}
