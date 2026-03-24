package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/example/agent-infra/internal/model"
	"github.com/redis/go-redis/v9"
)

func setupTestQueue(t *testing.T) (*PriorityQueue, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	queue := NewPriorityQueue(client)
	return queue, mr
}

func TestPriorityQueue_Enqueue(t *testing.T) {
	queue, mr := setupTestQueue(t)
	defer mr.Close()
	defer queue.client.Close()

	ctx := context.Background()
	now := time.Now()

	item := &QueueItem{
		TaskID:    "task-1",
		TenantID:  "tenant-1",
		Priority:  model.TaskPriorityNormal,
		CreatedAt: now,
	}

	err := queue.Enqueue(ctx, item)
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	// Verify task is in the queue
	size, err := queue.SizeByPriority(ctx, model.TaskPriorityNormal)
	if err != nil {
		t.Fatalf("failed to get queue size: %v", err)
	}
	if size != 1 {
		t.Errorf("expected queue size 1, got %d", size)
	}

	// Verify total size
	totalSize, err := queue.Size(ctx)
	if err != nil {
		t.Fatalf("failed to get total size: %v", err)
	}
	if totalSize != 1 {
		t.Errorf("expected total size 1, got %d", totalSize)
	}
}

func TestPriorityQueue_EnqueueNil(t *testing.T) {
	queue, mr := setupTestQueue(t)
	defer mr.Close()
	defer queue.client.Close()

	ctx := context.Background()

	err := queue.Enqueue(ctx, nil)
	if err == nil {
		t.Error("expected error for nil item, got nil")
	}
}

func TestPriorityQueue_EnqueueDifferentPriorities(t *testing.T) {
	queue, mr := setupTestQueue(t)
	defer mr.Close()
	defer queue.client.Close()

	ctx := context.Background()
	now := time.Now()

	items := []*QueueItem{
		{TaskID: "low-1", TenantID: "tenant-1", Priority: model.TaskPriorityLow, CreatedAt: now},
		{TaskID: "normal-1", TenantID: "tenant-1", Priority: model.TaskPriorityNormal, CreatedAt: now.Add(time.Millisecond)},
		{TaskID: "high-1", TenantID: "tenant-1", Priority: model.TaskPriorityHigh, CreatedAt: now.Add(2 * time.Millisecond)},
	}

	for _, item := range items {
		if err := queue.Enqueue(ctx, item); err != nil {
			t.Fatalf("failed to enqueue %s: %v", item.TaskID, err)
		}
	}

	// Verify each queue has correct count
	highSize, _ := queue.SizeByPriority(ctx, model.TaskPriorityHigh)
	normalSize, _ := queue.SizeByPriority(ctx, model.TaskPriorityNormal)
	lowSize, _ := queue.SizeByPriority(ctx, model.TaskPriorityLow)

	if highSize != 1 {
		t.Errorf("expected high queue size 1, got %d", highSize)
	}
	if normalSize != 1 {
		t.Errorf("expected normal queue size 1, got %d", normalSize)
	}
	if lowSize != 1 {
		t.Errorf("expected low queue size 1, got %d", lowSize)
	}
}

func TestPriorityQueue_Dequeue_PriorityOrder(t *testing.T) {
	queue, mr := setupTestQueue(t)
	defer mr.Close()
	defer queue.client.Close()

	ctx := context.Background()
	now := time.Now()

	// Add tasks in random order
	items := []*QueueItem{
		{TaskID: "low-1", TenantID: "tenant-1", Priority: model.TaskPriorityLow, CreatedAt: now},
		{TaskID: "high-1", TenantID: "tenant-1", Priority: model.TaskPriorityHigh, CreatedAt: now.Add(time.Millisecond)},
		{TaskID: "normal-1", TenantID: "tenant-1", Priority: model.TaskPriorityNormal, CreatedAt: now.Add(2 * time.Millisecond)},
	}

	for _, item := range items {
		if err := queue.Enqueue(ctx, item); err != nil {
			t.Fatalf("failed to enqueue: %v", err)
		}
	}

	// Dequeue should return in priority order: high, normal, low
	expectedOrder := []string{"high-1", "normal-1", "low-1"}
	for _, expectedID := range expectedOrder {
		item, err := queue.Dequeue(ctx)
		if err != nil {
			t.Fatalf("failed to dequeue: %v", err)
		}
		if item == nil {
			t.Fatalf("expected item %s, got nil", expectedID)
		}
		if item.TaskID != expectedID {
			t.Errorf("expected task %s, got %s", expectedID, item.TaskID)
		}
	}

	// Queue should be empty
	item, err := queue.Dequeue(ctx)
	if err != nil {
		t.Fatalf("unexpected error on empty dequeue: %v", err)
	}
	if item != nil {
		t.Errorf("expected nil on empty queue, got %v", item)
	}
}

func TestPriorityQueue_Dequeue_FIFO(t *testing.T) {
	queue, mr := setupTestQueue(t)
	defer mr.Close()
	defer queue.client.Close()

	ctx := context.Background()
	now := time.Now()

	// Add tasks with same priority but different timestamps
	items := []*QueueItem{
		{TaskID: "task-3", TenantID: "tenant-1", Priority: model.TaskPriorityNormal, CreatedAt: now.Add(2 * time.Millisecond)},
		{TaskID: "task-1", TenantID: "tenant-1", Priority: model.TaskPriorityNormal, CreatedAt: now},
		{TaskID: "task-2", TenantID: "tenant-1", Priority: model.TaskPriorityNormal, CreatedAt: now.Add(time.Millisecond)},
	}

	for _, item := range items {
		if err := queue.Enqueue(ctx, item); err != nil {
			t.Fatalf("failed to enqueue: %v", err)
		}
	}

	// Dequeue should return in FIFO order (by CreatedAt)
	expectedOrder := []string{"task-1", "task-2", "task-3"}
	for _, expectedID := range expectedOrder {
		item, err := queue.Dequeue(ctx)
		if err != nil {
			t.Fatalf("failed to dequeue: %v", err)
		}
		if item.TaskID != expectedID {
			t.Errorf("expected task %s, got %s", expectedID, item.TaskID)
		}
	}
}

func TestPriorityQueue_Dequeue_EmptyQueue(t *testing.T) {
	queue, mr := setupTestQueue(t)
	defer mr.Close()
	defer queue.client.Close()

	ctx := context.Background()

	item, err := queue.Dequeue(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item != nil {
		t.Errorf("expected nil on empty queue, got %v", item)
	}
}

func TestPriorityQueue_GetPosition(t *testing.T) {
	queue, mr := setupTestQueue(t)
	defer mr.Close()
	defer queue.client.Close()

	ctx := context.Background()
	now := time.Now()

	// Add tasks to different queues
	items := []*QueueItem{
		{TaskID: "high-1", TenantID: "tenant-1", Priority: model.TaskPriorityHigh, CreatedAt: now},
		{TaskID: "high-2", TenantID: "tenant-1", Priority: model.TaskPriorityHigh, CreatedAt: now.Add(time.Millisecond)},
		{TaskID: "normal-1", TenantID: "tenant-1", Priority: model.TaskPriorityNormal, CreatedAt: now.Add(2 * time.Millisecond)},
		{TaskID: "low-1", TenantID: "tenant-1", Priority: model.TaskPriorityLow, CreatedAt: now.Add(3 * time.Millisecond)},
	}

	for _, item := range items {
		if err := queue.Enqueue(ctx, item); err != nil {
			t.Fatalf("failed to enqueue: %v", err)
		}
	}

	tests := []struct {
		taskID       string
		expectedPos  int
	}{
		{"high-1", 1},
		{"high-2", 2},
		{"normal-1", 3},
		{"low-1", 4},
	}

	for _, tt := range tests {
		pos, err := queue.GetPosition(ctx, tt.taskID)
		if err != nil {
			t.Fatalf("failed to get position for %s: %v", tt.taskID, err)
		}
		if pos != tt.expectedPos {
			t.Errorf("task %s: expected position %d, got %d", tt.taskID, tt.expectedPos, pos)
		}
	}
}

func TestPriorityQueue_GetPosition_NotFound(t *testing.T) {
	queue, mr := setupTestQueue(t)
	defer mr.Close()
	defer queue.client.Close()

	ctx := context.Background()

	_, err := queue.GetPosition(ctx, "non-existent")
	if err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestPriorityQueue_Remove(t *testing.T) {
	queue, mr := setupTestQueue(t)
	defer mr.Close()
	defer queue.client.Close()

	ctx := context.Background()
	now := time.Now()

	item := &QueueItem{
		TaskID:    "task-1",
		TenantID:  "tenant-1",
		Priority:  model.TaskPriorityNormal,
		CreatedAt: now,
	}

	if err := queue.Enqueue(ctx, item); err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	// Remove the task
	err := queue.Remove(ctx, "task-1", model.TaskPriorityNormal)
	if err != nil {
		t.Fatalf("failed to remove: %v", err)
	}

	// Verify queue is empty
	size, _ := queue.Size(ctx)
	if size != 0 {
		t.Errorf("expected empty queue, got size %d", size)
	}

	// Position should return not found
	_, err = queue.GetPosition(ctx, "task-1")
	if err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound after remove, got %v", err)
	}
}

func TestPriorityQueue_Remove_NotFound(t *testing.T) {
	queue, mr := setupTestQueue(t)
	defer mr.Close()
	defer queue.client.Close()

	ctx := context.Background()

	err := queue.Remove(ctx, "non-existent", model.TaskPriorityNormal)
	if err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestPriorityQueue_Clear(t *testing.T) {
	queue, mr := setupTestQueue(t)
	defer mr.Close()
	defer queue.client.Close()

	ctx := context.Background()
	now := time.Now()

	// Add multiple tasks
	for i := 0; i < 5; i++ {
		item := &QueueItem{
			TaskID:    string(rune('a' + i)),
			TenantID:  "tenant-1",
			Priority:  model.TaskPriorityHigh,
			CreatedAt: now.Add(time.Duration(i) * time.Millisecond),
		}
		if err := queue.Enqueue(ctx, item); err != nil {
			t.Fatalf("failed to enqueue: %v", err)
		}
	}

	// Clear all queues
	if err := queue.Clear(ctx); err != nil {
		t.Fatalf("failed to clear: %v", err)
	}

	// Verify all queues are empty
	size, _ := queue.Size(ctx)
	if size != 0 {
		t.Errorf("expected empty queue after clear, got size %d", size)
	}
}
