package executor

import "errors"

// Executor errors
var (
	// ErrJobNotFound indicates the Job was not found in K8s.
	ErrJobNotFound = errors.New("job not found")

	// ErrPodNotFound indicates the Pod was not found for a task.
	ErrPodNotFound = errors.New("pod not found")

	// ErrTaskNotRunning indicates the task is not in a running state.
	ErrTaskNotRunning = errors.New("task is not running")

	// ErrTaskNotPaused indicates the task is not in a paused state.
	ErrTaskNotPaused = errors.New("task is not paused")

	// ErrJobAlreadyExists indicates a Job already exists for the task.
	ErrJobAlreadyExists = errors.New("job already exists for task")

	// ErrExecutorNotRunning indicates the executor is not running.
	ErrExecutorNotRunning = errors.New("executor is not running")

	// ErrExecutorAlreadyRunning indicates the executor is already running.
	ErrExecutorAlreadyRunning = errors.New("executor is already running")

	// ErrInvalidJobConfig indicates invalid Job configuration.
	ErrInvalidJobConfig = errors.New("invalid job configuration")

	// ErrWrapperUnavailable indicates the Wrapper is not responding.
	ErrWrapperUnavailable = errors.New("wrapper is unavailable")

	// ErrHeartbeatTimeout indicates the heartbeat was not received in time.
	ErrHeartbeatTimeout = errors.New("heartbeat timeout")

	// ErrJobCreationFailed indicates Job creation failed.
	ErrJobCreationFailed = errors.New("failed to create job")

	// ErrJobDeletionFailed indicates Job deletion failed.
	ErrJobDeletionFailed = errors.New("failed to delete job")
)
