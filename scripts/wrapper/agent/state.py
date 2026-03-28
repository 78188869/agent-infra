"""Thread-safe state machine for agent lifecycle management.

The state machine enforces valid transitions and prevents race conditions
during concurrent state changes.
"""
import asyncio
from typing import Dict, List, Set, Optional


class StateError(Exception):
    """Exception raised for invalid state transitions."""

    pass


class StateMachine:
    """Async-safe state machine for agent lifecycle.

    The state machine validates all transitions and provides thread-safe
    state mutation using asyncio locks.
    """

    # Valid state transitions: from_state -> [to_states]
    TRANSITIONS: Dict[str, Set[str]] = {
        "idle": {"starting"},
        "starting": {"streaming", "failed"},
        "streaming": {"interrupted", "completed", "failed"},
        "interrupted": {"streaming", "completed", "failed"},
        "completed": set(),  # Terminal state
        "failed": set(),  # Terminal state
    }

    def __init__(self, initial_state: str = "idle") -> None:
        """Initialize the state machine.

        Args:
            initial_state: Starting state, defaults to "idle".
        """
        self._state: str = initial_state
        self._lock = asyncio.Lock()

    @property
    def current(self) -> str:
        """Get the current state.

        Returns:
            Current state string.
        """
        return self._state

    async def transition(
        self, new_state: str, precondition: Optional[bool] = None
    ) -> None:
        """Transition to a new state with validation.

        Validates both the precondition (if provided) and that the transition
        is allowed according to the state machine rules.

        Args:
            new_state: Target state to transition to.
            precondition: Optional boolean condition that must be True.
                         If False or None, the transition is rejected.

        Raises:
            StateError: If precondition fails or transition is invalid.
        """
        async with self._lock:
            # Check precondition if provided
            if precondition is not None and not precondition:
                raise StateError(
                    f"Precondition failed for transition {self._state} -> {new_state}"
                )

            # Validate transition is allowed
            allowed = self.TRANSITIONS.get(self._state, set())
            if new_state not in allowed:
                raise StateError(
                    f"Invalid state transition: {self._state} -> {new_state}. "
                    f"Allowed transitions from {self._state}: {allowed}"
                )

            # Perform transition
            old_state = self._state
            self._state = new_state

    async def force_transition(self, new_state: str) -> None:
        """Force transition to a new state without validation.

        Used for error recovery and cleanup scenarios where the normal
        transition rules should be bypassed (e.g., completed -> failed).

        Args:
            new_state: Target state to transition to.
        """
        async with self._lock:
            self._state = new_state
