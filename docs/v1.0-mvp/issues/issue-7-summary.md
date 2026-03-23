# Issue #7: MVP Phase 3 - Task Scheduler Engine

> **Status**: in_progress
> **Created**: 2026-03-23
> **Assignee**: @claude

## Summary

根据 TRD 第4.2节，实现任务调度引擎，负责任务排队、限流和抢占调度。使用 Redis 实现分布式优先级队列，支持租户配额限流和高优先级任务抢占机制。

## Impact

**涉及模块**:
- `internal/scheduler/` - 新增调度器模块
- `internal/config/` - 新增 Redis 配置
- `internal/api/handler/health.go` - 增强 Redis 健康检查

**影响范围**:
- 任务创建流程：任务创建后进入调度队列
- 任务执行流程：Executor 从 Scheduler 获取可执行任务
- 租户配额管理：实时限流控制

## Scope

- [x] 优先级队列 (Redis Sorted Set)
- [x] 租户级限流 (并发数、每日任务数)
- [x] 全局并发限制
- [x] 抢占机制 (保存/恢复任务状态)
- [x] 排队位置查询
- [x] 单元测试覆盖 > 80%

## Related

- **PRD**: §4.4 调度策略
- **TRD**: `docs/v1.0-mvp/TRD.md` §4.2 任务调度器
- **Knowledge**: `docs/knowledge/scheduler.md`
- **Dependencies**: Issue #6 (Database Models) - Completed
- **Plan**: `docs/v1.0-mvp/plans/2026-03-23-issue-7-task-scheduler.md`

## Key Decisions

1. **Redis Sorted Set**: 使用时间戳作为 score 实现 FIFO
2. **三层队列**: high/normal/low 独立队列，按优先级顺序消费
3. **令牌桶限流**: 租户级和全局两级限流
4. **状态持久化**: 抢占时将任务状态保存到 Redis (24h TTL)

## Verification Criteria

| Criteria | Verification Method |
|----------|---------------------|
| 优先级队列正确排序 | Unit tests for high > normal > low |
| 租户配额限流生效 | Unit tests for quota exceeded |
| 全局并发限制生效 | Unit tests for global limit |
| 抢占机制正确工作 | Unit tests for preemption flow |
| 排队位置查询准确 | Unit tests for GetPosition |
| 单元测试覆盖率 > 80% | `go test -cover ./internal/scheduler/...` |

## Execution Plan

详见 `docs/v1.0-mvp/plans/2026-03-23-issue-7-task-scheduler.md`

## Change History

| 日期 | 变更内容 |
|------|---------|
| 2026-03-23 | 创建 Issue Summary |
| 2026-03-23 | 创建执行计划 |
