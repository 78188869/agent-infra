"""Tests for AgentClient with mocked ClaudeSDKClient."""
import asyncio

import pytest
from unittest.mock import AsyncMock, MagicMock, patch, PropertyMock

from agent.client import AgentClient
from models.schemas import AgentState


@pytest.fixture
def client():
    """Create an AgentClient with mocked ClaudeSDKClient."""
    with patch("agent.client.ClaudeSDKClient"):
        c = AgentClient(task_id="test-123", control_plane_url="http://localhost:8080")
        yield c


@pytest.mark.asyncio
async def test_initial_state_is_idle(client):
    """AgentClient should start in the idle state."""
    assert client.state == AgentState.idle


@pytest.mark.asyncio
async def test_get_status_returns_state(client):
    """get_status should return current state and no error initially."""
    status = client.get_status()
    assert status["state"] == AgentState.idle
    assert status["error"] is None


@pytest.mark.asyncio
async def test_interrupt_requires_streaming(client):
    """interrupt should raise StateError when not in streaming state."""
    from agent.state import StateError

    with pytest.raises(StateError):
        await client.interrupt()


@pytest.mark.asyncio
async def test_inject_requires_interrupted(client):
    """inject should raise StateError when not in interrupted state."""
    from agent.state import StateError

    with pytest.raises(StateError):
        await client.inject("new instruction")


@pytest.mark.asyncio
async def test_stop_cleans_up(client):
    """stop should cancel tasks and disconnect the SDK client."""
    client._sdk_client = MagicMock()
    client._sdk_client.disconnect = AsyncMock()

    # Create dummy tasks that are not yet done
    consumer = asyncio.create_task(asyncio.sleep(100))
    watchdog = asyncio.create_task(asyncio.sleep(100))
    timeout = asyncio.create_task(asyncio.sleep(100))
    client._consumer_task = consumer
    client._watchdog_task = watchdog
    client._timeout_task = timeout

    await client.stop()

    client._sdk_client.disconnect.assert_called_once()
    # After cancel(), tasks need a yield to propagate the cancellation
    await asyncio.sleep(0)
    assert consumer.cancelled() or consumer.done()
    assert watchdog.cancelled() or watchdog.done()
    assert timeout.cancelled() or timeout.done()


@pytest.mark.asyncio
async def test_stop_without_sdk_client(client):
    """stop should be safe when no SDK client is initialized."""
    client._sdk_client = None
    await client.stop()  # Should not raise


@pytest.mark.asyncio
async def test_stop_handles_disconnect_error(client):
    """stop should log warning when disconnect raises an error."""
    client._sdk_client = MagicMock()
    client._sdk_client.disconnect = AsyncMock(side_effect=RuntimeError("disconnect failed"))

    await client.stop()  # Should not raise
    client._sdk_client.disconnect.assert_called_once()


@pytest.mark.asyncio
async def test_start_success(client):
    """start should transition IDLE -> STARTING -> STREAMING and launch tasks."""
    mock_sdk = MagicMock()
    mock_sdk.connect = AsyncMock()

    with patch("agent.client.ClaudeSDKClient", return_value=mock_sdk):
        with patch("agent.client.config") as mock_config:
            mock_config.anthropic_api_key = "test-key"
            mock_config.allowed_tools = None
            await client.start("do something")

    assert client.state == AgentState.streaming
    mock_sdk.connect.assert_called_once_with("do something")
    assert client._consumer_task is not None
    assert client._watchdog_task is not None
    assert client._timeout_task is not None

    # Clean up background tasks
    await client.stop()


@pytest.mark.asyncio
async def test_start_with_options(client):
    """start should pass user options to _build_options."""
    mock_sdk = MagicMock()
    mock_sdk.connect = AsyncMock()

    with patch("agent.client.ClaudeSDKClient", return_value=mock_sdk):
        with patch("agent.client.config") as mock_config:
            mock_config.anthropic_api_key = "test-key"
            mock_config.allowed_tools = None
            await client.start("do something", options={"model": "claude-3"})

    assert client.state == AgentState.streaming
    await client.stop()


@pytest.mark.asyncio
async def test_start_connect_failure_transitions_to_failed(client):
    """start should transition to FAILED when connect raises."""
    mock_sdk = MagicMock()
    mock_sdk.connect = AsyncMock(side_effect=ConnectionError("refused"))

    with patch("agent.client.ClaudeSDKClient", return_value=mock_sdk):
        with patch("agent.client.config") as mock_config:
            mock_config.anthropic_api_key = "test-key"
            mock_config.allowed_tools = None
            # Mock the reporter to avoid writing to /workspace
            with patch.object(client._reporter, "report_failed", new_callable=AsyncMock):
                with pytest.raises(ConnectionError):
                    await client.start("do something")

    assert client.state == AgentState.failed
    assert client._sdk_error is not None
    assert "refused" in str(client._sdk_error)


@pytest.mark.asyncio
async def test_interrupt_calls_sdk_interrupt(client):
    """interrupt should call SDK interrupt when in streaming state."""
    mock_sdk = MagicMock()
    mock_sdk.interrupt = AsyncMock()
    client._sdk_client = mock_sdk

    # Manually set state to streaming
    await client._state.transition(AgentState.starting, precondition=True)
    await client._state.transition(AgentState.streaming, precondition=True)

    await client.interrupt()

    assert client.state == AgentState.interrupted
    mock_sdk.interrupt.assert_called_once()


@pytest.mark.asyncio
async def test_inject_calls_sdk_query(client):
    """inject should call SDK query and restart consumer task."""
    mock_sdk = MagicMock()
    mock_sdk.query = AsyncMock()
    client._sdk_client = mock_sdk

    # Manually set state to interrupted
    await client._state.transition(AgentState.starting, precondition=True)
    await client._state.transition(AgentState.streaming, precondition=True)
    await client._state.transition(AgentState.interrupted, precondition=True)

    await client.inject("new instruction")

    assert client.state == AgentState.streaming
    mock_sdk.query.assert_called_once_with("new instruction")
    assert client._consumer_task is not None

    await client.stop()


@pytest.mark.asyncio
async def test_get_status_with_error(client):
    """get_status should include error message when SDK has errored."""
    client._sdk_error = RuntimeError("test error")
    status = client.get_status()
    assert status["state"] == AgentState.idle
    assert "test error" in status["error"]
