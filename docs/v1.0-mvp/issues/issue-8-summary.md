# Issue #8: MVP Phase 4 - Task Executor Engine

> **Status**: pending
> **Created**: 2026-03-24
> **PR**: TBD

## Summary

根据 TRD 第4.3节和第5节，实现任务执行引擎，负责 K8s Job 的生命周期管理、状态同步和心跳检测。

执行引擎是控制面的核心组件，连接调度器和 K8s 集群，管理沙箱执行环境的完整生命周期。

## Scope

- [ ] Job 生命周期管理（创建、监控、清理）
- [ ] 状态同步机制（从 Wrapper 接收状态上报）
- [ ] 心跳检测与超时处理
- [ ] 执行接口（Execute/Pause/Resume/Cancel）
- [ ] 单元测试覆盖率 > 80%

## Knowledge References

- `docs/knowledge/executor.md` - 执行引擎设计
- `docs/knowledge/scheduler.md` - 调度器接口
- `docs/v1.0-mvp/TRD.md` §4.3, §5 - 技术设计

## Key Decisions

1. **K8s Job 资源**：使用原生 Job 而非 CRD，简化 MVP 开发
2. **Sidecar 模式**：cli-runner + wrapper + log-agent 容器架构
3. **HTTP 回调**：状态同步通过 Wrapper HTTP API 实现
4. **TTL 清理**：完成 1 小时后自动清理 Pod

## Execution Plan

详见 `docs/v1.0-mvp/plans/2026-03-24-issue-8-task-executor.md`

## Verification Criteria

| Criteria | Verification Method |
|----------|---------------------|
| K8s Job 正确创建和启动 | 集成测试 |
| 任务状态实时同步到数据库 | 单元测试 |
| 心跳超时正确触发失败处理 | 单元测试 |
| 暂停/恢复/取消操作正确执行 | 集成测试 |
| 单元测试覆盖率 > 80% | `go test -cover` |

## Change History

| 日期 | 变更内容 |
|------|---------|
| 2026-03-24 | 创建 Issue Summary |
