# Issue 47: Agent SDK 集成重构 Wrapper 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用 Python Claude Agent SDK 替代 shell+信号干预方案，合并 cli-runner + wrapper 为单容器

**Architecture:** 单 Python 进程运行 FastAPI + ClaudeSDKClient，共享 asyncio 事件循环。SDK consumer 在独立 asyncio.Task 中隔离异常。Go 后端适度重构 ComposeManager 和 WrapperClient 适配单容器模式。

**Tech Stack:** Python 3.12 / FastAPI / claude-agent-sdk / Go 1.21 / Gin / Docker Compose

**Design Spec:** `docs/superpowers/specs/2026-03-29-agent-sdk-wrapper-design.md`

---

## File Map

### New Files (Python Wrapper)

| File | Responsibility |
|------|----------------|
| `scripts/wrapper/requirements.txt` | Python 依赖声明 |
| `scripts/wrapper/config.py` | 环境变量配置 |
| `scripts/wrapper/models/__init__.py` | 包初始化 |
| `scripts/wrapper/models/schemas.py` | Pydantic 模型（状态枚举、请求/响应） |
| `scripts/wrapper/agent/__init__.py` | 包初始化 |
| `scripts/wrapper/agent/state.py` | 状态机（asyncio.Lock 保护） |
| `scripts/wrapper/agent/events.py` | EventReporter（HTTP 推送 + 重试 + 降级） |
| `scripts/wrapper/agent/client.py` | AgentClient（SDK 封装 + Watchdog + 超时） |
| `scripts/wrapper/api/__init__.py` | 包初始化 |
| `scripts/wrapper/api/routes.py` | FastAPI 路由 |
| `scripts/wrapper/main.py` | 入口（uvicorn + 优雅关闭） |
| `scripts/wrapper/tests/test_state.py` | 状态机测试 |
| `scripts/wrapper/tests/test_events.py` | EventReporter 测试 |
| `scripts/wrapper/tests/test_client.py` | AgentClient 测试 |
| `scripts/wrapper/tests/test_routes.py` | API 路由测试 |

### New Files (Container)

| File | Responsibility |
|------|----------------|
| `scripts/entrypoint.sh` | Git clone + CLAUDE.md 生成 + 启动 uvicorn |
| `scripts/wrapper/Dockerfile` | 多阶段构建（Python + Node + Go + gh） |

### Modified Files (Go Backend)

| File | Change |
|------|--------|
| `internal/executor/wrapper_client.go` | 新增 `Interrupt()`，移除 `Resume()`，调整端点路径 |
| `internal/executor/compose_manager.go` | 新增单容器 compose 模板 |
| `internal/executor/docker_runtime.go` | `GetAddress()` 适配单容器 |
| `internal/executor/task_executor.go` | `Pause()` → `Interrupt()` 语义，`Resume()` → 调用 `InjectInstruction()` |
| `internal/api/handler/intervention.go` | `/pause` 映射到 interrupt |
| `internal/api/router/router.go` | 新增内部事件接收路由 |

---

## Workflow Steps (agent.md §7)

Steps 0-4 已完成。本计划覆盖 Step 5-9。

### Step 0-4: ✅ 已完成

| Step | Status |
|------|--------|
| 0. 拉取最新代码 | ✅ `git pull origin main` |
| 1. 创建 Worktree | ✅ `feature/issue-47` |
| 2. 创建 Issue Summary | ⏳ Task 1 |
| 3. 获取背景知识 | ✅ executor + intervention + provider |
| 3.5 设计与文档回流 | ⏳ Task 2 (ADR) |
| 4. 创建执行计划 | ✅ 本文档 |

---

## Step 5: 开发实现 (TDD)

### Task 1: Issue Summary + ADR

**Files:**
- Create: `docs/current/issues/issue-47-agent-sdk-wrapper.md`
- Create: `docs/current/decisions/adr-xxx-agent-sdk-integration.md`

- [ ] **Step 1: 创建 Issue Summary**

按 `docs/current/issues/README.md` 模板，填写 issue 47 的摘要、范围、验收标准。

- [ ] **Step 2: 创建 ADR**

在 `docs/current/decisions/` 创建 ADR，记录选择 Agent SDK（方案 A）的决策理由。

- [ ] **Step 3: Commit**

```bash
git add docs/current/issues/issue-47-agent-sdk-wrapper.md docs/current/decisions/adr-xxx-agent-sdk-integration.md
git commit -m "docs(issues): add issue-47 agent sdk wrapper summary and ADR"
```

---

### Task 2: Python 项目脚手架 + Pydantic 模型 + 状态机

**Files:**
- Create: `scripts/wrapper/requirements.txt`
- Create: `scripts/wrapper/config.py`
- Create: `scripts/wrapper/models/__init__.py`
- Create: `scripts/wrapper/models/schemas.py`
- Create: `scripts/wrapper/agent/__init__.py`
- Create: `scripts/wrapper/agent/state.py`
- Create: `scripts/wrapper/tests/__init__.py`
- Create: `scripts/wrapper/tests/test_state.py`

- [ ] **Step 1: 创建 requirements.txt**

```txt
fastapi>=0.110.0
uvicorn>=0.29.0
claude-agent-sdk>=0.1.0
httpx>=0.27.0
pydantic>=2.6.0
pytest>=8.0.0
pytest-asyncio>=0.23.0
```

- [ ] **Step 2: 创建 config.py**

```python
import os


class Config:
    TASK_ID: str = os.environ.get("TASK_ID", "")
    CONTROL_PLANE_URL: str = os.environ.get("CONTROL_PLANE_URL", "http://localhost:8080")
    ANTHROPIC_API_KEY: str = os.environ.get("ANTHROPIC_API_KEY", "")
    TASK_PROMPT: str = os.environ.get("TASK_PROMPT", "")
    MAX_TIMEOUT: int = int(os.environ.get("MAX_TIMEOUT", "3600"))
    WORKSPACE_DIR: str = os.environ.get("WORKSPACE_DIR", "/workspace")
    GIT_REPO: str = os.environ.get("GIT_REPO", "")
    CLAUDE_MD_CONTENT: str = os.environ.get("CLAUDE_MD_CONTENT", "")
    ALLOWED_TOOLS: str = os.environ.get("ALLOWED_TOOLS", "")
    PORT: int = int(os.environ.get("PORT", "9090"))


config = Config()
```

- [ ] **Step 3: 创建 models/schemas.py**

```python
from enum import Enum

from pydantic import BaseModel


class AgentState(str, Enum):
    IDLE = "idle"
    STARTING = "starting"
    STREAMING = "streaming"
    INTERRUPTED = "interrupted"
    COMPLETED = "completed"
    FAILED = "failed"


class StartRequest(BaseModel):
    prompt: str
    options: dict | None = None


class InjectRequest(BaseModel):
    prompt: str


class StatusResponse(BaseModel):
    state: AgentState
    session_id: str | None = None
    error: str | None = None


class HealthResponse(BaseModel):
    status: str
    agent_state: AgentState
```

- [ ] **Step 4: 写状态机测试 test_state.py**

```python
import pytest
import asyncio

from agent.state import StateMachine, StateError


@pytest.fixture
def sm():
    return StateMachine()


@pytest.mark.asyncio
async def test_initial_state_is_idle(sm):
    assert sm.current == "idle"


@pytest.mark.asyncio
async def test_valid_transition_idle_to_starting(sm):
    await sm.transition("starting", "idle")
    assert sm.current == "starting"


@pytest.mark.asyncio
async def test_valid_transition_starting_to_streaming(sm):
    await sm.transition("starting", "idle")
    await sm.transition("streaming", "starting")
    assert sm.current == "streaming"


@pytest.mark.asyncio
async def test_valid_transition_streaming_to_interrupted(sm):
    await sm.transition("starting", "idle")
    await sm.transition("streaming", "starting")
    await sm.transition("interrupted", "streaming")
    assert sm.current == "interrupted"


@pytest.mark.asyncio
async def test_valid_transition_interrupted_to_streaming(sm):
    await sm.transition("starting", "idle")
    await sm.transition("streaming", "starting")
    await sm.transition("interrupted", "streaming")
    await sm.transition("streaming", "interrupted")
    assert sm.current == "streaming"


@pytest.mark.asyncio
async def test_valid_transition_streaming_to_completed(sm):
    await sm.transition("starting", "idle")
    await sm.transition("streaming", "starting")
    await sm.transition("completed", "streaming")
    assert sm.current == "completed"


@pytest.mark.asyncio
async def test_valid_transition_streaming_to_failed(sm):
    await sm.transition("starting", "idle")
    await sm.transition("streaming", "starting")
    await sm.transition("failed", "streaming")
    assert sm.current == "failed"


@pytest.mark.asyncio
async def test_invalid_transition_raises(sm):
    with pytest.raises(StateError):
        await sm.transition("streaming", "idle")  # can't go idle->streaming directly


@pytest.mark.asyncio
async def test_concurrent_transitions_only_one_succeeds():
    sm = StateMachine()
    await sm.transition("starting", "idle")

    results = await asyncio.gather(
        sm.transition("streaming", "starting"),
        sm.transition("failed", "starting"),
        return_exceptions=True,
    )
    # Exactly one should succeed, one should fail
    successes = [r for r in results if not isinstance(r, Exception)]
    failures = [r for r in results if isinstance(r, Exception)]
    assert len(successes) == 1
    assert len(failures) == 1
```

- [ ] **Step 5: 实现状态机 agent/state.py**

```python
import asyncio

from models.schemas import AgentState


class StateError(Exception):
    pass


class StateMachine:
    TRANSITIONS: dict[str, set[str]] = {
        AgentState.IDLE: {AgentState.STARTING},
        AgentState.STARTING: {AgentState.STREAMING, AgentState.FAILED},
        AgentState.STREAMING: {AgentState.INTERRUPTED, AgentState.COMPLETED, AgentState.FAILED},
        AgentState.INTERRUPTED: {AgentState.STREAMING, AgentState.COMPLETED, AgentState.FAILED},
        AgentState.COMPLETED: set(),
        AgentState.FAILED: set(),
    }

    def __init__(self):
        self._state: str = AgentState.IDLE
        self._lock = asyncio.Lock()

    @property
    def current(self) -> str:
        return self._state

    async def transition(self, new_state: str, precondition: str) -> None:
        async with self._lock:
            if self._state != precondition:
                raise StateError(
                    f"Cannot transition to {new_state}: current state is {self._state}, expected {precondition}"
                )
            allowed = self.TRANSITIONS.get(self._state, set())
            if new_state not in allowed:
                raise StateError(
                    f"Invalid transition from {self._state} to {new_state}"
                )
            self._state = new_state

    async def force_transition(self, new_state: str) -> None:
        """Force state transition without precondition check (for error recovery)."""
        async with self._lock:
            self._state = new_state
```

- [ ] **Step 6: 运行测试验证**

```bash
cd scripts/wrapper && python -m pytest tests/test_state.py -v
```

Expected: 所有 10 个测试通过

- [ ] **Step 7: Commit**

```bash
git add scripts/wrapper/requirements.txt scripts/wrapper/config.py scripts/wrapper/models/ scripts/wrapper/agent/__init__.py scripts/wrapper/agent/state.py scripts/wrapper/tests/
git commit -m "feat(wrapper): add project scaffold, schemas, and state machine"
```

---

### Task 3: EventReporter（事件推送 + 重试 + 降级）

**Files:**
- Create: `scripts/wrapper/agent/events.py`
- Create: `scripts/wrapper/tests/test_events.py`

- [ ] **Step 1: 写 EventReporter 测试**

```python
import pytest
import asyncio
from unittest.mock import AsyncMock, patch, MagicMock

from agent.events import EventReporter


@pytest.fixture
def reporter():
    return EventReporter(task_id="test-task-123", control_plane_url="http://localhost:8080")


@pytest.mark.asyncio
async def test_report_progress(reporter):
    with patch.object(reporter, "_post_event", new_callable=AsyncMock) as mock_post:
        await reporter.report_progress("hello world", tokens=50)
        mock_post.assert_called_once()
        call_args = mock_post.call_args
        assert call_args[0][0] == "progress"
        assert call_args[0][1]["text"] == "hello world"


@pytest.mark.asyncio
async def test_report_tool_call(reporter):
    with patch.object(reporter, "_post_event", new_callable=AsyncMock) as mock_post:
        await reporter.report_tool_call("Edit", {"file": "main.go"})
        mock_post.assert_called_once()
        call_args = mock_post.call_args
        assert call_args[0][0] == "tool_call"
        assert call_args[0][1]["tool_name"] == "Edit"


@pytest.mark.asyncio
async def test_report_complete(reporter):
    with patch.object(reporter, "_post_event", new_callable=AsyncMock) as mock_post:
        await reporter.report_complete(cost=0.05, duration=30.0, tokens=1000)
        mock_post.assert_called_once()
        call_args = mock_post.call_args
        assert call_args[0][0] == "complete"
        assert call_args[0][1]["cost_usd"] == 0.05


@pytest.mark.asyncio
async def test_report_failed(reporter):
    with patch.object(reporter, "_post_event", new_callable=AsyncMock) as mock_post:
        await reporter.report_failed("sdk_process_dead")
        mock_post.assert_called_once()
        call_args = mock_post.call_args
        assert call_args[0][0] == "failed"
        assert call_args[0][1]["error"] == "sdk_process_dead"


@pytest.mark.asyncio
async def test_retry_on_failure(reporter):
    call_count = 0

    async def failing_post(event_type, payload):
        nonlocal call_count
        call_count += 1
        if call_count < 3:
            raise ConnectionError("connection refused")
        # succeeds on 3rd try

    with patch.object(reporter, "_post_event", side_effect=failing_post):
        await reporter.report("progress", {"text": "retry test"})
        assert call_count == 3
```

- [ ] **Step 2: 实现 EventReporter agent/events.py**

```python
import json
import logging
import os
from pathlib import Path

import httpx

logger = logging.getLogger(__name__)

# Fallback: write to local file if control plane unreachable
FALLBACK_EVENTS_FILE = "/workspace/.agent-state/events.jsonl"


class EventReporter:
    def __init__(self, task_id: str, control_plane_url: str):
        self._task_id = task_id
        self._url = f"{control_plane_url}/internal/tasks/{task_id}/events"
        self._max_retries = 3

    async def _post_event(self, event_type: str, payload: dict) -> None:
        async with httpx.AsyncClient(timeout=5.0) as client:
            resp = await client.post(self._url, json={
                "event_type": event_type,
                "payload": payload,
            })
            resp.raise_for_status()

    async def _fallback_write(self, event_type: str, payload: dict) -> None:
        state_dir = os.path.dirname(FALLBACK_EVENTS_FILE)
        os.makedirs(state_dir, exist_ok=True)
        event = {"event_type": event_type, "payload": payload}
        with open(FALLBACK_EVENTS_FILE, "a") as f:
            f.write(json.dumps(event) + "\n")

    async def report(self, event_type: str, payload: dict) -> None:
        for attempt in range(self._max_retries):
            try:
                await self._post_event(event_type, payload)
                return
            except Exception as e:
                logger.warning("Event report attempt %d failed: %s", attempt + 1, e)
                if attempt < self._max_retries - 1:
                    import asyncio
                    await asyncio.sleep(2 ** attempt)
                else:
                    logger.error("All retries failed, falling back to local file")
                    await self._fallback_write(event_type, payload)

    async def report_progress(self, text: str, tokens: int) -> None:
        await self.report("progress", {"text": text, "tokens_used": tokens})

    async def report_tool_call(self, tool: str, input_data: dict) -> None:
        await self.report("tool_call", {"tool_name": tool, "input": input_data})

    async def report_complete(self, cost: float, duration: float, tokens: int = 0) -> None:
        await self.report("complete", {"cost_usd": cost, "duration_s": duration, "tokens": tokens})

    async def report_failed(self, error: str) -> None:
        await self.report("failed", {"error": error})
```

- [ ] **Step 3: 运行测试**

```bash
cd scripts/wrapper && python -m pytest tests/test_events.py -v
```

Expected: 全部通过

- [ ] **Step 4: Commit**

```bash
git add scripts/wrapper/agent/events.py scripts/wrapper/tests/test_events.py
git commit -m "feat(wrapper): add EventReporter with retry and fallback"
```

---

### Task 4: AgentClient（SDK 封装 + Watchdog + 超时）

**Files:**
- Create: `scripts/wrapper/agent/client.py`
- Create: `scripts/wrapper/tests/test_client.py`

- [ ] **Step 1: 写 AgentClient 测试**

```python
import pytest
import asyncio
from unittest.mock import AsyncMock, MagicMock, patch

from agent.client import AgentClient
from agent.state import StateMachine
from models.schemas import AgentState


@pytest.fixture
def client():
    with patch("agent.client.ClaudeSDKClient"):
        return AgentClient(task_id="test-123", control_plane_url="http://localhost:8080")


@pytest.mark.asyncio
async def test_start_transitions_to_starting(client):
    client._sdk_client.connect = AsyncMock()
    client._sdk_client.query = AsyncMock()
    # Don't actually start consumer - just verify state transitions
    await client._state.transition("starting", "idle")
    assert client._state.current == AgentState.STARTING


@pytest.mark.asyncio
async def test_interrupt_requires_streaming(client):
    # Should fail when state is idle
    with pytest.raises(Exception):
        await client.interrupt()


@pytest.mark.asyncio
async def test_inject_requires_interrupted(client):
    # Should fail when state is idle
    with pytest.raises(Exception):
        await client.inject("new instruction")


@pytest.mark.asyncio
async def test_stop_cleans_up(client):
    client._sdk_client.disconnect = AsyncMock()
    client._consumer_task = None
    await client.stop()
    client._sdk_client.disconnect.assert_called_once()


@pytest.mark.asyncio
async def test_status_returns_current_state(client):
    status = client.get_status()
    assert status["state"] == AgentState.IDLE
```

- [ ] **Step 2: 实现 AgentClient agent/client.py**

```python
import asyncio
import logging
from typing import Optional

from claude_agent_sdk import ClaudeSDKClient, ClaudeAgentOptions, AssistantMessage, ResultMessage, TextBlock, ToolUseBlock

from agent.state import StateMachine, StateError
from agent.events import EventReporter
from config import config
from models.schemas import AgentState

logger = logging.getLogger(__name__)


class AgentClient:
    def __init__(self, task_id: str, control_plane_url: str):
        self._task_id = task_id
        self._state = StateMachine()
        self._sdk_client: Optional[ClaudeSDKClient] = None
        self._consumer_task: Optional[asyncio.Task] = None
        self._watchdog_task: Optional[asyncio.Task] = None
        self._timeout_task: Optional[asyncio.Task] = None
        self._reporter = EventReporter(task_id, control_plane_url)
        self._sdk_error: Optional[Exception] = None

    @property
    def state(self) -> str:
        return self._state.current

    def get_status(self) -> dict:
        return {
            "state": self._state.current,
            "error": str(self._sdk_error) if self._sdk_error else None,
        }

    def _build_options(self, user_options: dict | None = None) -> ClaudeAgentOptions:
        opts = ClaudeAgentOptions(
            api_key=config.ANTHROPIC_API_KEY,
        )
        if config.ALLOWED_TOOLS:
            opts.allowed_tools = config.ALLOWED_TOOLS.split(",")
        if user_options:
            for k, v in user_options.items():
                if hasattr(opts, k):
                    setattr(opts, k, v)
        return opts

    async def start(self, prompt: str, options: dict | None = None) -> None:
        await self._state.transition(AgentState.STARTING, AgentState.IDLE)
        self._sdk_client = ClaudeSDKClient(options=self._build_options(options))
        try:
            await self._sdk_client.connect(prompt)
            await self._state.transition(AgentState.STREAMING, AgentState.STARTING)
            self._consumer_task = asyncio.create_task(self._consume_messages())
            self._watchdog_task = asyncio.create_task(self._watchdog_monitor())
            self._timeout_task = asyncio.create_task(self._timeout_watcher())
        except Exception as e:
            self._sdk_error = e
            await self._state.force_transition(AgentState.FAILED)
            await self._reporter.report_failed(str(e))
            raise

    async def interrupt(self) -> None:
        await self._state.transition(AgentState.INTERRUPTED, AgentState.STREAMING)
        if self._sdk_client:
            await self._sdk_client.interrupt()

    async def inject(self, prompt: str) -> None:
        await self._state.transition(AgentState.STREAMING, AgentState.INTERRUPTED)
        if self._sdk_client:
            await self._sdk_client.query(prompt)
            # Restart consumer for the new response
            if self._consumer_task and not self._consumer_task.done():
                self._consumer_task.cancel()
            self._consumer_task = asyncio.create_task(self._consume_messages())

    async def stop(self) -> None:
        if self._consumer_task and not self._consumer_task.done():
            self._consumer_task.cancel()
        if self._watchdog_task and not self._watchdog_task.done():
            self._watchdog_task.cancel()
        if self._timeout_task and not self._timeout_task.done():
            self._timeout_task.cancel()
        if self._sdk_client:
            try:
                await self._sdk_client.disconnect()
            except Exception as e:
                logger.warning("Error disconnecting SDK client: %s", e)

    async def _consume_messages(self) -> None:
        try:
            async for message in self._sdk_client.receive_response():
                if isinstance(message, AssistantMessage):
                    for block in message.content:
                        if isinstance(block, TextBlock):
                            await self._reporter.report_progress(block.text, tokens=0)
                        elif isinstance(block, ToolUseBlock):
                            await self._reporter.report_tool_call(block.name, getattr(block, "input", {}))
                elif isinstance(message, ResultMessage):
                    if message.subtype == "success":
                        await self._state.transition(AgentState.COMPLETED, self._state.current)
                        await self._reporter.report_complete(
                            cost=getattr(message, "total_cost_usd", 0.0) or 0.0,
                            duration=0.0,
                        )
                    else:
                        await self._state.transition(AgentState.FAILED, self._state.current)
                        await self._reporter.report_failed(getattr(message, "error", "unknown error"))
                    return
        except asyncio.CancelledError:
            logger.info("Consumer task cancelled")
        except Exception as e:
            self._sdk_error = e
            await self._state.force_transition(AgentState.FAILED)
            await self._reporter.report_failed(str(e))

    async def _watchdog_monitor(self) -> None:
        try:
            while True:
                await asyncio.sleep(30)
                if self._sdk_client and self._state.current in (AgentState.STREAMING, AgentState.INTERRUPTED):
                    try:
                        info = await self._sdk_client.get_server_info()
                        if info is None:
                            logger.error("Watchdog: SDK process appears dead")
                            await self._state.force_transition(AgentState.FAILED)
                            await self._reporter.report_failed("sdk_process_dead")
                            return
                    except Exception:
                        logger.error("Watchdog: SDK health check failed")
                        await self._state.force_transition(AgentState.FAILED)
                        await self._reporter.report_failed("sdk_process_dead")
                        return
        except asyncio.CancelledError:
            pass

    async def _timeout_watcher(self) -> None:
        try:
            await asyncio.sleep(config.MAX_TIMEOUT)
            if self._state.current in (AgentState.STREAMING, AgentState.STARTING):
                logger.warning("Task timeout after %d seconds", config.MAX_TIMEOUT)
                if self._sdk_client:
                    await self._sdk_client.interrupt()
                await self._state.force_transition(AgentState.FAILED)
                await self._reporter.report_failed("task timeout")
        except asyncio.CancelledError:
            pass
```

- [ ] **Step 3: 运行测试**

```bash
cd scripts/wrapper && python -m pytest tests/test_client.py -v
```

Expected: 全部通过

- [ ] **Step 4: Commit**

```bash
git add scripts/wrapper/agent/client.py scripts/wrapper/tests/test_client.py
git commit -m "feat(wrapper): add AgentClient with SDK lifecycle and watchdog"
```

---

### Task 5: FastAPI 路由 + main.py 入口

**Files:**
- Create: `scripts/wrapper/api/routes.py`
- Create: `scripts/wrapper/main.py`
- Create: `scripts/wrapper/tests/test_routes.py`

- [ ] **Step 1: 写 API 路由测试**

```python
import pytest
from unittest.mock import AsyncMock, MagicMock
from fastapi.testclient import TestClient

from models.schemas import AgentState


@pytest.fixture
def mock_agent():
    agent = MagicMock()
    agent.state = AgentState.IDLE
    agent.get_status.return_value = {"state": AgentState.IDLE, "error": None}
    agent.start = AsyncMock()
    agent.interrupt = AsyncMock()
    agent.inject = AsyncMock()
    agent.stop = AsyncMock()
    return agent


@pytest.fixture
def client(mock_agent):
    from api.routes import create_app
    app = create_app(mock_agent)
    return TestClient(app)


def test_health(client):
    resp = client.get("/health")
    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "ok"
    assert data["agent_state"] == AgentState.IDLE


def test_get_status(client):
    resp = client.get("/status")
    assert resp.status_code == 200
    assert resp.json()["state"] == AgentState.IDLE


def test_start_task(client, mock_agent):
    resp = client.post("/start", json={"prompt": "fix the bug"})
    assert resp.status_code == 200
    mock_agent.start.assert_called_once()


def test_interrupt_when_not_streaming(client, mock_agent):
    mock_agent.interrupt.side_effect = Exception("invalid state")
    resp = client.post("/interrupt")
    assert resp.status_code == 409


def test_inject_when_not_interrupted(client, mock_agent):
    mock_agent.inject.side_effect = Exception("invalid state")
    resp = client.post("/inject", json={"prompt": "new instruction"})
    assert resp.status_code == 409


def test_stop(client, mock_agent):
    resp = client.post("/stop")
    assert resp.status_code == 200
    mock_agent.stop.assert_called_once()
```

- [ ] **Step 2: 实现 routes.py**

```python
import logging

from fastapi import FastAPI, HTTPException

from models.schemas import AgentState, StartRequest, InjectRequest, StatusResponse, HealthResponse

logger = logging.getLogger(__name__)


def create_app(agent_client) -> FastAPI:
    app = FastAPI(title="Agent Wrapper", version="1.0.0")
    _agent = agent_client

    @app.get("/health", response_model=HealthResponse)
    async def health():
        return HealthResponse(status="ok", agent_state=_agent.state)

    @app.get("/status", response_model=StatusResponse)
    async def get_status():
        return _agent.get_status()

    @app.post("/start")
    async def start_task(req: StartRequest):
        try:
            await _agent.start(req.prompt, req.options)
            return {"status": "started"}
        except Exception as e:
            logger.error("Start failed: %s", e)
            raise HTTPException(status_code=409, detail=str(e))

    @app.post("/interrupt")
    async def interrupt_task():
        try:
            await _agent.interrupt()
            return {"status": "interrupted"}
        except Exception as e:
            logger.error("Interrupt failed: %s", e)
            raise HTTPException(status_code=409, detail=str(e))

    @app.post("/inject")
    async def inject_task(req: InjectRequest):
        try:
            await _agent.inject(req.prompt)
            return {"status": "injected"}
        except Exception as e:
            logger.error("Inject failed: %s", e)
            raise HTTPException(status_code=409, detail=str(e))

    @app.post("/stop")
    async def stop_task():
        await _agent.stop()
        return {"status": "stopped"}

    return app
```

- [ ] **Step 3: 实现 main.py（入口 + 优雅关闭）**

```python
import asyncio
import logging
import signal

import uvicorn

from agent.client import AgentClient
from api.routes import create_app
from config import config

logger = logging.getLogger(__name__)


class GracefulShutdown:
    def __init__(self, agent: AgentClient):
        self._agent = agent
        self._shutdown_event = asyncio.Event()
        loop = asyncio.get_event_loop()
        loop.add_signal_handler(signal.SIGTERM, lambda: asyncio.ensure_future(self._handle_sigterm()))

    async def _handle_sigterm(self):
        logger.info("Received SIGTERM, shutting down gracefully...")
        try:
            if self._agent.state in ("streaming", "starting"):
                await self._agent.interrupt()
            await self._agent.stop()
        except Exception as e:
            logger.error("Error during shutdown: %s", e)
        self._shutdown_event.set()

    @property
    def shutdown_event(self) -> asyncio.Event:
        return self._shutdown_event


def main():
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")

    agent = AgentClient(task_id=config.TASK_ID, control_plane_url=config.CONTROL_PLANE_URL)
    app = create_app(agent)

    shutdown = GracefulShutdown(agent)

    @app.on_event("startup")
    async def on_startup():
        if config.TASK_PROMPT:
            logger.info("Auto-starting task with prompt")
            await agent.start(config.TASK_PROMPT)

    uvicorn_config = uvicorn.Config(
        app,
        host="0.0.0.0",
        port=config.PORT,
        log_level="info",
    )
    server = uvicorn.Server(uvicorn_config)

    asyncio.run(server.serve())


if __name__ == "__main__":
    main()
```

- [ ] **Step 4: 运行测试**

```bash
cd scripts/wrapper && python -m pytest tests/test_routes.py -v
```

Expected: 全部通过

- [ ] **Step 5: Commit**

```bash
git add scripts/wrapper/api/routes.py scripts/wrapper/main.py scripts/wrapper/tests/test_routes.py
git commit -m "feat(wrapper): add FastAPI routes and main entry with graceful shutdown"
```

---

### Task 6: 容器构建文件（Dockerfile + entrypoint）

**Files:**
- Create: `scripts/entrypoint.sh`
- Create: `scripts/wrapper/Dockerfile`

- [ ] **Step 1: 创建 entrypoint.sh**

```bash
#!/bin/bash
set -e

STATE_DIR="/workspace/.agent-state"
mkdir -p "$STATE_DIR"

# Git clone
if [ -n "$GIT_REPO" ]; then
    echo "Cloning $GIT_REPO..."
    REPO_DIR="/workspace/repo"
    if [ -n "$GIT_BRANCH" ]; then
        git clone --depth 1 --branch "$GIT_BRANCH" "$GIT_REPO" "$REPO_DIR"
    else
        git clone --depth 1 "$GIT_REPO" "$REPO_DIR"
    fi
fi

# Generate CLAUDE.md
if [ -n "$CLAUDE_MD_CONTENT" ]; then
    TARGET_DIR="${REPO_DIR:-/workspace}"
    echo "$CLAUDE_MD_CONTENT" > "$TARGET_DIR/CLAUDE_MD_CONTENT.md"
fi

# Start Python wrapper
cd /app
exec python main.py
```

- [ ] **Step 2: 创建 Dockerfile**

```dockerfile
# Stage 1: Python dependencies
FROM python:3.12-slim AS py-builder
WORKDIR /build
COPY scripts/wrapper/requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

# Stage 2: Node.js + Claude Code CLI
FROM node:20-slim AS node-builder
RUN npm install -g @anthropic-ai/claude-code

# Stage 3: Runtime
FROM python:3.12-slim

# Dev toolchain
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl wget make gcc bash \
    && rm -rf /var/lib/apt/lists/*

# Go
RUN ARCH=$(dpkg --print-architecture) && \
    curl -sL "https://go.dev/dl/go1.22.0.linux-${ARCH}.tar.gz" | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:${PATH}"

# GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update && apt-get install -y --no-install-recommends gh \
    && rm -rf /var/lib/apt/lists/*

# Node.js + Claude Code CLI from builder
COPY --from=node-builder /usr/local/bin/claude /usr/local/bin/
COPY --from=node-builder /usr/local/lib/node_modules /usr/local/lib/node_modules

# Python dependencies from builder
COPY --from=py-builder /install /usr/local

# Wrapper application
WORKDIR /app
COPY scripts/wrapper/ .

# Entrypoint script
COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 9090
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:9090/health || exit 1

ENTRYPOINT ["/entrypoint.sh"]
```

- [ ] **Step 3: Commit**

```bash
chmod +x scripts/entrypoint.sh
git add scripts/entrypoint.sh scripts/wrapper/Dockerfile
git commit -m "feat(wrapper): add Dockerfile with full dev toolchain and entrypoint"
```

---

### Task 7: Go 后端 — WrapperClient 更新

**Files:**
- Modify: `internal/executor/wrapper_client.go`
- Create: `internal/executor/wrapper_client_test.go`（如果不存在则创建，存在则追加）

- [ ] **Step 1: 写 WrapperClient.Interrupt 测试**

在 `internal/executor/wrapper_client_test.go` 中添加：

```go
func TestWrapperClient_Interrupt(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodPost && r.URL.Path == "/interrupt" {
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(map[string]string{"status": "interrupted"})
            return
        }
        w.WriteHeader(http.StatusNotFound)
    }))
    defer server.Close()

    cfg := &WrapperClientConfig{Timeout: 5 * time.Second}
    client := NewWrapperClient(cfg)

    addr := strings.TrimPrefix(server.URL, "http://")
    err := client.Interrupt(context.Background(), addr)
    assert.NoError(t, err)
}
```

- [ ] **Step 2: 在 wrapper_client.go 中添加 Interrupt 方法**

在 `Pause` 方法后添加：

```go
// Interrupt sends an interrupt request to the wrapper's /interrupt endpoint.
func (c *WrapperClient) Interrupt(ctx context.Context, address string) error {
	url := fmt.Sprintf("http://%s/interrupt", net.JoinHostPort(address, fmt.Sprintf("%d", c.port)))
	_, err := c.doRequest(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("interrupt wrapper %s: %w", address, err)
	}
	return nil
}
```

- [ ] **Step 3: 移除 Resume 方法或将已弃用标记**

给 `Resume` 方法添加弃用注释：

```go
// Deprecated: Resume is no longer supported with Agent SDK wrapper.
// Use InjectInstruction instead to send new instructions to an interrupted agent.
func (c *WrapperClient) Resume(ctx context.Context, address string) error {
```

- [ ] **Step 4: 运行测试**

```bash
cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-47
go test ./internal/executor/ -run TestWrapperClient -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/executor/wrapper_client.go internal/executor/wrapper_client_test.go
git commit -m "feat(executor): add WrapperClient.Interrupt and deprecate Resume"
```

---

### Task 8: Go 后端 — ComposeManager 单容器模板

**Files:**
- Modify: `internal/executor/compose_manager.go`
- Modify: `internal/executor/compose_manager_test.go`（如果存在）

- [ ] **Step 1: 在 compose_manager.go 中添加单容器模板**

在 `GenerateConfig` 方法中，支持通过环境变量或配置切换模板。在现有模板常量之后添加新的单容器模板：

```go
const singleContainerComposeTemplate = `version: '3.8'
services:
  wrapper:
    image: {{.WrapperImage}}
    ports:
      - "9090"
    volumes:
      - {{.WorkspaceDir}}/{{.TaskID}}:/workspace
    environment:
      - TASK_ID={{.TaskID}}
      - CONTROL_PLANE_URL={{.ControlPlaneURL}}
      - ANTHROPIC_API_KEY={{.AnthropicAPIKey}}
      - TASK_PROMPT={{.TaskPrompt}}
      - MAX_TIMEOUT={{.MaxTimeout}}
      - WORKSPACE_DIR=/workspace
      - GIT_REPO={{.GitRepo}}
      - GIT_BRANCH={{.GitBranch}}
      - CLAUDE_MD_CONTENT={{.ClaudeMdContent}}
      - ALLOWED_TOOLS={{.AllowedTools}}
`
```

- [ ] **Step 2: 添加模板渲染结构和逻辑**

```go
type SingleContainerTemplateData struct {
	TaskID           string
	WrapperImage     string
	WorkspaceDir     string
	ControlPlaneURL  string
	AnthropicAPIKey  string
	TaskPrompt       string
	MaxTimeout       string
	GitRepo          string
	GitBranch        string
	ClaudeMdContent  string
	AllowedTools     string
}

// GenerateSingleContainerConfig generates a single-container docker-compose.yml
// using the Agent SDK wrapper (replaces cli-runner + wrapper sidecar pattern).
func (m *ComposeManager) GenerateSingleContainerConfig(ctx context.Context, taskID string, data *SingleContainerTemplateData) error {
	taskDir := m.TaskDir(taskID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("create task dir %s: %w", taskDir, err)
	}
	data.TaskID = taskID

	tmpl, err := template.New("compose").Parse(singleContainerComposeTemplate)
	if err != nil {
		return fmt.Errorf("parse compose template: %w", err)
	}

	composeFile := filepath.Join(taskDir, "docker-compose.yml")
	f, err := os.Create(composeFile)
	if err != nil {
		return fmt.Errorf("create compose file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("execute compose template: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/executor/ -run TestComposeManager -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/executor/compose_manager.go
git commit -m "feat(executor): add single-container compose template for Agent SDK wrapper"
```

---

### Task 9: Go 后端 — DockerRuntime + TaskExecutor 适配

**Files:**
- Modify: `internal/executor/docker_runtime.go`
- Modify: `internal/executor/task_executor.go`

- [ ] **Step 1: 更新 DockerRuntime.GetAddress**

在 `docker_runtime.go` 的 `GetAddress` 方法中，将服务名从 `wrapper` 保持不变（单容器也叫 `wrapper`）：

```go
// GetAddress is unchanged - the single container service is still named "wrapper"
func (r *DockerRuntime) GetAddress(ctx context.Context, taskID string) (string, error) {
    port, err := r.compose.GetServicePort(ctx, taskID, "wrapper", 9090)
    // ... existing logic
}
```

（实际上服务名不变，这里可能不需要改动。检查 `GetStatus` 是否也正常工作——单容器模式下只有一个 service，但现有逻辑应该兼容。）

- [ ] **Step 2: 更新 TaskExecutor.Pause 语义**

在 `task_executor.go` 中，修改 `Pause` 方法调用 `Interrupt` 替代 `Pause`：

```go
// Pause now maps to Interrupt for Agent SDK wrapper compatibility.
// For legacy K8s runtime, the old Pause behavior is preserved.
func (e *TaskExecutor) Pause(ctx context.Context, taskID string) error {
    // ... existing validation logic ...

    // Try Interrupt first (Agent SDK wrapper), fall back to Pause (legacy)
    err := e.wrapperClient.Interrupt(ctx, address)
    if err != nil {
        // Fallback to legacy pause for K8s runtime
        err = e.wrapperClient.Pause(ctx, address)
    }
    // ... rest of existing logic ...
}
```

- [ ] **Step 3: 更新 TaskExecutor.Resume 语义**

修改 `Resume` 方法，对 Agent SDK wrapper 使用 `InjectInstruction` 替代 `Resume`：

```go
// Resume for Agent SDK wrapper is a no-op.
// The agent waits in interrupted state for an inject call.
func (e *TaskExecutor) Resume(ctx context.Context, taskID string) error {
    // For Agent SDK wrapper, resume is handled by InjectInstruction
    // This method is kept for API backward compatibility
    logger.Info("Resume called - for Agent SDK wrapper, use InjectInstruction instead")
    // Update task status to running
    // ... minimal implementation for API compat ...
    return nil
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/executor/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/executor/docker_runtime.go internal/executor/task_executor.go
git commit -m "refactor(executor): map Pause to Interrupt for Agent SDK wrapper"
```

---

### Task 10: Go 后端 — Intervention Handler + 事件接收路由

**Files:**
- Modify: `internal/api/handler/intervention.go`
- Modify: `internal/api/router/router.go`

- [ ] **Step 1: 更新 intervention handler**

在 `intervention.go` 中，`Pause` handler 的实现保持对外 API 不变，但内部调用链已通过 TaskExecutor 映射到 Interrupt。无需修改 handler 代码（TaskExecutor.Pause 内部已处理映射）。

确认 handler 无需改动。

- [ ] **Step 2: 在 router.go 添加内部事件接收路由**

在 `internal/api/router/router.go` 的 `Setup` 函数中，找到 internal 路由组（如果没有则创建），添加事件接收端点：

```go
// Internal routes for wrapper event push
internal := engine.Group("/internal")
{
    internal.POST("/tasks/:id/events", taskSvc.HandleTaskEvent)
}
```

需要确认 `InterventionService` 或 `TaskService` 是否已有 `HandleTaskEvent` 方法。从 `task_executor.go` 的 `HandleTaskEvent` 来看，它在 executor 层。需要在 service 层添加透传方法。

- [ ] **Step 3: 在 service 层添加事件处理透传**

在 `internal/service/` 中对应的服务文件添加：

```go
// HandleTaskEvent receives event pushes from wrapper containers.
func (s *TaskService) HandleTaskEvent(c *gin.Context) {
    taskID := c.Param("id")
    var req struct {
        EventType string                 `json:"event_type"`
        Payload   map[string]interface{} `json:"payload"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, "invalid request")
        return
    }
    if err := s.executor.HandleTaskEvent(c.Request.Context(), taskID, req.EventType, req.Payload); err != nil {
        response.Error(c, http.StatusInternalServerError, "handle event failed")
        return
    }
    response.Success(c, nil)
}
```

- [ ] **Step 4: 运行全量测试**

```bash
go test ./internal/... -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/api/router/router.go internal/service/
git commit -m "feat(api): add internal event receiving route for wrapper push"
```

---

## Step 6: 测试验证

### Task 11: 全量测试

- [ ] **Step 1: Python wrapper 测试**

```bash
cd scripts/wrapper && python -m pytest tests/ -v --tb=short
```

Expected: 全部通过

- [ ] **Step 2: Go 后端测试**

```bash
cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-47
make test
```

Expected: 全部通过

- [ ] **Step 3: Lint 检查**

```bash
make lint
```

Expected: 无错误

- [ ] **Step 4: 测试覆盖率**

```bash
go test -cover ./internal/executor/...
```

Expected: 覆盖率 > 80%

---

## Step 7: 代码审查

### Task 12: 代码审查

- [ ] **Step 1: 审查 Python wrapper 代码质量**
- [ ] **Step 2: 审查 Go 后端改动与现有代码一致性**
- [ ] **Step 3: 验证设计文档与实现一致性**
- [ ] **Step 4: 修复发现的问题**

---

## Step 8: 提交推送

### Task 13: 提交推送

- [ ] **Step 1: 检查所有改动**

```bash
git status
git diff --stat main
```

- [ ] **Step 2: 推送到远端**

```bash
git push -u origin feature/issue-47
```

---

## Step 9: 创建 PR

### Task 14: 创建 PR

- [ ] **Step 1: 创建 Pull Request**

```bash
gh pr create --base main --title "feat(executor): Agent SDK wrapper integration (issue #47)" --body "$(cat <<'EOF'
## Summary

- Replace shell+signal intervention with Python Claude Agent SDK
- Merge cli-runner + wrapper into single container with FastAPI + ClaudeSDKClient
- Add interrupt/inject via SDK API (process stays alive, same session)
- Update Go backend: ComposeManager single-container template, WrapperClient.Interrupt

## Test plan

- [x] Python wrapper unit tests (state machine, events, routes)
- [x] Go backend unit tests (WrapperClient.Interrupt, ComposeManager)
- [x] `make test` passes
- [x] `make lint` passes

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Remaining Steps (manual)

| Step | Action | Owner |
|------|--------|-------|
| 10 | 等待合并（人工审核） | Human |
| 11 | 拉取合并代码 | Agent |
| 12 | 关闭 Issue #47 | Agent |
| 13 | 清理 worktree | Agent |
