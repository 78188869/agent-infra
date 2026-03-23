# Scheduler Knowledge

> **Last Updated**: 2026-03-23
> **PRD Version**: v0.7-draft
> **TRD Version**: v2.4

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

```go
// Scheduler 调度器接口
type Scheduler interface {
    // 入队
    Enqueue(ctx context.Context, task *Task) error

    // 出队（阻塞等待）
    Dequeue(ctx context.Context) (*Task, error)

    // 查询排队位置
    GetQueuePosition(ctx context.Context, taskID string) (int, error)

    // 移除任务
    Remove(ctx context.Context, taskID string) error
}

// RateLimiter 限流器接口
type RateLimiter interface {
    // 检查是否允许
    Allow(ctx context.Context, tenantID string) (bool, error)

    // 获取当前使用量
    GetUsage(ctx context.Context, tenantID string) (*QuotaUsage, error)

    // 预留资源
    Reserve(ctx context.Context, tenantID string, resources *ResourceRequest) error

    // 释放资源
    Release(ctx context.Context, tenantID string, resources *ResourceRequest) error
}
```

### 3.4 Redis 队列设计

```
# 优先级队列 (Sorted Set)
scheduler:queue:high     # 高优先级队列，score = 创建时间戳
scheduler:queue:normal   # 普通优先级队列
scheduler:queue:low      # 低优先级队列

# 租户配额 (Hash)
scheduler:quota:{tenant_id}
  ├── concurrency:current   # 当前并发数
  ├── concurrency:max       # 最大并发数
  ├── daily:current         # 当日任务数
  └── daily:max             # 每日上限

# 任务元数据 (Hash)
scheduler:task:{task_id}
  ├── tenant_id
  ├── priority
  ├── created_at
  └── status
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

### 4.1 关键实现要点

1. **原子操作**：使用 Redis Lua 脚本保证队列操作原子性
2. **心跳机制**：调度器定期更新心跳，支持主备切换
3. **优雅关闭**：关闭时将正在处理的任务重新入队
4. **监控指标**：队列长度、等待时间、吞吐量

### 4.2 限流算法

使用**令牌桶算法**实现租户级限流：
- 每个租户独立的令牌桶
- 令牌按速率补充（对应并发数）
- 每个任务消耗一个令牌
- 令牌不足时拒绝或排队

### 4.3 抢占调度（可选）

MVP 阶段暂不实现抢占调度，后续版本可考虑：
1. 高优先级任务到达时，检查是否有低优先级任务运行
2. 选择合适的低优先级任务暂停
3. 高优先级任务获得资源执行
4. 低优先级任务恢复排队

## 5. Change History

| Date | Version | Issue | PRD Ref | TRD Ref | Changes |
|------|---------|-------|---------|---------|---------|
| 2026-03-23 | v1.0 | - | §4.4 | §4.1 | 初始定义：任务调度引擎 |
