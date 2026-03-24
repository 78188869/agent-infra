package scheduler

import (
	"context"
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
