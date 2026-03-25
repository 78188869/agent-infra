package executor

import (
	"context"
	"log/slog"

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

// SecurityConfig holds security-related configuration for Pods and containers.
type SecurityConfig struct {
	// RunAsNonRoot indicates that the container must run as a non-root user.
	RunAsNonRoot bool `yaml:"run_as_non_root"` // Default: true

	// RunAsUser is the UID to run the entrypoint of the container process.
	RunAsUser int64 `yaml:"run_as_user"` // Default: 1000

	// RunAsGroup is the GID to run the entrypoint of the container process.
	RunAsGroup int64 `yaml:"run_as_group"` // Default: 1000

	// ReadOnlyRootFilesystem mounts the container's root filesystem as read-only.
	ReadOnlyRootFilesystem bool `yaml:"read_only_root_filesystem"` // Default: true

	// AllowPrivilegeEscalation controls whether a process can gain more privileges.
	// Default is false for security.
	AllowPrivilegeEscalation *bool `yaml:"allow_privilege_escalation"` // Default: false

	// FSGroup is a special supplemental group that applies to all containers in a pod.
	FSGroup *int64 `yaml:"fs_group"` // Default: 1000
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

	// Resource limits for CLI Runner container
	DefaultCPULimit      string `yaml:"default_cpu_limit"`      // e.g., "2"
	DefaultMemoryLimit   string `yaml:"default_memory_limit"`   // e.g., "4Gi"
	DefaultCPURequest    string `yaml:"default_cpu_request"`    // e.g., "500m"
	DefaultMemoryRequest string `yaml:"default_memory_request"` // e.g., "1Gi"

	// Resource limits for Wrapper container
	WrapperCPULimit      string `yaml:"wrapper_cpu_limit"`      // e.g., "100m"
	WrapperMemoryLimit   string `yaml:"wrapper_memory_limit"`   // e.g., "128Mi"
	WrapperCPURequest    string `yaml:"wrapper_cpu_request"`    // e.g., "50m"
	WrapperMemoryRequest string `yaml:"wrapper_memory_request"` // e.g., "64Mi"

	// TTL configuration
	TTLSecondsAfterFinished int32 `yaml:"ttl_seconds_after_finished"` // Default: 3600

	// Timeout configuration
	DefaultTimeoutSeconds int64 `yaml:"default_timeout_seconds"` // Default: 3600

	// Wrapper configuration
	WrapperPort int `yaml:"wrapper_port"` // Default: 9090

	// Environment configuration
	ControlPlaneURL string `yaml:"control_plane_url"`

	// Security configuration for Pods and containers
	// If nil, secure defaults will be used.
	Security *SecurityConfig `yaml:"security"`

	// ServiceAccountName is the name of the ServiceAccount to use for the Pod.
	// If empty, the namespace's default ServiceAccount will be used.
	ServiceAccountName string `yaml:"service_account_name"`

	// Labels and annotations
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

// DefaultSecurityConfig returns a SecurityConfig with secure default values.
func DefaultSecurityConfig() *SecurityConfig {
	allowPrivilegeEscalation := false
	fsGroup := int64(1000)
	return &SecurityConfig{
		RunAsNonRoot:            true,
		RunAsUser:               1000,
		RunAsGroup:              1000,
		ReadOnlyRootFilesystem:  true,
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		FSGroup:                 &fsGroup,
	}
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
		WrapperCPULimit:         "100m",
		WrapperMemoryLimit:      "128Mi",
		WrapperCPURequest:       "50m",
		WrapperMemoryRequest:    "64Mi",
		TTLSecondsAfterFinished: 3600,
		DefaultTimeoutSeconds:   3600,
		WrapperPort:             9090,
		Security:                DefaultSecurityConfig(),
		Labels:                  make(map[string]string),
		Annotations:             make(map[string]string),
	}
}

// MetricsRecorder defines the interface for recording metrics.
type MetricsRecorder interface {
	RecordTaskExecution(taskID string, status string)
	RecordTaskCancelled(taskID string, reason string)
	RecordTaskCompleted(taskID string)
	RecordTaskFailed(taskID string, errMsg string)
}

// ExecutorConfig holds configuration for the TaskExecutor.
type ExecutorConfig struct {
	// JobConfig for creating Jobs
	JobConfig *JobConfig

	// Logger for structured logging
	Logger *slog.Logger

	// Metrics for recording observability data
	Metrics MetricsRecorder

	// Callbacks for external integration
	UpdateTaskStatus func(ctx context.Context, taskID string, status string, message string) error
	GetTask          func(ctx context.Context, taskID string) (*model.Task, error)
	OnTaskComplete   func(ctx context.Context, taskID string, result map[string]interface{}) error
	OnTaskFailed     func(ctx context.Context, taskID string, err error) error
}
