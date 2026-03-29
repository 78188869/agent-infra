"""Entry point for the agent wrapper service.

Creates the FastAPI application with graceful shutdown handling and
optional auto-start from TASK_PROMPT environment variable.
"""
import asyncio
import logging
import signal

import uvicorn

from agent.client import AgentClient
from api.routes import create_app
from config import config
from models.schemas import AgentState

logger = logging.getLogger(__name__)


class GracefulShutdown:
    """Handles SIGTERM by interrupting, stopping, and reporting agent state."""

    def __init__(self, agent: AgentClient) -> None:
        self._agent = agent
        self._shutdown_event = asyncio.Event()

    def register(self) -> None:
        """Register the SIGTERM handler on the current event loop."""
        loop = asyncio.get_event_loop()
        loop.add_signal_handler(
            signal.SIGTERM,
            lambda: asyncio.ensure_future(self._handle_sigterm()),
        )

    async def _handle_sigterm(self) -> None:
        """Interrupt active session, stop agent, and signal shutdown."""
        logger.info("Received SIGTERM, shutting down gracefully...")
        try:
            if self._agent.state in (AgentState.streaming, AgentState.starting):
                await self._agent.interrupt()
            await self._agent.stop()
        except Exception as exc:
            logger.error("Error during shutdown: %s", exc)
        self._shutdown_event.set()


def main() -> None:
    """Create agent, app, and run uvicorn server."""
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )

    agent = AgentClient(
        task_id=config.task_id,
        control_plane_url=config.control_plane_url,
    )
    app = create_app(agent)
    shutdown = GracefulShutdown(agent)

    @app.on_event("startup")
    async def on_startup() -> None:
        shutdown.register()
        if config.task_prompt:
            logger.info("Auto-starting task")
            await agent.start(config.task_prompt)

    uvicorn.run(app, host="0.0.0.0", port=config.port)


if __name__ == "__main__":
    main()
