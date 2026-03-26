# Issue #12: Monitoring & Logging System

## Summary

实现 MVP 阶段的任务监控和日志采集系统，支持任务执行过程的可观测性。

**主要功能:**
- 阿里云 SLS 日志采集（execution_logs 索引表）
- 实时 WebSocket 状态推送（task.status_changed, task.progress_updated, task.log_entry, task.completed）
- 监控指标 API（/metrics/dashboard, /metrics/tasks, /metrics/resources, /metrics/tenants）
- Dashboard 指标（活跃任务数、排队任务数、今日完成/失败数、平均执行时间、Token 消耗）
- 日志查询 API（支持时间范围和关键词过滤）

## Impact

- 涉及模块: monitoring, executor, API handlers
- 用户角色: operator（监控任务、处理异常）, developer（查看任务执行状态）
- 依赖方: #5 (Backend Core API), #6 (Database Models), #8 (Task Executor)

## Status

In Progress

## Related

- PRD: `docs/current/PRD.md`
- TRD: `docs/current/TRD.md` §9
- ADR: `docs/knowledge/monitoring.md`
- Plan: `docs/current/plans/2026-03-26-issue-12-monitoring-logging.md`

## Knowledge Required

- `docs/knowledge/monitoring.md`
- `docs/current/TRD.md` §7.5 (WebSocket API), §7.1.7 (Metrics API), §9 (监控告警设计)

## Resolution

<!-- 解决方案（完成后填写） -->

## Change History

| 日期 | 变更内容 |
|------|---------|
| 2026-03-26 | 创建 Issue Summary，生成执行计划 |
