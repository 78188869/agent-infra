package executor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"k8s.io/client-go/kubernetes"

	"github.com/example/agent-infra/internal/model"
)

// RedisClient interface for Redis operations (allows mocking).
type RedisClient interface {
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HGet(ctx context.Context, key, field string) *redis.StringCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	Pipeline() redis.Pipeliner
}

// TaskExecutor implements the Executor interface using K8s Jobs.
type TaskExecutor struct {
	jobManager    *JobManager
	wrapperClient *WrapperClient
	heartbeat     *HeartbeatManager

	// Configuration
	config *ExecutorConfig

	// State
	running atomic.Bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewTaskExecutor creates a new TaskExecutor instance.
func NewTaskExecutor(k8sClient kubernetes.Interface, redisClient RedisClient, cfg *ExecutorConfig) (*TaskExecutor, error) {
	if cfg == nil {
		cfg = &ExecutorConfig{}
	}
	if cfg.JobConfig == nil {
		cfg.JobConfig = DefaultJobConfig()
	}

	jobManager := NewJobManager(k8sClient, cfg.JobConfig)
	wrapperClient := NewWrapperClient(&WrapperClientConfig{
		Port: cfg.JobConfig.WrapperPort,
	})

	heartbeatCfg := &HeartbeatManagerConfig{
		Interval: 5 * time.Second,
		Timeout:  15 * time.Second,
	}
	heartbeat := NewHeartbeatManager(redisClient, heartbeatCfg)

	return &TaskExecutor{
		jobManager:    jobManager,
		wrapperClient: wrapperClient,
		heartbeat:     heartbeat,
		config:        cfg,
		stopCh:        make(chan struct{}),
	}, nil
}

// Execute creates and starts a K8s Job for the given task.
func (e *TaskExecutor) Execute(ctx context.Context, task *model.Task) (*JobInfo, error) {
	if task == nil {
		return nil, ErrInvalidJobConfig
	}

	// Validate task status
	if !e.canExecute(task) {
		return nil, fmt.Errorf("task cannot be executed in status: %s", task.Status)
	}

	// Create the K8s Job
	jobInfo, err := e.jobManager.CreateJob(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Register for heartbeat monitoring
	taskID := task.ID.String()
	podIP := "" // Will be set when Pod starts
	e.heartbeat.Register(taskID, podIP)

	// Update task status to running
	if e.config.UpdateTaskStatus != nil {
		if err := e.config.UpdateTaskStatus(ctx, taskID, model.TaskStatusRunning, "job created"); err != nil {
			// Log error but don't fail - job is already created
		}
	}

	return jobInfo, nil
}

// GetStatus returns the current status of a running Job.
func (e *TaskExecutor) GetStatus(ctx context.Context, taskID string) (*JobStatus, error) {
	return e.jobManager.GetJobStatus(ctx, taskID)
}

// Pause pauses a running Job.
func (e *TaskExecutor) Pause(ctx context.Context, taskID string) error {
	// Get the task to check status
	if e.config.GetTask != nil {
		task, err := e.config.GetTask(ctx, taskID)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}
		if task.Status != model.TaskStatusRunning {
			return ErrTaskNotRunning
		}
	}

	// Get Pod IP
	podIP, err := e.jobManager.GetPodAddress(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get pod address: %w", err)
	}

	// Call Wrapper pause API
	if err := e.wrapperClient.Pause(ctx, podIP); err != nil {
		return fmt.Errorf("failed to pause wrapper: %w", err)
	}

	// Update task status
	if e.config.UpdateTaskStatus != nil {
		if err := e.config.UpdateTaskStatus(ctx, taskID, model.TaskStatusPaused, "task paused"); err != nil {
			// Log error but don't fail
		}
	}

	return nil
}

// Resume resumes a paused Job.
func (e *TaskExecutor) Resume(ctx context.Context, taskID string) error {
	// Get the task to check status
	if e.config.GetTask != nil {
		task, err := e.config.GetTask(ctx, taskID)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}
		if task.Status != model.TaskStatusPaused {
			return ErrTaskNotPaused
		}
	}

	// Get Pod IP
	podIP, err := e.jobManager.GetPodAddress(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get pod address: %w", err)
	}

	// Call Wrapper resume API
	if err := e.wrapperClient.Resume(ctx, podIP); err != nil {
		return fmt.Errorf("failed to resume wrapper: %w", err)
	}

	// Update task status
	if e.config.UpdateTaskStatus != nil {
		if err := e.config.UpdateTaskStatus(ctx, taskID, model.TaskStatusRunning, "task resumed"); err != nil {
			// Log error but don't fail
		}
	}

	return nil
}

// Cancel cancels a running Job and cleans up resources.
func (e *TaskExecutor) Cancel(ctx context.Context, taskID string, reason string) error {
	// Delete the K8s Job
	if err := e.jobManager.DeleteJob(ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	// Unregister from heartbeat monitoring
	e.heartbeat.Unregister(taskID)

	// Update task status
	if e.config.UpdateTaskStatus != nil {
		if err := e.config.UpdateTaskStatus(ctx, taskID, model.TaskStatusCancelled, reason); err != nil {
			// Log error but don't fail
		}
	}

	return nil
}

// GetPodAddress returns the Pod IP address for a task's Job.
func (e *TaskExecutor) GetPodAddress(ctx context.Context, taskID string) (string, error) {
	return e.jobManager.GetPodAddress(ctx, taskID)
}

// Start begins the executor's processing loop.
func (e *TaskExecutor) Start(ctx context.Context) error {
	if e.running.Load() {
		return ErrExecutorAlreadyRunning
	}

	e.running.Store(true)

	// Start heartbeat monitoring
	if err := e.heartbeat.Start(ctx); err != nil {
		return fmt.Errorf("failed to start heartbeat manager: %w", err)
	}

	return nil
}

// Stop gracefully stops the executor.
func (e *TaskExecutor) Stop(ctx context.Context) error {
	if !e.running.Load() {
		return nil
	}

	e.running.Store(false)
	close(e.stopCh)

	// Stop heartbeat monitoring
	if err := e.heartbeat.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop heartbeat manager: %w", err)
	}

	// Wait for any in-progress operations
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IsRunning returns whether the executor is currently running.
func (e *TaskExecutor) IsRunning() bool {
	return e.running.Load()
}

// canExecute checks if a task can be executed.
func (e *TaskExecutor) canExecute(task *model.Task) bool {
	return task.Status == model.TaskStatusScheduled ||
		task.Status == model.TaskStatusPending ||
		task.Status == model.TaskStatusRetrying
}

// HandleHeartbeat handles a heartbeat from the Wrapper.
func (e *TaskExecutor) HandleHeartbeat(ctx context.Context, taskID string, status string, progress int) error {
	return e.heartbeat.UpdateHeartbeat(ctx, taskID, status, progress)
}

// HandleTaskEvent handles an event from the Wrapper.
func (e *TaskExecutor) HandleTaskEvent(ctx context.Context, taskID string, eventType string, payload map[string]interface{}) error {
	switch eventType {
	case "status_change":
		if status, ok := payload["status"].(string); ok {
			if e.config.UpdateTaskStatus != nil {
				message := ""
				if m, ok := payload["message"].(string); ok {
					message = m
				}
				e.config.UpdateTaskStatus(ctx, taskID, status, message)
			}
		}

	case "heartbeat":
		status := ""
		progress := 0
		if s, ok := payload["status"].(string); ok {
			status = s
		}
		if p, ok := payload["progress"].(float64); ok {
			progress = int(p)
		}
		e.heartbeat.UpdateHeartbeat(ctx, taskID, status, progress)

	case "complete":
		e.heartbeat.Unregister(taskID)
		if e.config.OnTaskComplete != nil {
			result := make(map[string]interface{})
			if r, ok := payload["result"].(map[string]interface{}); ok {
				result = r
			}
			e.config.OnTaskComplete(ctx, taskID, result)
		}

	case "failed":
		e.heartbeat.Unregister(taskID)
		if e.config.OnTaskFailed != nil {
			errMsg := ""
			if m, ok := payload["error"].(string); ok {
				errMsg = m
			}
			e.config.OnTaskFailed(ctx, taskID, fmt.Errorf(errMsg))
		}
	}

	return nil
}

// InjectInstruction injects an instruction into a running task.
func (e *TaskExecutor) InjectInstruction(ctx context.Context, taskID string, content string) error {
	// Get the task to check status
	if e.config.GetTask != nil {
		task, err := e.config.GetTask(ctx, taskID)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}
		if task.Status != model.TaskStatusRunning {
			return ErrTaskNotRunning
		}
	}

	// Get Pod IP
	podIP, err := e.jobManager.GetPodAddress(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get pod address: %w", err)
	}

	// Call Wrapper inject API
	if err := e.wrapperClient.Inject(ctx, podIP, content); err != nil {
		return fmt.Errorf("failed to inject instruction: %w", err)
	}

	return nil
}

// GetJobManager returns the JobManager for direct access.
func (e *TaskExecutor) GetJobManager() *JobManager {
	return e.jobManager
}

// GetHeartbeatManager returns the HeartbeatManager for direct access.
func (e *TaskExecutor) GetHeartbeatManager() *HeartbeatManager {
	return e.heartbeat
}
