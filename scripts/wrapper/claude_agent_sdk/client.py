"""Stub ClaudeSDKClient for development and testing."""

from typing import Any, AsyncIterator, Optional

from .types import ClaudeAgentOptions


class ClaudeSDKClient:
    """Client for interacting with the Claude Agent SDK.

    This is a stub implementation. In production, this would be provided
    by the actual claude-agent-sdk package.
    """

    def __init__(self, options: Optional[ClaudeAgentOptions] = None) -> None:
        self._options = options
        self._connected = False

    async def connect(self, prompt: str) -> None:
        """Connect to the SDK and start a session."""
        self._connected = True

    async def disconnect(self) -> None:
        """Disconnect from the SDK session."""
        self._connected = False

    async def interrupt(self) -> None:
        """Send an interrupt signal to the running session."""
        pass

    async def query(self, prompt: str) -> None:
        """Send a new query to the session."""
        pass

    async def receive_response(self) -> AsyncIterator[Any]:
        """Receive messages from the SDK as an async iterator."""
        yield None  # Stub: no messages

    async def get_server_info(self) -> Optional[dict]:
        """Get server information for health checks."""
        if self._connected:
            return {"status": "ok"}
        return None
