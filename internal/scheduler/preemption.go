package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// PreemptedTasksKey is the Redis key for tracking preempted tasks.
	PreemptedTasksKey = "scheduler:preempted:tasks"
	// TaskStateKeyPattern is the pattern for task state keys.
	TaskStateKeyPattern = "scheduler:task:%s:state"
	// TaskStateTTL is the TTL for task state storage.
	TaskStateTTL = 24 * time.Hour
)

// TaskState represents the saved state of a preempted task.
type TaskState struct {
	TaskID      string                 `json:"task_id"`
	Status      string                 `json:"status"`
	Progress    int                    `json:"progress"`
	Checkpoint  map[string]interface{} `json:"checkpoint,omitempty"`
	PreemptedAt time.Time              `json:"preempted_at"`
}

// PreemptionManager handles task preemption and state management.
type PreemptionManager struct {
	client *redis.Client
	queue  *PriorityQueue
}

// NewPreemptionManager creates a new PreemptionManager instance.
func NewPreemptionManager(client *redis.Client, queue *PriorityQueue) *PreemptionManager {
	return &PreemptionManager{
		client: client,
		queue:  queue,
	}
}

// Preempt saves the current task state and re-queues the task for later execution.
func (p *PreemptionManager) Preempt(ctx context.Context, item *QueueItem, status string, progress int) error {
	if item == nil {
		return fmt.Errorf("cannot preempt nil item")
	}

	// 1. Save current task state
	state := &TaskState{
		TaskID:      item.TaskID,
		Status:      status,
		Progress:    progress,
		PreemptedAt: time.Now(),
	}
	if err := p.SaveTaskState(ctx, state); err != nil {
		return fmt.Errorf("failed to save task state: %w", err)
	}

	// 2. Add to preempted tasks set for tracking
	if err := p.client.SAdd(ctx, PreemptedTasksKey, item.TaskID).Err(); err != nil {
		return fmt.Errorf("failed to track preempted task: %w", err)
	}

	// 3. Re-queue the preempted task with new timestamp for fair re-queuing
	requeuedItem := &QueueItem{
		TaskID:    item.TaskID,
		TenantID:  item.TenantID,
		Priority:  item.Priority,
		CreatedAt: time.Now(), // New timestamp for fair re-queuing
	}
	if err := p.queue.Enqueue(ctx, requeuedItem); err != nil {
		return fmt.Errorf("failed to requeue preempted task: %w", err)
	}

	return nil
}

// SaveTaskState saves the state of a task to Redis.
func (p *PreemptionManager) SaveTaskState(ctx context.Context, state *TaskState) error {
	if state == nil {
		return fmt.Errorf("cannot save nil state")
	}

	key := fmt.Sprintf(TaskStateKeyPattern, state.TaskID)
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal task state: %w", err)
	}

	return p.client.Set(ctx, key, data, TaskStateTTL).Err()
}

// GetTaskState retrieves the saved state of a task.
func (p *PreemptionManager) GetTaskState(ctx context.Context, taskID string) (*TaskState, error) {
	key := fmt.Sprintf(TaskStateKeyPattern, taskID)
	data, err := p.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task state: %w", err)
	}

	var state TaskState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task state: %w", err)
	}

	return &state, nil
}

// ClearTaskState removes the saved state of a task.
func (p *PreemptionManager) ClearTaskState(ctx context.Context, taskID string) error {
	key := fmt.Sprintf(TaskStateKeyPattern, taskID)
	p.client.Del(ctx, key)
	p.client.SRem(ctx, PreemptedTasksKey, taskID)
	return nil
}

// IsPreempted checks if a task has been preempted.
func (p *PreemptionManager) IsPreempted(ctx context.Context, taskID string) (bool, error) {
	return p.client.SIsMember(ctx, PreemptedTasksKey, taskID).Result()
}

// GetPreemptedTasks returns all preempted task IDs.
func (p *PreemptionManager) GetPreemptedTasks(ctx context.Context) ([]string, error) {
	return p.client.SMembers(ctx, PreemptedTasksKey).Result()
}

// ClearPreemptedTracking removes all preempted task tracking.
func (p *PreemptionManager) ClearPreemptedTracking(ctx context.Context) error {
	return p.client.Del(ctx, PreemptedTasksKey).Err()
}
