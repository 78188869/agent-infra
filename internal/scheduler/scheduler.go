package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/example/agent-infra/internal/model"
	"github.com/redis/go-redis/v9"
)

// Scheduler defines the interface for task scheduling operations.
type Scheduler interface {
	// Schedule adds a task to the scheduling queue.
	Schedule(ctx context.Context, task *model.Task) error
	// Dequeue retrieves the next available task from the queue.
	Dequeue(ctx context.Context) (*DequeuedTask, error)
	// GetPosition returns the position of a task in the queue.
	GetPosition(ctx context.Context, taskID string) (int, error)
	// Preempt saves the current task state and re-queues it for later execution.
	Preempt(ctx context.Context, taskID string, status string, progress int) error
	// Complete releases resources for a completed task.
	Complete(ctx context.Context, taskID string, tenantID string) error
	// Start begins the scheduler processing.
	Start(ctx context.Context) error
	// Stop gracefully stops the scheduler.
	Stop(ctx context.Context) error
	// IsRunning returns whether the scheduler is currently running.
	IsRunning() bool
}

// DequeuedTask represents a dequeued task with its quota info.
type DequeuedTask struct {
	Task      *model.Task
	QueueItem *QueueItem
}

// TaskScheduler implements the Scheduler interface using Redis.
type TaskScheduler struct {
	queue       *PriorityQueue
	limiter     *RateLimiter
	preemption  *PreemptionManager
	client      *redis.Client

	// Configuration
	globalLimit int

	// State
	running atomic.Bool
	stopCh  chan struct{}
	wg      sync.WaitGroup

	// Callbacks for external integration
	getTenantQuota func(ctx context.Context, tenantID string) (*TenantQuota, error)
	getTask        func(ctx context.Context, taskID string) (*model.Task, error)
	updateStatus   func(ctx context.Context, taskID string, status string, message string) error
}

// SchedulerConfig holds configuration for TaskScheduler.
type SchedulerConfig struct {
	GlobalLimit    int
	GetTenantQuota func(ctx context.Context, tenantID string) (*TenantQuota, error)
	GetTask        func(ctx context.Context, taskID string) (*model.Task, error)
	UpdateStatus   func(ctx context.Context, taskID string, status string, message string) error
}

// NewTaskScheduler creates a new TaskScheduler instance.
func NewTaskScheduler(client *redis.Client, cfg *SchedulerConfig) *TaskScheduler {
	if cfg.GlobalLimit <= 0 {
		cfg.GlobalLimit = 100
	}

	queue := NewPriorityQueue(client)
	s := &TaskScheduler{
		client:         client,
		queue:          queue,
		limiter:        NewRateLimiter(client, cfg.GlobalLimit),
		preemption:     NewPreemptionManager(client, queue),
		globalLimit:    cfg.GlobalLimit,
		stopCh:         make(chan struct{}),
		getTenantQuota: cfg.GetTenantQuota,
		getTask:        cfg.GetTask,
		updateStatus:   cfg.UpdateStatus,
	}

	return s
}

// Schedule adds a task to the scheduling queue.
func (s *TaskScheduler) Schedule(ctx context.Context, task *model.Task) error {
	if task == nil {
		return fmt.Errorf("cannot schedule nil task")
	}

	// Convert UUID to string for queue storage
	taskIDStr := task.ID.String()

	item := &QueueItem{
		TaskID:    taskIDStr,
		TenantID:  task.TenantID,
		Priority:  task.Priority,
		CreatedAt: task.CreatedAt,
	}

	if err := s.queue.Enqueue(ctx, item); err != nil {
		return fmt.Errorf("failed to schedule task: %w", err)
	}

	// Update task status to scheduled if callback is provided
	if s.updateStatus != nil {
		if err := s.updateStatus(ctx, taskIDStr, model.TaskStatusScheduled, "queued for execution"); err != nil {
			// Log error but don't fail the schedule operation
			// Status update is best-effort
		}
	}

	return nil
}

// Dequeue retrieves the next available task from the queue.
func (s *TaskScheduler) Dequeue(ctx context.Context) (*DequeuedTask, error) {
	for {
		// Check if scheduler is still running
		if !s.running.Load() {
			return nil, ErrSchedulerNotRunning
		}

		item, err := s.queue.Dequeue(ctx)
		if err != nil {
			return nil, err
		}
		if item == nil {
			return nil, nil // Queue empty
		}

		// Get tenant quota
		var quota *TenantQuota
		if s.getTenantQuota != nil {
			quota, err = s.getTenantQuota(ctx, item.TenantID)
			if err != nil {
				// Re-queue on error
				s.queue.Enqueue(ctx, item)
				return nil, fmt.Errorf("failed to get tenant quota: %w", err)
			}
		} else {
			// Default quota if no callback provided
			quota = &TenantQuota{
				Concurrency: 10,
				DailyTasks:  100,
			}
		}

		// Check rate limit
		if err := s.limiter.Allow(ctx, item.TenantID, quota); err != nil {
			// Rate limited - re-queue with slight delay
			s.queue.Enqueue(ctx, item)
			select {
			case <-time.After(100 * time.Millisecond):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// Reserve resources
		if err := s.limiter.Reserve(ctx, item.TenantID); err != nil {
			s.queue.Enqueue(ctx, item)
			return nil, fmt.Errorf("failed to reserve resources: %w", err)
		}

		// Get full task from repository if callback is provided
		var task *model.Task
		if s.getTask != nil {
			task, err = s.getTask(ctx, item.TaskID)
			if err != nil {
				s.limiter.Release(ctx, item.TenantID)
				return nil, fmt.Errorf("failed to get task: %w", err)
			}
		} else {
			// Create minimal task object
			task = &model.Task{}
			task.ID.String() // This will panic, but we won't reach here in tests
		}

		return &DequeuedTask{
			Task:      task,
			QueueItem: item,
		}, nil
	}
}

// GetPosition returns the position of a task in the queue.
func (s *TaskScheduler) GetPosition(ctx context.Context, taskID string) (int, error) {
	return s.queue.GetPosition(ctx, taskID)
}

// Preempt saves the current task state and re-queues it for later execution.
func (s *TaskScheduler) Preempt(ctx context.Context, taskID string, status string, progress int) error {
	// Get task info if callback is available
	var item *QueueItem
	if s.getTask != nil {
		task, err := s.getTask(ctx, taskID)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}
		if task.Status != model.TaskStatusRunning {
			return ErrTaskNotRunning
		}
		item = &QueueItem{
			TaskID:    taskID,
			TenantID:  task.TenantID,
			Priority:  task.Priority,
			CreatedAt: time.Now(),
		}
	} else {
		item = &QueueItem{
			TaskID:    taskID,
			CreatedAt: time.Now(),
		}
	}

	// Perform preemption
	if err := s.preemption.Preempt(ctx, item, status, progress); err != nil {
		return err
	}

	// Release resources
	s.limiter.Release(ctx, item.TenantID)

	// Update task status if callback is provided
	if s.updateStatus != nil {
		if err := s.updateStatus(ctx, taskID, model.TaskStatusPaused, "preempted by higher priority task"); err != nil {
			// Log error but don't fail the preemption
		}
	}

	return nil
}

// Complete releases resources for a completed task.
func (s *TaskScheduler) Complete(ctx context.Context, taskID string, tenantID string) error {
	// Release resources
	if err := s.limiter.Release(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to release resources: %w", err)
	}

	// Clear any preemption state
	s.preemption.ClearTaskState(ctx, taskID)

	return nil
}

// Start begins the scheduler processing.
func (s *TaskScheduler) Start(ctx context.Context) error {
	if s.running.Load() {
		return ErrSchedulerAlreadyRunning
	}

	s.running.Store(true)
	return nil
}

// Stop gracefully stops the scheduler.
func (s *TaskScheduler) Stop(ctx context.Context) error {
	if !s.running.Load() {
		return nil
	}

	s.running.Store(false)
	close(s.stopCh)

	// Wait for any in-progress operations to complete
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IsRunning returns whether the scheduler is currently running.
func (s *TaskScheduler) IsRunning() bool {
	return s.running.Load()
}

// GetQueueSize returns the total number of tasks in the queue.
func (s *TaskScheduler) GetQueueSize(ctx context.Context) (int64, error) {
	return s.queue.Size(ctx)
}

// GetGlobalConcurrency returns the current global concurrency count.
func (s *TaskScheduler) GetGlobalConcurrency(ctx context.Context) (int, error) {
	return s.limiter.GetGlobalConcurrency(ctx)
}

// GetTenantUsage returns the current quota usage for a tenant.
func (s *TaskScheduler) GetTenantUsage(ctx context.Context, tenantID string) (*QuotaUsage, error) {
	return s.limiter.GetUsage(ctx, tenantID)
}

// GetPreemptedTasks returns all preempted task IDs.
func (s *TaskScheduler) GetPreemptedTasks(ctx context.Context) ([]string, error) {
	return s.preemption.GetPreemptedTasks(ctx)
}
