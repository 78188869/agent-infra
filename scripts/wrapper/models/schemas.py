"""Request and response schemas for the wrapper API."""
from enum import Enum
from typing import Optional

from pydantic import BaseModel, Field


class AgentState(str, Enum):
    """Agent execution states following the state machine in agent/state.py."""

    idle = "idle"
    starting = "starting"
    streaming = "streaming"
    interrupted = "interrupted"
    completed = "completed"
    failed = "failed"


class StartRequest(BaseModel):
    """Request to start an agent session."""

    prompt: str = Field(..., description="The task prompt for the agent")
    options: Optional[dict] = Field(None, description="Optional execution options")


class InjectRequest(BaseModel):
    """Request to inject input into an interrupted session."""

    prompt: str = Field(..., description="The input to inject into the session")


class StatusResponse(BaseModel):
    """Response reporting the current agent status.

    Fields match Go's WrapperStatus struct for protocol compatibility.
    """

    status: str = Field(..., description="Current agent state as string")
    progress: int = Field(0, description="Task progress percentage")
    stage: str = Field("", description="Current execution stage")
    timestamp: float = Field(0.0, description="Unix timestamp of status")
    message: str = Field("", description="Status message")
    error: Optional[str] = Field(None, description="Error message if in failed state")


class HealthResponse(BaseModel):
    """Health check response.

    Fields match Go's WrapperHealth struct for protocol compatibility.
    """

    status: str = Field(..., description="Overall service health status")
    agent_state: AgentState = Field(..., description="Current agent state")
    task_id: str = Field("", description="Task ID assigned to this wrapper")
    uptime: float = Field(0.0, description="Seconds since wrapper started")
    version: str = Field("1.0.0", description="Wrapper version")
