"""Type stubs for the Claude Agent SDK."""

from typing import Any, Dict, List, Optional


class ClaudeAgentOptions:
    """Options for configuring a Claude Agent session."""

    def __init__(self, api_key: str = "", **kwargs: Any) -> None:
        self.api_key = api_key
        self.allowed_tools: Optional[List[str]] = None
        for k, v in kwargs.items():
            setattr(self, k, v)


class TextBlock:
    """A text content block in an assistant message."""

    def __init__(self, text: str = "") -> None:
        self.text = text


class ToolUseBlock:
    """A tool use block in an assistant message."""

    def __init__(self, name: str = "", input: Optional[Dict[str, Any]] = None) -> None:
        self.name = name
        self.input = input or {}


class AssistantMessage:
    """An assistant message containing content blocks."""

    def __init__(self, content: Optional[List[Any]] = None) -> None:
        self.content = content or []


class ResultMessage:
    """A result message indicating completion or failure."""

    def __init__(
        self,
        subtype: str = "success",
        total_cost_usd: float = 0.0,
        error: Optional[str] = None,
    ) -> None:
        self.subtype = subtype
        self.total_cost_usd = total_cost_usd
        self.error = error
