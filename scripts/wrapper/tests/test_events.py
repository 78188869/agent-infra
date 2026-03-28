"""Tests for EventReporter with retry and fallback behavior."""
import json
import os
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from agent.events import EventReporter


@pytest.fixture
def reporter():
    """Create an EventReporter instance for testing."""
    return EventReporter(task_id="task-123", control_plane_url="http://localhost:8080")


@pytest.mark.asyncio
async def test_report_progress(reporter):
    """report_progress should send correct event_type and payload."""
    with patch.object(reporter, "_post_event", new_callable=AsyncMock) as mock_post:
        await reporter.report_progress("working...", tokens=42)

    mock_post.assert_called_once()
    call_args = mock_post.call_args
    assert call_args[0][0] == "progress"
    assert call_args[0][1] == {"text": "working...", "tokens": 42}


@pytest.mark.asyncio
async def test_report_tool_call(reporter):
    """report_tool_call should send tool_name and input in payload."""
    with patch.object(reporter, "_post_event", new_callable=AsyncMock) as mock_post:
        await reporter.report_tool_call("bash", {"command": "ls -la"})

    mock_post.assert_called_once()
    call_args = mock_post.call_args
    assert call_args[0][0] == "tool_call"
    assert call_args[0][1]["tool_name"] == "bash"
    assert call_args[0][1]["input"] == {"command": "ls -la"}


@pytest.mark.asyncio
async def test_report_complete(reporter):
    """report_complete should send cost_usd, duration_s, and tokens."""
    with patch.object(reporter, "_post_event", new_callable=AsyncMock) as mock_post:
        await reporter.report_complete(cost=0.05, duration=30.5, tokens=1000)

    mock_post.assert_called_once()
    call_args = mock_post.call_args
    assert call_args[0][0] == "complete"
    assert call_args[0][1] == {"cost_usd": 0.05, "duration_s": 30.5, "tokens": 1000}


@pytest.mark.asyncio
async def test_report_complete_default_tokens(reporter):
    """report_complete should default tokens to 0."""
    with patch.object(reporter, "_post_event", new_callable=AsyncMock) as mock_post:
        await reporter.report_complete(cost=0.05, duration=30.5)

    call_args = mock_post.call_args
    assert call_args[0][1]["tokens"] == 0


@pytest.mark.asyncio
async def test_report_failed(reporter):
    """report_failed should send the error field."""
    with patch.object(reporter, "_post_event", new_callable=AsyncMock) as mock_post:
        await reporter.report_failed("something went wrong")

    mock_post.assert_called_once()
    call_args = mock_post.call_args
    assert call_args[0][0] == "failed"
    assert call_args[0][1] == {"error": "something went wrong"}


@pytest.mark.asyncio
async def test_retry_on_failure_then_success(reporter):
    """report should retry up to 3 times: fail twice then succeed."""
    call_count = 0

    async def mock_post(event_type, payload):
        nonlocal call_count
        call_count += 1
        if call_count < 3:
            raise ConnectionError("connection refused")

    with patch.object(reporter, "_post_event", side_effect=mock_post):
        await reporter.report("progress", {"text": "retry test", "tokens": 0})

    assert call_count == 3


@pytest.mark.asyncio
async def test_fallback_after_all_retries_exhausted(reporter, tmp_path):
    """report should write to local file when all retries fail."""
    fallback_file = tmp_path / "events.jsonl"
    reporter._fallback_path = fallback_file

    with patch.object(
        reporter, "_post_event", new_callable=AsyncMock, side_effect=ConnectionError("down")
    ):
        await reporter.report("progress", {"text": "fallback test", "tokens": 0})

    # Verify the fallback file was written
    assert fallback_file.exists()
    lines = fallback_file.read_text().strip().split("\n")
    assert len(lines) == 1
    entry = json.loads(lines[0])
    assert entry["event_type"] == "progress"
    assert entry["payload"]["text"] == "fallback test"
    assert entry["task_id"] == "task-123"
    assert "timestamp" in entry


@pytest.mark.asyncio
async def test_report_url_format(reporter):
    """EventReporter should construct the correct URL."""
    assert reporter._url == "http://localhost:8080/internal/tasks/task-123/events"


@pytest.mark.asyncio
async def test_report_url_strips_trailing_slash():
    """EventReporter should handle trailing slashes in control_plane_url."""
    r = EventReporter(task_id="t-1", control_plane_url="http://host:8080/")
    assert r._url == "http://host:8080/internal/tasks/t-1/events"


@pytest.mark.asyncio
async def test_close_cleans_up_client(reporter):
    """close should close the internal httpx client."""
    mock_client = AsyncMock()
    mock_client.is_closed = False
    reporter._client = mock_client

    await reporter.close()

    mock_client.aclose.assert_called_once()
    assert reporter._client is None


@pytest.mark.asyncio
async def test_close_idempotent(reporter):
    """close should be safe when client is already None or closed."""
    # No client set
    await reporter.close()  # Should not raise

    # Closed client
    mock_client = AsyncMock()
    mock_client.is_closed = True
    reporter._client = mock_client

    await reporter.close()  # Should not call aclose
    mock_client.aclose.assert_not_called()


@pytest.mark.asyncio
async def test_get_client_creates_client(reporter):
    """_get_client should create a new client if none exists."""
    client = await reporter._get_client()
    assert client is not None
    assert not client.is_closed
    # Clean up
    await reporter.close()


@pytest.mark.asyncio
async def test_get_client_reuses_existing(reporter):
    """_get_client should reuse existing client if not closed."""
    client1 = await reporter._get_client()
    client2 = await reporter._get_client()
    assert client1 is client2
    # Clean up
    await reporter.close()
