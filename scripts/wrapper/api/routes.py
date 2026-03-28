"""FastAPI route definitions for the agent wrapper service.

Uses factory pattern: create_app(agent_client) returns a configured FastAPI
application with all routes bound to the provided agent client instance.
"""
import logging
from typing import Any, Dict

from fastapi import FastAPI
from fastapi.responses import JSONResponse

from agent.state import StateError
from models.schemas import (
    HealthResponse,
    InjectRequest,
    StartRequest,
)

logger = logging.getLogger(__name__)


def create_app(agent_client: Any) -> FastAPI:
    """Create and configure a FastAPI application bound to the given agent client.

    Args:
        agent_client: An AgentClient instance with start/interrupt/inject/stop
                      methods and a `state` property.

    Returns:
        Configured FastAPI application.
    """
    app = FastAPI(title="Agent Wrapper", version="0.1.0")

    @app.get("/health", response_model=HealthResponse)
    async def health() -> HealthResponse:
        """Health check endpoint returning service and agent state."""
        return HealthResponse(status="ok", agent_state=agent_client.state)

    @app.get("/status")
    async def get_status() -> Dict[str, Any]:
        """Return the current agent status including state and error info."""
        return agent_client.get_status()

    @app.post("/start")
    async def start_task(req: StartRequest) -> JSONResponse:
        """Start an agent session with the given prompt and optional options.

        Returns 200 on success, 409 if the agent is in an invalid state.
        """
        try:
            await agent_client.start(req.prompt, req.options)
            return JSONResponse(content={"status": "started"}, status_code=200)
        except (StateError, Exception) as exc:
            logger.warning("Start failed: %s", exc)
            return JSONResponse(
                content={"error": str(exc)}, status_code=409
            )

    @app.post("/interrupt")
    async def interrupt() -> JSONResponse:
        """Interrupt the current streaming session.

        Returns 200 on success, 409 if the agent is not in a streaming state.
        """
        try:
            await agent_client.interrupt()
            return JSONResponse(content={"status": "interrupted"}, status_code=200)
        except (StateError, Exception) as exc:
            logger.warning("Interrupt failed: %s", exc)
            return JSONResponse(
                content={"error": str(exc)}, status_code=409
            )

    @app.post("/inject")
    async def inject(req: InjectRequest) -> JSONResponse:
        """Inject a prompt into an interrupted session.

        Returns 200 on success, 409 if the agent is not in an interrupted state.
        """
        try:
            await agent_client.inject(req.prompt)
            return JSONResponse(content={"status": "injected"}, status_code=200)
        except (StateError, Exception) as exc:
            logger.warning("Inject failed: %s", exc)
            return JSONResponse(
                content={"error": str(exc)}, status_code=409
            )

    @app.post("/stop")
    async def stop() -> JSONResponse:
        """Stop all background tasks and disconnect the SDK client.

        Returns 200 on success.
        """
        await agent_client.stop()
        return JSONResponse(content={"status": "stopped"}, status_code=200)

    return app
