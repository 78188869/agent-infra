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

**Completed** (PR #22 merged 2026-03-26)

## Related

- PRD: `docs/current/PRD.md`
- TRD: `docs/current/TRD.md` §9
- ADR: `docs/knowledge/monitoring.md`
- Plan: `docs/current/plans/2026-03-26-issue-12-monitoring-logging.md`

## Knowledge Required

- `docs/knowledge/monitoring.md`
- `docs/current/TRD.md` §7.5 (WebSocket API), §7.1.7 (Metrics API), §9 (监控告警设计)

## Resolution

通过 PR #22 实现，包含以下组件：

| 组件 | 文件 | 说明 |
|------|------|------|
| WebSocket Hub | `internal/monitoring/ws_hub.go` | 内存级租户广播中心，使用 `WSConn` 接口便于测试 |
| WS Handler | `internal/api/handler/ws_handler.go` | Token 认证 + gorilla/websocket 升级，遵循 TRD §7.5.1 |
| SLS Client | `pkg/aliyun/sls/client.go` | 阿里云 SLS 配置 + MVP stdout 日志（生产环境使用 SDK） |
| MonitoringService | `internal/service/monitoring_service.go` | `RecordTaskStatusChange`, `RecordLogEntry`, `RecordTaskProgress`, `BroadcastTaskCompletion` |
| Metrics Handlers | `internal/api/handler/metrics.go` | `GET /metrics/dashboard`, `/tasks`, `/resources`, `/tenants`, `GET /tasks/:id/logs`，遵循 TRD §7.1.7 |
| Types | `internal/monitoring/types.go` | WebSocket 消息类型和事件结构体 |

**测试覆盖率:** monitoring 包 91.3%（超过 80% 目标）

**Acceptance Criteria 完成情况:**
- [x] SLS 日志正确采集（MVP 使用 stdout，预留 SDK 接口）
- [x] WebSocket 连接稳定（含租户隔离和连接管理）
- [x] 实时状态推送延迟 < 1s（内存广播）
- [x] 监控指标 API 正确返回数据（4 个 endpoint）
- [x] 日志查询支持时间范围和关键词过滤
- [x] 单元测试覆盖率 > 80%（实际 91.3%）

**额外修复:**
- 修复了 executor 测试中的 3 个既有 bug（JobConfig 资源限制、nil UUID 验证）
- 解决了与 PR #21（intervention system）的合并冲突

## Change History

| 日期 | 变更内容 |
|------|---------|
| 2026-03-26 | 创建 Issue Summary，生成执行计划 |
| 2026-03-26 | 完成 WebSocket Hub 实现 (`80421ae0`) |
| 2026-03-26 | 完成完整监控系统实现 (`5077c0dc`) — SLS client, MonitoringService, Metrics API, WS Handler |
| 2026-03-26 | 解决与 PR #21 合并冲突 (`97342f7a`) |
| 2026-03-26 | 修复重复 ExecutorConfig 字段 (`adcdd7bc`) |
| 2026-03-26 | PR #22 合并到 main (`2be92bc2`) |
| 2026-03-28 | 更新 Issue Summary，关闭 Issue |
