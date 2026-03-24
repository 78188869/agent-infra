package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/example/agent-infra/internal/model"
	"github.com/redis/go-redis/v9"
)

func setupTestPreemption(t *testing.T) (*PreemptionManager, *PriorityQueue, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	queue := NewPriorityQueue(client)
	preemption := NewPreemptionManager(client, queue)
	return preemption, queue, mr
}

func TestPreemptionManager_SaveTaskState(t *testing.T) {
	preemption, _, mr := setupTestPreemption(t)
	defer mr.Close()
	defer preemption.client.Close()

	ctx := context.Background()

	state := &TaskState{
		TaskID:      "task-1",
		Status:      model.TaskStatusRunning,
		Progress:    50,
		PreemptedAt: time.Now(),
	}

	err := preemption.SaveTaskState(ctx, state)
	if err != nil {
		t.Fatalf("failed to save task state: %v", err)
	}

	// Verify state was saved
	retrieved, err := preemption.GetTaskState(ctx, "task-1")
	if err != nil {
		t.Fatalf("failed to get task state: %v", err)
	}

	if retrieved.TaskID != state.TaskID {
		t.Errorf("expected task ID %s, got %s", state.TaskID, retrieved.TaskID)
	}
	if retrieved.Status != state.Status {
		t.Errorf("expected status %s, got %s", state.Status, retrieved.Status)
	}
	if retrieved.Progress != state.Progress {
		t.Errorf("expected progress %d, got %d", state.Progress, retrieved.Progress)
	}
}

func TestPreemptionManager_SaveTaskState_Nil(t *testing.T) {
	preemption, _, mr := setupTestPreemption(t)
	defer mr.Close()
	defer preemption.client.Close()

	ctx := context.Background()

	err := preemption.SaveTaskState(ctx, nil)
	if err == nil {
		t.Error("expected error for nil state, got nil")
	}
}

func TestPreemptionManager_GetTaskState_NotFound(t *testing.T) {
	preemption, _, mr := setupTestPreemption(t)
	defer mr.Close()
	defer preemption.client.Close()

	ctx := context.Background()

	_, err := preemption.GetTaskState(ctx, "non-existent")
	if err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestPreemptionManager_Preempt(t *testing.T) {
	preemption, queue, mr := setupTestPreemption(t)
	defer mr.Close()
	defer preemption.client.Close()

	ctx := context.Background()

	item := &QueueItem{
		TaskID:    "task-1",
		TenantID:  "tenant-1",
		Priority:  model.TaskPriorityNormal,
		CreatedAt: time.Now(),
	}

	// Preempt the task
	err := preemption.Preempt(ctx, item, model.TaskStatusRunning, 75)
	if err != nil {
		t.Fatalf("failed to preempt: %v", err)
	}

	// Verify state was saved
	state, err := preemption.GetTaskState(ctx, "task-1")
	if err != nil {
		t.Fatalf("failed to get task state: %v", err)
	}
	if state.Progress != 75 {
		t.Errorf("expected progress 75, got %d", state.Progress)
	}

	// Verify task is marked as preempted
	isPreempted, err := preemption.IsPreempted(ctx, "task-1")
	if err != nil {
		t.Fatalf("failed to check preempted status: %v", err)
	}
	if !isPreempted {
		t.Error("expected task to be marked as preempted")
	}

	// Verify task was re-queued
	position, err := queue.GetPosition(ctx, "task-1")
	if err != nil {
		t.Fatalf("failed to get position: %v", err)
	}
	if position == 0 {
		t.Error("expected task to be in queue")
	}
}

func TestPreemptionManager_Preempt_Nil(t *testing.T) {
	preemption, _, mr := setupTestPreemption(t)
	defer mr.Close()
	defer preemption.client.Close()

	ctx := context.Background()

	err := preemption.Preempt(ctx, nil, model.TaskStatusRunning, 50)
	if err == nil {
		t.Error("expected error for nil item, got nil")
	}
}

func TestPreemptionManager_IsPreempted(t *testing.T) {
	preemption, _, mr := setupTestPreemption(t)
	defer mr.Close()
	defer preemption.client.Close()

	ctx := context.Background()

	// Initially not preempted
	isPreempted, err := preemption.IsPreempted(ctx, "task-1")
	if err != nil {
		t.Fatalf("failed to check preempted status: %v", err)
	}
	if isPreempted {
		t.Error("expected task not to be preempted initially")
	}

	// Preempt the task
	item := &QueueItem{
		TaskID:    "task-1",
		TenantID:  "tenant-1",
		Priority:  model.TaskPriorityNormal,
		CreatedAt: time.Now(),
	}
	preemption.Preempt(ctx, item, model.TaskStatusRunning, 50)

	// Now should be preempted
	isPreempted, err = preemption.IsPreempted(ctx, "task-1")
	if err != nil {
		t.Fatalf("failed to check preempted status: %v", err)
	}
	if !isPreempted {
		t.Error("expected task to be preempted")
	}
}

func TestPreemptionManager_GetPreemptedTasks(t *testing.T) {
	preemption, _, mr := setupTestPreemption(t)
	defer mr.Close()
	defer preemption.client.Close()

	ctx := context.Background()

	// Preempt multiple tasks
	for i := 0; i < 3; i++ {
		item := &QueueItem{
			TaskID:    string(rune('a' + i)),
			TenantID:  "tenant-1",
			Priority:  model.TaskPriorityNormal,
			CreatedAt: time.Now(),
		}
		if err := preemption.Preempt(ctx, item, model.TaskStatusRunning, i*10); err != nil {
			t.Fatalf("failed to preempt task %d: %v", i, err)
		}
	}

	// Get all preempted tasks
	tasks, err := preemption.GetPreemptedTasks(ctx)
	if err != nil {
		t.Fatalf("failed to get preempted tasks: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("expected 3 preempted tasks, got %d", len(tasks))
	}
}

func TestPreemptionManager_ClearTaskState(t *testing.T) {
	preemption, _, mr := setupTestPreemption(t)
	defer mr.Close()
	defer preemption.client.Close()

	ctx := context.Background()

	// Save and preempt a task
	item := &QueueItem{
		TaskID:    "task-1",
		TenantID:  "tenant-1",
		Priority:  model.TaskPriorityNormal,
		CreatedAt: time.Now(),
	}
	preemption.Preempt(ctx, item, model.TaskStatusRunning, 50)

	// Clear the state
	if err := preemption.ClearTaskState(ctx, "task-1"); err != nil {
		t.Fatalf("failed to clear task state: %v", err)
	}

	// State should be gone
	_, err := preemption.GetTaskState(ctx, "task-1")
	if err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}

	// Should not be in preempted set
	isPreempted, _ := preemption.IsPreempted(ctx, "task-1")
	if isPreempted {
		t.Error("expected task not to be in preempted set after clear")
	}
}

func TestPreemptionManager_ClearPreemptedTracking(t *testing.T) {
	preemption, _, mr := setupTestPreemption(t)
	defer mr.Close()
	defer preemption.client.Close()

	ctx := context.Background()

	// Preempt some tasks
	for i := 0; i < 3; i++ {
		item := &QueueItem{
			TaskID:    string(rune('a' + i)),
			TenantID:  "tenant-1",
			Priority:  model.TaskPriorityNormal,
			CreatedAt: time.Now(),
		}
		preemption.Preempt(ctx, item, model.TaskStatusRunning, i*10)
	}

	// Clear all tracking
	if err := preemption.ClearPreemptedTracking(ctx); err != nil {
		t.Fatalf("failed to clear preempted tracking: %v", err)
	}

	// Should be empty
	tasks, err := preemption.GetPreemptedTasks(ctx)
	if err != nil {
		t.Fatalf("failed to get preempted tasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 preempted tasks, got %d", len(tasks))
	}
}

func TestPreemptionManager_TaskStateWithCheckpoint(t *testing.T) {
	preemption, _, mr := setupTestPreemption(t)
	defer mr.Close()
	defer preemption.client.Close()

	ctx := context.Background()

	state := &TaskState{
		TaskID:   "task-1",
		Status:   model.TaskStatusRunning,
		Progress: 50,
		Checkpoint: map[string]interface{}{
			"last_step":    10,
			"current_file": "/path/to/file.go",
			"context":      "some context data",
		},
		PreemptedAt: time.Now(),
	}

	err := preemption.SaveTaskState(ctx, state)
	if err != nil {
		t.Fatalf("failed to save task state: %v", err)
	}

	retrieved, err := preemption.GetTaskState(ctx, "task-1")
	if err != nil {
		t.Fatalf("failed to get task state: %v", err)
	}

	if retrieved.Checkpoint == nil {
		t.Fatal("expected checkpoint to be saved")
	}

	if retrieved.Checkpoint["last_step"].(float64) != 10 {
		t.Errorf("expected last_step 10, got %v", retrieved.Checkpoint["last_step"])
	}
}
