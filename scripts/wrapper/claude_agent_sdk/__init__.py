"""Stub for claude_agent_sdk package.

This is a placeholder module providing type stubs for the Claude Agent SDK.
In production, this would be replaced by the actual SDK package.
"""

from .client import ClaudeSDKClient
from .types import AssistantMessage, ClaudeAgentOptions, ResultMessage, TextBlock, ToolUseBlock

__all__ = [
    "ClaudeSDKClient",
    "ClaudeAgentOptions",
    "AssistantMessage",
    "ResultMessage",
    "TextBlock",
    "ToolUseBlock",
]
