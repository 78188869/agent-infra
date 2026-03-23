# Executor Knowledge

> **Last Updated**: 2026-03-23
> **PRD Version**: v0.7-draft
> **TRD Version**: v2.4

## 1. Overview

Executor 模块负责沙箱 Job 的生命周期管理和状态同步。

**模块职责**：
- K8s Job 创建与销毁
- Pod 状态监控
- 心跳检测与超时处理
- 状态上报与同步

**核心概念**：
- **Job**: K8s Job 资源，封装沙箱执行环境
- **Pod**: K8s Pod，包含 cli-runner 和 wrapper 容器
- **Wrapper**: Sidecar 容器，负责状态上报和干预处理

## 2. Product Requirements (from PRD)

### 2.1 执行流程

```
Create → Validate → Schedule → Start → Execute → Check → Complete
                                                      ↓
                                                  Reflect → Adjust → Retry
```

| 阶段 | 说明 |
|------|------|
| Create | 创建任务实例，加载模板配置 |
| Validate | 校验参数、权限、资源配额 |
| Schedule | 调度到执行队列 |
| Start | 创建沙箱环境，初始化能力，加载上下文 |
| Execute | Agent 在沙箱中执行任务 |
| Check | 检查成功指标是否达成 |
| Complete | 收集结果，更新状态，清理资源 |

### 2.2 执行策略

| 策略 | 说明 |
|------|------|
| 执行模式 | 同步/异步/定时 |
| 超时行为 | 终止任务并标记失败 |
| 重试策略 | 最大重试次数、退避策略 |

## 3. Technical Design (from TRD)

### 3.1 Task-Job 绑定设计

> **设计决策**：MVP 阶段使用 K8s 原生 `Job` 资源管理沙箱执行环境。

```
Task (DB)                    Job (K8s)                    Pod (K8s)
────────                     ─────────                    ─────────
pending
    │
    ▼  调度器取出任务
scheduled  ──────────────▶  Created
    │                           │
    ▼  Executor 创建 Job       ▼  创建 Pod
running    ◀──────────────  Running   ◀─────────────────  Pending
    │                           │                           │
    │                           │                           ▼
    │                           │                        Running
    │                           │                           │
    ▼  任务完成                 ▼                           ▼
succeeded  ◀──────────────  Complete  ◀─────────────────  Completed

    或

failed     ◀──────────────  Failed    ◀─────────────────  Error
```

### 3.2 Job 规格配置

| 配置项 | 值 | 说明 |
|--------|-----|------|
| `backoffLimit` | 0 | MVP 阶段禁用自动重试 |
| `ttlSecondsAfterFinished` | 3600 | 完成 1 小时后自动清理 |
| `activeDeadlineSeconds` | 按模板配置 | 任务超时时间 |
| `restartPolicy` | Never | 不自动重启 |
| `parallelism` | 1 | 单任务单 Pod |
| `completions` | 1 | 一次性任务 |

### 3.3 沙箱 Job 架构（Sidecar 模式）

```
┌─────────────────────────────────────────────────────────────────────┐
│                    沙箱 Job (Sidecar 模式)                          │
│                    资源名: sandbox-{task-id}                        │
├─────────────────────────────────────────────────────────────────────┤
│  共享配置: shareProcessNamespace: true                              │
│  共享卷: workspace (emptyDir)                                       │
│                                                                      │
│  /workspace/ (共享 Volume)                                          │
│  ├── CLAUDE.md          # Claude Code 项目配置                      │
│  ├── .claude/           # Claude Code 配置目录                      │
│  ├── .mcp.json          # MCP工具配置                               │
│  ├── src/               # 业务代码                                  │
│  └── .agent-state/      # 容器间通信状态文件                        │
│      ├── status.json    # CLI 当前状态                              │
│      ├── events.jsonl   # 事件流                                    │
│      └── inject.json    # 待注入指令                                │
│                                                                      │
│  ┌──────────────────────────┐  ┌──────────────────────────┐        │
│  │  主容器: cli-runner      │  │  Sidecar: wrapper        │        │
│  │  • 克隆代码仓库          │  │  • HTTP Server (:9090)   │        │
│  │  • 生成CLAUDE.md         │  │  • 心跳服务 (5s)         │        │
│  │  • 启动CLI               │  │  • 状态监控              │        │
│  │  • 写入状态文件          │  │  • 干预处理              │        │
│  └──────────────────────────┘  └──────────────────────────┘        │
│                                                                      │
│  ┌──────────────────────────┐                                       │
│  │  Sidecar: log-agent      │                                       │
│  │  • 采集所有容器日志      │                                       │
│  │  • 实时上报到阿里云SLS   │                                       │
│  └──────────────────────────┘                                       │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.4 核心接口

```go
// Executor 执行器接口
type Executor interface {
    // 创建并启动 Job
    CreateJob(ctx context.Context, task *Task) (*JobInfo, error)

    // 获取 Job 状态
    GetJobStatus(ctx context.Context, taskID string) (*JobStatus, error)

    // 暂停 Job
    PauseJob(ctx context.Context, taskID string) error

    // 恢复 Job
    ResumeJob(ctx context.Context, taskID string) error

    // 取消 Job
    CancelJob(ctx context.Context, taskID string, reason string) error

    // 获取 Pod 地址（用于干预）
    GetPodAddress(ctx context.Context, taskID string) (string, error)
}

// JobInfo Job 信息
type JobInfo struct {
    Name      string
    Namespace string
    PodName   string
    Status    JobStatus
    CreatedAt time.Time
}

// JobStatus Job 状态
type JobStatus struct {
    Phase      string    // Pending, Running, Succeeded, Failed
    StartTime  *time.Time
    CompletionTime *time.Time
    Message    string
}
```

### 3.5 模块结构

```
internal/executor/
├── executor.go        # 执行管理器主逻辑
├── job_manager.go     # K8s Job 生命周期管理
├── wrapper_client.go  # Wrapper HTTP 客户端
└── heartbeat.go       # 心跳检测
```

## 4. Implementation Notes

### 4.1 关键实现要点

1. **K8s Client**: 使用 `client-go` 与 K8s API 交互
2. **状态同步**: 通过 HTTP 回调从 wrapper 接收状态更新
3. **心跳检测**: 每 5 秒检测一次，超时 30 秒标记异常
4. **资源清理**: 使用 TTL 机制自动清理完成的 Job

### 4.2 Wrapper HTTP API

Wrapper 暴露的 HTTP 接口：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /health | 健康检查 |
| GET | /status | 获取 CLI 状态 |
| POST | /pause | 暂停 CLI |
| POST | /resume | 恢复 CLI |
| POST | /inject | 注入指令 |

### 4.3 状态同步机制

```
┌─────────────────────────────────────────────────────────────────────┐
│                        状态同步流程                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Wrapper ──▶ POST /internal/tasks/:id/events ──▶ Control Plane     │
│     │                                              │                 │
│     │                                              ▼                 │
│     │                                         Update DB              │
│     │                                              │                 │
│     │                                              ▼                 │
│     │                                         WebSocket Push         │
│     │                                              │                 │
│     └──────────────────────────────────────────────┘                 │
│                                                                      │
│  事件类型:                                                           │
│  - status_change: 状态变更                                          │
│  - tool_call: 工具调用                                              │
│  - tool_result: 工具结果                                            │
│  - heartbeat: 心跳                                                  │
│  - metric: 指标上报                                                 │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## 5. Change History

| Date | Version | Issue | PRD Ref | TRD Ref | Changes |
|------|---------|-------|---------|---------|---------|
| 2026-03-23 | v1.0 | - | §4.2 | §5.1, §5.2, §5.3 | 初始定义：执行引擎设计 |
