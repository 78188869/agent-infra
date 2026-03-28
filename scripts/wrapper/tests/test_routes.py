"""Tests for FastAPI routes in api/routes.py.

Uses FastAPI TestClient with a mocked AgentClient to verify route
responses without depending on the real SDK or network.
"""
import pytest
from unittest.mock import AsyncMock, MagicMock

from fastapi.testclient import TestClient

from models.schemas import AgentState


@pytest.fixture
def mock_agent():
    """Create a mocked AgentClient with async method stubs."""
    agent = MagicMock()
    agent.state = AgentState.idle
    agent.get_status.return_value = {"state": AgentState.idle, "error": None}
    agent.start = AsyncMock()
    agent.interrupt = AsyncMock()
    agent.inject = AsyncMock()
    agent.stop = AsyncMock()
    return agent


@pytest.fixture
def client(mock_agent):
    """Create a TestClient with routes bound to the mocked agent."""
    from api.routes import create_app

    app = create_app(mock_agent)
    return TestClient(app)


def test_health(client):
    """GET /health returns 200 with status ok and agent state."""
    resp = client.get("/health")
    assert resp.status_code == 200
    body = resp.json()
    assert body["status"] == "ok"
    assert body["agent_state"] == AgentState.idle


def test_get_status(client):
    """GET /status returns 200 with agent state and no error."""
    resp = client.get("/status")
    assert resp.status_code == 200
    body = resp.json()
    assert body["state"] == AgentState.idle
    assert body["error"] is None


def test_start_task(client, mock_agent):
    """POST /start returns 200 and calls agent.start with prompt."""
    resp = client.post("/start", json={"prompt": "fix the bug"})
    assert resp.status_code == 200
    assert resp.json()["status"] == "started"
    mock_agent.start.assert_awaited_once_with("fix the bug", None)


def test_start_task_with_options(client, mock_agent):
    """POST /start passes optional execution options to agent.start."""
    opts = {"allowed_tools": ["Read", "Write"]}
    resp = client.post(
        "/start", json={"prompt": "do stuff", "options": opts}
    )
    assert resp.status_code == 200
    mock_agent.start.assert_awaited_once_with("do stuff", opts)


def test_start_conflict(client, mock_agent):
    """POST /start returns 409 when agent is in an invalid state."""
    mock_agent.start.side_effect = Exception("invalid state")
    resp = client.post("/start", json={"prompt": "fix the bug"})
    assert resp.status_code == 409
    assert "invalid state" in resp.json()["error"]


def test_interrupt(client, mock_agent):
    """POST /interrupt returns 200 and calls agent.interrupt."""
    resp = client.post("/interrupt")
    assert resp.status_code == 200
    assert resp.json()["status"] == "interrupted"
    mock_agent.interrupt.assert_awaited_once()


def test_interrupt_conflict(client, mock_agent):
    """POST /interrupt returns 409 when not in streaming state."""
    mock_agent.interrupt.side_effect = Exception("invalid state")
    resp = client.post("/interrupt")
    assert resp.status_code == 409
    assert "invalid state" in resp.json()["error"]


def test_inject(client, mock_agent):
    """POST /inject returns 200 and calls agent.inject with prompt."""
    resp = client.post("/inject", json={"prompt": "new instruction"})
    assert resp.status_code == 200
    assert resp.json()["status"] == "injected"
    mock_agent.inject.assert_awaited_once_with("new instruction")


def test_inject_conflict(client, mock_agent):
    """POST /inject returns 409 when not in interrupted state."""
    mock_agent.inject.side_effect = Exception("invalid state")
    resp = client.post("/inject", json={"prompt": "new instruction"})
    assert resp.status_code == 409
    assert "invalid state" in resp.json()["error"]


def test_stop(client, mock_agent):
    """POST /stop returns 200 and calls agent.stop."""
    resp = client.post("/stop")
    assert resp.status_code == 200
    assert resp.json()["status"] == "stopped"
    mock_agent.stop.assert_awaited_once()
