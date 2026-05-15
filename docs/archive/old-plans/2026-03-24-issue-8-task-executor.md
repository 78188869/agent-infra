# Plan: MVP Phase 4 - Task Executor Engine

## Context

**Issue**: #8 - MVP Phase 4 - Task Executor Engine
**Status**: Not Started
**Dependencies**: 
- Issue #6 (Database Models) - ✅ Completed
- Issue #7 (Task Scheduler) - ✅ Completed

**背景**：根据 TRD 第4.3节和第5节，实现任务执行引擎，负责 K8s Job 的生命周期管理、状态同步和心跳检测。执行引擎是控制面的核心组件，连接调度器和 K8s 集群。

**技术选型**：
- K8s 原生 Job 资源管理沙箱执行环境
- client-go 操作 K8s API
- HTTP 回调机制同步状态
- Sidecar 模式容器架构

## Objectives

1. 实现 K8s Job 生命周期管理（创建、监控、清理）
2. 实现状态同步机制（从 Wrapper 接收状态上报）
3. 实现心跳检测与超时处理
4. 实现执行接口（Execute/Pause/Resume/Cancel）
5. 单元测试覆盖率 > 80%

## Knowledge Required

- [x] docs/knowledge/executor.md - 执行引擎设计
- [x] docs/knowledge/scheduler.md - 调度器接口
- [x] docs/v1.0-mvp/TRD.md §4.3, §5 - 技术设计
- [x] internal/model/task.go - Task 模型定义

## Tasks

### Phase 1: 基础设施与接口定义

- [ ] 创建 `internal/executor/` 目录结构
- [ ] 定义核心接口 `Executor`（基于 TRD §5.3）
- [ ] 创建 K8s client 配置和初始化
- [ ] 定义 `JobConfig` 结构体（Pod 模板配置）
- [ ] 添加 K8s 相关依赖（client-go, apimachinery）

### Phase 2: Job 生命周期管理

- [ ] 实现 `job_manager.go` - Job 创建逻辑
  - 生成 Job 名称 `sandbox-{task-id}`
  - 配置 Sidecar 容器（cli-runner, wrapper）
  - 配置共享 Volume 和 PID Namespace
  - 设置 TTL 自动清理
- [ ] 实现 Job 状态监控
  - 监听 Job 状态变更事件
  - 同步 Task 状态到数据库
- [ ] 实现 Job 清理逻辑
  - 支持手动取消
  - TTL 自动清理

### Phase 3: 状态同步机制

- [ ] 创建 `wrapper_client.go` - Wrapper HTTP 客户端
- [ ] 创建内部 API 端点 `/internal/tasks/:id/events`
- [ ] 实现事件处理逻辑
  - 状态变更事件
  - 工具调用事件
  - 指标上报事件
- [ ] 实现 WebSocket 推送（可选，MVP 后续迭代）

### Phase 4: 心跳检测

- [ ] 创建 `heartbeat.go` - 心跳检测器
- [ ] 实现心跳记录存储（Redis）
- [ ] 实现超时检测逻辑
  - 5秒间隔检测
  - 15秒超时标记异常
- [ ] 实现告警触发机制

### Phase 5: 执行接口实现

- [ ] 实现 `Execute(task *Task) error`
  - 准备 Job 配置
  - 创建 K8s Job
  - 更新 Task 状态为 running
- [ ] 实现 `Pause(taskID string) error`
  - 调用 Wrapper `/pause` API
  - 更新 Task 状态为 paused
- [ ] 实现 `Resume(taskID string) error`
  - 调用 Wrapper `/resume` API
  - 更新 Task 状态为 running
- [ ] 实现 `Cancel(taskID string, reason string) error`
  - 删除 K8s Job
  - 更新 Task 状态为 cancelled
- [ ] 实现 `GetStatus(taskID string) (*TaskStatus, error)`

### Phase 6: 测试与文档

- [ ] 编写单元测试
  - Job 创建测试
  - 状态同步测试
  - 心跳检测测试
  - Mock K8s API 测试
- [ ] 更新 `docs/knowledge/executor.md` Change History
- [ ] 创建 Issue Summary

## Dependencies

| 依赖 | 状态 | 说明 |
|------|------|------|
| Issue #6 (Database Models) | ✅ Completed | Task 模型已定义 |
| Issue #7 (Task Scheduler) | ✅ Completed | 调度器接口可用 |

**解除阻塞**：所有依赖项已完成，可立即开始开发。

## Risks

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| K8s API 限流 | Job 创建延迟 | 实现 Rate Limit 和重试机制 |
| Wrapper 通信失败 | 状态同步中断 | HTTP 重试 + 状态恢复机制 |
| 心跳网络抖动 | 误判任务失败 | 增加重试和超时阈值 |

## Technical Details

### 核心文件结构

```
internal/executor/
├── executor.go        # Executor 接口定义和主逻辑
├── job_manager.go     # K8s Job 生命周期管理
├── wrapper_client.go  # Wrapper HTTP 客户端
├── heartbeat.go       # 心跳检测器
├── config.go          # Job 配置生成
└── executor_test.go   # 单元测试
```

### Job 配置关键参数

```yaml
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 3600
  activeDeadlineSeconds: {from template}
  restartPolicy: Never
  parallelism: 1
  completions: 1
  shareProcessNamespace: true
```

### Task-Job 状态映射

| Job 状态 | Task 状态 |
|---------|---------|
| Created | scheduled |
| Running | running |
| Complete | succeeded |
| Failed | failed |

## Status

Not Started
