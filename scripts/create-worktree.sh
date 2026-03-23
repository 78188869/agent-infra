#!/bin/bash
set -e

ISSUE_NUMBER=$1
WORKTREE_DIR="../worktrees/$ISSUE_NUMBER"

if [ -z "$ISSUE_NUMBER" ]; then
    echo "Usage: $0 <issue-number>"
    echo "Example: $0 123"
    exit 1
fi

# Fetch latest main
git fetch origin main

# Create worktree
echo "Creating worktree for issue #$ISSUE_NUMBER..."
git worktree add "$WORKTREE_DIR" -b "issue-$ISSUE_NUMBER" origin/main

echo ""
echo "✅ Worktree created at: $WORKTREE_DIR"
echo "   Branch: issue-$ISSUE_NUMBER"
echo ""
echo "To start working:"
echo "  cd $WORKTREE_DIR"
