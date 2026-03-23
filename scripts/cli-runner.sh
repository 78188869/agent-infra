#!/bin/bash
# CLI Runner Script
# Runs inside the cli-runner container to execute Claude Code CLI tasks

set -e

# Configuration from environment
TASK_ID="${TASK_ID:-unknown}"
GIT_REPO_URL="${GIT_REPO_URL:-}"
GIT_BRANCH="${GIT_BRANCH:-main}"
GIT_TOKEN="${GIT_TOKEN:-}"
CLAUDE_MD_CONTENT="${CLAUDE_MD_CONTENT:-}"
TASK_PROMPT="${TASK_PROMPT:-}"
MAX_TOKENS="${MAX_TOKENS:-128000}"
ALLOWED_TOOLS="${ALLOWED_TOOLS:-}"
WORKSPACE="/workspace"

# State directory for communication with wrapper
STATE_DIR="${WORKSPACE}/.agent-state"
EVENTS_FILE="${STATE_DIR}/events.jsonl"
STATUS_FILE="${STATE_DIR}/status.json"
INJECT_FILE="${STATE_DIR}/inject.json"

log() {
    echo "[$(date -Iseconds)] [cli-runner] $*"
}

# Signal handling - graceful shutdown
cleanup() {
    log "Received termination signal, saving state..."
    echo "{\"status\": \"interrupted\", \"exit_code\": 130, \"timestamp\": $(date +%s)}" > "${STATUS_FILE}"
    if [ -n "$CLI_PID" ] && kill -0 "$CLI_PID" 2>/dev/null; then
        kill -TERM "$CLI_PID" 2>/dev/null || true
        wait "$CLI_PID" 2>/dev/null || true
    fi
    exit 130
}
trap cleanup SIGTERM SIGINT

# Initialize state directory
mkdir -p "${STATE_DIR}"
echo "{\"status\": \"initializing\", \"timestamp\": $(date +%s)}" > "${STATUS_FILE}"

# 1. Clone Git repository
log "Cloning repository: ${GIT_REPO_URL}"
if [ -n "$GIT_TOKEN" ]; then
    # Inject token into URL if provided
    REPO_URL_WITH_TOKEN=$(echo "$GIT_REPO_URL" | sed "s|https://|https://${GIT_TOKEN}@|")
else
    REPO_URL_WITH_TOKEN="$GIT_REPO_URL"
fi

git clone --depth 1 --branch "${GIT_BRANCH}" "${REPO_URL_WITH_TOKEN}" "${WORKSPACE}/src"
cd "${WORKSPACE}/src"
log "Repository cloned successfully"

# 2. Generate CLAUDE.md
log "Generating CLAUDE.md"
cat > "${WORKSPACE}/CLAUDE.md" <<EOF
${CLAUDE_MD_CONTENT}
EOF

# 3. Generate .mcp.json if needed (future capability)
# TODO: Generate MCP configuration based on template capabilities

# Update status
echo "{\"status\": \"running\", \"timestamp\": $(date +%s)}" > "${STATUS_FILE}"

# 4. Start CLI and capture output
log "Starting Claude Code CLI"
claude -p "${TASK_PROMPT}" \
       --max-tokens "${MAX_TOKENS}" \
       ${ALLOWED_TOOLS:+--allowedTools "${ALLOWED_TOOLS}"} \
       --output-format stream-json 2>&1 | while read -r line; do
    echo "$line" >> "${EVENTS_FILE}"
    # Extract key status updates
    if echo "$line" | grep -q '"type"[[:space:]]*:[[:space:]]*"status"'; then
        echo "$line" > "${STATUS_FILE}"
    fi
done &

CLI_PID=$!
log "CLI started with PID: ${CLI_PID}"

# Wait for CLI to complete
wait "$CLI_PID"
EXIT_CODE=$?

# 5. Write final status
log "CLI completed with exit code: ${EXIT_CODE}"
if [ $EXIT_CODE -eq 0 ]; then
    FINAL_STATUS="completed"
else
    FINAL_STATUS="failed"
fi

echo "{\"status\": \"${FINAL_STATUS}\", \"exit_code\": ${EXIT_CODE}, \"timestamp\": $(date +%s)}" > "${STATUS_FILE}"

exit $EXIT_CODE
