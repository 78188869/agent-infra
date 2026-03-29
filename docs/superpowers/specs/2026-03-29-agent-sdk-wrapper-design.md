# Issue 47: Agent SDK 集成重构 Wrapper 设计文档

> **Date**: 2026-03-29
> **Status**: Approved
> **Issue**: #47 — Agent SDK 集成重构 Wrapper，实现优雅中断与指令注入

---

## 1. 背景与目标

### 当前问题

1. **SIGSTOP/SIGCONT 是硬暂停**：冻结进程，无法实现"中断当前操作 → 追加纠偏指令"
2. **inject.json 文件轮询不优雅**：延迟高、可靠性差
3. **双容器 Sidecar 模式过重**：每个任务需 cli-runner + wrapper 两个容器

### 目标

用 Python Claude Agent SDK 的 `ClaudeSDKClient` 替代 shell + 信号方案，合并 cli-runner + wrapper 为单容器。

### 不做什么

- 不改 K8s 运行时（生产环境保持现有 Sidecar 模式）
- 不改上游 InterventionService / TaskExecutor 接口签名

---

## 2. 架构方案

### 2.1 方案选择：单进程双循环（Approach A）

一个 Python 进程同时运行 FastAPI HTTP server 和 `ClaudeSDKClient`，通过 asyncio 事件循环共享。

```
┌──────────────────────────────────────────┐
│         scripts/wrapper/ (单容器)         │
│                                          │
│  ┌─────────────┐    ┌─────────────────┐ │
│  │  FastAPI     │    │ ClaudeSDKClient │ │
│  │  HTTP Server │    │ (Agent SDK)     │ │
│  │  :9090       │    │                 │ │
│  └──────┬───────┘    └────────┬────────┘ │
│         │                     │          │
│         └─────── asyncio ─────┘          │
│              Event Loop                   │
└──────────────────────────────────────────┘
```

**选择理由**：
- SDK 的 `interrupt()` 和 `query()` 都是 async 方法，与 FastAPI 的 async 路由天然兼容
- 真正的单进程，零 IPC 开销
- FastAPI 提供自动 OpenAPI 文档，便于调试

**关键防护措施**（架构评审 P0）：
- SDK consumer 运行在独立 `asyncio.Task` 中，异常隔离不波及 FastAPI
- `asyncio.Lock` 保护状态转换和 SDK 调用，防止竞态
- Watchdog 机制检测 SDK 进程死亡，主动通知 Go 后端

---

## 3. 项目结构

```
scripts/wrapper/
├── Dockerfile              # 多阶段构建，含完整开发工具链
├── requirements.txt        # fastapi, uvicorn, claude-agent-sdk, httpx
├── main.py                 # 入口：启动 FastAPI + Agent SDK
├── agent/
│   ├── __init__.py
│   ├── client.py           # ClaudeSDKClient 封装，管理生命周期
│   └── events.py           # 事件处理：SDK 消息 → 状态变更 → 推送
├── api/
│   ├── __init__.py
│   └── routes.py           # FastAPI 路由（/health, /status, /interrupt, /inject）
├── models/
│   ├── __init__.py
│   └── schemas.py          # Pydantic 模型（请求/响应/状态）
└── config.py               # 环境变量配置
```

### 职责划分

| 模块 | 职责 |
|------|------|
| `client.py` | 封装 `ClaudeSDKClient`，暴露 `start()`, `interrupt()`, `inject()`, `stop()`；维护状态机；Watchdog 监控 |
| `events.py` | 消费 `receive_response()` 消息流，转换为结构化事件，通过 HTTP 推送到 Control Plane |
| `routes.py` | FastAPI 路由，调用 `client.py` 方法，返回 HTTP 响应 |
| `schemas.py` | Pydantic 模型定义（状态枚举、请求/响应结构） |
| `config.py` | 从环境变量读取配置（Control Plane URL、超时时间、API Key 等） |
| `main.py` | `asyncio.gather()` 并发运行 uvicorn + SDK 事件循环 |

---

## 4. 核心设计

### 4.1 状态机

```
idle → starting → streaming ⇄ interrupted
                   ↓                  ↓
               completed          completed
                   ↓                  ↓
                 failed              failed
```

**状态说明**：

| 状态 | 含义 | 可转换到 |
|------|------|---------|
| `idle` | 初始状态，未启动 | `starting` |
| `starting` | SDK connect 中，尚未收到第一条消息 | `streaming`, `failed` |
| `streaming` | SDK 正在产出消息 | `interrupted`, `completed`, `failed` |
| `interrupted` | interrupt() 后等待新指令 | `streaming` (via inject), `completed` |
| `completed` | 任务正常完成（收到 ResultMessage） | 终态 |
| `failed` | 任务失败（SDK 异常、进程死亡、超时） | 终态 |

**状态转换保护**（P0 竞态防护）：

```python
class StateMachine:
    def __init__(self):
        self._state = "idle"
        self._lock = asyncio.Lock()
        self._state_changed = asyncio.Condition()

    async def transition(self, new_state: str, precondition: str):
        async with self._lock:
            if self._state != precondition:
                raise StateError(f"Cannot go to {new_state} from {self._state}")
            self._state = new_state
            self._state_changed.notify_all()
```

### 4.2 AgentClient

```python
class AgentClient:
    """ClaudeSDKClient 封装，带异常隔离和竞态保护"""

    _state: StateMachine
    _client: Optional[ClaudeSDKClient]
    _consumer_task: Optional[asyncio.Task]  # 隔离的消费者 Task
    _watchdog: AgentWatchdog
    _reporter: EventReporter
    _sdk_error: Optional[Exception]

    async def start(self, prompt: str) -> None:
        """idle → starting，connect + query，启动 _consumer_task"""

    async def interrupt(self) -> None:
        """streaming → interrupted，调用 SDK interrupt()"""

    async def inject(self, prompt: str) -> None:
        """interrupted → streaming，调用 SDK query() 注入新指令"""

    async def stop(self) -> None:
        """disconnect + 取消 consumer task + 清理资源"""

    async def _consume_messages(self) -> None:
        """独立 asyncio.Task，try/except 包裹
        正常结束(ResultMessage) → completed
        异常 → failed + 通知 Go 后端"""

    async def _watchdog_monitor(self) -> None:
        """每 30s 检查 SDK 进程存活（get_server_info）
        死亡 → 通知 Go 后端"""
```

### 4.3 Watchdog（进程死亡检测）

```python
class AgentWatchdog:
    """监控 SDK 连接健康状态"""

    async def start(self):
        self._task = asyncio.create_task(self._monitor())

    async def _monitor(self):
        while self._running:
            await asyncio.sleep(30)
            if self._client is not None:
                try:
                    info = await self._client.get_server_info()
                    if info is None:
                        await self._handle_dead_process()
                except Exception:
                    await self._handle_dead_process()

    async def _handle_dead_process(self):
        self._agent.transition("failed")
        await self._reporter.report_failed("sdk_process_dead")
```

---

## 5. Go 后端集成变更

### 5.1 ComposeManager 单容器模板

```yaml
services:
  wrapper:
    image: ${WRAPPER_IMAGE}
    ports:
      - "9090"
    volumes:
      - ${WORKSPACE_DIR}/${TASK_ID}:/workspace
    environment:
      - TASK_ID=${TASK_ID}
      - CONTROL_PLANE_URL=${CONTROL_PLANE_URL}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - TASK_PROMPT=${TASK_PROMPT}
      - MAX_TIMEOUT=${MAX_TIMEOUT}
```

### 5.2 Pause/Interrupt 语义映射

| 旧 API | 旧语义 | 新映射 | 说明 |
|--------|--------|--------|------|
| `Pause()` | SIGSTOP 硬冻结 | `Interrupt()` | 中断当前生成，进程存活 |
| `Resume()` | SIGCONT 恢复 | 废弃 | interrupt 后用 Inject 恢复 |
| `Inject()` | 写文件 + 轮询 | `Inject()` | 直接 SDK query() |

Go 侧改动：
- `TaskExecutor` 新增 `Interrupt()` 方法（替代 `Pause()`）
- `WrapperClient` 端点调整：`/pause` → `/interrupt`，移除 `/resume`
- API Handler 层保持对外兼容（`POST /tasks/:id/pause` 映射到 interrupt）

### 5.3 事件接收

Wrapper 主动推送事件到 Control Plane，复用已有事件处理管道：

```
Wrapper POST /internal/tasks/:id/events
    → TaskExecutor.HandleTaskEvent()
    → 已有逻辑（status_change / heartbeat / complete / failed）
```

### 5.4 变更文件清单

| 文件 | 变更 |
|------|------|
| `internal/executor/compose_manager.go` | 新增单容器模板生成逻辑 |
| `internal/executor/docker_runtime.go` | `GetAddress()` 适配单容器服务名 |
| `internal/executor/wrapper_client.go` | 新增 `Interrupt()` 方法，移除 `Resume()`，调整端点 |
| `internal/executor/task_executor.go` | `Pause()` → `Interrupt()` 语义更新 |
| `internal/api/handler/intervention.go` | `/pause` 映射到 interrupt，保持 API 兼容 |
| `internal/api/router/router.go` | 新增 wrapper 事件推送接收路由 |

---

## 6. 容器设计

### 6.1 Dockerfile（多阶段构建 + 完整工具链）

```dockerfile
# Stage 1: Python 依赖
FROM python:3.12-slim AS builder
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Stage 2: Node.js + Claude Code CLI
FROM node:20-slim AS node-runtime
RUN npm install -g @anthropic-ai/claude-code

# Stage 3: 运行时
FROM python:3.12-slim

# 开发工具链
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl wget make gcc bash \
    && rm -rf /var/lib/apt/lists/*

# Go
RUN curl -sL https://go.dev/dl/go1.22.0.linux-$(dpkg --print-architecture).tar.gz \
    | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:${PATH}"

# GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update && apt-get install -y --no-install-recommends gh \
    && rm -rf /var/lib/apt/lists/*

# Node.js + Claude Code CLI
COPY --from=node-runtime /usr/local/bin/claude /usr/local/bin/
COPY --from=node-runtime /usr/local/lib/node_modules /usr/local/lib/node_modules

# Python deps
COPY --from=builder /usr/local/lib/python3.12 /usr/local/lib/python3.12

WORKDIR /app
COPY scripts/wrapper/ .
RUN pip install --no-cache-dir -r requirements.txt

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 9090
ENTRYPOINT ["/entrypoint.sh"]
```

### 6.2 工具清单

| 工具 | 用途 | 来源 |
|------|------|------|
| `bash` | Shell 执行 | apt (base) |
| `git` | 仓库操作 | apt |
| `go` 1.22 | Go 编译/测试 | 官方 tarball |
| `node`/`npm` | Node 项目 | 多阶段拷贝 |
| `gh` | GitHub PR/Issue | 官方 apt 源 |
| `make` | 构建工具 | apt |
| `curl`/`wget` | HTTP 请求 | apt |
| `gcc` | C 编译（CGO 等） | apt |
| `python` 3.12 | Wrapper 运行时 | base image |
| `claude` CLI | Agent SDK 底层 | npm 全局安装 |

### 6.3 ENTRYPOINT 脚本

关注点分离：环境准备由 bash 完成，Agent 逻辑由 Python 负责。

```bash
#!/bin/bash
set -e

# Git clone（如果需要）
if [ -n "$GIT_REPO" ]; then
    git clone --depth 1 "$GIT_REPO" /workspace/repo
fi

# 生成 CLAUDE.md（如果提供）
if [ -n "$CLAUDE_MD_CONTENT" ]; then
    echo "$CLAUDE_MD_CONTENT" > /workspace/repo/CLAUDE.md
fi

# 启动 Python wrapper
exec python -m uvicorn main:app --host 0.0.0.0 --port 9090
```

### 6.4 优雅关闭

```python
class GracefulShutdown:
    def __init__(self, agent: AgentClient):
        self._agent = agent
        signal.signal(signal.SIGTERM, self._handle_sigterm)

    async def _handle_sigterm(self):
        if self._agent.state in ("streaming", "starting"):
            await self._agent.interrupt()
        await self._agent.stop()
        await self._report_final_status()
```

### 6.5 超时处理

```python
# config.py
TASK_TIMEOUT = int(os.environ.get("MAX_TIMEOUT", "3600"))

# client.py
async def _timeout_watcher(self):
    await asyncio.sleep(config.TASK_TIMEOUT)
    if self.state in ("streaming", "starting"):
        await self.interrupt()
        await self._reporter.report_failed("task timeout")
```

---

## 7. 事件推送与错误处理

### 7.1 事件流

```
ClaudeSDKClient
    ↓ receive_response()
AgentClient._consume_messages()
    ↓ 解析 Message 类型
EventReporter
    ↓ HTTP POST (带重试)
Control Plane /internal/tasks/:id/events
    ↓
TaskExecutor.HandleTaskEvent()
```

### 7.2 事件类型映射

| SDK 消息类型 | 转换为 | 推送内容 |
|-------------|--------|---------|
| `AssistantMessage` (TextBlock) | `progress` | `{text, tokens_used}` |
| `AssistantMessage` (ToolUseBlock) | `tool_call` | `{tool_name, input}` |
| `ResultMessage` (success) | `complete` | `{cost, duration, tokens}` |
| `ResultMessage` (error) | `failed` | `{error, exit_code}` |
| SDK 异常 | `failed` | `{error: str(exception)}` |
| Watchdog 检测死亡 | `failed` | `{error: "sdk_process_dead"}` |

### 7.3 EventReporter

```python
class EventReporter:
    """将 SDK 消息转换为事件并推送到 Control Plane"""

    def __init__(self, task_id: str, control_plane_url: str):
        self._url = f"{control_plane_url}/internal/tasks/{task_id}/events"
        self._http: httpx.AsyncClient

    async def report(self, event_type: str, payload: dict) -> None:
        # 带重试（最多 3 次，指数退避）
        # 失败则降级写本地 events.jsonl

    async def report_progress(self, text: str, tokens: int) -> None: ...
    async def report_tool_call(self, tool: str, input_data: dict) -> None: ...
    async def report_complete(self, cost: float, duration: float) -> None: ...
    async def report_failed(self, error: str) -> None: ...
```

### 7.4 错误处理策略

| 场景 | 处理 |
|------|------|
| SDK 进程崩溃 | Watchdog 检测 → `report_failed()` → Go 后端决定重建或标记失败 |
| Control Plane 不可达 | 重试 3 次 → 降级写本地 `events.jsonl` |
| interrupt/inject 竞态 | `asyncio.Lock` 保护，拒绝非法状态转换，返回 HTTP 409 |
| 任务超时 | `_timeout_watcher` 触发 interrupt → `report_failed("timeout")` |
| 容器 SIGTERM | `GracefulShutdown` → interrupt + stop + 上报最终状态 |
| SDK 版本不兼容 | `requirements.txt` 锁版本 + `client.py` 薄抽象层 |

---

## 8. API 端点

### Wrapper HTTP API（FastAPI :9090）

| 方法 | 路径 | 说明 | 请求体 |
|------|------|------|--------|
| GET | `/health` | 健康检查 | — |
| GET | `/status` | 当前 Agent 状态 | — |
| POST | `/start` | 启动 Agent 任务 | `{prompt: string, options?: object}` |
| POST | `/interrupt` | 中断当前生成 | — |
| POST | `/inject` | 注入新指令 | `{prompt: string}` |
| POST | `/stop` | 停止 Agent | — |

### Control Plane 新增路由

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/internal/tasks/:id/events` | 接收 wrapper 推送的事件 |
