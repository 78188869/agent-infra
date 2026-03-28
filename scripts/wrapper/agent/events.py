"""Event reporter with retry and local fallback for control plane communication.

The EventReporter sends task lifecycle events to the control plane via HTTP.
When the control plane is unreachable, events are persisted to a local JSONL
file so that no event data is lost.
"""
import json
import os
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, Optional

import httpx


# Default fallback directory relative to workspace
_FALLBACK_DIR = "/workspace/.agent-state"
_FALLBACK_FILE = "events.jsonl"


class EventReporter:
    """Reports agent lifecycle events to the control plane with retry and fallback.

    Events are retried up to 3 times with exponential backoff. If all retries
    fail, the event is written to a local JSONL file for later recovery.
    """

    MAX_RETRIES = 3
    TIMEOUT_SECONDS = 5

    def __init__(self, task_id: str, control_plane_url: str) -> None:
        """Initialize the EventReporter.

        Args:
            task_id: Unique identifier for the task.
            control_plane_url: Base URL of the control plane API.
        """
        self._task_id = task_id
        self._control_plane_url = control_plane_url.rstrip("/")
        self._url = f"{self._control_plane_url}/internal/tasks/{task_id}/events"
        self._fallback_path = Path(os.getenv("FALLBACK_DIR", _FALLBACK_DIR)) / _FALLBACK_FILE

    async def _post_event(self, event_type: str, payload: Dict[str, Any]) -> None:
        """Send an event to the control plane via HTTP POST.

        Args:
            event_type: Type of the event (progress, tool_call, complete, failed).
            payload: Event-specific data.

        Raises:
            httpx.HTTPError: If the request fails after timeout.
        """
        body = {
            "event_type": event_type,
            "payload": payload,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }
        async with httpx.AsyncClient(timeout=self.TIMEOUT_SECONDS) as client:
            response = await client.post(self._url, json=body)
            response.raise_for_status()

    async def _fallback_write(self, event_type: str, payload: Dict[str, Any]) -> None:
        """Write an event to the local fallback JSONL file.

        Ensures the fallback directory exists before writing.

        Args:
            event_type: Type of the event.
            payload: Event-specific data.
        """
        self._fallback_path.parent.mkdir(parents=True, exist_ok=True)
        entry = {
            "event_type": event_type,
            "payload": payload,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "task_id": self._task_id,
        }
        with open(self._fallback_path, "a") as f:
            f.write(json.dumps(entry) + "\n")

    async def report(self, event_type: str, payload: Dict[str, Any]) -> None:
        """Report an event with retry and local fallback.

        Retries up to MAX_RETRIES times with exponential backoff (2^attempt
        seconds). On final failure, writes the event to a local JSONL file.

        Args:
            event_type: Type of the event.
            payload: Event-specific data.
        """
        last_error: Optional[Exception] = None
        for attempt in range(self.MAX_RETRIES):
            try:
                await self._post_event(event_type, payload)
                return
            except Exception as exc:
                last_error = exc
                # Exponential backoff: 1s, 2s, 4s
                backoff = 2 ** attempt
                import asyncio

                await asyncio.sleep(backoff)

        # All retries exhausted, fall back to local file
        await self._fallback_write(event_type, payload)

    async def report_progress(self, text: str, tokens: int) -> None:
        """Report a progress event.

        Args:
            text: Progress message text.
            tokens: Number of tokens consumed so far.
        """
        await self.report("progress", {"text": text, "tokens": tokens})

    async def report_tool_call(self, tool: str, input_data: Any) -> None:
        """Report a tool call event.

        Args:
            tool: Name of the tool being called.
            input_data: Input data passed to the tool.
        """
        await self.report("tool_call", {"tool_name": tool, "input": input_data})

    async def report_complete(
        self, cost: float, duration: float, tokens: int = 0
    ) -> None:
        """Report task completion.

        Args:
            cost: Total cost in USD.
            duration: Execution duration in seconds.
            tokens: Total tokens consumed.
        """
        await self.report(
            "complete",
            {"cost_usd": cost, "duration_s": duration, "tokens": tokens},
        )

    async def report_failed(self, error: str) -> None:
        """Report task failure.

        Args:
            error: Error message describing the failure.
        """
        await self.report("failed", {"error": error})
