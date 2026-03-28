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
    """Response reporting the current agent status."""

    state: AgentState = Field(..., description="Current agent state")
    session_id: Optional[str] = Field(None, description="Active session ID if any")
    error: Optional[str] = Field(None, description="Error message if in failed state")


class HealthResponse(BaseModel):
    """Health check response."""

    status: str = Field(..., description="Overall service health status")
    agent_state: AgentState = Field(..., description="Current agent state")
