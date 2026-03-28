"""Tests for the agent state machine."""
import asyncio

import pytest

from agent.state import StateError, StateMachine


def test_initial_state_is_idle():
    """State machine should start in the idle state."""
    sm = StateMachine()
    assert sm.current == "idle"


@pytest.mark.asyncio
async def test_valid_transition_idle_to_starting():
    """Transition from idle to starting should succeed."""
    sm = StateMachine()
    await sm.transition("starting", precondition=True)
    assert sm.current == "starting"


@pytest.mark.asyncio
async def test_valid_transition_starting_to_streaming():
    """Transition from starting to streaming should succeed."""
    sm = StateMachine(initial_state="starting")
    await sm.transition("streaming", precondition=True)
    assert sm.current == "streaming"


@pytest.mark.asyncio
async def test_valid_transition_streaming_to_interrupted():
    """Transition from streaming to interrupted should succeed."""
    sm = StateMachine(initial_state="streaming")
    await sm.transition("interrupted", precondition=True)
    assert sm.current == "interrupted"


@pytest.mark.asyncio
async def test_valid_transition_interrupted_to_streaming():
    """Transition from interrupted back to streaming (inject flow) should succeed."""
    sm = StateMachine(initial_state="interrupted")
    await sm.transition("streaming", precondition=True)
    assert sm.current == "streaming"


@pytest.mark.asyncio
async def test_valid_transition_streaming_to_completed():
    """Transition from streaming to completed should succeed."""
    sm = StateMachine(initial_state="streaming")
    await sm.transition("completed", precondition=True)
    assert sm.current == "completed"


@pytest.mark.asyncio
async def test_valid_transition_streaming_to_failed():
    """Transition from streaming to failed should succeed."""
    sm = StateMachine(initial_state="streaming")
    await sm.transition("failed", precondition=True)
    assert sm.current == "failed"


@pytest.mark.asyncio
async def test_invalid_transition_raises():
    """Transitioning to a state not in the allowed set should raise StateError."""
    sm = StateMachine()
    with pytest.raises(StateError, match="Invalid state transition"):
        await sm.transition("streaming", precondition=True)
    assert sm.current == "idle"  # State should remain unchanged


@pytest.mark.asyncio
async def test_concurrent_transitions_only_one_succeeds():
    """When multiple coroutines race to transition from the same state,
    only the first one should succeed; the second should fail because
    the state has already moved."""
    sm = StateMachine(initial_state="streaming")

    results = []

    async def try_transition(target: str):
        try:
            await sm.transition(target, precondition=True)
            results.append(target)
        except StateError:
            results.append("error")

    # Race: both coroutines try to transition from streaming.
    # "completed" is a terminal state, so the second one will fail
    # because there are no valid transitions from "completed".
    await asyncio.gather(
        try_transition("completed"),
        try_transition("completed"),
    )

    # Exactly one should succeed, the other should error
    assert results.count("completed") == 1
    assert "error" in results
    assert sm.current == "completed"
