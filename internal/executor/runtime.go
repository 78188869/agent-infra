package executor

import (
	"context"

	"github.com/example/agent-infra/internal/model"
)

// ContainerRuntime abstracts containerized execution environment operations.
// Implementations may use K8s Jobs, Docker containers, or other runtimes.
type ContainerRuntime interface {
	// Create creates a containerized execution environment for the task.
	Create(ctx context.Context, task *model.Task) (*RuntimeInfo, error)

	// GetStatus returns the current status of the execution environment.
	GetStatus(ctx context.Context, taskID string) (*RuntimeStatus, error)

	// Delete removes the execution environment and cleans up resources.
	Delete(ctx context.Context, taskID string) error

	// GetAddress returns the network address for reaching the runtime.
	// The format depends on the runtime (e.g., Pod IP, container hostname).
	GetAddress(ctx context.Context, taskID string) (string, error)
}

// RuntimeInfo contains information about a created runtime environment.
type RuntimeInfo struct {
	Name      string        `json:"name"`
	Namespace string        `json:"namespace"`
	Status    RuntimeStatus `json:"status"`
	CreatedAt int64         `json:"created_at"`
}

// RuntimeStatus represents the status of a runtime environment.
type RuntimeStatus struct {
	Phase          string `json:"phase"`           // Pending, Running, Succeeded, Failed
	Message        string `json:"message"`         // Human-readable message
	StartTime      *int64 `json:"start_time"`      // Unix timestamp
	CompletionTime *int64 `json:"completion_time"` // Unix timestamp
	ExitCode       *int32 `json:"exit_code"`       // Container exit code
}
