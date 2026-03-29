#!/bin/bash
set -e

STATE_DIR="/workspace/.agent-state"
mkdir -p "$STATE_DIR"

# Git clone
if [ -n "$GIT_REPO" ]; then
    echo "Cloning $GIT_REPO..."
    REPO_DIR="/workspace/repo"
    if [ -n "$GIT_BRANCH" ]; then
        git clone --depth 1 --branch "$GIT_BRANCH" "$GIT_REPO" "$REPO_DIR"
    else
        git clone --depth 1 "$GIT_REPO" "$REPO_DIR"
    fi
fi

# Generate CLAUDE.md
if [ -n "$CLAUDE_MD_CONTENT" ]; then
    TARGET_DIR="${REPO_DIR:-/workspace}"
    printf '%s' "$CLAUDE_MD_CONTENT" > "$TARGET_DIR/CLAUDE.md"
fi

# Start Python wrapper
cd /app
exec python main.py
