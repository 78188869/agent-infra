"""Configuration management for the wrapper service.

Loads settings from environment variables with sensible defaults.
"""
import os
from typing import Optional


class Config:
    """Configuration loaded from environment variables."""

    # Required: Task identification
    task_id: str = os.getenv("TASK_ID", "")
    control_plane_url: str = os.getenv("CONTROL_PLANE_URL", "")
    anthropic_api_key: str = os.getenv("ANTHROPIC_API_KEY", "")
    task_prompt: str = os.getenv("TASK_PROMPT", "")

    # Optional: Execution settings
    max_timeout: int = int(os.getenv("MAX_TIMEOUT", "3600"))
    workspace_dir: str = os.getenv("WORKSPACE_DIR", "/workspace")

    # Optional: Git and context
    git_repo: Optional[str] = os.getenv("GIT_REPO")
    claude_md_content: Optional[str] = os.getenv("CLAUDE_MD_CONTENT")
    allowed_tools: Optional[str] = os.getenv("ALLOWED_TOOLS")

    # Optional: Server binding
    port: int = int(os.getenv("PORT", "9090"))

    def validate(self) -> None:
        """Validate that required configuration is present.

        Raises:
            ValueError: If required configuration is missing.
        """
        missing = []
        if not self.task_id:
            missing.append("TASK_ID")
        if not self.control_plane_url:
            missing.append("CONTROL_PLANE_URL")
        if not self.anthropic_api_key:
            missing.append("ANTHROPIC_API_KEY")
        if not self.task_prompt:
            missing.append("TASK_PROMPT")

        if missing:
            raise ValueError(f"Missing required environment variables: {', '.join(missing)}")


# Global config instance
config = Config()
