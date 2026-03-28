package executor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

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
	runtime       ContainerRuntime
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
func NewTaskExecutor(runtime ContainerRuntime, redisClient RedisClient, cfg *ExecutorConfig) (*TaskExecutor, error) {
	if runtime == nil {
		return nil, ErrNilContainerRuntime
	}
	if redisClient == nil {
		return nil, ErrNilRedisClient
	}

	if cfg == nil {
		cfg = &ExecutorConfig{}
	}

	// Initialize logger
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default().With("component", "task_executor")
	}

	wrapperPort := cfg.WrapperPort
	if wrapperPort == 0 {
		wrapperPort = 9090
	}
	wrapperClient := NewWrapperClient(&WrapperClientConfig{
		Port: wrapperPort,
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
		runtime:       runtime,
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

	// Validate task ID is not zero UUID
	if task.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: task ID cannot be nil UUID", ErrInvalidTaskID)
	}

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

	// Create the runtime environment
	runtimeInfo, err := e.runtime.Create(ctx, task)
	if err != nil {
		e.logger.Error("failed to create runtime environment",
			"task_id", taskID,
			"tenant_id", task.TenantID,
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		e.recordMetric("task_execution_failed", taskID, "job_creation_error")
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	jobInfo := &JobInfo{
		Name:      runtimeInfo.Name,
		Namespace: runtimeInfo.Namespace,
		Status:    JobStatus{Phase: runtimeInfo.Status.Phase},
		CreatedAt: runtimeInfo.CreatedAt,
	}

	// Register for heartbeat monitoring
	address := "" // Will be resolved when runtime starts
	e.heartbeat.Register(taskID, address)

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

	runtimeStatus, err := e.runtime.GetStatus(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return &JobStatus{
		Phase:          runtimeStatus.Phase,
		Message:        runtimeStatus.Message,
		StartTime:      runtimeStatus.StartTime,
		CompletionTime: runtimeStatus.CompletionTime,
		ExitCode:       runtimeStatus.ExitCode,
	}, nil
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

	// Get runtime address
	address, err := e.runtime.GetAddress(ctx, taskID)
	if err != nil {
		e.logger.Error("failed to get runtime address for pause",
			"task_id", taskID,
			"error", err,
		)
		return fmt.Errorf("failed to get runtime address: %w", err)
	}

	// Try Interrupt first (Agent SDK wrapper), fall back to Pause (legacy K8s)
	err = e.wrapperClient.Interrupt(ctx, address)
	if err != nil {
		// Fallback to legacy pause for K8s runtime
		e.logger.Warn("Interrupt failed, falling back to legacy Pause",
			"task_id", taskID,
			"address", address,
			"error", err,
		)
		err = e.wrapperClient.Pause(ctx, address)
	}
	if err != nil {
		e.logger.Error("failed to pause wrapper",
			"task_id", taskID,
			"address", address,
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
// For Agent SDK wrapper, use InjectInstruction to send new instructions to an interrupted agent.
// This method is kept for backward compatibility with K8s runtime.
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

	// Get runtime address
	address, err := e.runtime.GetAddress(ctx, taskID)
	if err != nil {
		e.logger.Error("failed to get runtime address for resume",
			"task_id", taskID,
			"error", err,
		)
		return fmt.Errorf("failed to get runtime address: %w", err)
	}

	// Call Wrapper resume API
	if err := e.wrapperClient.Resume(ctx, address); err != nil {
		e.logger.Error("failed to resume wrapper",
			"task_id", taskID,
			"address", address,
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

	// Delete the runtime environment
	if err := e.runtime.Delete(ctx, taskID); err != nil {
		e.logger.Error("failed to delete runtime during cancel",
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

// GetAddress returns the network address for a task's runtime environment.
func (e *TaskExecutor) GetAddress(ctx context.Context, taskID string) (string, error) {
	// Validate task ID
	if err := validateTaskID(taskID); err != nil {
		return "", err
	}

	return e.runtime.GetAddress(ctx, taskID)
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
	parsed, err := uuid.Parse(taskID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTaskID, err)
	}
	if parsed == uuid.Nil {
		return fmt.Errorf("%w: task ID cannot be nil UUID", ErrInvalidTaskID)
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
		if err := e.heartbeat.UpdateHeartbeat(ctx, taskID, status, progress); err != nil {
			e.logger.Error("failed to update heartbeat",
				"task_id", taskID,
				"error", err,
			)
		}

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
			if err := e.config.OnTaskFailed(ctx, taskID, errors.New(errMsg)); err != nil {
				e.logger.Error("OnTaskFailed callback failed",
					"task_id", taskID,
					"error", err,
				)
				return fmt.Errorf("OnTaskFailed callback failed for task %s: %w", taskID, err)
			}
		}

	case "progress":
		e.logger.Info("Task progress",
			"task_id", taskID,
			"text", payload["text"],
		)

	case "tool_call":
		e.logger.Info("Task tool call",
			"task_id", taskID,
			"tool_name", payload["tool_name"],
		)
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

	// Get runtime address
	address, err := e.runtime.GetAddress(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get runtime address: %w", err)
	}

	// Call Wrapper inject API
	if err := e.wrapperClient.Inject(ctx, address, content); err != nil {
		return fmt.Errorf("failed to inject instruction: %w", err)
	}

	return nil
}

// GetHeartbeatManager returns the HeartbeatManager for direct access.
func (e *TaskExecutor) GetHeartbeatManager() *HeartbeatManager {
	return e.heartbeat
}

// recordMetric records a metric event if metrics recorder is configured.
func (e *TaskExecutor) recordMetric(event string, taskID string, detail string) {
	if e.metrics != nil {
		switch event {
		case "task_execution_started":
			e.metrics.RecordTaskExecution(taskID, "started")
		case "task_execution_failed":
			e.metrics.RecordTaskExecution(taskID, "failed")
		case "task_cancelled":
			e.metrics.RecordTaskCancelled(taskID, detail)
		case "task_completed":
			e.metrics.RecordTaskCompleted(taskID)
		case "task_failed":
			e.metrics.RecordTaskFailed(taskID, detail)
		}
	}
}
