# Monitoring Knowledge

> **Last Updated**: 2026-03-23
> **PRD Version**: v0.7-draft
> **TRD Version**: v2.4

## 1. Overview

Monitoring 模块负责日志采集、指标监控和告警通知。

**模块职责**：
- 执行日志采集与存储
- 系统指标收集
- 告警规则配置
- 通知推送

**核心概念**：
- **Execution Log**: 任务执行日志
- **Metric**: 性能指标（延迟、吞吐量、资源使用）
- **Alert**: 告警事件
- **Notification**: 通知推送

## 2. Product Requirements (from PRD)

### 2.1 用户故事

| 故事ID | 描述 | 验收标准 |
|--------|------|---------|
| US-D02 | 任务执行监控 | 实时显示执行进度和日志，流式输出执行日志，显示资源消耗 |
| US-O01 | 任务监控 | 任务列表支持按状态、时间、租户筛选，实时显示 CPU、内存、Token 消耗，异常任务自动告警 |

### 2.2 监控维度

| 维度 | 指标 | 说明 |
|------|------|------|
| 任务指标 | 任务数、成功率、平均执行时间 | 业务健康度 |
| 资源指标 | CPU、内存、Token 消耗 | 资源使用 |
| 系统指标 | API 延迟、错误率、吞吐量 | 系统性能 |
| 调度指标 | 队列长度、等待时间 | 调度效率 |

### 2.3 日志类型

| 类型 | 说明 | 存储位置 |
|---------|------|---------|
| 执行日志 | CLI 输出、工具调用、状态变更 | 阿里云 SLS |
| 访问日志 | API 请求/响应 | MSE 网关 |
| 系统日志 | 服务运行日志 | 阿里云 SLS |
| 审计日志 | 操作审计记录 | 数据库 + SLS |

## 3. Technical Design (from TRD)

### 3.1 日志架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                        日志采集架构                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  沙箱 Job                                                     │   │
│  │  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐ │   │
│  │  │  cli-runner    │  │  wrapper       │  │  log-agent     │ │   │
│  │  │  stdout        │  │  events        │  │  (Sidecar)     │ │   │
│  │  │  events.jsonl  │  │  heartbeat     │  │  采集所有日志  │ │   │
│  │  └────────────────┘  └────────────────┘  └────────────────┘ │   │
│  │                                                │              │   │
│  └────────────────────────────────────────────────│──────────────┘   │
│                                                   │                  │
│                                                   ▼                  │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                     阿里云 SLS                               │    │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐              │    │
│  │  │ 执行日志   │ │ 系统日志   │ │ 访问日志   │              │    │
│  │  │ logstore   │ │ logstore   │ │ logstore   │              │    │
│  │  └────────────┘ └────────────┘ └────────────┘              │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 日志格式

**执行日志格式**（写入 SLS）：

```json
{
    "task_id": "task-uuid",
    "tenant_id": "tenant-uuid",
    "timestamp": "2026-03-23T10:30:00.123Z",
    "event_type": "tool_call",
    "event_name": "Write",
    "content": {
        "file": "main.go",
        "action": "create"
    },
    "source": "cli-runner"
}
```

**事件类型枚举**：

| event_type | 说明 |
|------------|------|
| status_change | 状态变更 |
| tool_call | 工具调用 |
| tool_result | 工具结果 |
| llm_input | LLM 输入 |
| llm_output | LLM 输出 |
| error | 错误 |
| heartbeat | 心跳 |
| intervention | 干预 |
| metric | 指标上报 |
| checkpoint | 检查点 |

### 3.3 监控指标

**Prometheus 指标定义**：

```yaml
# 任务指标
- task_total{tenant_id, status}
- task_duration_seconds{tenant_id}
- task_token_usage{tenant_id}

# 调度指标
- scheduler_queue_length{priority}
- scheduler_wait_time_seconds{priority}

# 执行器指标
- executor_active_jobs
- executor_job_duration_seconds

# API 指标
- http_request_duration_seconds{method, path, status}
- http_requests_total{method, path, status}

# 资源指标
- container_cpu_usage_seconds_total
- container_memory_working_set_bytes
```

### 3.4 告警规则

**告警规则配置**：

| 告警名称 | 条件 | 级别 | 通知方式 |
|----------|------|------|---------|
| 任务失败率过高 | 5min 内失败率 > 10% | warning | 钉钉 |
| 任务执行超时 | 单任务执行 > 1h | warning | 钉钉 |
| 队列堆积 | 队列长度 > 100 | warning | 钉钉 |
| API 错误率过高 | 5min 内错误率 > 5% | critical | 钉钉 + 短信 |
| 资源使用过高 | CPU > 80% 持续 5min | warning | 钉钉 |

### 3.5 通知服务

**NotifyService 接口**：

```go
type NotifyService interface {
    // 任务通知
    NotifyTaskStatusChange(ctx context.Context, taskID string, oldStatus, newStatus TaskStatus) error
    NotifyTaskCompletion(ctx context.Context, taskID string, result *TaskResult) error
    NotifyTaskFailure(ctx context.Context, taskID string, err error) error

    // 告警通知
    SendAlert(ctx context.Context, alert *Alert) error

    // WebSocket 推送
    PushToClient(ctx context.Context, userID string, event *WebSocketEvent) error
    BroadcastToTenant(ctx context.Context, tenantID string, event *WebSocketEvent) error

    // Webhook
    TriggerWebhook(ctx context.Context, webhookURL string, payload *WebhookPayload) error
}
```

## 4. Implementation Notes

### 4.1 关键实现要点

1. **日志采集**: 使用 Sidecar 模式的 log-agent 采集容器日志
2. **结构化日志**: 使用 JSON 格式，便于查询分析
3. **标签索引**: 日志添加 task_id、tenant_id 等标签
4. **实时推送**: 使用 WebSocket 推送任务状态变更

### 4.2 SLS 配置

```yaml
# 日志库配置
logstores:
  - name: execution-logs
    ttl: 30d
    shard_count: 10

  - name: system-logs
    ttl: 7d
    shard_count: 5

# 索引配置
indexes:
  - keys: ["task_id", "tenant_id", "event_type"]
    type: text
  - keys: ["timestamp"]
    type: time
```

### 4.3 性能指标目标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| API P99 延迟 | < 500ms | 控制面 API |
| 日志延迟 | < 5s | 从产生到可查询 |
| 告警延迟 | < 1min | 从触发到通知 |
| WebSocket 延迟 | < 1s | 状态推送 |

### 4.4 技术决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 日志存储 | 阿里云 SLS | 托管服务、查询能力强 |
| 指标存储 | Prometheus | 云原生标准、生态成熟 |
| 实时推送 | WebSocket | 低延迟、双向通信 |
| 通知渠道 | 钉钉 + 短信 | 企业常用、覆盖面广 |

### 4.5 Local Persistent Logging

本地开发环境下（`APP_ENV=local`），日志持久化到本地 JSONL 文件。

**配置项**（`config.local.yaml`）：

```yaml
log:
  outputs: both        # stdout | file | both
  file:
    dir: logs          # 日志目录
    max_size_mb: 100
    max_backups: 7
    max_age_days: 30
```

**文件命名**：
- `logs/business-YYYY-MM-DD.jsonl` — 业务执行日志（状态变更、工具调用、错误）
- `logs/http-YYYY-MM-DD.jsonl` — HTTP 请求/响应日志

**环境切换**：
- 本地开发：`outputs: both`，同时输出到 stdout + 文件
- 生产环境：`outputs: stdout`（默认），仅输出到终端，SLS 负责持久化

**查询示例**：

```bash
# 查询所有业务错误
cat logs/business-*.jsonl | jq 'select(.level=="ERROR")'

# 查询指定任务的日志
grep "task-123" logs/business-*.jsonl

# 查询 HTTP 4xx/5xx 请求
cat logs/http-*.jsonl | jq 'select(.status >= 400)'
```

## 5. Change History

| Date | Version | Issue | PRD Ref | TRD Ref | Changes |
|------|---------|-------|---------|---------|---------|
| 2026-03-23 | v1.0 | - | §4.9, §6.1 | §9 | 初始定义：监控告警设计 |
| 2026-03-28 | v1.1 | #33 | - | §9 | 新增本地持久化日志支持 |
