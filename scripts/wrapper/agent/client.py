"""AgentClient wrapping ClaudeSDKClient with lifecycle, watchdog, and timeout.

The AgentClient manages the full lifecycle of a Claude Code SDK session,
including message consumption, watchdog monitoring, and timeout enforcement.
"""
import asyncio
import logging
from typing import Any, Dict, Optional

from claude_agent_sdk import (
    AssistantMessage,
    ClaudeAgentOptions,
    ClaudeSDKClient,
    ResultMessage,
    TextBlock,
    ToolUseBlock,
)

from agent.events import EventReporter
from agent.state import StateError, StateMachine
from config import config
from models.schemas import AgentState

logger = logging.getLogger(__name__)


class AgentClient:
    """Wraps ClaudeSDKClient with lifecycle management, watchdog, and timeout.

    Provides start/interrupt/inject/stop operations that coordinate the state
    machine, event reporter, and background tasks for message consumption,
    watchdog monitoring, and timeout enforcement.
    """

    def __init__(self, task_id: str, control_plane_url: str) -> None:
        """Initialize the AgentClient.

        Args:
            task_id: Unique identifier for the task.
            control_plane_url: Base URL of the control plane API.
        """
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
        """Get the current agent state."""
        return self._state.current

    def get_status(self) -> Dict[str, Any]:
        """Get the current agent status as a dictionary.

        Returns:
            Dictionary with state and optional error message.
        """
        return {
            "state": self._state.current,
            "error": str(self._sdk_error) if self._sdk_error else None,
        }

    def _build_options(self, user_options: Optional[Dict[str, Any]] = None) -> ClaudeAgentOptions:
        """Build ClaudeAgentOptions from config and user overrides.

        Args:
            user_options: Optional dict of user-provided option overrides.

        Returns:
            Configured ClaudeAgentOptions instance.
        """
        opts = ClaudeAgentOptions(api_key=config.anthropic_api_key)
        if config.allowed_tools:
            opts.allowed_tools = config.allowed_tools.split(",")
        if user_options:
            for k, v in user_options.items():
                if hasattr(opts, k):
                    setattr(opts, k, v)
        return opts

    async def start(self, prompt: str, options: Optional[Dict[str, Any]] = None) -> None:
        """Start an agent session.

        Transitions from IDLE -> STARTING -> STREAMING and launches background
        tasks for message consumption, watchdog monitoring, and timeout.

        Args:
            prompt: The task prompt for the agent.
            options: Optional execution options.

        Raises:
            StateError: If the current state does not allow starting.
            Exception: If the SDK client fails to connect.
        """
        await self._state.transition(AgentState.starting, precondition=True)
        self._sdk_client = ClaudeSDKClient(options=self._build_options(options))
        try:
            await self._sdk_client.connect(prompt)
            await self._state.transition(AgentState.streaming, precondition=True)
            self._consumer_task = asyncio.create_task(self._consume_messages())
            self._watchdog_task = asyncio.create_task(self._watchdog_monitor())
            self._timeout_task = asyncio.create_task(self._timeout_watcher())
        except Exception as e:
            self._sdk_error = e
            await self._state.force_transition(AgentState.failed)
            await self._reporter.report_failed(str(e))
            raise

    async def interrupt(self) -> None:
        """Interrupt the current streaming session.

        Transitions from STREAMING -> INTERRUPTED and sends an interrupt
        signal to the SDK client.

        Raises:
            StateError: If not in STREAMING state.
        """
        await self._state.transition(AgentState.interrupted, precondition=True)
        if self._sdk_client:
            await self._sdk_client.interrupt()

    async def inject(self, prompt: str) -> None:
        """Inject a new prompt into an interrupted session.

        Transitions from INTERRUPTED -> STREAMING, sends the prompt to
        the SDK client, and restarts the message consumer.

        Args:
            prompt: The input to inject into the session.

        Raises:
            StateError: If not in INTERRUPTED state.
        """
        await self._state.transition(AgentState.streaming, precondition=True)
        if self._sdk_client:
            await self._sdk_client.query(prompt)
            if self._consumer_task and not self._consumer_task.done():
                self._consumer_task.cancel()
            self._consumer_task = asyncio.create_task(self._consume_messages())

    async def stop(self) -> None:
        """Stop all background tasks and disconnect the SDK client.

        Cancels consumer, watchdog, and timeout tasks, then disconnects
        the SDK client. Safe to call multiple times.
        """
        for task in (self._consumer_task, self._watchdog_task, self._timeout_task):
            if task and not task.done():
                task.cancel()
        if self._sdk_client:
            try:
                await self._sdk_client.disconnect()
            except Exception as e:
                logger.warning("Error disconnecting SDK client: %s", e)

    async def _consume_messages(self) -> None:
        """Consume messages from the SDK client and report events.

        Iterates over the SDK response stream, dispatching progress events
        for text blocks, tool call events for tool use blocks, and handling
        result messages for completion or failure.

        Handles CancelledError gracefully (task cancellation during shutdown).
        """
        try:
            async for message in self._sdk_client.receive_response():
                if isinstance(message, AssistantMessage):
                    for block in message.content:
                        if isinstance(block, TextBlock):
                            await self._reporter.report_progress(block.text, tokens=0)
                        elif isinstance(block, ToolUseBlock):
                            await self._reporter.report_tool_call(
                                block.name, getattr(block, "input", {})
                            )
                elif isinstance(message, ResultMessage):
                    if message.subtype == "success":
                        await self._state.transition(
                            AgentState.completed, precondition=True
                        )
                        await self._reporter.report_complete(
                            cost=getattr(message, "total_cost_usd", 0.0) or 0.0,
                            duration=0.0,
                        )
                    else:
                        await self._state.transition(
                            AgentState.failed, precondition=True
                        )
                        await self._reporter.report_failed(
                            getattr(message, "error", "unknown error")
                        )
                    return
        except asyncio.CancelledError:
            logger.info("Consumer task cancelled")
        except Exception as e:
            self._sdk_error = e
            await self._state.force_transition(AgentState.failed)
            await self._reporter.report_failed(str(e))

    async def _watchdog_monitor(self) -> None:
        """Monitor SDK process health periodically.

        Checks every 30 seconds that the SDK process is still alive.
        If the process is dead, forces a transition to FAILED state.
        """
        try:
            while True:
                await asyncio.sleep(30)
                if self._sdk_client and self._state.current in (
                    AgentState.streaming,
                    AgentState.interrupted,
                ):
                    try:
                        info = await self._sdk_client.get_server_info()
                        if info is None:
                            await self._state.force_transition(AgentState.failed)
                            await self._reporter.report_failed("sdk_process_dead")
                            return
                    except Exception:
                        await self._state.force_transition(AgentState.failed)
                        await self._reporter.report_failed("sdk_process_dead")
                        return
        except asyncio.CancelledError:
            pass

    async def _timeout_watcher(self) -> None:
        """Enforce maximum task execution timeout.

        If the task exceeds MAX_TIMEOUT seconds, interrupts the SDK client
        and forces a transition to FAILED state.
        """
        try:
            await asyncio.sleep(config.max_timeout)
            if self._state.current in (AgentState.streaming, AgentState.starting):
                logger.warning("Task timeout after %d seconds", config.max_timeout)
                if self._sdk_client:
                    await self._sdk_client.interrupt()
                await self._state.force_transition(AgentState.failed)
                await self._reporter.report_failed("task timeout")
        except asyncio.CancelledError:
            pass
