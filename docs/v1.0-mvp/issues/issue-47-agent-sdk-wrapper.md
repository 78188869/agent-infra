# Issue #47: Agent SDK 集成重构 Wrapper，实现优雅中断与指令注入

> **Status**: in_progress
> **Created**: 2026-03-28
> **Closed**: YYYY-MM-DD (如果已完成)
> **PR**: #{PR Number} (如果已合并)

## Summary

重构当前的 CLI Wrapper 架构，将 Claude Agent SDK 集成到执行容器中，替换现有的 shell + 信号机制，实现更优雅的干预控制和指令注入能力。合并 cli-runner 和 wrapper 为单一容器，简化部署架构。

## Scope

- [ ] 集成 Python Claude Agent SDK (ClaudeSDKClient) 到 wrapper 容器
- [ ] 实现单进程双循环架构：FastAPI HTTP + Agent SDK 事件循环
- [ ] 重构干预机制：从 SIGSTOP/SIGCONT 改为 SDK pause/resume API
- [ ] 实现指令注入：SDK input API 支持动态插入指令
- [ ] 合并容器镜像：将 cli-runner 和 wrapper 统一为单一容器
- [ ] 更新 Go backend：调整干预 API 调用新的 SDK 接口
- [ ] 添加 P0 保护机制：asyncio.Task 隔离、状态机锁、Watchdog
- [ ] 编写集成测试验证中断与注入场景

## Knowledge References

- `knowledge/intervention.md` - 人工干预机制设计
- `knowledge/executor.md` - 执行引擎架构
- `knowledge/provider.md` - Agent 运行时配置

## Key Decisions

1. **架构选择**：采用单进程双循环方案，而非双进程或纯 SDK-first 方案
   - 理由：更简单的部署，更好的中断语义，较低的运维复杂度

2. **SDK 集成方式**：使用 Claude Agent SDK 的 pause/resume/input API
   - 理由：标准化接口，避免自定义信号处理，更好的错误处理

3. **状态管理**：基于 Lock 的状态机保护核心状态转换
   - 理由：防止竞态条件，确保 pause/execute/approval 互斥

4. **容器合并**：cli-runner 功能迁移到 wrapper，单一容器部署
   - 理由：减少 sidecar 通信开销，简化 K8s 配置

## Execution Plan

详见 `plans/2026-03-28-agent-sdk-wrapper-refactor.md`

## Acceptance Criteria

1. ✅ Agent SDK 成功集成，能正常启动和执行任务
2. ✅ 通过 HTTP API 触发 pause/resume，Agent 代码正确暂停/恢复
3. ✅ 通过 input API 注入指令，Agent 能接收并执行
4. ✅ 单一容器部署成功，移除 cli-runner sidecar
5. ✅ P0 保护机制生效，无竞态条件导致的崩溃
6. ✅ 集成测试覆盖所有干预场景（暂停、恢复、指令注入、审批）
7. ✅ Go backend API 调用新接口成功，无兼容性问题

## Technical Notes

### P0 风险缓解

- **asyncio.Task 隔离**：HTTP 和 SDK 事件循环在独立 Task 中运行，异常不互相影响
- **Lock-based State Machine**：保护 pause/execute/approval 状态转换，防止并发冲突
- **Watchdog Timer**：监控 SDK 心跳，超过阈值自动重启 Agent 进程

### 与现有架构的差异

| 维度 | 当前架构 | 新架构 |
|------|---------|--------|
| 容器 | cli-runner + wrapper (sidecar) | 单一容器 |
| 暂停机制 | SIGSTOP → 进程冻结 | SDK pause → 协作式暂停 |
| 指令注入 | 文件轮询 | SDK input API |
| 状态同步 | Redis 共享状态 | 内存状态机 + Redis 持久化 |
| 中断语义 | 硬冻结（可能丢失状态） | 软暂停（保存状态） |

## References

- Original Issue: https://github.com/yang/agent-infra/issues/47
- ADR: `docs/current/decisions/adr-002-agent-sdk-integration.md`
