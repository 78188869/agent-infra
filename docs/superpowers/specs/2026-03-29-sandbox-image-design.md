# Sandbox Development Image Design

> **Issue**: #36 (revised after #47)
> **Date**: 2026-03-29
> **Status**: Approved
> **Review**: Code review identified 4 Critical + 4 Important issues (all included)

## 1. Overview

Fix and complete the sandbox Docker image to support local development task execution. After issue #47 merged cli-runner and wrapper into a single container, the `scripts/wrapper/Dockerfile` became the sole execution image but has toolchain gaps and the surrounding build infrastructure (Makefile, ComposeManager) still references the old two-container architecture.

## 2. Changes

### 2.1 Fix Dockerfile toolchain (`scripts/wrapper/Dockerfile`)

**Problem**: Missing `node`, `npm`, `npx` binaries in runtime stage. Only `claude` CLI was copied from node-builder.

**Fix**: Copy full Node.js toolchain from node-builder stage.

```dockerfile
# Node.js + Claude Code CLI from builder
COPY --from=node-builder /usr/local/bin/node /usr/local/bin/
COPY --from=node-builder /usr/local/bin/npm /usr/local/bin/
COPY --from=node-builder /usr/local/bin/npx /usr/local/bin/
COPY --from=node-builder /usr/local/bin/claude /usr/local/bin/
COPY --from=node-builder /usr/local/lib/node_modules /usr/local/lib/node_modules
```

**Problem**: Missing C++ compiler and SSH client.

**Fix**: Update apt-get install line:

```dockerfile
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl wget make gcc g++ bash openssh-client ca-certificates \
    && rm -rf /var/lib/apt/lists/*
```

**Decision**: Use `gcc + g++` instead of `build-essential` to avoid pulling unnecessary packages (dpkg-dev, lintian, etc.). Sufficient for node-gyp.

### 2.2 Add non-root user [Critical #1]

**Problem**: Container runs as root. K8s SecurityConfig has `RunAsNonRoot: true` and `RunAsUser: 1000`, which will cause the container to fail to start.

**Fix**: Add `runner` user with UID 1000:

```dockerfile
RUN useradd -m -u 1000 -s /bin/bash runner && \
    chown -R runner:runner /workspace /app
USER runner
```

### 2.3 Fix exit code mapping in DockerRuntime [Critical #4]

**Problem**: `docker_runtime.go:mapDockerStateToPhase()` maps all `exited` containers to `Succeeded`, regardless of exit code. A failed git clone (exit 1) is reported as success.

**Fix**: Check exit code when state is `exited`:

```go
case "exited":
    // Need to check exit code via docker inspect
    exitCode := getContainerExitCode(ctx, containerID)
    if exitCode == 0 {
        return "Succeeded"
    }
    return "Failed"
```

This requires updating `GetStatus()` to query exit code from Docker when state is `exited`. The `ComposeManager` needs a new method `GetExitCode(ctx, taskID) (int, error)` using `docker compose ps --format json` to extract the exit code.

### 2.4 Sanitize compose template values [Critical #3]

**Problem**: `text/template` directly interpolates values into YAML. Values containing YAML special characters break the compose file or enable injection.

**Fix**: Use Go's `text/template` with proper YAML string quoting. Wrap all environment variable values in a helper that quotes and escapes:

```go
// yamlQuote wraps a string in single quotes for safe YAML embedding.
// Single quotes inside the value are escaped as ''.
func yamlQuote(s string) string {
    return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
```

Apply `yamlQuote` to all template values: TaskPrompt, ClaudeMdContent, GitRepoURL, etc.

### 2.5 Update ComposeManager to single-container template [Critical #2]

**Problem**: `composeTemplate` defines two services (cli-runner + wrapper). `DockerRuntime.Create()` calls old `GenerateConfig()`, not `GenerateSingleContainerConfig()`.

**Fix**:
- Remove old `composeTemplate` and `GenerateConfig()` method
- Rename `singleContainerComposeTemplate` to `composeTemplate`
- Rename `GenerateSingleContainerConfig()` to `GenerateConfig()`
- Update `DockerRuntime.Create()` to call the renamed method
- Apply defaults from `cm.config` when template data fields are empty
- Add `CLAUDE_MD_CONTENT` size validation (warn if > 64KB)

**Code changes**:
- `DockerConfig`: Remove `CLIRunnerImage` field
- `DefaultDockerConfig()`: Remove `CLIRunnerImage` default
- `DefaultJobConfig()`: Remove `CLIRunnerImage` references
- `DockerRuntime.GetAddress()`: Service name `"wrapper"` -> `"sandbox"`
- Pass `GIT_BRANCH` and `CLAUDE_MD_CONTENT` from envVars
- Update all tests to match new single-container template

### 2.6 Separate test dependencies from production [Important #5]

**Problem**: `requirements.txt` includes `pytest` and `pytest-asyncio` in the production image.

**Fix**: Split into two files:
- `requirements.txt` — production deps only (fastapi, uvicorn, claude-agent-sdk, httpx, pydantic)
- `requirements-dev.txt` — adds `pytest>=8.0.0` and `pytest-asyncio>=0.23.0`

Dockerfile only installs `requirements.txt`. CI/dev workflow installs `requirements-dev.txt`.

### 2.7 Fix HEALTHCHECK start-period [Important #6]

**Problem**: `--start-period=5s` is too short for large repository clones. Container may be marked unhealthy during normal initialization.

**Fix**: Increase to 60s:

```dockerfile
HEALTHCHECK --interval=10s --timeout=3s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:9090/health || exit 1
```

### 2.8 Add .dockerignore [Important #7]

**Problem**: `COPY scripts/wrapper/ .` copies everything including `tests/` and `__pycache__/`.

**Fix**: Add `.dockerignore` at project root:

```
scripts/wrapper/tests/
scripts/wrapper/**/__pycache__/
scripts/wrapper/**/*.pyc
.git/
.claude/
```

### 2.9 Update Makefile

Remove old targets:
- `docker-build-cli-runner`
- `docker-build-wrapper`

Add new target:
```makefile
docker-build-sandbox:
	docker build -f scripts/wrapper/Dockerfile -t $(REGISTRY)/sandbox:$(VERSION) .
```

Update `docker-build-all` and `docker-push` accordingly.

### 2.10 Clean up legacy files

Delete:
- `deploy/dockerfiles/Dockerfile.cli-runner` (replaced by `scripts/wrapper/Dockerfile`)
- `deploy/dockerfiles/Dockerfile.agent-wrapper` (replaced by `scripts/wrapper/Dockerfile`)
- `scripts/cli-runner.sh` (replaced by `scripts/entrypoint.sh`)

### 2.11 Use ARG for Go version [Nice-to-have]

Parameterize Go version for easier updates:

```dockerfile
ARG GO_VERSION=1.22.0
RUN ARCH=$(dpkg --print-architecture) && \
    curl -sL "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz" | tar -C /usr/local -xz
```

## 3. Acceptance Criteria

- [ ] `docker build -f scripts/wrapper/Dockerfile .` succeeds locally
- [ ] Built image contains: git, go, node, npm, npx, claude, gh, python3, pip, make, gcc, g++, bash, curl, wget, openssh-client
- [ ] Container runs as non-root user (UID 1000)
- [ ] `make docker-build-sandbox` works
- [ ] ComposeManager generates single-container compose YAML with proper YAML escaping
- [ ] DockerRuntime.Create() uses single-container template
- [ ] Exited containers with non-zero exit code report as `Failed`
- [ ] No references to deleted files remain in codebase
- [ ] `.dockerignore` excludes test files and caches
- [ ] `requirements.txt` does not contain test dependencies
- [ ] HEALTHCHECK start-period is 60s
- [ ] All existing tests pass

## 4. Out of Scope

- Docker runtime integration testing (needs running Docker daemon)
- Container resource limits
- Log streaming from container
- K8s manifest updates (future issue)
- Go module cache volume mounts
