# Task Scheduler Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the task scheduler engine with priority queue, rate limiting, and preemption mechanism based on Redis.

**Architecture:** The scheduler consists of three main components: PriorityQueue (Redis Sorted Set based), RateLimiter (token bucket for tenant quotas), and PreemptionManager (high-priority task preemption). Uses Redis for distributed state management.

**Tech Stack:** Go 1.22 + go-redis/redis v9 + context + sync

**Related Issue:** #7 - MVP Phase 3 - Task Scheduler Engine

**Reference Documents:**
- TRD: `docs/v1.0-mvp/TRD.md` §4.2
- Knowledge: `docs/knowledge/scheduler.md`
- Dependency: Issue #6 (Database Models) - Completed

---

## Verification Criteria

Based on Issue #7 Acceptance Criteria:

| Criteria | Verification Method |
|----------|---------------------|
| 优先级队列正确排序 | Unit tests for high > normal > low, FIFO within same priority |
| 租户配额限流生效 | Unit tests for concurrency limit exceeded scenarios |
| 全局并发限制生效 | Unit tests for global limit enforcement |
| 抢占机制正确保存和恢复任务状态 | Unit tests for preemption flow |
| 排队位置查询准确 | Unit tests for GetPosition |
| 单元测试覆盖率 > 80% | `go test -cover ./internal/scheduler/...` |

**Acceptance Test Checklist:**
- [ ] Schedule() adds task to correct priority queue
- [ ] Dequeue() returns tasks in priority order (high > normal > low)
- [ ] Same priority tasks are returned in FIFO order
- [ ] Tenant concurrency limit blocks tasks when quota exceeded
- [ ] Global concurrency limit blocks tasks when exceeded
- [ ] Daily task limit blocks tasks when exceeded
- [ ] Preempt() saves current task state and re-queues
- [ ] GetPosition() returns accurate queue position
- [ ] Test coverage > 80% verified

---

## File Structure

### New Files to Create

```
internal/
├── scheduler/
│   ├── scheduler.go           # Scheduler interface and main implementation
│   ├── scheduler_test.go      # Scheduler unit tests
│   ├── queue.go               # Priority queue management (Redis Sorted Set)
│   ├── queue_test.go          # Queue unit tests
│   ├── ratelimiter.go         # Rate limiter (tenant quota, global limit)
│   ├── ratelimiter_test.go    # Rate limiter unit tests
│   ├── preemption.go          # Preemption manager
│   ├── preemption_test.go     # Preemption unit tests
│   └── errors.go              # Scheduler-specific errors
│
├── config/
│   ├── redis.go               # Redis connection configuration (NEW)
│   └── redis_test.go          # Redis config tests (NEW)
│
└── repository/
    └── task_quota_repo.go     # Task quota tracking repository (NEW)
```

### Files to Modify

- `go.mod` - Add go-redis/redis v9 dependency
- `internal/config/database.go` - May need to coordinate with quota queries

---

## Task 1: Redis Configuration and Infrastructure

**Files:**
- Modify: `go.mod`
- Create: `internal/config/redis.go`
- Create: `internal/scheduler/errors.go`

- [ ] **Step 1: Add Redis dependency to go.mod**

```bash
go get github.com/redis/go-redis/v9
```

- [ ] **Step 2: Write failing test for Redis configuration**

Create `internal/config/redis_test.go`:
```go
func TestNewRedisClient(t *testing.T) {
    cfg := DefaultRedisConfig()
    client, err := NewRedisClient(cfg)
    // Test should pass with miniredis or fail gracefully without Redis
}
```

- [ ] **Step 3: Implement Redis configuration**

Create `internal/config/redis.go`:
```go
type RedisConfig struct {
    Addr         string
    Password     string
    DB           int
    PoolSize     int
    MinIdleConns int
}

type RedisClient struct {
    *redis.Client
    Config RedisConfig
}

func NewRedisClient(cfg RedisConfig) (*RedisClient, error) {
    client := redis.NewClient(&redis.Options{
        Addr:         cfg.Addr,
        Password:     cfg.Password,
        DB:           cfg.DB,
        PoolSize:     cfg.PoolSize,
        MinIdleConns: cfg.MinIdleConns,
    })
    // Verify connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("failed to connect to Redis: %w", err)
    }
    return &RedisClient{Client: client, Config: cfg}, nil
}
```

- [ ] **Step 4: Create scheduler error types**

Create `internal/scheduler/errors.go`:
```go
var (
    ErrTaskNotFound      = errors.New("task not found in queue")
    ErrQueueFull         = errors.New("queue is full")
    ErrQuotaExceeded     = errors.New("tenant quota exceeded")
    ErrGlobalLimitExceeded = errors.New("global concurrency limit exceeded")
    ErrDailyLimitExceeded  = errors.New("daily task limit exceeded")
    ErrPreemptionFailed    = errors.New("preemption failed")
)
```

- [ ] **Step 5: Run tests and verify**

Run: `go test ./internal/config/... -v`

- [ ] **Step 6: Commit infrastructure**

```bash
git add go.mod go.sum internal/config/redis.go internal/config/redis_test.go internal/scheduler/errors.go
git commit -m "feat(scheduler): add Redis configuration and error types"
```

---

## Task 2: Priority Queue Implementation

**Files:**
- Create: `internal/scheduler/queue.go`
- Create: `internal/scheduler/queue_test.go`

- [ ] **Step 1: Write failing test for PriorityQueue interface**

Create `internal/scheduler/queue_test.go`:
```go
func TestPriorityQueue_Enqueue(t *testing.T) {
    // Test adding tasks to different priority queues
}

func TestPriorityQueue_Dequeue_PriorityOrder(t *testing.T) {
    // Test that high priority tasks are returned first
    // Add low, normal, high tasks
    // Dequeue should return: high, normal, low
}

func TestPriorityQueue_Dequeue_FIFO(t *testing.T) {
    // Test that same priority tasks are returned in FIFO order
}

func TestPriorityQueue_GetPosition(t *testing.T) {
    // Test position query
}
```

- [ ] **Step 2: Define PriorityQueue interface and types**

Create `internal/scheduler/queue.go`:
```go
const (
    QueueKeyHigh   = "scheduler:queue:high"
    QueueKeyNormal = "scheduler:queue:normal"
    QueueKeyLow    = "scheduler:queue:low"
    TaskMetaKey    = "scheduler:task:%s:meta"
)

// QueueItem represents an item in the priority queue
type QueueItem struct {
    TaskID    string    `json:"task_id"`
    TenantID  string    `json:"tenant_id"`
    Priority  Priority  `json:"priority"`
    CreatedAt time.Time `json:"created_at"`
}

// PriorityQueue manages task queues with priority support
type PriorityQueue struct {
    client *redis.Client
}

func NewPriorityQueue(client *redis.Client) *PriorityQueue {
    return &PriorityQueue{client: client}
}
```

- [ ] **Step 3: Implement Enqueue method**

```go
func (q *PriorityQueue) Enqueue(ctx context.Context, item *QueueItem) error {
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
    metaKey := fmt.Sprintf(TaskMetaKey, item.TaskID)
    return q.client.HSet(ctx, metaKey, map[string]interface{}{
        "tenant_id":  item.TenantID,
        "priority":   string(item.Priority),
        "created_at": item.CreatedAt.Unix(),
    }).Err()
}
```

- [ ] **Step 4: Implement Dequeue method**

```go
func (q *PriorityQueue) Dequeue(ctx context.Context) (*QueueItem, error) {
    // Try queues in priority order: high -> normal -> low
    queues := []string{QueueKeyHigh, QueueKeyNormal, QueueKeyLow}

    for _, queueKey := range queues {
        // Use ZPOPMIN for FIFO within priority
        result, err := q.client.ZPopMin(ctx, queueKey).Result()
        if err == redis.Nil {
            continue // Queue empty, try next
        }
        if err != nil {
            return nil, fmt.Errorf("failed to dequeue: %w", err)
        }

        taskID := result.Member.(string)
        return q.getTaskMeta(ctx, taskID)
    }

    return nil, nil // All queues empty
}
```

- [ ] **Step 5: Implement GetPosition method**

```go
func (q *PriorityQueue) GetPosition(ctx context.Context, taskID string) (int, error) {
    // Find task in all queues and calculate position
    // This requires checking all priority queues
}
```

- [ ] **Step 6: Run tests and verify**

Run: `go test ./internal/scheduler/... -v -run TestPriorityQueue`

- [ ] **Step 7: Commit priority queue**

```bash
git add internal/scheduler/queue.go internal/scheduler/queue_test.go
git commit -m "feat(scheduler): implement priority queue with Redis Sorted Set"
```

---

## Task 3: Rate Limiter Implementation

**Files:**
- Create: `internal/scheduler/ratelimiter.go`
- Create: `internal/scheduler/ratelimiter_test.go`

- [ ] **Step 1: Write failing test for RateLimiter interface**

Create `internal/scheduler/ratelimiter_test.go`:
```go
func TestRateLimiter_TenantConcurrency(t *testing.T) {
    // Test that tenant concurrency limit is enforced
}

func TestRateLimiter_GlobalConcurrency(t *testing.T) {
    // Test that global concurrency limit is enforced
}

func TestRateLimiter_DailyTaskLimit(t *testing.T) {
    // Test that daily task limit is enforced
}

func TestRateLimiter_ReserveAndRelease(t *testing.T) {
    // Test resource reservation and release
}
```

- [ ] **Step 2: Define RateLimiter types**

Create `internal/scheduler/ratelimiter.go`:
```go
const (
    TenantQuotaKey   = "scheduler:tenant:%s:quota"
    GlobalQuotaKey   = "scheduler:global:quota"
)

type QuotaUsage struct {
    CurrentConcurrency int   `json:"current_concurrency"`
    TodayTasks         int   `json:"today_tasks"`
}

type RateLimiter struct {
    client        *redis.Client
    globalLimit   int
}

func NewRateLimiter(client *redis.Client, globalLimit int) *RateLimiter {
    return &RateLimiter{
        client:      client,
        globalLimit: globalLimit,
    }
}
```

- [ ] **Step 3: Implement Allow method (tenant quota check)**

```go
func (r *RateLimiter) Allow(ctx context.Context, tenantID string, tenantQuota *model.ResourceQuota) error {
    // Check tenant concurrency
    usage, err := r.GetUsage(ctx, tenantID)
    if err != nil {
        return err
    }

    if usage.CurrentConcurrency >= tenantQuota.Concurrency {
        return ErrQuotaExceeded
    }

    if usage.TodayTasks >= tenantQuota.DailyTasks {
        return ErrDailyLimitExceeded
    }

    // Check global limit
    globalUsage, err := r.getGlobalUsage(ctx)
    if err != nil {
        return err
    }
    if globalUsage.CurrentConcurrency >= r.globalLimit {
        return ErrGlobalLimitExceeded
    }

    return nil
}
```

- [ ] **Step 4: Implement Reserve and Release methods**

```go
func (r *RateLimiter) Reserve(ctx context.Context, tenantID string) error {
    // Atomically increment tenant concurrency counter
    key := fmt.Sprintf(TenantQuotaKey, tenantID)
    err := r.client.HIncrBy(ctx, key, "current_concurrency", 1).Err()
    if err != nil {
        return err
    }
    // Increment global counter
    return r.client.HIncrBy(ctx, GlobalQuotaKey, "current_concurrency", 1).Err()
}

func (r *RateLimiter) Release(ctx context.Context, tenantID string) error {
    // Decrement counters
    key := fmt.Sprintf(TenantQuotaKey, tenantID)
    r.client.HIncrBy(ctx, key, "current_concurrency", -1)
    return r.client.HIncrBy(ctx, GlobalQuotaKey, "current_concurrency", -1).Err()
}
```

- [ ] **Step 5: Implement GetUsage method**

```go
func (r *RateLimiter) GetUsage(ctx context.Context, tenantID string) (*QuotaUsage, error) {
    key := fmt.Sprintf(TenantQuotaKey, tenantID)
    result, err := r.client.HGetAll(ctx, key).Result()
    if err != nil {
        return nil, err
    }

    usage := &QuotaUsage{}
    if v, ok := result["current_concurrency"]; ok {
        usage.CurrentConcurrency, _ = strconv.Atoi(v)
    }
    if v, ok := result["today_tasks"]; ok {
        usage.TodayTasks, _ = strconv.Atoi(v)
    }

    return usage, nil
}
```

- [ ] **Step 6: Run tests and verify**

Run: `go test ./internal/scheduler/... -v -run TestRateLimiter`

- [ ] **Step 7: Commit rate limiter**

```bash
git add internal/scheduler/ratelimiter.go internal/scheduler/ratelimiter_test.go
git commit -m "feat(scheduler): implement rate limiter for tenant and global quotas"
```

---

## Task 4: Preemption Manager Implementation

**Files:**
- Create: `internal/scheduler/preemption.go`
- Create: `internal/scheduler/preemption_test.go`

- [ ] **Step 1: Write failing test for PreemptionManager**

Create `internal/scheduler/preemption_test.go`:
```go
func TestPreemptionManager_Preempt(t *testing.T) {
    // Test preempting a lower priority task
}

func TestPreemptionManager_SaveTaskState(t *testing.T) {
    // Test saving task state before preemption
}

func TestPreemptionManager_RequeuePreemptedTask(t *testing.T) {
    // Test re-queuing preempted task
}
```

- [ ] **Step 2: Define PreemptionManager types**

Create `internal/scheduler/preemption.go`:
```go
const (
    PreemptedTasksKey = "scheduler:preempted:tasks"
    TaskStateKey      = "scheduler:task:%s:state"
)

type TaskState struct {
    TaskID      string                 `json:"task_id"`
    Status      model.TaskStatus       `json:"status"`
    Progress    int                    `json:"progress"`
    Checkpoint  map[string]interface{} `json:"checkpoint"`
    PreemptedAt time.Time              `json:"preempted_at"`
}

type PreemptionManager struct {
    client *redis.Client
    queue  *PriorityQueue
}

func NewPreemptionManager(client *redis.Client, queue *PriorityQueue) *PreemptionManager {
    return &PreemptionManager{
        client: client,
        queue:  queue,
    }
}
```

- [ ] **Step 3: Implement Preempt method**

```go
func (p *PreemptionManager) Preempt(ctx context.Context, taskID string, task *model.Task) error {
    // 1. Save current task state
    state := &TaskState{
        TaskID:      taskID,
        Status:      task.Status,
        Progress:    task.Progress,
        PreemptedAt: time.Now(),
    }
    if err := p.SaveTaskState(ctx, state); err != nil {
        return fmt.Errorf("failed to save task state: %w", err)
    }

    // 2. Re-queue the preempted task
    item := &QueueItem{
        TaskID:    taskID,
        TenantID:  task.TenantID,
        Priority:  task.Priority,
        CreatedAt: time.Now(), // New timestamp for fair re-queuing
    }
    if err := p.queue.Enqueue(ctx, item); err != nil {
        return fmt.Errorf("failed to requeue preempted task: %w", err)
    }

    return nil
}
```

- [ ] **Step 4: Implement SaveTaskState and GetTaskState**

```go
func (p *PreemptionManager) SaveTaskState(ctx context.Context, state *TaskState) error {
    key := fmt.Sprintf(TaskStateKey, state.TaskID)
    data, err := json.Marshal(state)
    if err != nil {
        return err
    }
    return p.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (p *PreemptionManager) GetTaskState(ctx context.Context, taskID string) (*TaskState, error) {
    key := fmt.Sprintf(TaskStateKey, taskID)
    data, err := p.client.Get(ctx, key).Bytes()
    if err == redis.Nil {
        return nil, ErrTaskNotFound
    }
    if err != nil {
        return nil, err
    }

    var state TaskState
    if err := json.Unmarshal(data, &state); err != nil {
        return nil, err
    }
    return &state, nil
}
```

- [ ] **Step 5: Run tests and verify**

Run: `go test ./internal/scheduler/... -v -run TestPreemption`

- [ ] **Step 6: Commit preemption manager**

```bash
git add internal/scheduler/preemption.go internal/scheduler/preemption_test.go
git commit -m "feat(scheduler): implement preemption manager for task state saving"
```

---

## Task 5: Main Scheduler Implementation

**Files:**
- Create: `internal/scheduler/scheduler.go`
- Create: `internal/scheduler/scheduler_test.go`

- [ ] **Step 1: Write failing test for Scheduler interface**

Create `internal/scheduler/scheduler_test.go`:
```go
func TestScheduler_Schedule(t *testing.T) {
    // Test scheduling a task into queue
}

func TestScheduler_Dequeue_WithRateLimit(t *testing.T) {
    // Test dequeue respects rate limits
}

func TestScheduler_GetPosition(t *testing.T) {
    // Test position query
}

func TestScheduler_Preempt(t *testing.T) {
    // Test preemption through scheduler
}

func TestScheduler_GracefulShutdown(t *testing.T) {
    // Test graceful shutdown
}
```

- [ ] **Step 2: Define Scheduler interface and main struct**

Create `internal/scheduler/scheduler.go`:
```go
// Scheduler coordinates priority queue, rate limiting, and preemption
type Scheduler interface {
    Schedule(ctx context.Context, task *model.Task) error
    Dequeue(ctx context.Context) (*model.Task, error)
    GetPosition(ctx context.Context, taskID string) (int, error)
    Preempt(ctx context.Context, taskID string) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}

type TaskScheduler struct {
    queue       *PriorityQueue
    limiter     *RateLimiter
    preemption  *PreemptionManager
    taskRepo    repository.TaskRepository
    tenantRepo  repository.TenantRepository

    running     atomic.Bool
    stopCh      chan struct{}
}

func NewTaskScheduler(
    client *redis.Client,
    taskRepo repository.TaskRepository,
    tenantRepo repository.TenantRepository,
    globalLimit int,
) *TaskScheduler {
    queue := NewPriorityQueue(client)
    return &TaskScheduler{
        queue:      queue,
        limiter:    NewRateLimiter(client, globalLimit),
        preemption: NewPreemptionManager(client, queue),
        taskRepo:   taskRepo,
        tenantRepo: tenantRepo,
        stopCh:     make(chan struct{}),
    }
}
```

- [ ] **Step 3: Implement Schedule method**

```go
func (s *TaskScheduler) Schedule(ctx context.Context, task *model.Task) error {
    item := &QueueItem{
        TaskID:    task.ID,
        TenantID:  task.TenantID,
        Priority:  task.Priority,
        CreatedAt: task.CreatedAt,
    }

    if err := s.queue.Enqueue(ctx, item); err != nil {
        return fmt.Errorf("failed to schedule task: %w", err)
    }

    // Update task status to scheduled
    return s.taskRepo.UpdateStatus(ctx, task.ID, model.TaskStatusScheduled, "queued")
}
```

- [ ] **Step 4: Implement Dequeue method with rate limiting**

```go
func (s *TaskScheduler) Dequeue(ctx context.Context) (*model.Task, error) {
    for {
        item, err := s.queue.Dequeue(ctx)
        if err != nil {
            return nil, err
        }
        if item == nil {
            return nil, nil // Queue empty
        }

        // Get tenant quota
        tenant, err := s.tenantRepo.GetByID(ctx, item.TenantID)
        if err != nil {
            // Re-queue on error
            s.queue.Enqueue(ctx, item)
            continue
        }

        quota := &model.ResourceQuota{
            Concurrency: tenant.QuotaConcurrency,
            DailyTasks:  tenant.QuotaDailyTasks,
        }

        // Check rate limit
        if err := s.limiter.Allow(ctx, item.TenantID, quota); err != nil {
            // Rate limited - re-queue with delay
            time.Sleep(100 * time.Millisecond)
            s.queue.Enqueue(ctx, item)
            continue
        }

        // Reserve resources
        if err := s.limiter.Reserve(ctx, item.TenantID); err != nil {
            s.queue.Enqueue(ctx, item)
            continue
        }

        // Get full task from repository
        task, err := s.taskRepo.GetByID(ctx, item.TaskID)
        if err != nil {
            s.limiter.Release(ctx, item.TenantID)
            return nil, err
        }

        return task, nil
    }
}
```

- [ ] **Step 5: Implement GetPosition method**

```go
func (s *TaskScheduler) GetPosition(ctx context.Context, taskID string) (int, error) {
    return s.queue.GetPosition(ctx, taskID)
}
```

- [ ] **Step 6: Implement Preempt method**

```go
func (s *TaskScheduler) Preempt(ctx context.Context, taskID string) error {
    task, err := s.taskRepo.GetByID(ctx, taskID)
    if err != nil {
        return err
    }

    if !task.IsRunning() {
        return fmt.Errorf("task is not running, cannot preempt")
    }

    // Perform preemption
    if err := s.preemption.Preempt(ctx, taskID, task); err != nil {
        return err
    }

    // Release resources
    s.limiter.Release(ctx, task.TenantID)

    // Update task status
    return s.taskRepo.UpdateStatus(ctx, taskID, model.TaskStatusPaused, "preempted")
}
```

- [ ] **Step 7: Implement Start and Stop for graceful shutdown**

```go
func (s *TaskScheduler) Start(ctx context.Context) error {
    s.running.Store(true)
    // Start background processing if needed
    return nil
}

func (s *TaskScheduler) Stop(ctx context.Context) error {
    s.running.Store(false)
    close(s.stopCh)
    // Wait for current operations to complete
    return nil
}
```

- [ ] **Step 8: Run tests and verify**

Run: `go test ./internal/scheduler/... -v`

- [ ] **Step 9: Commit main scheduler**

```bash
git add internal/scheduler/scheduler.go internal/scheduler/scheduler_test.go
git commit -m "feat(scheduler): implement main TaskScheduler with rate limiting and preemption"
```

---

## Task 6: Integration and Final Verification

**Files:**
- Modify: `internal/api/handler/health.go` (add Redis health check)
- Run: Full test suite

- [ ] **Step 1: Enhance health check with Redis**

Modify `internal/api/handler/health.go` to include Redis connectivity check.

- [ ] **Step 2: Run full test suite**

```bash
go test ./internal/scheduler/... -v -cover
```

- [ ] **Step 3: Verify coverage > 80%**

```bash
go test ./internal/scheduler/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

- [ ] **Step 4: Run all tests**

```bash
go test ./... -v
```

- [ ] **Step 5: Commit integration changes**

```bash
git add internal/api/handler/health.go
git commit -m "feat(scheduler): add Redis health check and finalize integration"
```

- [ ] **Step 6: Create PR**

```bash
git push origin feature/issue-7-task-scheduler
gh pr create --title "feat(scheduler): implement Task Scheduler Engine" --body "..."
```

---

## Dependency Graph

```
Task 1 (Redis Config + Errors)
    │
    ├── Task 2 (Priority Queue) ──┐
    │                             │
    ├── Task 3 (Rate Limiter) ────┼──→ Task 5 (Main Scheduler)
    │                             │
    └── Task 4 (Preemption) ──────┘
                                        │
                                        ▼
                                  Task 6 (Integration)
```

## Estimated Effort

| Task | Estimated Time |
|------|---------------|
| Task 1: Redis Config | 0.5 day |
| Task 2: Priority Queue | 1 day |
| Task 3: Rate Limiter | 0.5 day |
| Task 4: Preemption | 0.5 day |
| Task 5: Main Scheduler | 1 day |
| Task 6: Integration | 0.5 day |
| **Total** | **3-4 days** |

---

## Notes for Implementer

1. **Use miniredis for testing**: Use `github.com/alicebob/miniredis/v2` for unit tests without real Redis
2. **Lua scripts for atomicity**: Consider using Redis Lua scripts for complex atomic operations
3. **TDD approach**: Write failing tests first, then implement
4. **Graceful shutdown**: Ensure in-progress tasks can be re-queued on shutdown
5. **Monitoring hooks**: Add metrics collection points for queue length, wait time
6. **Redis key expiration**: Set TTL on task metadata to prevent memory leaks
