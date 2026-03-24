package executor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
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
	running  atomic.Bool
	stopCh   chan struct{}
	wg       sync.WaitGroup
	stopOnce sync.Once

	// Observability
	logger  *slog.Logger
	metrics MetricsRecorder
}

// NewTaskExecutor creates a new TaskExecutor instance.
func NewTaskExecutor(k8sClient kubernetes.Interface, redisClient RedisClient, cfg *ExecutorConfig) (*TaskExecutor, error) {
	if redisClient == nil {
		return nil, ErrNilRedisClient
	}

	if cfg == nil {
		cfg = &ExecutorConfig{}
	}
	if cfg.JobConfig == nil {
		cfg.JobConfig = DefaultJobConfig()
	}

	// Initialize logger
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default().With("component", "task_executor")
	}

	jobManager := NewJobManager(k8sClient, cfg.JobConfig)
	wrapperClient := NewWrapperClient(&WrapperClientConfig{
		Port: cfg.JobConfig.WrapperPort,
	})

	heartbeatCfg := &HeartbeatManagerConfig{
		Interval: 5 * time.Second,
		Timeout:  15 * time.Second,
	}
	heartbeat, err := NewHeartbeatManager(redisClient, heartbeatCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create heartbeat manager: %w", err)
	}

	return &TaskExecutor{
		jobManager:    jobManager,
		wrapperClient: wrapperClient,
		heartbeat:     heartbeat,
		config:        cfg,
		stopCh:        make(chan struct{}),
		logger:        logger,
		metrics:       cfg.Metrics, // May be nil, will use no-op if not set
	}, nil
}

// Execute creates and starts a K8s Job for the given task.
func (e *TaskExecutor) Execute(ctx context.Context, task *model.Task) (*JobInfo, error) {
	startTime := time.Now()

	if task == nil {
		return nil, ErrInvalidJobConfig
	}

	taskID := task.ID.String()

	// Validate task ID
	if err := validateTaskID(taskID); err != nil {
		e.logger.Warn("task execution rejected: invalid task ID",
			"task_id", taskID,
			"error", err,
		)
		return nil, err
	}

	// Validate task status
	if !e.canExecute(task) {
		e.logger.Warn("task execution rejected: invalid status",
			"task_id", taskID,
			"status", task.Status,
			"tenant_id", task.TenantID,
		)
		return nil, fmt.Errorf("task cannot be executed in status: %s", task.Status)
	}

	e.logger.Info("starting task execution",
		"task_id", taskID,
		"tenant_id", task.TenantID,
		"status", task.Status,
	)

	// Create the K8s Job
	jobInfo, err := e.jobManager.CreateJob(ctx, task)
	if err != nil {
		e.logger.Error("failed to create K8s job",
			"task_id", taskID,
			"tenant_id", task.TenantID,
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		e.recordMetric("task_execution_failed", taskID, "job_creation_error")
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Register for heartbeat monitoring
	podIP := "" // Will be set when Pod starts
	e.heartbeat.Register(taskID, podIP)

	// Update task status to running
	if e.config.UpdateTaskStatus != nil {
		if err := e.config.UpdateTaskStatus(ctx, taskID, model.TaskStatusRunning, "job created"); err != nil {
			// Log error but don't fail - job is already created
			e.logger.Warn("UpdateTaskStatus callback failed",
				"task_id", taskID,
				"new_status", model.TaskStatusRunning,
				"error", err,
			)
		}
	}

	e.logger.Info("task execution started successfully",
		"task_id", taskID,
		"tenant_id", task.TenantID,
		"job_name", jobInfo.Name,
		"namespace", jobInfo.Namespace,
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	e.recordMetric("task_execution_started", taskID, "")

	return jobInfo, nil
}

// GetStatus returns the current status of a running Job.
func (e *TaskExecutor) GetStatus(ctx context.Context, taskID string) (*JobStatus, error) {
	// Validate task ID
	if err := validateTaskID(taskID); err != nil {
		return nil, err
	}

	return e.jobManager.GetJobStatus(ctx, taskID)
}

// Pause pauses a running Job.
func (e *TaskExecutor) Pause(ctx context.Context, taskID string) error {
	startTime := time.Now()

	// Validate task ID
	if err := validateTaskID(taskID); err != nil {
		e.logger.Warn("pause rejected: invalid task ID",
			"task_id", taskID,
			"error", err,
		)
		return err
	}

	e.logger.Info("pausing task",
		"task_id", taskID,
	)

	// Get the task to check status
	if e.config.GetTask != nil {
		task, err := e.config.GetTask(ctx, taskID)
		if err != nil {
			e.logger.Error("failed to get task for pause",
				"task_id", taskID,
				"error", err,
			)
			return fmt.Errorf("failed to get task: %w", err)
		}
		if task.Status != model.TaskStatusRunning {
			e.logger.Warn("pause rejected: task not running",
				"task_id", taskID,
				"current_status", task.Status,
			)
			return ErrTaskNotRunning
		}
	}

	// Get Pod IP
	podIP, err := e.jobManager.GetPodAddress(ctx, taskID)
	if err != nil {
		e.logger.Error("failed to get pod address for pause",
			"task_id", taskID,
			"error", err,
		)
		return fmt.Errorf("failed to get pod address: %w", err)
	}

	// Call Wrapper pause API
	if err := e.wrapperClient.Pause(ctx, podIP); err != nil {
		e.logger.Error("failed to pause wrapper",
			"task_id", taskID,
			"pod_ip", podIP,
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return fmt.Errorf("failed to pause wrapper: %w", err)
	}

	// Update task status
	if e.config.UpdateTaskStatus != nil {
		if err := e.config.UpdateTaskStatus(ctx, taskID, model.TaskStatusPaused, "task paused"); err != nil {
			// Log error but don't fail
			e.logger.Warn("UpdateTaskStatus callback failed for pause",
				"task_id", taskID,
				"new_status", model.TaskStatusPaused,
				"error", err,
			)
		}
	}

	e.logger.Info("task paused successfully",
		"task_id", taskID,
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	return nil
}

// Resume resumes a paused Job.
func (e *TaskExecutor) Resume(ctx context.Context, taskID string) error {
	startTime := time.Now()

	// Validate task ID
	if err := validateTaskID(taskID); err != nil {
		e.logger.Warn("resume rejected: invalid task ID",
			"task_id", taskID,
			"error", err,
		)
		return err
	}

	e.logger.Info("resuming task",
		"task_id", taskID,
	)

	// Get the task to check status
	if e.config.GetTask != nil {
		task, err := e.config.GetTask(ctx, taskID)
		if err != nil {
			e.logger.Error("failed to get task for resume",
				"task_id", taskID,
				"error", err,
			)
			return fmt.Errorf("failed to get task: %w", err)
		}
		if task.Status != model.TaskStatusPaused {
			e.logger.Warn("resume rejected: task not paused",
				"task_id", taskID,
				"current_status", task.Status,
			)
			return ErrTaskNotPaused
		}
	}

	// Get Pod IP
	podIP, err := e.jobManager.GetPodAddress(ctx, taskID)
	if err != nil {
		e.logger.Error("failed to get pod address for resume",
			"task_id", taskID,
			"error", err,
		)
		return fmt.Errorf("failed to get pod address: %w", err)
	}

	// Call Wrapper resume API
	if err := e.wrapperClient.Resume(ctx, podIP); err != nil {
		e.logger.Error("failed to resume wrapper",
			"task_id", taskID,
			"pod_ip", podIP,
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return fmt.Errorf("failed to resume wrapper: %w", err)
	}

	// Update task status
	if e.config.UpdateTaskStatus != nil {
		if err := e.config.UpdateTaskStatus(ctx, taskID, model.TaskStatusRunning, "task resumed"); err != nil {
			// Log error but don't fail
			e.logger.Warn("UpdateTaskStatus callback failed for resume",
				"task_id", taskID,
				"new_status", model.TaskStatusRunning,
				"error", err,
			)
		}
	}

	e.logger.Info("task resumed successfully",
		"task_id", taskID,
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	return nil
}

// Cancel cancels a running Job and cleans up resources.
func (e *TaskExecutor) Cancel(ctx context.Context, taskID string, reason string) error {
	startTime := time.Now()

	// Validate task ID
	if err := validateTaskID(taskID); err != nil {
		e.logger.Warn("cancel rejected: invalid task ID",
			"task_id", taskID,
			"error", err,
		)
		return err
	}

	e.logger.Info("cancelling task",
		"task_id", taskID,
		"reason", reason,
	)

	// Delete the K8s Job
	if err := e.jobManager.DeleteJob(ctx, taskID); err != nil {
		e.logger.Error("failed to delete K8s job during cancel",
			"task_id", taskID,
			"error", err,
		)
		return fmt.Errorf("failed to delete job: %w", err)
	}

	// Unregister from heartbeat monitoring
	e.heartbeat.Unregister(taskID)

	// Update task status
	if e.config.UpdateTaskStatus != nil {
		if err := e.config.UpdateTaskStatus(ctx, taskID, model.TaskStatusCancelled, reason); err != nil {
			// Log error but don't fail
			e.logger.Warn("UpdateTaskStatus callback failed for cancel",
				"task_id", taskID,
				"new_status", model.TaskStatusCancelled,
				"error", err,
			)
		}
	}

	e.logger.Info("task cancelled successfully",
		"task_id", taskID,
		"reason", reason,
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	e.recordMetric("task_cancelled", taskID, reason)

	return nil
}

// GetPodAddress returns the Pod IP address for a task's Job.
func (e *TaskExecutor) GetPodAddress(ctx context.Context, taskID string) (string, error) {
	// Validate task ID
	if err := validateTaskID(taskID); err != nil {
		return "", err
	}

	return e.jobManager.GetPodAddress(ctx, taskID)
}

// Start begins the executor's processing loop.
func (e *TaskExecutor) Start(ctx context.Context) error {
	if e.running.Load() {
		e.logger.Warn("executor already running")
		return ErrExecutorAlreadyRunning
	}

	e.logger.Info("starting task executor")

	e.running.Store(true)

	// Start heartbeat monitoring
	if err := e.heartbeat.Start(ctx); err != nil {
		e.logger.Error("failed to start heartbeat manager",
			"error", err,
		)
		return fmt.Errorf("failed to start heartbeat manager: %w", err)
	}

	e.logger.Info("task executor started successfully")

	return nil
}

// Stop gracefully stops the executor.
// It uses sync.Once to ensure cleanup operations are only performed once,
// even if Stop is called multiple times concurrently.
func (e *TaskExecutor) Stop(ctx context.Context) error {
	var stopErr error
	e.stopOnce.Do(func() {
		e.logger.Info("stopping task executor")

		if !e.running.Load() {
			e.logger.Debug("executor already stopped")
			return
		}

		e.running.Store(false)
		close(e.stopCh)

		// Stop heartbeat monitoring
		if err := e.heartbeat.Stop(ctx); err != nil {
			e.logger.Error("failed to stop heartbeat manager",
				"error", err,
			)
			stopErr = fmt.Errorf("failed to stop heartbeat manager: %w", err)
			return
		}

		// Wait for any in-progress operations
		done := make(chan struct{})
		go func() {
			e.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			e.logger.Info("task executor stopped successfully")
		case <-ctx.Done():
			e.logger.Warn("task executor stop timed out",
				"error", ctx.Err(),
			)
			stopErr = ctx.Err()
		}
	})
	return stopErr
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

// validateTaskID validates that a taskID is a valid non-empty UUID.
func validateTaskID(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("%w: task ID cannot be empty", ErrInvalidTaskID)
	}
	if _, err := uuid.Parse(taskID); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTaskID, err)
	}
	return nil
}

// HandleHeartbeat handles a heartbeat from the Wrapper.
func (e *TaskExecutor) HandleHeartbeat(ctx context.Context, taskID string, status string, progress int) error {
	// Validate task ID
	if err := validateTaskID(taskID); err != nil {
		return err
	}

	return e.heartbeat.UpdateHeartbeat(ctx, taskID, status, progress)
}

// HandleTaskEvent handles an event from the Wrapper.
func (e *TaskExecutor) HandleTaskEvent(ctx context.Context, taskID string, eventType string, payload map[string]interface{}) error {
	// Validate task ID
	if err := validateTaskID(taskID); err != nil {
		e.logger.Warn("task event rejected: invalid task ID",
			"task_id", taskID,
			"event_type", eventType,
			"error", err,
		)
		return err
	}

	e.logger.Debug("handling task event",
		"task_id", taskID,
		"event_type", eventType,
	)

	switch eventType {
	case "status_change":
		if status, ok := payload["status"].(string); ok {
			e.logger.Info("task status change event",
				"task_id", taskID,
				"new_status", status,
			)
			if e.config.UpdateTaskStatus != nil {
				message := ""
				if m, ok := payload["message"].(string); ok {
					message = m
				}
				if err := e.config.UpdateTaskStatus(ctx, taskID, status, message); err != nil {
					e.logger.Error("UpdateTaskStatus callback failed for status_change",
						"task_id", taskID,
						"status", status,
						"error", err,
					)
					return fmt.Errorf("UpdateTaskStatus callback failed for task %s: %w", taskID, err)
				}
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
		e.logger.Debug("heartbeat event received",
			"task_id", taskID,
			"status", status,
			"progress", progress,
		)
		e.heartbeat.UpdateHeartbeat(ctx, taskID, status, progress)

	case "complete":
		e.logger.Info("task completed event",
			"task_id", taskID,
		)
		e.heartbeat.Unregister(taskID)
		e.recordMetric("task_completed", taskID, "")
		if e.config.OnTaskComplete != nil {
			result := make(map[string]interface{})
			if r, ok := payload["result"].(map[string]interface{}); ok {
				result = r
			}
			if err := e.config.OnTaskComplete(ctx, taskID, result); err != nil {
				e.logger.Error("OnTaskComplete callback failed",
					"task_id", taskID,
					"error", err,
				)
				return fmt.Errorf("OnTaskComplete callback failed for task %s: %w", taskID, err)
			}
		}

	case "failed":
		errMsg := ""
		if m, ok := payload["error"].(string); ok {
			errMsg = m
		}
		e.logger.Warn("task failed event",
			"task_id", taskID,
			"error", errMsg,
		)
		e.heartbeat.Unregister(taskID)
		e.recordMetric("task_failed", taskID, errMsg)
		if e.config.OnTaskFailed != nil {
			if err := e.config.OnTaskFailed(ctx, taskID, fmt.Errorf(errMsg)); err != nil {
				e.logger.Error("OnTaskFailed callback failed",
					"task_id", taskID,
					"error", err,
				)
				return fmt.Errorf("OnTaskFailed callback failed for task %s: %w", taskID, err)
			}
		}
	}

	return nil
}

// InjectInstruction injects an instruction into a running task.
func (e *TaskExecutor) InjectInstruction(ctx context.Context, taskID string, content string) error {
	// Validate task ID
	if err := validateTaskID(taskID); err != nil {
		return err
	}

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
