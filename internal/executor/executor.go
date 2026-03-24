package executor

import (
	"context"

	"github.com/example/agent-infra/internal/model"
)

// TaskStatus type alias for convenience.
type TaskStatus = string

// Task result constants.
const (
	TaskStatusPending         TaskStatus = model.TaskStatusPending
	TaskStatusScheduled       TaskStatus = model.TaskStatusScheduled
	TaskStatusRunning         TaskStatus = model.TaskStatusRunning
	TaskStatusPaused          TaskStatus = model.TaskStatusPaused
	TaskStatusWaitingApproval TaskStatus = model.TaskStatusWaitingApproval
	TaskStatusRetrying        TaskStatus = model.TaskStatusRetrying
	TaskStatusSucceeded       TaskStatus = model.TaskStatusSucceeded
	TaskStatusFailed          TaskStatus = model.TaskStatusFailed
	TaskStatusCancelled       TaskStatus = model.TaskStatusCancelled
)

// Executor defines the interface for task execution operations.
// The Executor is responsible for managing the lifecycle of K8s Jobs
// that run tasks in sandbox environments.
type Executor interface {
	// Execute creates and starts a K8s Job for the given task.
	// It returns JobInfo containing the Job name and initial status.
	Execute(ctx context.Context, task *model.Task) (*JobInfo, error)

	// GetStatus returns the current status of a running Job.
	GetStatus(ctx context.Context, taskID string) (*JobStatus, error)

	// Pause pauses a running Job by calling the Wrapper's pause API.
	Pause(ctx context.Context, taskID string) error

	// Resume resumes a paused Job by calling the Wrapper's resume API.
	Resume(ctx context.Context, taskID string) error

	// Cancel cancels a running Job and cleans up resources.
	Cancel(ctx context.Context, taskID string, reason string) error

	// GetPodAddress returns the Pod IP address for a task's Job.
	// This is used for intervention operations.
	GetPodAddress(ctx context.Context, taskID string) (string, error)

	// Start begins the executor's processing loop.
	Start(ctx context.Context) error

	// Stop gracefully stops the executor.
	Stop(ctx context.Context) error

	// IsRunning returns whether the executor is currently running.
	IsRunning() bool
}

// JobInfo contains information about a K8s Job.
type JobInfo struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	PodName   string    `json:"pod_name"`
	Status    JobStatus `json:"status"`
	CreatedAt int64     `json:"created_at"`
}

// JobStatus represents the status of a K8s Job.
type JobStatus struct {
	Phase          string `json:"phase"`           // Pending, Running, Succeeded, Failed
	Message        string `json:"message"`         // Human-readable message
	StartTime      *int64 `json:"start_time"`      // Unix timestamp
	CompletionTime *int64 `json:"completion_time"` // Unix timestamp
	ExitCode       *int32 `json:"exit_code"`       // Container exit code
}

// JobConfig holds the configuration for creating a K8s Job.
type JobConfig struct {
	// Job naming
	NamePrefix string `yaml:"name_prefix"` // Default: "sandbox-"

	// Namespace configuration
	Namespace string `yaml:"namespace"` // K8s namespace for Jobs

	// Container images
	CLIRunnerImage string `yaml:"cli_runner_image"`
	WrapperImage   string `yaml:"wrapper_image"`
	LogAgentImage  string `yaml:"log_agent_image"`

	// Resource limits
	DefaultCPULimit      string `yaml:"default_cpu_limit"`      // e.g., "2"
	DefaultMemoryLimit   string `yaml:"default_memory_limit"`   // e.g., "4Gi"
	DefaultCPURequest    string `yaml:"default_cpu_request"`    // e.g., "500m"
	DefaultMemoryRequest string `yaml:"default_memory_request"` // e.g., "1Gi"

	// TTL configuration
	TTLSecondsAfterFinished int32 `yaml:"ttl_seconds_after_finished"` // Default: 3600

	// Timeout configuration
	DefaultTimeoutSeconds int64 `yaml:"default_timeout_seconds"` // Default: 3600

	// Wrapper configuration
	WrapperPort int `yaml:"wrapper_port"` // Default: 9090

	// Environment configuration
	ControlPlaneURL string `yaml:"control_plane_url"`

	// Labels and annotations
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

// DefaultJobConfig returns a JobConfig with default values.
func DefaultJobConfig() *JobConfig {
	return &JobConfig{
		NamePrefix:              "sandbox-",
		Namespace:               "sandbox",
		CLIRunnerImage:          "agent-infra/cli-runner:latest",
		WrapperImage:            "agent-infra/wrapper:latest",
		LogAgentImage:           "agent-infra/log-agent:latest",
		DefaultCPULimit:         "2",
		DefaultMemoryLimit:      "4Gi",
		DefaultCPURequest:       "500m",
		DefaultMemoryRequest:    "1Gi",
		TTLSecondsAfterFinished: 3600,
		DefaultTimeoutSeconds:   3600,
		WrapperPort:             9090,
		Labels:                  make(map[string]string),
		Annotations:             make(map[string]string),
	}
}

// ExecutorConfig holds configuration for the TaskExecutor.
type ExecutorConfig struct {
	// JobConfig for creating Jobs
	JobConfig *JobConfig

	// Callbacks for external integration
	UpdateTaskStatus func(ctx context.Context, taskID string, status string, message string) error
	GetTask          func(ctx context.Context, taskID string) (*model.Task, error)
	OnTaskComplete   func(ctx context.Context, taskID string, result map[string]interface{}) error
	OnTaskFailed     func(ctx context.Context, taskID string, err error) error
}
