# ADR-002: Agent SDK 集成架构设计

## Status

Accepted

## Context

当前人工干预机制存在以下问题：

1. **SIGSTOP/SIGCONT 是硬暂停**：进程冻结导致无法优雅保存状态，可能丢失内存中数据
2. **文件轮询不可靠**：指令注入通过文件轮询实现，延迟高（秒级）且可能丢失事件
3. **双容器开销**：cli-runner 和 wrapper 分离部署，增加 sidecar 通信复杂度和资源消耗
4. **信号处理复杂**：需要处理 SIGSTOP/SIGCONT/SIGTERM 等多种信号，易出错

Claude Agent SDK 提供了标准的 pause/resume/input API，可以实现协作式暂停和指令注入，比基于信号的机制更优雅。

## Decision

采用 **单进程双循环架构**，将 Claude Agent SDK 集成到 wrapper 容器中，替换现有的 shell + 信号机制：

### 核心架构

```
┌─────────────────────────────────────────────────────────┐
│               Wrapper Container (Single Process)        │
│                                                          │
│  ┌──────────────────────┐      ┌──────────────────────┐ │
│  │  FastAPI HTTP Server │      │  Claude Agent SDK    │ │
│  │                      │      │  Event Loop          │ │
│  │  - POST /pause       │◄────►│  - pause()           │ │
│  │  - POST /resume      │      │  - resume()          │ │
│  │  - POST /input       │      │  - input(text)       │ │
│  │  - GET /status       │      │  - on_pause callback │ │
│  └──────────────────────┘      └──────────────────────┘ │
│              │                                │          │
│              └────────────┬───────────────────┘          │
│                           ▼                              │
│                  ┌─────────────────┐                     │
│                  │ State Machine   │                     │
│                  │ (Lock-protected)│                     │
│                  └─────────────────┘                     │
└─────────────────────────────────────────────────────────┘
```

### 关键组件

1. **FastAPI HTTP Server**：暴露干预接口（pause/resume/input/status）
2. **Claude Agent SDK Event Loop**：管理 Agent 生命周期和执行状态
3. **Lock-based State Machine**：保护状态转换，防止竞态条件
4. **asyncio.Task 隔离**：HTTP 和 SDK 在独立 Task 中运行

### API 设计

```python
# FastAPI Endpoints
POST /api/v1/intervene/pause
POST /api/v1/intervene/resume
POST /api/v1/intervene/input
GET  /api/v1/intervene/status

# SDK Integration
client = ClaudeSDKClient()
client.pause()      # 协作式暂停
client.resume()     # 恢复执行
client.input(text)  # 注入指令
```

### P0 保护机制

1. **asyncio.Task 隔离**
   - HTTP Server 和 SDK Event Loop 在独立 Task 中运行
   - 使用 `asyncio.create_task()` 和异常隔离
   - 单个组件崩溃不影响另一个

2. **Lock-based State Machine**
   ```python
   state_lock = asyncio.Lock()
   current_state = "running"  # running | paused | suspended

   async def pause():
       async with state_lock:
           if current_state == "running":
               await client.pause()
               current_state = "paused"
   ```

3. **Watchdog Timer**
   - 监控 SDK 心跳（`client.get_status()`）
   - 超过阈值（如 30s）无响应，重启 Agent 进程
   - 日志记录异常事件用于排查

## Consequences

### Positive

1. **优雅的中断语义**：协作式暂停可保存状态，避免数据丢失
2. **实时指令注入**：SDK input API 比文件轮询更可靠、低延迟
3. **简化部署**：单容器架构，移除 sidecar 通信复杂度
4. **标准化接口**：使用官方 SDK API，减少自定义代码维护成本
5. **更好的可观测性**：SDK 提供标准事件和日志接口

### Negative

1. **SDK 依赖风险**：依赖 Python Claude SDK 的稳定性和维护状态
2. **协作式暂停限制**：Agent 代码必须响应暂停信号，无法强制冻结
3. **单点故障**：单进程架构，进程崩溃会导致 HTTP 和 SDK 同时不可用
4. **学习曲线**：团队需要熟悉 Agent SDK 的使用和最佳实践

### Neutral

1. **资源消耗**：单进程可能比双进程占用略少的内存，但需要实际测试验证
2. **性能影响**：asyncio 开销较小，但对极端高并发场景需要压测验证

## Alternatives Considered

| 方案 | 描述 | 优点 | 缺点 | 为何未选择 |
|------|------|------|------|-----------|
| **A. 单进程双循环** | FastAPI + SDK 在同一进程，使用 asyncio.Task 隔离 | 简单部署，低延迟 | SDK 崩溃影响 HTTP | **选择** - 在 P0 保护下可接受风险 |
| **B. 双进程 + Unix Socket** | HTTP Server 和 SDK 分离进程，通过 Unix Socket 通信 | 进程隔离，故障域小 | 复杂度高，延迟大 | 双容器问题未解决，增加通信开销 |
| **C. SDK-first 嵌入式 HTTP** | 完全依赖 SDK 的 HTTP 能力，不额外启动 FastAPI | 代码最少 | SDK HTTP 功能有限，无法扩展 | 缺乏灵活性，无法满足自定义 API 需求 |

## Implementation Notes

### 迁移步骤

1. **Phase 1**: 集成 SDK 到 wrapper，保留旧机制兼容
2. **Phase 2**: 并行运行新旧机制，验证稳定性
3. **Phase 3**: 移除 cli-runner 和信号处理代码
4. **Phase 4**: 更新 Go backend 调用新 API

### 测试策略

- 单元测试：状态机转换逻辑、Lock 并发安全性
- 集成测试：pause/resume/input 端到端流程
- 压力测试：高频并发请求下的稳定性
- 故障注入：模拟 SDK 崩溃，验证 Watchdog 和重启机制

### 监控指标

- API 响应延迟（p50/p95/p99）
- 暂停/恢复成功率
- SDK 崩溃次数和 Watchdog 触发次数
- 指令注入延迟（从 API 调用到 Agent 接收）

## References

- [PRD - 人工干预机制](../PRD.md#人工干预)
- [TRD - 执行引擎设计](../TRD.md#执行引擎)
- [Claude Agent SDK Docs](https://docs.anthropic.com/claude/docs/agent-sdk)
- Issue #47: https://github.com/yang/agent-infra/issues/47
