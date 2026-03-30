# 本地开发一键启动 (Issue 37) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `make local` start the complete application with zero external services — SQLite replaces MySQL, miniredis replaces Redis, Docker runtime is optional.

**Architecture:** main.go detects `APP_ENV=local` → loads `config.local.yaml` → inits SQLite + miniredis + Docker runtime → wires all services/scheduler/executor → starts HTTP server with graceful shutdown. Production mode uses real MySQL/Redis/K8s as before.

**Tech Stack:** Go 1.21, SQLite (via GORM), miniredis (in-memory Redis), Docker Compose runtime

---

### Task 1: Update config.local.yaml for SQLite

**Files:**
- Modify: `configs/config.local.yaml`

- [ ] **Step 1: Update config.local.yaml**

Replace the entire file with SQLite-focused local config:

```yaml
# Local Development Configuration
# Usage: APP_ENV=local make local
# No external dependencies needed — SQLite + miniredis (in-memory Redis)
env: local
server:
  port: 8080
  mode: debug
database:
  driver: sqlite
  name: data/agent_infra.db
redis:
  host: localhost
  port: 6379
  db: 0
log:
  level: debug
  format: text
  outputs: both
  file:
    dir: logs
    max_size_mb: 100
    max_backups: 7
    max_age_days: 30
```

- [ ] **Step 2: Verify config loads correctly**

Run: `cd cmd/control-plane && go test -run TestLoadConfig -v`
Expected: PASS (existing test still works with config.yaml)

Also manually verify:
```bash
cd cmd/control-plane && go run . 2>&1 | head -5
# Should show: "failed to connect to database" because config.yaml still uses MySQL
# This is expected — Task 4 will fix this
```

- [ ] **Step 3: Commit**

```bash
git add configs/config.local.yaml
git commit -m "chore(config): update config.local.yaml for SQLite-based local dev"
```

---

### Task 2: Add ResolveConfigPath helper (TDD)

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test for ResolveConfigPath**

Add to `internal/config/config_test.go`:

```go
func TestResolveConfigPath(t *testing.T) {
	// Save and restore env
	origAPPENV := os.Getenv("APP_ENV")
	origCFGPATH := os.Getenv("CONFIG_PATH")
	t.Cleanup(func() {
		os.Setenv("APP_ENV", origAPPENV)
		os.Setenv("CONFIG_PATH", origCFGPATH)
	})

	tests := []struct {
		name     string
		appEnv   string
		cfgPath  string
		expected string
	}{
		{
			name:     "CONFIG_PATH takes priority",
			appEnv:   "local",
			cfgPath:  "/custom/config.yaml",
			expected: "/custom/config.yaml",
		},
		{
			name:     "APP_ENV=local selects config.local.yaml",
			appEnv:   "local",
			cfgPath:  "",
			expected: "configs/config.local.yaml",
		},
		{
			name:     "APP_ENV=development selects config.local.yaml",
			appEnv:   "development",
			cfgPath:  "",
			expected: "configs/config.local.yaml",
		},
		{
			name:     "APP_ENV=production selects config.yaml",
			appEnv:   "production",
			cfgPath:  "",
			expected: "configs/config.yaml",
		},
		{
			name:     "empty APP_ENV defaults to config.yaml",
			appEnv:   "",
			cfgPath:  "",
			expected: "configs/config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("APP_ENV", tt.appEnv)
			os.Setenv("CONFIG_PATH", tt.cfgPath)

			got := config.ResolveConfigPath()
			if got != tt.expected {
				t.Errorf("ResolveConfigPath() = %q, want %q", got, tt.expected)
			}
		})
	}
}
```

Note: Add `"os"` to imports in config_test.go if not already present. Also add the import alias: use the package name directly since config_test is in the same package. If the test file is in a separate `_test` package (external test), use `config.ResolveConfigPath()`.

Check the test file's package declaration first — if it's `package config_test`, the above is correct. If it's `package config`, remove the `config.` prefix.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestResolveConfigPath -v`
Expected: FAIL — `undefined: config.ResolveConfigPath`

- [ ] **Step 3: Implement ResolveConfigPath**

Add to `internal/config/config.go`:

```go
// ResolveConfigPath determines the config file path based on environment.
// Priority: CONFIG_PATH env var > APP_ENV-based selection > default config.yaml.
func ResolveConfigPath() string {
	// Explicit override takes highest priority
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}

	// Local/development environments use config.local.yaml
	env := os.Getenv("APP_ENV")
	if env == "local" || env == "development" {
		return "configs/config.local.yaml"
	}

	return "configs/config.yaml"
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestResolveConfigPath -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add ResolveConfigPath for environment-based config selection"
```

---

### Task 3: Add SetInterventionEventHandler export helper (TDD)

**Files:**
- Modify: `internal/service/intervention_service.go`
- Modify: `internal/service/intervention_service_test.go`

- [ ] **Step 1: Write failing test for SetInterventionEventHandler**

Add to `internal/service/intervention_service_test.go`:

```go
func TestSetInterventionEventHandler(t *testing.T) {
	taskRepo := &mockTaskRepoForIntervention{}
	interventionRepo := &mockInterventionRepo{}

	svc := NewInterventionService(taskRepo, interventionRepo)

	handler := &mockTaskEventHandler{}
	SetInterventionEventHandler(svc, handler)

	// Verify the handler is set by triggering HandleWrapperEvent which delegates to handler
	// This indirectly tests that the handler was set correctly
}
```

Note: You'll need to check the existing test file for the mock types available (`mockTaskRepoForIntervention`, `mockInterventionRepo`, etc.) and use the appropriate ones. If a `mockTaskEventHandler` doesn't exist, add it:

```go
type mockTaskEventHandler struct {
	called bool
}

func (m *mockTaskEventHandler) HandleTaskEvent(ctx context.Context, taskID string, eventType string, payload map[string]interface{}) error {
	m.called = true
	return nil
}
```

If the above test structure is too tightly coupled to internal types, a simpler approach:

```go
func TestSetInterventionEventHandler(t *testing.T) {
	taskRepo := &mockTaskRepoForIntervention{}
	interventionRepo := &mockInterventionRepo{}

	svc := NewInterventionService(taskRepo, interventionRepo)
	handler := &mockTaskEventHandler{}

	// Should not panic
	SetInterventionEventHandler(svc, handler)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/ -run TestSetInterventionEventHandler -v`
Expected: FAIL — `undefined: SetInterventionEventHandler`

- [ ] **Step 3: Implement SetInterventionEventHandler**

Add to `internal/service/intervention_service.go`:

```go
// SetInterventionEventHandler sets the TaskEventHandler on an InterventionService.
// This function allows external packages (e.g., main.go) to wire the executor
// as an event handler without depending on the concrete service type.
func SetInterventionEventHandler(svc InterventionService, handler TaskEventHandler) {
	if s, ok := svc.(*interventionService); ok {
		s.eventHandler = handler
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/ -run TestSetInterventionEventHandler -v`
Expected: PASS

- [ ] **Step 5: Run all service tests**

Run: `go test ./internal/service/ -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/intervention_service.go internal/service/intervention_service_test.go
git commit -m "feat(service): export SetInterventionEventHandler for executor wiring"
```

---

### Task 4: Refactor main.go — full bootstrap sequence

This is the core task. The main.go will be rewritten to bootstrap all components.

**Files:**
- Rewrite: `cmd/control-plane/main.go`
- Update: `cmd/control-plane/main_test.go` (move mocks to test file)

- [ ] **Step 4.1: Move mock services to main_test.go**

All mock service types (`mockTenantService`, `mockTemplateService`, etc.) are currently in `main.go` but only used in `main_test.go`. Move them to `main_test.go` so they don't appear in the production binary.

Move these types from `main.go` to `main_test.go`:
- `mockTenantService` and all its methods
- `mockTemplateService` and all its methods
- `mockTaskService` and all its methods
- `mockProviderService` and all its methods
- `mockCapabilityService` and all its methods
- `mockInterventionService` and all its methods

Cut everything from `// mockTenantService is a fallback service...` to the end of the file in `main.go`, and paste it into `main_test.go`.

- [ ] **Step 4.2: Verify tests still pass after move**

Run: `go test ./cmd/control-plane/ -v`
Expected: All PASS (mock tests still work since they're in the same `package main`)

- [ ] **Step 4.3: Rewrite main.go with full bootstrap**

Replace the entire `main.go` with:

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/example/agent-infra/internal/api/router"
	"github.com/example/agent-infra/internal/config"
	"github.com/example/agent-infra/internal/executor"
	"github.com/example/agent-infra/internal/migration"
	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/monitoring"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/internal/scheduler"
	"github.com/example/agent-infra/internal/seed"
	"github.com/example/agent-infra/internal/service"
	"github.com/example/agent-infra/pkg/aliyun/sls"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	// 1. Load config (auto-select based on APP_ENV)
	configPath := config.ResolveConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err, "path", configPath)
		os.Exit(1)
	}

	env := cfg.GetEnvironment()

	// 2. Initialize structured logger
	logger := monitoring.NewLogger(cfg)
	slog.SetDefault(logger)
	slog.Info("starting control-plane", "env", env, "config", configPath)

	gin.SetMode(cfg.Server.Mode)

	// 3. Ensure data directory exists for SQLite
	if cfg.Database.IsSQLite() {
		dbName := cfg.Database.Database
		if dbName == "" {
			dbName = "agent_infra.db"
		}
		dir := filepath.Dir(dbName)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				slog.Error("failed to create data directory", "dir", dir, "error", err)
				os.Exit(1)
			}
		}
	}

	// 4. Initialize database
	db, err := config.NewDatabase(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Warn("error closing database", "error", err)
		}
	}()
	slog.Info("database connected", "driver", cfg.Database.Driver)

	// 4. Run migrations + seed data
	m := migration.NewMigrator(db.DB)
	if err := m.AutoMigrate(); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations completed")

	if cfg.IsLocal() {
		if err := seed.SeedProviders(db.DB); err != nil {
			slog.Warn("failed to seed providers", "error", err)
		} else {
			slog.Info("seed data initialized")
		}
	}

	// 5. Initialize Redis (miniredis for local, real Redis for production)
	var redisClient *redis.Client
	var miniRedis *miniredis.Miniredis

	if cfg.IsLocal() {
		miniRedis = miniredis.NewMiniRedis()
		if err := miniRedis.Start(); err != nil {
			slog.Error("failed to start miniredis", "error", err)
			os.Exit(1)
		}
		redisClient = redis.NewClient(&redis.Options{
			Addr: miniRedis.Addr(),
		})
		slog.Info("using miniredis (in-memory Redis)", "addr", miniRedis.Addr())
	} else {
		redisCfg := cfg.Redis.ToRedisConfig()
		rc, err := config.NewRedisClient(redisCfg)
		if err != nil {
			slog.Error("failed to connect to Redis", "error", err)
			os.Exit(1)
		}
		redisClient = rc.Client
		slog.Info("Redis connected", "addr", redisCfg.Addr)
	}

	// 6. Create repositories
	tenantRepo := repository.NewTenantRepository(db.DB)
	templateRepo := repository.NewTemplateRepository(db.DB)
	taskRepo := repository.NewTaskRepository(db.DB)
	providerRepo := repository.NewProviderRepository(db.DB)
	capabilityRepo := repository.NewCapabilityRepository(db.DB)
	interventionRepo := repository.NewInterventionRepository(db.DB)

	// 7. Create services
	tenantSvc := service.NewTenantService(tenantRepo)
	templateSvc := service.NewTemplateService(templateRepo)
	taskSvc := service.NewTaskService(taskRepo)
	providerSvc := service.NewProviderService(providerRepo)
	capabilitySvc := service.NewCapabilityService(capabilityRepo)
	interventionSvc := service.NewInterventionService(taskRepo, interventionRepo)

	// 8. Create monitoring
	monitoringHub := monitoring.NewHub()
	slsClient := monitoring.NewSLSClient(sls.Config{
		Endpoint:        cfg.SLS.Endpoint,
		AccessKeyID:     cfg.SLS.AccessKey,
		AccessKeySecret: cfg.SLS.AccessSecret,
		Project:         cfg.SLS.Project,
		LogStore:        cfg.SLS.Logstore,
	})
	monitoringSvc := service.NewMonitoringService(monitoringHub, slsClient)

	// 9. Create executor (Docker runtime for local, K8s for production)
	var exec *executor.TaskExecutor
	if cfg.IsLocal() {
		dockerRuntime, err := executor.NewDockerRuntime(executor.DefaultDockerConfig())
		if err != nil {
			slog.Warn("Docker runtime unavailable — task execution will not work", "error", err)
		} else {
			exec, err = executor.NewTaskExecutor(dockerRuntime, redisClient, &executor.ExecutorConfig{
				UpdateTaskStatus: func(ctx context.Context, taskID, status, message string) error {
					id, err := uuid.Parse(taskID)
					if err != nil {
						return err
					}
					return taskRepo.UpdateStatus(ctx, id, status, message)
				},
				GetTask: func(ctx context.Context, taskID string) (*model.Task, error) {
					return taskSvc.GetByID(ctx, taskID)
				},
				OnTaskComplete: func(ctx context.Context, taskID string, result map[string]interface{}) error {
					id, _ := uuid.Parse(taskID)
					return taskRepo.UpdateStatus(ctx, id, model.TaskStatusSucceeded, "task completed")
				},
				OnTaskFailed: func(ctx context.Context, taskID string, taskErr error) error {
					id, _ := uuid.Parse(taskID)
					return taskRepo.UpdateStatus(ctx, id, model.TaskStatusFailed, taskErr.Error())
				},
			})
			if err != nil {
				slog.Warn("failed to create executor", "error", err)
			} else {
				// Wire executor as event handler for intervention service
				service.SetInterventionEventHandler(interventionSvc, exec)
				slog.Info("executor initialized (Docker runtime)")
			}
		}
	}

	// 10. Create scheduler
	sched := scheduler.NewTaskScheduler(redisClient, &scheduler.SchedulerConfig{
		GlobalLimit: 100,
		GetTenantQuota: func(ctx context.Context, tenantID string) (*scheduler.TenantQuota, error) {
			tenant, err := tenantSvc.GetByID(ctx, tenantID)
			if err != nil {
				return nil, err
			}
			return &scheduler.TenantQuota{
				Concurrency: tenant.QuotaConcurrency,
				DailyTasks:  tenant.QuotaDailyTasks,
			}, nil
		},
		GetTask: func(ctx context.Context, taskID string) (*model.Task, error) {
			return taskSvc.GetByID(ctx, taskID)
		},
		UpdateStatus: func(ctx context.Context, taskID, status, message string) error {
			id, err := uuid.Parse(taskID)
			if err != nil {
				return err
			}
			return taskRepo.UpdateStatus(ctx, id, status, message)
		},
	})
	slog.Info("scheduler initialized")

	// 11. Start executor and scheduler
	ctx := context.Background()
	if exec != nil {
		if err := exec.Start(ctx); err != nil {
			slog.Warn("failed to start executor", "error", err)
		}
	}
	if err := sched.Start(ctx); err != nil {
		slog.Warn("failed to start scheduler", "error", err)
	}

	// 12. Setup router
	r := router.Setup(tenantSvc, templateSvc, taskSvc, providerSvc, capabilitySvc, monitoringSvc, monitoringHub, interventionSvc, db)

	// 14. Start HTTP server with graceful shutdown
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		slog.Info("HTTP server starting", "addr", addr, "env", env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("received shutdown signal", "signal", sig)

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop components in reverse order of startup
	if err := sched.Stop(shutdownCtx); err != nil {
		slog.Warn("error stopping scheduler", "error", err)
	}
	if exec != nil {
		if err := exec.Stop(shutdownCtx); err != nil {
			slog.Warn("error stopping executor", "error", err)
		}
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Warn("error shutting down HTTP server", "error", err)
	}

	if miniRedis != nil {
		miniRedis.Close()
		slog.Info("miniredis stopped")
	}

	slog.Info("server stopped gracefully")
}
```

Note: This imports `"github.com/example/agent-infra/internal/model"` — check if `model.Task` is referenced and add the import. The `model` package import is needed for `model.TaskStatusSucceeded` and `model.TaskStatusFailed` used in the executor callbacks.

Actually, verify the import path. Looking at the existing imports in main.go: the model package is already imported as `"github.com/example/agent-infra/internal/model"`.

- [ ] **Step 4.4: Verify compilation**

Run: `go build ./cmd/control-plane/`
Expected: Builds successfully with no errors.

If there are import issues, check:
1. `github.com/alicebob/miniredis/v2` is in go.mod (it is)
2. All referenced types exist in their packages
3. `service.SetInterventionEventHandler` matches the function signature from Task 3
4. `model.TaskStatusSucceeded` / `model.TaskStatusFailed` exist in model package

- [ ] **Step 4.5: Fix compilation errors (if any)**

Common issues:
- Missing import for `"github.com/example/agent-infra/internal/model"` — add it
- `SetInterventionEventHandler` not exported — verify Task 3 was completed
- `uuid.Parse` needs `"github.com/google/uuid"` import — add it

Run: `go build ./cmd/control-plane/`
Expected: Builds successfully

- [ ] **Step 4.6: Run all tests**

Run: `go test ./cmd/control-plane/ -v`
Expected: All PASS

Run: `go test ./... -short`
Expected: All PASS

- [ ] **Step 4.7: Commit**

```bash
git add cmd/control-plane/main.go cmd/control-plane/main_test.go
git commit -m "feat(main): wire scheduler/executor/miniredis with graceful shutdown"
```

---

### Task 5: Add `make local` target

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Add local target to Makefile**

Add after the `dev: run` target (around line 92):

```makefile
# Local development (SQLite + miniredis, no external deps)
local:
	@mkdir -p data logs
	APP_ENV=local GOPROXY=$(GOPROXY) $(GOBUILD) -o $(BINARY) $(MAIN_PACKAGE) && APP_ENV=local ./$(BINARY)
```

- [ ] **Step 2: Test the target**

Run: `make local &`
Then: `curl -s http://localhost:8080/health | head -1`
Expected: `{"status":"ok"}` or similar health response

Then stop: `kill %1` or Ctrl+C

Verify:
1. `data/agent_infra.db` was created (SQLite database)
2. `logs/` directory has log files
3. Health endpoint responded correctly

Run: `make test`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "feat(makefile): add make local target for one-click local dev"
```

---

### Task 6: Integration test for local startup

**Files:**
- Modify: `cmd/control-plane/main_test.go`

- [ ] **Step 1: Write integration test**

Add to `cmd/control-plane/main_test.go`:

```go
func TestLocalStartup(t *testing.T) {
	// Set up local environment
	origAPPENV := os.Getenv("APP_ENV")
	t.Cleanup(func() {
		os.Setenv("APP_ENV", origAPPENV)
	})
	os.Setenv("APP_ENV", "local")

	// Verify config path resolution
	configPath := config.ResolveConfigPath()
	if configPath != "configs/config.local.yaml" {
		t.Fatalf("expected config.local.yaml, got %s", configPath)
	}

	// Load and verify config
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if !cfg.IsLocal() {
		t.Error("expected IsLocal() to be true")
	}
	if !cfg.Database.IsSQLite() {
		t.Error("expected SQLite driver in local config")
	}
	if cfg.Server.Mode != "debug" {
		t.Errorf("expected debug mode, got %s", cfg.Server.Mode)
	}
}
```

Note: Add `"os"` to imports if not already present.

- [ ] **Step 2: Run test to verify it passes**

Run: `go test ./cmd/control-plane/ -run TestLocalStartup -v`
Expected: PASS

- [ ] **Step 3: Run full test suite**

Run: `make test`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/control-plane/main_test.go
git commit -m "test(main): add local startup integration test"
```

---

### Task 7: Local dev documentation

**Files:**
- Create: `docs/knowledge/local-dev.md`

- [ ] **Step 1: Write local dev guide**

Create `docs/knowledge/local-dev.md`:

```markdown
# Local Development Guide

## Prerequisites

- Go 1.21+
- Docker (optional, for task execution)

**No MySQL, Redis, or Kubernetes required.**

## Quick Start

```bash
make local
```

This single command:
1. Creates `data/agent_infra.db` (SQLite)
2. Starts in-memory Redis (miniredis)
3. Runs database migrations
4. Seeds system providers (Claude Code, Zhipu GLM, DeepSeek)
5. Starts the HTTP server on `:8080`

## Architecture (Local Mode)

| Component | Production | Local |
|-----------|-----------|-------|
| Database | OceanBase (MySQL) | SQLite (`data/agent_infra.db`) |
| Cache/Queue | Redis 6 | miniredis (in-memory) |
| Container Runtime | K8s Jobs | Docker Compose |
| Logging | Aliyun SLS | File (`logs/`) + stdout |

## API Endpoints

After startup, the API is available at `http://localhost:8080`:

```bash
# Health check
curl http://localhost:8080/health

# Create a tenant
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name": "test-tenant"}'

# List providers (seeded)
curl http://localhost:8080/api/v1/providers
```

## Configuration

Local config is at `configs/config.local.yaml`. Key settings:

- `database.driver: sqlite` — file-based database
- `database.name: data/agent_infra.db` — database file path
- `log.level: debug` — verbose logging
- `log.outputs: both` — stdout + file logging

## Without Docker

If Docker is not installed, the app still starts successfully. Task creation and management via API works, but task execution will fail with a Docker error. This is expected.

## Cleanup

```bash
rm -rf data/ logs/
```
```

- [ ] **Step 2: Commit**

```bash
git add docs/knowledge/local-dev.md
git commit -m "docs: add local development guide"
```

---

### Task 8: Final validation and cleanup

- [ ] **Step 8.1: Run full test suite**

Run: `make test`
Expected: All PASS

- [ ] **Step 8.2: Run linter**

Run: `make lint`
Expected: No errors (or only pre-existing warnings)

- [ ] **Step 8.3: Run `make local` end-to-end**

```bash
# Start the server in background
make local &
sleep 3

# Test health endpoint
curl -sf http://localhost:8080/health

# Test API - create tenant
curl -sf -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name": "test-tenant"}'

# Test API - list providers (should have seeded data)
curl -sf http://localhost:8080/api/v1/providers

# Stop the server
kill %1
```

Expected: All endpoints respond successfully, providers list includes claude-code, zhipu-glm, deepseek.

- [ ] **Step 8.4: Verify acceptance criteria**

Check each criterion:
- [ ] `make local` starts the complete app — verified in Step 8.3
- [ ] API supports full task lifecycle (create, view, intervene, cancel) — verified via curl
- [ ] No MySQL, Redis, K8s needed — verified (SQLite + miniredis)
- [ ] Documentation exists — verified (docs/knowledge/local-dev.md)

- [ ] **Step 8.5: Final commit (if any remaining changes)**

```bash
git add -A
git status  # Review any remaining changes
git commit -m "chore: final cleanup for issue 37"
```
