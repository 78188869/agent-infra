package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/example/agent-infra/internal/model"
	"github.com/redis/go-redis/v9"
)

const (
	// QueueKeyHigh is the Redis key for high priority queue.
	QueueKeyHigh = "scheduler:queue:high"
	// QueueKeyNormal is the Redis key for normal priority queue.
	QueueKeyNormal = "scheduler:queue:normal"
	// QueueKeyLow is the Redis key for low priority queue.
	QueueKeyLow = "scheduler:queue:low"
	// TaskMetaKeyPattern is the pattern for task metadata keys.
	TaskMetaKeyPattern = "scheduler:task:%s:meta"
)

// QueueItem represents an item in the priority queue.
type QueueItem struct {
	TaskID    string    `json:"task_id"`
	TenantID  string    `json:"tenant_id"`
	Priority  string    `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
}

// PriorityQueue manages task queues with priority support using Redis Sorted Sets.
type PriorityQueue struct {
	client *redis.Client
}

// NewPriorityQueue creates a new PriorityQueue instance.
func NewPriorityQueue(client *redis.Client) *PriorityQueue {
	return &PriorityQueue{client: client}
}

// getQueueKey returns the Redis key for the given priority level.
func (q *PriorityQueue) getQueueKey(priority string) string {
	switch priority {
	case model.TaskPriorityHigh:
		return QueueKeyHigh
	case model.TaskPriorityLow:
		return QueueKeyLow
	default:
		return QueueKeyNormal
	}
}

// Enqueue adds a task to the priority queue.
// Uses timestamp as score for FIFO ordering within same priority.
func (q *PriorityQueue) Enqueue(ctx context.Context, item *QueueItem) error {
	if item == nil {
		return fmt.Errorf("cannot enqueue nil item")
	}

	queueKey := q.getQueueKey(item.Priority)
	score := float64(item.CreatedAt.UnixNano())

	// Use Redis Sorted Set with timestamp as score for FIFO within priority
	err := q.client.ZAdd(ctx, queueKey, redis.Z{
		Score:  score,
		Member: item.TaskID,
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	// Store task metadata
	metaKey := fmt.Sprintf(TaskMetaKeyPattern, item.TaskID)
	return q.client.HSet(ctx, metaKey, map[string]interface{}{
		"task_id":    item.TaskID,
		"tenant_id":  item.TenantID,
		"priority":   item.Priority,
		"created_at": item.CreatedAt.UnixNano(),
	}).Err()
}

// Dequeue removes and returns the highest priority task from the queue.
// Returns nil if all queues are empty.
func (q *PriorityQueue) Dequeue(ctx context.Context) (*QueueItem, error) {
	// Try queues in priority order: high -> normal -> low
	queues := []string{QueueKeyHigh, QueueKeyNormal, QueueKeyLow}

	for _, queueKey := range queues {
		// Use ZPOPMIN for FIFO within priority (lowest score = earliest time)
		result, err := q.client.ZPopMin(ctx, queueKey, 1).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to dequeue: %w", err)
		}
		if len(result) == 0 {
			continue // Queue empty, try next
		}

		taskID, ok := result[0].Member.(string)
		if !ok {
			return nil, fmt.Errorf("invalid task id type in queue")
		}

		item, err := q.getTaskMeta(ctx, taskID)
		if err != nil {
			// Clean up metadata on error
			q.client.Del(ctx, fmt.Sprintf(TaskMetaKeyPattern, taskID))
			return nil, err
		}

		// Clean up metadata after successful dequeue
		q.client.Del(ctx, fmt.Sprintf(TaskMetaKeyPattern, taskID))

		return item, nil
	}

	return nil, nil // All queues empty
}

// getTaskMeta retrieves task metadata from Redis.
func (q *PriorityQueue) getTaskMeta(ctx context.Context, taskID string) (*QueueItem, error) {
	metaKey := fmt.Sprintf(TaskMetaKeyPattern, taskID)
	result, err := q.client.HGetAll(ctx, metaKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get task metadata: %w", err)
	}

	if len(result) == 0 {
		return nil, ErrTaskNotFound
	}

	// Parse created_at from string to int64 (nanoseconds)
	var createdAtNano int64
	if v, ok := result["created_at"]; ok {
		if _, err := fmt.Sscanf(v, "%d", &createdAtNano); err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}
	}

	return &QueueItem{
		TaskID:    result["task_id"],
		TenantID:  result["tenant_id"],
		Priority:  result["priority"],
		CreatedAt: time.Unix(0, createdAtNano),
	}, nil
}

// GetPosition returns the position of a task in the queue.
// Position 1 means the task is at the front of the queue.
// Returns 0 if the task is not found in any queue.
func (q *PriorityQueue) GetPosition(ctx context.Context, taskID string) (int, error) {
	// First, find which queue the task is in and its priority
	metaKey := fmt.Sprintf(TaskMetaKeyPattern, taskID)
	result, err := q.client.HGetAll(ctx, metaKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get task metadata: %w", err)
	}

	if len(result) == 0 {
		return 0, ErrTaskNotFound
	}

	priority := result["priority"]
	queueKey := q.getQueueKey(priority)

	// Get the task's rank in its queue (0-indexed)
	rank, err := q.client.ZRank(ctx, queueKey, taskID).Result()
	if err == redis.Nil {
		return 0, ErrTaskNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get task rank: %w", err)
	}

	// Calculate overall position considering higher priority queues
	position := int(rank) + 1

	// Add counts of higher priority queues
	switch priority {
	case model.TaskPriorityNormal:
		highCount, err := q.client.ZCard(ctx, QueueKeyHigh).Result()
		if err != nil {
			return 0, err
		}
		position += int(highCount)
	case model.TaskPriorityLow:
		highCount, err := q.client.ZCard(ctx, QueueKeyHigh).Result()
		if err != nil {
			return 0, err
		}
		normalCount, err := q.client.ZCard(ctx, QueueKeyNormal).Result()
		if err != nil {
			return 0, err
		}
		position += int(highCount) + int(normalCount)
	}

	return position, nil
}

// Remove removes a task from the queue.
func (q *PriorityQueue) Remove(ctx context.Context, taskID string, priority string) error {
	queueKey := q.getQueueKey(priority)
	removed, err := q.client.ZRem(ctx, queueKey, taskID).Result()
	if err != nil {
		return fmt.Errorf("failed to remove task from queue: %w", err)
	}

	if removed == 0 {
		return ErrTaskNotFound
	}

	// Clean up metadata
	metaKey := fmt.Sprintf(TaskMetaKeyPattern, taskID)
	q.client.Del(ctx, metaKey)

	return nil
}

// Size returns the total number of tasks in all queues.
func (q *PriorityQueue) Size(ctx context.Context) (int64, error) {
	high, err := q.client.ZCard(ctx, QueueKeyHigh).Result()
	if err != nil {
		return 0, err
	}
	normal, err := q.client.ZCard(ctx, QueueKeyNormal).Result()
	if err != nil {
		return 0, err
	}
	low, err := q.client.ZCard(ctx, QueueKeyLow).Result()
	if err != nil {
		return 0, err
	}
	return high + normal + low, nil
}

// SizeByPriority returns the number of tasks in a specific priority queue.
func (q *PriorityQueue) SizeByPriority(ctx context.Context, priority string) (int64, error) {
	queueKey := q.getQueueKey(priority)
	return q.client.ZCard(ctx, queueKey).Result()
}

// Clear removes all tasks from all queues.
func (q *PriorityQueue) Clear(ctx context.Context) error {
	// Get all task IDs from all queues to clean up metadata
	queues := []string{QueueKeyHigh, QueueKeyNormal, QueueKeyLow}
	for _, queueKey := range queues {
		taskIDs, err := q.client.ZRange(ctx, queueKey, 0, -1).Result()
		if err != nil {
			return err
		}
		for _, taskID := range taskIDs {
			q.client.Del(ctx, fmt.Sprintf(TaskMetaKeyPattern, taskID))
		}
		q.client.Del(ctx, queueKey)
	}
	return nil
}
