# Docker Container Execution Engine Design

> **Issue**: #35
> **Date**: 2026-03-28
> **Status**: Draft

## 1. Overview

Provide a Docker-based container execution engine as an alternative to K8s, enabling local development to run the full task lifecycle (create, execute, monitor, intervene, complete) in containerized isolation with production-parity behavior.

## 2. Architecture

```
                    ┌─────────────────────────┐
                    │      TaskExecutor       │
                    │  (unchanged core logic) │
                    └───────────┬─────────────┘
                                │ ContainerRuntime interface
                    ┌───────────┴───────────┐
                    │                       │
            ┌───────▼───────┐       ┌───────▼───────┐
            │  K8sRuntime   │       │ DockerRuntime │  ← NEW
            │  (existing)   │       │  (new)        │
            └───────────────┘       └───────┬───────┘
                                            │
                                    ┌───────▼───────┐
                                    │ ComposeManager│  ← NEW
                                    │ generate/manage│
                                    │ compose YAML  │
                                    └───────────────┘
```

**Core principle**: Only add a `DockerRuntime` implementation. `TaskExecutor` and all upstream code remain unchanged.

## 3. ComposeManager

Generates per-task `docker-compose.yml` files and manages lifecycle via Docker CLI.

### 3.1 Struct

```go
type ComposeManager struct {
    workspaceDir string           // host workspace root (e.g. ./workspace)
    composeDir   string           // temp dir for compose files
    templates    ComposeTemplates // image names and config
}

type ComposeTemplates struct {
    CLIRunnerImage string
    WrapperImage   string
    WrapperPort    int
}
```

### 3.2 Compose YAML template

Per-task file at `{compose-dir}/task-{task-id}/docker-compose.yml`:

```yaml
services:
  cli-runner:
    image: ${CLIRUNNER_IMAGE}
    volumes:
      - ${WORKSPACE_DIR}/${TASK_ID}:/workspace
    environment:
      - TASK_ID=${TASK_ID}
      - GIT_REPO_URL=${GIT_REPO_URL}
      - TASK_PROMPT=${TASK_PROMPT}
      - AGENT_STATE_DIR=/workspace/.agent-state

  wrapper:
    image: ${WRAPPER_IMAGE}
    ports:
      - "9090"                    # dynamic host port
    volumes:
      - ${WORKSPACE_DIR}/${TASK_ID}:/workspace
    environment:
      - TASK_ID=${TASK_ID}
      - SHARED_STATE_DIR=/workspace/.agent-state
    depends_on:
      - cli-runner
```

Key points:
- Two containers per task (cli-runner + wrapper), mirroring K8s Pod sidecar pattern
- Shared volume via host bind mount
- Wrapper port mapped dynamically
- No log-agent container (file-based logging for local dev)

### 3.3 Methods

| Method | Implementation | Description |
|--------|---------------|-------------|
| `GenerateConfig()` | Write YAML to filesystem | Create per-task compose directory |
| `Up()` | `docker compose up -d` | Start container group |
| `Down()` | `docker compose down` | Stop and remove containers |
| `GetStatus()` | `docker compose ps --format json` | Parse container states |
| `GetServicePort()` | `docker compose port` | Get wrapper mapped port |

### 3.4 Docker-to-system status mapping

| Docker state | System state |
|-------------|-------------|
| running | running |
| exited (0) | succeeded |
| exited (non-0) | failed |
| paused | paused |
| not found | terminated |

## 4. DockerRuntime

Implements the existing `ContainerRuntime` interface.

### 4.1 Struct

```go
type DockerRuntime struct {
    compose *ComposeManager
}
```

### 4.2 Interface implementation

| Method | Logic |
|--------|-------|
| `Create(ctx, task)` | `compose.GenerateConfig()` → `compose.Up()` → return `RuntimeInfo{TaskID, Port}` |
| `GetStatus(ctx, taskID)` | `compose.GetStatus()` → map Docker state to `RuntimeStatus` |
| `Delete(ctx, taskID)` | `compose.Down()` → cleanup compose directory |
| `GetAddress(ctx, taskID)` | `compose.GetServicePort()` → return `http://localhost:{port}` |

### 4.3 Intervention support

All existing intervention mechanisms work unchanged through the wrapper sidecar:

| Operation | Signal/Action | Flow |
|-----------|--------------|------|
| pause | SIGSTOP | wrapper → cli-runner process |
| resume | SIGCONT | wrapper → cli-runner process |
| inject | write inject.json | wrapper → file → cli-runner polls |
| cancel | Delete() | DockerRuntime removes containers |

**Note**: interrupt (SIGINT) and Agent SDK integration are deferred to a follow-up issue.

## 5. Configuration

```yaml
# configs/config.yaml
executor:
  runtime_type: docker          # "docker" or "k8s"
  docker:
    workspace_dir: ./workspace   # host path for task workspaces
    compose_dir: /tmp/agent-infra/compose  # temp dir for compose files
    cli_runner_image: cli-runner:latest
    wrapper_image: agent-wrapper:latest
```

Initialization in `NewTaskExecutor()` selects runtime based on `runtime_type`:

```go
var runtime ContainerRuntime
switch cfg.RuntimeType {
case "k8s":
    runtime = NewK8sRuntime(...)
case "docker":
    runtime = NewDockerRuntime(...)
}
```

## 6. File changes

| File | Change | Description |
|------|--------|-------------|
| `internal/executor/docker_runtime.go` | NEW | DockerRuntime implementing ContainerRuntime |
| `internal/executor/compose_manager.go` | NEW | Compose YAML generation and CLI management |
| `internal/executor/compose_manager_test.go` | NEW | Unit tests for ComposeManager |
| `internal/executor/docker_runtime_test.go` | NEW | Unit tests for DockerRuntime |
| `internal/executor/runtime.go` | MODIFY | Add runtime type constants |
| `internal/executor/executor.go` | MODIFY | Runtime selection based on config |
| `internal/config/config.go` | MODIFY | Add Docker runtime config struct |

## 7. Out of scope (follow-up issues)

- **interrupt (SIGINT + --resume)**: Graceful interrupt and instruction injection via session resume
- **Agent SDK integration**: Replace shell-based cli-runner with Python Agent SDK `ClaudeSDKClient`, merge cli-runner and wrapper into single container
- **Resource limits**: CPU/memory constraints per container (future: map from template config)
- **Container log streaming**: Real-time log forwarding to control plane

## 8. Acceptance criteria

- [ ] Tasks execute inside Docker containers, isolated from host
- [ ] Can query container task status (running/completed/failed)
- [ ] Can stop/cleanup running task containers
- [ ] Intervention operations (pause, resume, inject) work via wrapper HTTP API
- [ ] Containers share workspace via host bind mount
- [ ] Runtime selection (Docker/K8s) is transparent to callers
