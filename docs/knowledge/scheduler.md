# Scheduler Knowledge

> **Last Updated**: 2026-03-24
> **PRD Version**: v0.7-draft
> **TRD Version**: v2.4
> **Implementation Status**: ✅ Completed (Issue #7)

## 1. Overview

Scheduler 模块负责任务调度、排队管理和限流控制。

**模块职责**：
- 优先级队列管理
- 租户级限流控制
- 抢占式调度
- 调度状态同步

**核心概念**：
- **Priority Queue**: 基于 Redis 的优先级队列
- **Rate Limiter**: 令牌桶限流器
- **Preemption**: 高优先级任务抢占低优先级任务资源

## 2. Product Requirements (from PRD)

### 2.1 调度策略

| 策略 | 说明 |
|------|------|
| 优先级调度 | high > normal > low，同优先级 FIFO |
| 公平调度 | 租户级并发限制，防止单租户占用过多资源 |
| 抢占调度 | 高优先级任务可抢占低优先级任务资源（可选） |

### 2.2 资源配额

| 配额项 | 说明 | 默认值 |
|--------|------|--------|
| quota_concurrency | 最大并发任务数 | 50 |
| quota_daily_tasks | 每日任务数上限 | 1000 |
| quota_cpu | CPU 核心数上限 | 100 |
| quota_memory | 内存上限 (GB) | 200 |

## 3. Technical Design (from TRD)

### 3.1 架构设计

```
┌─────────────────────────────────────────────────────────────────────┐
│                        调度引擎架构                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │  Task Scheduler                                                │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │ │
│  │  │  Queue      │  │ RateLimiter │  │ Preemption  │            │ │
│  │  │  Manager    │  │             │  │  Manager    │            │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘            │ │
│  └────────────────────────────────────────────────────────────────┘ │
│         │                   │                   │                   │
│         ▼                   ▼                   ▼                   │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                     Redis                                    │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │   │
│  │  │ High     │ │ Normal   │ │ Low      │ │ Tenant   │       │   │
│  │  │ Queue    │ │ Queue    │ │ Queue    │ │ Quota    │       │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 模块结构

```
internal/scheduler/
├── scheduler.go      # 调度器主逻辑
├── queue.go          # 优先级队列管理
├── ratelimiter.go    # 限流器
└── preemption.go     # 抢占逻辑
```

### 3.3 核心接口

> **Note**: 以下为实际实现的接口定义 (Issue #7)

```go
// Scheduler 调度器接口
type Scheduler interface {
    Schedule(ctx context.Context, task *model.Task) error
    Dequeue(ctx context.Context) (*DequeuedTask, error)
    GetPosition(ctx context.Context, taskID string) (int, error)
    Preempt(ctx context.Context, taskID string, status string, progress int) error
    Complete(ctx context.Context, taskID string, tenantID string) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    IsRunning() bool
}

// DequeuedTask 出队任务结构
type DequeuedTask struct {
    Task      *model.Task
    QueueItem *QueueItem
}

// RateLimiter 限流器
type RateLimiter struct {
    client      *redis.Client
    globalLimit int
}

func (r *RateLimiter) Allow(ctx context.Context, tenantID string, quota *TenantQuota) error
func (r *RateLimiter) Reserve(ctx context.Context, tenantID string) error
func (r *RateLimiter) Release(ctx context.Context, tenantID string) error
func (r *RateLimiter) GetUsage(ctx context.Context, tenantID string) (*QuotaUsage, error)
func (r *RateLimiter) GetGlobalConcurrency(ctx context.Context) (int, error)

// PreemptionManager 抢占管理器
type PreemptionManager struct {
    client *redis.Client
    queue  *PriorityQueue
}

func (p *PreemptionManager) Preempt(ctx context.Context, item *QueueItem, status string, progress int) error
func (p *PreemptionManager) SaveTaskState(ctx context.Context, state *TaskState) error
func (p *PreemptionManager) GetTaskState(ctx context.Context, taskID string) (*TaskState, error)
func (p *PreemptionManager) ClearTaskState(ctx context.Context, taskID string) error
```

### 3.4 Redis 队列设计

> **Actual Implementation**: 使用单一 Sorted Set 配合优先级编码 score 实现优先级队列

```
# 优先级队列 (Sorted Set) - 使用优先级编码 score
# score = priority * 10^15 + timestamp
# high=3, normal=2, low=1
scheduler:queue:tasks      # 单一队列，score 编码了优先级和时间戳

# 任务元数据 (Hash)
scheduler:task:{task_id}:meta
  ├── tenant_id
  ├── priority
  └── created_at

# 租户配额 (Hash)
scheduler:tenant:{tenant_id}:quota
  └── current_concurrency   # 当前并发数

# 租户每日任务计数 (String)
scheduler:tenant:{tenant_id}:daily:{YYYY-MM-DD}
  └── 每日任务数，自动在午夜过期

# 全局配额 (Hash)
scheduler:global:quota
  └── current_concurrency   # 全局当前并发数

# 抢占任务追踪 (Set)
scheduler:preempted:tasks   # 被抢占的任务 ID 集合

# 任务状态 (String, JSON)
scheduler:task:{task_id}:state
  ├── task_id
  ├── status
  ├── progress
  ├── checkpoint (optional)
  └── preempted_at
  TTL: 24h
```

### 3.5 调度流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                        任务调度流程                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. 任务入队                                                        │
│     Enqueue() → Redis ZADD queue:{priority} {timestamp} {task_id}  │
│                                                                      │
│  2. 调度循环                                                        │
│     ┌──────────────────────────────────────────────────────────┐    │
│     │  Loop:                                                    │    │
│     │    1. 按优先级顺序检查队列 (high → normal → low)         │    │
│     │    2. ZPOP 任务                                           │    │
│     │    3. 检查租户配额 (RateLimiter.Allow)                    │    │
│     │       ├── 允许 → 预留资源，返回任务给 Executor            │    │
│     │       └── 不允许 → 重新入队，继续检查下一个任务           │    │
│     └──────────────────────────────────────────────────────────┘    │
│                                                                      │
│  3. 任务出队                                                        │
│     Dequeue() → 返回可执行的任务给 Executor                         │
│                                                                      │
│  4. 资源释放                                                        │
│     任务完成/失败/取消 → RateLimiter.Release()                      │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## 4. Implementation Notes

> **Implemented in Issue #7** - See `internal/scheduler/` for source code

### 4.1 实际实现架构

```
internal/scheduler/
├── scheduler.go       # TaskScheduler 主实现，协调各组件
├── queue.go           # PriorityQueue (Redis Sorted Set)
├── ratelimiter.go     # RateLimiter (租户/全局限流)
├── preemption.go      # PreemptionManager (抢占管理)
└── errors.go          # 错误定义
```

### 4.2 关键设计决策

1. **Callback-based Integration**: 使用回调函数 (GetTenantQuota, GetTask, UpdateStatus) 而非直接依赖 Repository，提高灵活性
2. **单一 Sorted Set**: 使用优先级编码 score 实现优先级队列，避免多队列复杂性
3. **Graceful Shutdown**: 使用 atomic.Bool 和 channel 实现优雅关闭
4. **Daily TTL**: 每日任务计数 key 在午夜自动过期，确保日边界准确

### 4.3 错误类型

```go
var (
    ErrTaskNotFound            // 任务未找到
    ErrQueueFull               // 队列已满
    ErrQuotaExceeded           // 租户配额超限
    ErrGlobalLimitExceeded     // 全局并发超限
    ErrDailyLimitExceeded      // 每日任务数超限
    ErrSchedulerNotRunning     // 调度器未运行
    ErrSchedulerAlreadyRunning // 调度器已运行
    ErrTaskNotRunning          // 任务未运行（抢占时）
)
```

### 4.4 测试覆盖

- 单元测试覆盖率: **81.5%**
- 使用 miniredis 进行内存测试
- 包含优先级抢占场景测试

## 5. Change History

| Date | Version | Issue | PRD Ref | TRD Ref | Changes |
|------|---------|-------|---------|---------|---------|
| 2026-03-24 | v1.1 | #7 | §4.4 | §4.2 | 更新实际实现接口、Redis key 设计、错误类型 |
| 2026-03-23 | v1.0 | - | §4.4 | §4.1 | 初始定义：任务调度引擎 |

## 6. Related Files

- **Source Code**: `internal/scheduler/`
- **Test Files**: `internal/scheduler/*_test.go`
- **Issue Summary**: `docs/v1.0-mvp/issues/issue-7-summary.md`
- **Implementation Plan**: `docs/v1.0-mvp/plans/2026-03-23-issue-7-task-scheduler.md`
