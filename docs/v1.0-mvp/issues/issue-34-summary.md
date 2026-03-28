# Issue #34: [TASK] Executor 层抽象重构

> **Status**: completed
> **Created**: 2026-03-28

## Summary
将执行引擎的核心逻辑与 K8s 实现解耦，通过 `ContainerRuntime` 接口使其可扩展其他容器运行时。

## Problem
1. TaskExecutor 直接依赖 K8s JobManager 的具体实现，无法在非 K8s 环境下运行任务
2. 无法在测试中替换为 mock，测试困难
3. 执行引擎的核心职责（创建/查询/停止/寻址）是通用的，不应绑定特定容器编排系统

## Scope
- [x] 定义 `ContainerRuntime` 接口（Create/GetStatus/Delete/GetAddress）
- [x] 实现 `K8sRuntime` 封装现有 `JobManager`
- [x] `TaskExecutor` 依赖接口而非具体类型
- [x] `GetPodAddress` → `GetAddress`，地址验证放宽
- [x] `ExecutorConfig` 移除 `JobConfig` 依赖
- [x] 纯重构，所有现有测试通过，无功能变更

## Knowledge References
- `knowledge/executor.md`

## Key Decisions
1. `ContainerRuntime` 接口放在 `internal/executor/` 包内（与 `JobManager` 同包），而非独立包
2. 使用 `RuntimeInfo`/`RuntimeStatus` 新类型而非复用 `JobInfo`/`JobStatus`，保持抽象层独立性
3. `MockContainerRuntime` 使用 function-injection 模式，比 testify/mock 更轻量
4. WrapperClient 参数名 `podIP` 重命名为 `address`，验证从 `isValidPodIP` 放宽为 `isValidAddress`

## Execution Plan
详见 `plans/issue-34-executor-abstraction.md`

## PR
- PR #46: https://github.com/78188869/agent-infra/pull/46
