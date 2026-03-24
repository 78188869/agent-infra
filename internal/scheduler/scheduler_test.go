package scheduler

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/example/agent-infra/internal/model"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func setupTestScheduler(t *testing.T) (*TaskScheduler, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cfg := &SchedulerConfig{
		GlobalLimit: 10,
		GetTenantQuota: func(ctx context.Context, tenantID string) (*TenantQuota, error) {
			return &TenantQuota{
				Concurrency: 5,
				DailyTasks:  100,
			}, nil
		},
		GetTask: func(ctx context.Context, taskID string) (*model.Task, error) {
			return nil, nil
		},
		UpdateStatus: func(ctx context.Context, taskID string, status string, message string) error {
			return nil
		},
	}

	scheduler := NewTaskScheduler(client, cfg)
	return scheduler, mr
}

func TestTaskScheduler_Schedule(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	now := time.Now()
	item := &QueueItem{
		TaskID:    "task-1",
		TenantID:  "tenant-1",
		Priority:  model.TaskPriorityNormal,
		CreatedAt: now,
	}

	err := scheduler.queue.Enqueue(ctx, item)
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	size, err := scheduler.queue.Size(ctx)
	if err != nil {
		t.Fatalf("failed to get queue size: %v", err)
	}
	if size != 1 {
		t.Errorf("expected queue size 1, got %d", size)
	}
}

func TestTaskScheduler_ScheduleWithModel(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	task := &model.Task{
		TenantID:  "tenant-1",
		Name:      "test-task",
		Status:    model.TaskStatusPending,
		Priority:  model.TaskPriorityHigh,
	}
	task.ID = uuid.New()
	task.CreatedAt = time.Now()

	err := scheduler.Schedule(ctx, task)
	if err != nil {
		t.Fatalf("failed to schedule task: %v", err)
	}

	size, err := scheduler.GetQueueSize(ctx)
	if err != nil {
		t.Fatalf("failed to get queue size: %v", err)
	}
	if size != 1 {
		t.Errorf("expected queue size 1, got %d", size)
	}
}

func TestTaskScheduler_ScheduleNil(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	err := scheduler.Schedule(ctx, nil)
	if err == nil {
		t.Error("expected error for nil task, got nil")
	}
}

func TestTaskScheduler_Dequeue_PriorityOrder(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	now := time.Now()
	items := []*QueueItem{
		{TaskID: "task-low", TenantID: "tenant-1", Priority: model.TaskPriorityLow, CreatedAt: now},
		{TaskID: "task-high", TenantID: "tenant-1", Priority: model.TaskPriorityHigh, CreatedAt: now.Add(time.Millisecond)},
		{TaskID: "task-normal", TenantID: "tenant-1", Priority: model.TaskPriorityNormal, CreatedAt: now.Add(2 * time.Millisecond)},
	}

	for _, item := range items {
		if err := scheduler.queue.Enqueue(ctx, item); err != nil {
			t.Fatalf("failed to enqueue: %v", err)
		}
	}

	expectedOrder := []string{"task-high", "task-normal", "task-low"}
	for _, expectedID := range expectedOrder {
		result, err := scheduler.Dequeue(ctx)
		if err != nil {
			t.Fatalf("failed to dequeue: %v", err)
		}
		if result == nil {
			t.Fatalf("expected task %s, got nil", expectedID)
		}
		if result.QueueItem.TaskID != expectedID {
			t.Errorf("expected task %s, got %s", expectedID, result.QueueItem.TaskID)
		}
		scheduler.Complete(ctx, result.QueueItem.TaskID, result.QueueItem.TenantID)
	}
}

func TestTaskScheduler_Dequeue_NotRunning(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	// Don't start the scheduler
	_, err := scheduler.Dequeue(ctx)
	if err != ErrSchedulerNotRunning {
		t.Errorf("expected ErrSchedulerNotRunning, got %v", err)
	}
}

func TestTaskScheduler_StartStop(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	if scheduler.IsRunning() {
		t.Error("expected scheduler to not be running initially")
	}

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	if !scheduler.IsRunning() {
		t.Error("expected scheduler to be running after start")
	}

	if err := scheduler.Start(ctx); err != ErrSchedulerAlreadyRunning {
		t.Errorf("expected ErrSchedulerAlreadyRunning, got %v", err)
	}

	if err := scheduler.Stop(ctx); err != nil {
		t.Fatalf("failed to stop scheduler: %v", err)
	}
	if scheduler.IsRunning() {
		t.Error("expected scheduler to not be running after stop")
	}
}

func TestTaskScheduler_GetPosition(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	now := time.Now()
	items := []*QueueItem{
		{TaskID: "task-1", TenantID: "tenant-1", Priority: model.TaskPriorityNormal, CreatedAt: now},
		{TaskID: "task-2", TenantID: "tenant-1", Priority: model.TaskPriorityNormal, CreatedAt: now.Add(time.Millisecond)},
		{TaskID: "task-3", TenantID: "tenant-1", Priority: model.TaskPriorityNormal, CreatedAt: now.Add(2 * time.Millisecond)},
	}

	for _, item := range items {
		if err := scheduler.queue.Enqueue(ctx, item); err != nil {
			t.Fatalf("failed to enqueue: %v", err)
		}
	}

	pos, err := scheduler.GetPosition(ctx, "task-1")
	if err != nil {
		t.Fatalf("failed to get position: %v", err)
	}
	if pos != 1 {
		t.Errorf("expected position 1 for task-1, got %d", pos)
	}

	pos, err = scheduler.GetPosition(ctx, "task-3")
	if err != nil {
		t.Fatalf("failed to get position: %v", err)
	}
	if pos != 3 {
		t.Errorf("expected position 3 for task-3, got %d", pos)
	}
}

func TestTaskScheduler_GetQueueSize(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	// Initially empty
	size, err := scheduler.GetQueueSize(ctx)
	if err != nil {
		t.Fatalf("failed to get queue size: %v", err)
	}
	if size != 0 {
		t.Errorf("expected queue size 0, got %d", size)
	}

	// Add some tasks
	now := time.Now()
	for i := 0; i < 3; i++ {
		item := &QueueItem{
			TaskID:    string(rune('a' + i)),
			TenantID:  "tenant-1",
			Priority:  model.TaskPriorityNormal,
			CreatedAt: now.Add(time.Duration(i) * time.Millisecond),
		}
		if err := scheduler.queue.Enqueue(ctx, item); err != nil {
			t.Fatalf("failed to enqueue: %v", err)
		}
	}

	size, err = scheduler.GetQueueSize(ctx)
	if err != nil {
		t.Fatalf("failed to get queue size: %v", err)
	}
	if size != 3 {
		t.Errorf("expected queue size 3, got %d", size)
	}
}

func TestTaskScheduler_GetGlobalConcurrency(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	count, err := scheduler.GetGlobalConcurrency(ctx)
	if err != nil {
		t.Fatalf("failed to get global concurrency: %v", err)
	}
	if count != 0 {
		t.Errorf("expected global concurrency 0, got %d", count)
	}
}

func TestTaskScheduler_GetTenantUsage(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	usage, err := scheduler.GetTenantUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("failed to get tenant usage: %v", err)
	}
	if usage.CurrentConcurrency != 0 {
		t.Errorf("expected concurrency 0, got %d", usage.CurrentConcurrency)
	}
}

func TestTaskScheduler_GetPreemptedTasks(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	// Initially empty
	tasks, err := scheduler.GetPreemptedTasks(ctx)
	if err != nil {
		t.Fatalf("failed to get preempted tasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 preempted tasks, got %d", len(tasks))
	}

	// Preempt a task
	item := &QueueItem{
		TaskID:    "task-1",
		TenantID:  "tenant-1",
		Priority:  model.TaskPriorityNormal,
		CreatedAt: time.Now(),
	}
	if err := scheduler.preemption.Preempt(ctx, item, model.TaskStatusRunning, 50); err != nil {
		t.Fatalf("failed to preempt: %v", err)
	}

	tasks, err = scheduler.GetPreemptedTasks(ctx)
	if err != nil {
		t.Fatalf("failed to get preempted tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 preempted task, got %d", len(tasks))
	}
}

func TestTaskScheduler_Complete(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	// Reserve some resources first
	if err := scheduler.limiter.Reserve(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to reserve: %v", err)
	}

	// Complete should release resources
	if err := scheduler.Complete(ctx, "task-1", "tenant-1"); err != nil {
		t.Fatalf("failed to complete: %v", err)
	}

	// Verify resources were released
	usage, err := scheduler.GetTenantUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("failed to get tenant usage: %v", err)
	}
	if usage.CurrentConcurrency != 0 {
		t.Errorf("expected concurrency 0 after complete, got %d", usage.CurrentConcurrency)
	}
}

func TestTaskScheduler_NewTaskScheduler_Defaults(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// Test with zero global limit (should default to 100)
	cfg := &SchedulerConfig{
		GlobalLimit: 0,
	}
	scheduler := NewTaskScheduler(client, cfg)

	if scheduler.globalLimit != 100 {
		t.Errorf("expected global limit 100, got %d", scheduler.globalLimit)
	}
}

func TestTaskScheduler_StopNotRunning(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	// Stop on non-running scheduler should succeed
	if err := scheduler.Stop(ctx); err != nil {
		t.Errorf("expected no error stopping non-running scheduler, got %v", err)
	}
}

func TestTaskScheduler_Preempt(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	task := &model.Task{
		TenantID: "tenant-1",
		Name:     "test-task",
		Status:   model.TaskStatusRunning,
		Priority: model.TaskPriorityNormal,
	}
	task.ID = uuid.New()

	cfg := &SchedulerConfig{
		GlobalLimit: 10,
		GetTask: func(ctx context.Context, taskID string) (*model.Task, error) {
			return task, nil
		},
		UpdateStatus: func(ctx context.Context, taskID string, status string, message string) error {
			return nil
		},
	}

	scheduler := NewTaskScheduler(client, cfg)

	ctx := context.Background()

	// Reserve resources for the task first
	if err := scheduler.limiter.Reserve(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to reserve: %v", err)
	}

	// Preempt the task
	if err := scheduler.Preempt(ctx, task.ID.String(), model.TaskStatusRunning, 50); err != nil {
		t.Fatalf("failed to preempt: %v", err)
	}

	// Verify task was re-queued
	pos, err := scheduler.GetPosition(ctx, task.ID.String())
	if err != nil {
		t.Fatalf("failed to get position: %v", err)
	}
	if pos == 0 {
		t.Error("expected task to be re-queued")
	}

	// Verify resources were released
	usage, err := scheduler.GetTenantUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("failed to get tenant usage: %v", err)
	}
	if usage.CurrentConcurrency != 0 {
		t.Errorf("expected concurrency 0 after preempt, got %d", usage.CurrentConcurrency)
	}
}

func TestTaskScheduler_Preempt_TaskNotRunning(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	task := &model.Task{
		TenantID: "tenant-1",
		Name:     "test-task",
		Status:   model.TaskStatusPending, // Not running
		Priority: model.TaskPriorityNormal,
	}
	task.ID = uuid.New()

	cfg := &SchedulerConfig{
		GlobalLimit: 10,
		GetTask: func(ctx context.Context, taskID string) (*model.Task, error) {
			return task, nil
		},
	}

	scheduler := NewTaskScheduler(client, cfg)

	ctx := context.Background()

	// Preempt should fail for non-running task
	err = scheduler.Preempt(ctx, task.ID.String(), model.TaskStatusPending, 50)
	if err != ErrTaskNotRunning {
		t.Errorf("expected ErrTaskNotRunning, got: %v", err)
	}
}

func TestTaskScheduler_Dequeue_Empty(t *testing.T) {
	scheduler, mr := setupTestScheduler(t)
	defer mr.Close()
	defer scheduler.client.Close()

	ctx := context.Background()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// Dequeue from empty queue should return nil
	result, err := scheduler.Dequeue(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for empty queue, got %v", result)
	}
}

func TestTaskScheduler_Dequeue_WithoutGetTask(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// Create scheduler without getTask callback
	cfg := &SchedulerConfig{
		GlobalLimit: 10,
		GetTenantQuota: func(ctx context.Context, tenantID string) (*TenantQuota, error) {
			return &TenantQuota{Concurrency: 5, DailyTasks: 100}, nil
		},
	}
	scheduler := NewTaskScheduler(client, cfg)

	ctx := context.Background()
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// Enqueue a task
	now := time.Now()
	item := &QueueItem{
		TaskID:    "task-1",
		TenantID:  "tenant-1",
		Priority:  model.TaskPriorityNormal,
		CreatedAt: now,
	}
	if err := scheduler.queue.Enqueue(ctx, item); err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	// Dequeue should fail without getTask callback
	_, err = scheduler.Dequeue(ctx)
	if err == nil {
		t.Error("expected error for dequeue without getTask callback")
	}
}

func TestTaskScheduler_Preempt_WithoutGetTask(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// Create scheduler without getTask callback
	cfg := &SchedulerConfig{
		GlobalLimit: 10,
	}
	scheduler := NewTaskScheduler(client, cfg)

	ctx := context.Background()

	// Preempt should fail without getTask callback
	err = scheduler.Preempt(ctx, "task-1", model.TaskStatusRunning, 50)
	if err == nil {
		t.Error("expected error for preempt without getTask callback")
	}
}

func TestTaskScheduler_Preempt_InvalidInput(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	task := &model.Task{
		TenantID: "tenant-1",
		Name:     "test-task",
		Status:   model.TaskStatusRunning,
		Priority: model.TaskPriorityNormal,
	}
	task.ID = uuid.New()

	cfg := &SchedulerConfig{
		GlobalLimit: 10,
		GetTask: func(ctx context.Context, taskID string) (*model.Task, error) {
			return task, nil
		},
	}
	scheduler := NewTaskScheduler(client, cfg)

	ctx := context.Background()

	// Test empty taskID
	err = scheduler.Preempt(ctx, "", model.TaskStatusRunning, 50)
	if err == nil {
		t.Error("expected error for empty taskID")
	}

	// Test invalid progress (< 0)
	err = scheduler.Preempt(ctx, task.ID.String(), model.TaskStatusRunning, -1)
	if err == nil {
		t.Error("expected error for negative progress")
	}

	// Test invalid progress (> 100)
	err = scheduler.Preempt(ctx, task.ID.String(), model.TaskStatusRunning, 101)
	if err == nil {
		t.Error("expected error for progress > 100")
	}
}

// TestTaskScheduler_PriorityPreemptionScenario tests the complete priority-based preemption flow:
// 1. A low-priority task is running (occupying the only slot)
// 2. A high-priority task arrives but can't start (no available slots)
// 3. The low-priority task is preempted to make room
// 4. The high-priority task gets executed first
// 5. After high-priority completes, the preempted task resumes
func TestTaskScheduler_PriorityPreemptionScenario(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// Create tasks with different priorities
	lowPriorityTask := &model.Task{
		TenantID: "tenant-1",
		Name:     "low-priority-task",
		Status:   model.TaskStatusRunning,
		Priority: model.TaskPriorityLow,
	}
	lowPriorityTask.ID = uuid.New()

	highPriorityTask := &model.Task{
		TenantID: "tenant-1",
		Name:     "high-priority-task",
		Status:   model.TaskStatusPending,
		Priority: model.TaskPriorityHigh,
	}
	highPriorityTask.ID = uuid.New()

	// Track status updates
	statusUpdates := make(map[string]string)
	var statusMutex sync.Mutex

	// Create scheduler with concurrency limit of 1
	cfg := &SchedulerConfig{
		GlobalLimit: 1, // Only 1 concurrent task allowed
		GetTenantQuota: func(ctx context.Context, tenantID string) (*TenantQuota, error) {
			return &TenantQuota{
				Concurrency: 1, // Only 1 concurrent task per tenant
				DailyTasks:  100,
			}, nil
		},
		GetTask: func(ctx context.Context, taskID string) (*model.Task, error) {
			if taskID == lowPriorityTask.ID.String() {
				return lowPriorityTask, nil
			}
			if taskID == highPriorityTask.ID.String() {
				return highPriorityTask, nil
			}
			return nil, fmt.Errorf("task not found")
		},
		UpdateStatus: func(ctx context.Context, taskID string, status string, message string) error {
			statusMutex.Lock()
			defer statusMutex.Unlock()
			statusUpdates[taskID] = status
			return nil
		},
	}

	scheduler := NewTaskScheduler(client, cfg)
	ctx := context.Background()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// Step 1: Enqueue and dequeue the low-priority task
	now := time.Now()
	lowItem := &QueueItem{
		TaskID:    lowPriorityTask.ID.String(),
		TenantID:  "tenant-1",
		Priority:  model.TaskPriorityLow,
		CreatedAt: now,
	}
	if err := scheduler.queue.Enqueue(ctx, lowItem); err != nil {
		t.Fatalf("failed to enqueue low priority task: %v", err)
	}

	// Dequeue the low-priority task (it starts running)
	dequeuedLow, err := scheduler.Dequeue(ctx)
	if err != nil {
		t.Fatalf("failed to dequeue low priority task: %v", err)
	}
	if dequeuedLow.QueueItem.TaskID != lowPriorityTask.ID.String() {
		t.Fatalf("expected low priority task, got %s", dequeuedLow.QueueItem.TaskID)
	}

	// Verify concurrency is at limit
	usage, err := scheduler.GetTenantUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("failed to get tenant usage: %v", err)
	}
	if usage.CurrentConcurrency != 1 {
		t.Errorf("expected concurrency 1, got %d", usage.CurrentConcurrency)
	}

	// Step 2: Enqueue the high-priority task
	highItem := &QueueItem{
		TaskID:    highPriorityTask.ID.String(),
		TenantID:  "tenant-1",
		Priority:  model.TaskPriorityHigh,
		CreatedAt: now.Add(time.Millisecond),
	}
	if err := scheduler.queue.Enqueue(ctx, highItem); err != nil {
		t.Fatalf("failed to enqueue high priority task: %v", err)
	}

	// Verify high-priority task is in queue at position 1 (before low-priority if re-queued)
	highPos, err := scheduler.GetPosition(ctx, highPriorityTask.ID.String())
	if err != nil {
		t.Fatalf("failed to get high priority task position: %v", err)
	}
	if highPos != 1 {
		t.Errorf("expected high priority task at position 1, got %d", highPos)
	}

	// Step 3: Preempt the low-priority task to make room for high-priority
	// Simulate the low-priority task has made some progress
	err = scheduler.Preempt(ctx, lowPriorityTask.ID.String(), model.TaskStatusRunning, 30)
	if err != nil {
		t.Fatalf("failed to preempt low priority task: %v", err)
	}

	// Verify the low-priority task's state was saved
	state, err := scheduler.preemption.GetTaskState(ctx, lowPriorityTask.ID.String())
	if err != nil {
		t.Fatalf("failed to get preempted task state: %v", err)
	}
	if state.Progress != 30 {
		t.Errorf("expected progress 30, got %d", state.Progress)
	}
	if state.Status != model.TaskStatusRunning {
		t.Errorf("expected status %s, got %s", model.TaskStatusRunning, state.Status)
	}

	// Verify the low-priority task is marked as preempted
	isPreempted, err := scheduler.preemption.IsPreempted(ctx, lowPriorityTask.ID.String())
	if err != nil {
		t.Fatalf("failed to check preempted status: %v", err)
	}
	if !isPreempted {
		t.Error("expected low priority task to be marked as preempted")
	}

	// Verify resources were released (concurrency should be 0)
	usage, err = scheduler.GetTenantUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("failed to get tenant usage after preempt: %v", err)
	}
	if usage.CurrentConcurrency != 0 {
		t.Errorf("expected concurrency 0 after preempt, got %d", usage.CurrentConcurrency)
	}

	// Verify the low-priority task was re-queued
	lowPos, err := scheduler.GetPosition(ctx, lowPriorityTask.ID.String())
	if err != nil {
		t.Fatalf("failed to get low priority task position after preempt: %v", err)
	}
	if lowPos == 0 {
		t.Error("expected low priority task to be re-queued")
	}

	// Step 4: Dequeue the high-priority task (should be first due to priority)
	dequeuedHigh, err := scheduler.Dequeue(ctx)
	if err != nil {
		t.Fatalf("failed to dequeue high priority task: %v", err)
	}
	if dequeuedHigh.QueueItem.TaskID != highPriorityTask.ID.String() {
		t.Errorf("expected high priority task to be dequeued, got %s", dequeuedHigh.QueueItem.TaskID)
	}

	// Step 5: Complete the high-priority task
	err = scheduler.Complete(ctx, highPriorityTask.ID.String(), "tenant-1")
	if err != nil {
		t.Fatalf("failed to complete high priority task: %v", err)
	}

	// Verify concurrency is back to 0
	usage, err = scheduler.GetTenantUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("failed to get tenant usage after complete: %v", err)
	}
	if usage.CurrentConcurrency != 0 {
		t.Errorf("expected concurrency 0 after complete, got %d", usage.CurrentConcurrency)
	}

	// Step 6: Dequeue the preempted low-priority task (it resumes)
	dequeuedResumed, err := scheduler.Dequeue(ctx)
	if err != nil {
		t.Fatalf("failed to dequeue resumed task: %v", err)
	}
	if dequeuedResumed.QueueItem.TaskID != lowPriorityTask.ID.String() {
		t.Errorf("expected resumed low priority task, got %s", dequeuedResumed.QueueItem.TaskID)
	}

	// Step 7: Complete the resumed task
	err = scheduler.Complete(ctx, lowPriorityTask.ID.String(), "tenant-1")
	if err != nil {
		t.Fatalf("failed to complete resumed task: %v", err)
	}

	// Verify the preempted state was cleared
	_, err = scheduler.preemption.GetTaskState(ctx, lowPriorityTask.ID.String())
	if err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound for cleared state, got %v", err)
	}

	t.Log("Priority preemption scenario completed successfully!")
}

// TestTaskScheduler_MultiplePriorityPreemption tests preemption with multiple tasks at different priorities
func TestTaskScheduler_MultiplePriorityPreemption(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// Create tasks with different priorities
	tasks := make(map[string]*model.Task)
	for i, priority := range []string{model.TaskPriorityLow, model.TaskPriorityNormal, model.TaskPriorityHigh} {
		task := &model.Task{
			TenantID: "tenant-1",
			Name:     fmt.Sprintf("task-%s-%d", priority, i),
			Status:   model.TaskStatusPending,
			Priority: priority,
		}
		task.ID = uuid.New()
		tasks[task.ID.String()] = task
	}

	// Get task IDs for easy reference
	var lowTaskID, normalTaskID, highTaskID string
	for id, task := range tasks {
		switch task.Priority {
		case model.TaskPriorityLow:
			lowTaskID = id
		case model.TaskPriorityNormal:
			normalTaskID = id
		case model.TaskPriorityHigh:
			highTaskID = id
		}
	}

	// Mark low task as running (simulating it was already dequeued)
	tasks[lowTaskID].Status = model.TaskStatusRunning

	cfg := &SchedulerConfig{
		GlobalLimit: 1, // Only 1 concurrent task
		GetTenantQuota: func(ctx context.Context, tenantID string) (*TenantQuota, error) {
			return &TenantQuota{Concurrency: 1, DailyTasks: 100}, nil
		},
		GetTask: func(ctx context.Context, taskID string) (*model.Task, error) {
			if task, ok := tasks[taskID]; ok {
				return task, nil
			}
			return nil, fmt.Errorf("task not found")
		},
		UpdateStatus: func(ctx context.Context, taskID string, status string, message string) error {
			if task, ok := tasks[taskID]; ok {
				task.Status = status
			}
			return nil
		},
	}

	scheduler := NewTaskScheduler(client, cfg)
	ctx := context.Background()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// Simulate low-priority task is running (reserve resources)
	if err := scheduler.limiter.Reserve(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to reserve resources: %v", err)
	}

	// Enqueue normal and high priority tasks
	now := time.Now()
	for _, item := range []struct {
		taskID   string
		priority string
		delay    time.Duration
	}{
		{normalTaskID, model.TaskPriorityNormal, time.Millisecond},
		{highTaskID, model.TaskPriorityHigh, 2 * time.Millisecond},
	} {
		queueItem := &QueueItem{
			TaskID:    item.taskID,
			TenantID:  "tenant-1",
			Priority:  item.priority,
			CreatedAt: now.Add(item.delay),
		}
		if err := scheduler.queue.Enqueue(ctx, queueItem); err != nil {
			t.Fatalf("failed to enqueue task: %v", err)
		}
	}

	// Verify queue order: high should be before normal (position 1 vs 2)
	highPos, _ := scheduler.GetPosition(ctx, highTaskID)
	normalPos, _ := scheduler.GetPosition(ctx, normalTaskID)
	if highPos >= normalPos {
		t.Errorf("expected high priority (pos %d) before normal priority (pos %d)", highPos, normalPos)
	}

	// Preempt the low-priority task
	err = scheduler.Preempt(ctx, lowTaskID, model.TaskStatusRunning, 50)
	if err != nil {
		t.Fatalf("failed to preempt low priority task: %v", err)
	}

	// Verify all three tasks are now in queue, high should still be first
	queueSize, err := scheduler.GetQueueSize(ctx)
	if err != nil {
		t.Fatalf("failed to get queue size: %v", err)
	}
	if queueSize != 3 {
		t.Errorf("expected queue size 3, got %d", queueSize)
	}

	// Dequeue should return high-priority task first
	result, err := scheduler.Dequeue(ctx)
	if err != nil {
		t.Fatalf("failed to dequeue: %v", err)
	}
	if result.QueueItem.TaskID != highTaskID {
		t.Errorf("expected high priority task, got %s", result.QueueItem.TaskID)
	}

	t.Log("Multiple priority preemption scenario completed successfully!")
}

// TestTaskScheduler_PreemptionStatePreservation tests that task state is properly preserved during preemption
func TestTaskScheduler_PreemptionStatePreservation(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	task := &model.Task{
		TenantID: "tenant-1",
		Name:     "task-with-checkpoint",
		Status:   model.TaskStatusRunning,
		Priority: model.TaskPriorityNormal,
	}
	task.ID = uuid.New()

	cfg := &SchedulerConfig{
		GlobalLimit: 10,
		GetTenantQuota: func(ctx context.Context, tenantID string) (*TenantQuota, error) {
			return &TenantQuota{Concurrency: 5, DailyTasks: 100}, nil
		},
		GetTask: func(ctx context.Context, taskID string) (*model.Task, error) {
			return task, nil
		},
		UpdateStatus: func(ctx context.Context, taskID string, status string, message string) error {
			task.Status = status
			return nil
		},
	}

	scheduler := NewTaskScheduler(client, cfg)
	ctx := context.Background()

	// Reserve resources (simulate running task)
	if err := scheduler.limiter.Reserve(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to reserve: %v", err)
	}

	// Preempt with progress and checkpoint data
	progress := 75
	err = scheduler.Preempt(ctx, task.ID.String(), model.TaskStatusRunning, progress)
	if err != nil {
		t.Fatalf("failed to preempt: %v", err)
	}

	// Verify state was saved with correct values
	state, err := scheduler.preemption.GetTaskState(ctx, task.ID.String())
	if err != nil {
		t.Fatalf("failed to get task state: %v", err)
	}

	if state.TaskID != task.ID.String() {
		t.Errorf("expected task ID %s, got %s", task.ID.String(), state.TaskID)
	}
	if state.Status != model.TaskStatusRunning {
		t.Errorf("expected status %s, got %s", model.TaskStatusRunning, state.Status)
	}
	if state.Progress != progress {
		t.Errorf("expected progress %d, got %d", progress, state.Progress)
	}
	if state.PreemptedAt.IsZero() {
		t.Error("expected PreemptedAt to be set")
	}

	// Verify task was re-queued
	pos, err := scheduler.GetPosition(ctx, task.ID.String())
	if err != nil {
		t.Fatalf("failed to get position: %v", err)
	}
	if pos == 0 {
		t.Error("expected task to be in queue")
	}

	// Verify resources were released
	usage, err := scheduler.GetTenantUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("failed to get usage: %v", err)
	}
	if usage.CurrentConcurrency != 0 {
		t.Errorf("expected concurrency 0, got %d", usage.CurrentConcurrency)
	}

	t.Log("Preemption state preservation test completed successfully!")
}
