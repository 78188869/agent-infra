# Environment-Aware Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add environment-aware configuration that lets developers activate local mode with a single `APP_ENV=local` variable, automatically selecting appropriate defaults for database, Redis, executor, and logging.

**Architecture:** Introduce a unified `AppConfig` struct in `internal/config/` that loads YAML with `${VAR:default}` expansion, detects environment from `APP_ENV` (or `env` field in YAML), and provides environment-specific defaults. The `main.go` entry point uses this instead of its current ad-hoc `Config` struct. Local mode uses debug logging, no SLS, and skips K8s executor.

**Tech Stack:** Go 1.21, yaml.v3, regex for env var expansion, os.Getenv

---

## File Structure

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `internal/config/config.go` | Unified `AppConfig` struct, `Load()` function, `ExpandEnv()`, env detection |
| Create | `internal/config/config_test.go` | Tests for config loading, env expansion, environment detection, field mapping |
| Modify | `internal/config/database.go` | Add YAML tags to `DatabaseConfig` |
| Modify | `internal/config/redis.go` | Add YAML tags to `RedisConfig` |
| Modify | `cmd/control-plane/main.go` | Replace ad-hoc `Config` with `config.AppConfig` |
| Create | `configs/config.local.yaml` | Local development defaults (debug mode, no SLS) |
| Modify | `configs/config.yaml` | Add `env` field, align SLS env var names |

---

## Task 1: Config Loader Core — ExpandEnv + AppConfig + YAML Tags

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Modify: `internal/config/database.go` (add YAML tags)
- Modify: `internal/config/redis.go` (add YAML tags)

This task delivers the complete config module foundation: env var expansion, the unified `AppConfig` struct, YAML field compatibility, and the `Load` function. All in one task to avoid type inconsistency between steps.

- [ ] **Step 1: Write the failing tests**

Create `internal/config/config_test.go`:

```go
package config

import (
	"os"
	"testing"
)

func TestExpandEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envSetup map[string]string
		expected string
	}{
		{
			name:     "no placeholders",
			input:    "host: localhost",
			expected: "host: localhost",
		},
		{
			name:     "env var with value",
			input:    "host: ${DB_HOST:localhost}",
			envSetup: map[string]string{"DB_HOST": "prod-db.example.com"},
			expected: "host: prod-db.example.com",
		},
		{
			name:     "env var with default",
			input:    "host: ${DB_HOST:localhost}",
			envSetup: map[string]string{},
			expected: "host: localhost",
		},
		{
			name:     "env var without default and not set",
			input:    "password: ${DB_PASSWORD}",
			envSetup: map[string]string{},
			expected: "password: ",
		},
		{
			name:     "env var without default and set",
			input:    "password: ${DB_PASSWORD}",
			envSetup: map[string]string{"DB_PASSWORD": "secret"},
			expected: "password: secret",
		},
		{
			name:     "multiple placeholders in one line",
			input:    "addr: ${REDIS_HOST:localhost}:${REDIS_PORT:6379}",
			envSetup: map[string]string{"REDIS_HOST": "10.0.0.1"},
			expected: "addr: 10.0.0.1:6379",
		},
		{
			name:     "empty default",
			input:    "kubeconfig: ${KUBECONFIG:}",
			envSetup: map[string]string{},
			expected: "kubeconfig: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envSetup {
				t.Setenv(k, v)
			}
			got := ExpandEnv(tt.input)
			if got != tt.expected {
				t.Errorf("ExpandEnv() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLoad_ValidYAML(t *testing.T) {
	content := `
env: local
server:
  port: 9090
  mode: debug
database:
  host: localhost
  port: 3306
  name: testdb
  user: root
  password: ""
redis:
  host: localhost
  port: 6379
  db: 0
log:
  level: debug
  format: text
`
	tmpDir := t.TempDir()
	path := tmpDir + "/config.yaml"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != "local" {
		t.Errorf("Env = %q, want %q", cfg.Env, "local")
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 9090)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "localhost")
	}
	if cfg.Database.Username != "root" {
		t.Errorf("Database.Username = %q, want %q", cfg.Database.Username, "root")
	}
	if cfg.Database.Database != "testdb" {
		t.Errorf("Database.Database = %q, want %q", cfg.Database.Database, "testdb")
	}
	if cfg.Redis.Host != "localhost" {
		t.Errorf("Redis.Host = %q, want %q", cfg.Redis.Host, "localhost")
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("Redis.Port = %d, want %d", cfg.Redis.Port, 6379)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "debug")
	}
}

func TestLoad_EnvVarExpansion(t *testing.T) {
	t.Setenv("TEST_DB_HOST", "expanded-host")

	content := `
env: ${APP_ENV:production}
database:
  host: ${TEST_DB_HOST:localhost}
  port: 3306
`
	tmpDir := t.TempDir()
	path := tmpDir + "/config.yaml"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.Host != "expanded-host" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "expanded-host")
	}
	if cfg.Env != "production" {
		t.Errorf("Env = %q, want %q", cfg.Env, "production")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoad_DefaultsEnvFromEnvVar(t *testing.T) {
	t.Setenv("APP_ENV", "staging")

	content := `
server:
  port: 8080
`
	tmpDir := t.TempDir()
	path := tmpDir + "/config.yaml"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if resolved := cfg.GetEnvironment(); resolved != "staging" {
		t.Errorf("GetEnvironment() = %q, want %q", resolved, "staging")
	}
}

func TestLoad_DatabaseFieldMapping(t *testing.T) {
	content := `
database:
  host: db.example.com
  port: 2881
  name: mydb
  user: admin
  password: s3cret
  max_connections: 50
`
	tmpDir := t.TempDir()
	path := tmpDir + "/config.yaml"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.Username != "admin" {
		t.Errorf("Database.Username = %q, want %q", cfg.Database.Username, "admin")
	}
	if cfg.Database.Database != "mydb" {
		t.Errorf("Database.Database = %q, want %q", cfg.Database.Database, "mydb")
	}
	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "db.example.com")
	}
	if cfg.Database.MaxOpenConns != 50 {
		t.Errorf("Database.MaxOpenConns = %d, want %d", cfg.Database.MaxOpenConns, 50)
	}
}

func TestRedisYAMLConfig_ToRedisConfig(t *testing.T) {
	redis := RedisYAMLConfig{
		Host:     "10.0.0.1",
		Port:     6380,
		Password: "secret",
		DB:       2,
	}
	got := redis.ToRedisConfig()
	if got.Addr != "10.0.0.1:6380" {
		t.Errorf("Addr = %q, want %q", got.Addr, "10.0.0.1:6380")
	}
	if got.Password != "secret" {
		t.Errorf("Password = %q, want %q", got.Password, "secret")
	}
	if got.DB != 2 {
		t.Errorf("DB = %d, want %d", got.DB, 2)
	}
}

func TestRedisYAMLConfig_ToRedisConfig_Empty(t *testing.T) {
	redis := RedisYAMLConfig{}
	got := redis.ToRedisConfig()
	if got.Addr != "localhost:6379" {
		t.Errorf("Addr = %q, want %q", got.Addr, "localhost:6379")
	}
}

func TestAppConfig_IsLocal(t *testing.T) {
	cfg := &AppConfig{Env: "local"}
	if !cfg.IsLocal() {
		t.Error("IsLocal() = false, want true for env=local")
	}

	cfg.Env = "production"
	if cfg.IsLocal() {
		t.Error("IsLocal() = true, want false for env=production")
	}
}

func TestAppConfig_ApplyDefaults_Local(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	cfg := &AppConfig{}
	cfg.ApplyDefaults()
	if cfg.Server.Mode != "debug" {
		t.Errorf("Server.Mode = %q, want %q", cfg.Server.Mode, "debug")
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "debug")
	}
	if cfg.Log.Format != "text" {
		t.Errorf("Log.Format = %q, want %q", cfg.Log.Format, "text")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 8080)
	}
}

func TestAppConfig_ApplyDefaults_Production(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	cfg := &AppConfig{}
	cfg.ApplyDefaults()
	if cfg.Server.Mode != "release" {
		t.Errorf("Server.Mode = %q, want %q", cfg.Server.Mode, "release")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "info")
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %q, want %q", cfg.Log.Format, "json")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-31 && GOPROXY=https://goproxy.cn,direct go test ./internal/config/ -v`
Expected: FAIL — `ExpandEnv`, `Load`, `AppConfig` etc. undefined

- [ ] **Step 3: Create `internal/config/config.go`**

```go
package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Environment constants
const (
	EnvLocal      = "local"
	EnvProduction = "production"
)

// envPattern matches ${VAR} or ${VAR:default}
var envPattern = regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)

// ExpandEnv replaces ${VAR} and ${VAR:default} placeholders with environment variable values.
func ExpandEnv(s string) string {
	return envPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := envPattern.FindStringSubmatch(match)
		name := sub[1]
		defaultVal := ""
		if len(sub) > 2 {
			defaultVal = sub[2]
		}
		if val, ok := os.LookupEnv(name); ok {
			return val
		}
		return defaultVal
	})
}

// AppConfig is the unified application configuration.
type AppConfig struct {
	Env      string           `yaml:"env"`
	Server   ServerConfig     `yaml:"server"`
	Database DatabaseConfig   `yaml:"database"`
	Redis    RedisYAMLConfig  `yaml:"redis"`
	Log      LogConfig        `yaml:"log"`
	K8s      K8sConfig        `yaml:"k8s"`
	SLS      SLSConfig        `yaml:"sls"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// K8sConfig holds Kubernetes configuration.
type K8sConfig struct {
	Kubeconfig string             `yaml:"kubeconfig"`
	Namespace  K8sNamespaceConfig `yaml:"namespace"`
}

// K8sNamespaceConfig holds K8s namespace configuration.
type K8sNamespaceConfig struct {
	ControlPlane string `yaml:"control_plane"`
	Sandbox      string `yaml:"sandbox"`
}

// SLSConfig holds Aliyun SLS logging configuration.
type SLSConfig struct {
	Endpoint     string `yaml:"endpoint"`
	Project      string `yaml:"project"`
	LogStore     string `yaml:"logstore"`
	AccessKey    string `yaml:"access_key"`
	AccessSecret string `yaml:"access_secret"`
}

// RedisYAMLConfig matches the YAML structure with separate host/port fields.
type RedisYAMLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// ToRedisConfig converts RedisYAMLConfig to RedisConfig for use with NewRedisClient.
func (r RedisYAMLConfig) ToRedisConfig() RedisConfig {
	addr := "localhost:6379"
	if r.Host != "" {
		addr = fmt.Sprintf("%s:%d", r.Host, r.Port)
	}
	return RedisConfig{
		Addr:         addr,
		Password:     r.Password,
		DB:           r.DB,
		PoolSize:     100,
		MinIdleConns: 10,
	}
}

// GetEnvironment returns the effective environment name.
// Priority: APP_ENV env var > config file env field > "production" default.
func (c *AppConfig) GetEnvironment() string {
	if env := os.Getenv("APP_ENV"); env != "" {
		return env
	}
	if c.Env != "" {
		return c.Env
	}
	return EnvProduction
}

// IsLocal returns true if running in local development mode.
func (c *AppConfig) IsLocal() bool {
	return c.GetEnvironment() == EnvLocal
}

// Load reads a YAML config file, expands environment variables, and unmarshals it.
func Load(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	expanded := ExpandEnv(string(data))

	var cfg AppConfig
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// ApplyDefaults fills in zero values with environment-appropriate defaults.
func (c *AppConfig) ApplyDefaults() {
	env := c.GetEnvironment()

	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Mode == "" {
		if env == EnvLocal {
			c.Server.Mode = "debug"
		} else {
			c.Server.Mode = "release"
		}
	}
	if c.Log.Level == "" {
		if env == EnvLocal {
			c.Log.Level = "debug"
		} else {
			c.Log.Level = "info"
		}
	}
	if c.Log.Format == "" {
		if env == EnvLocal {
			c.Log.Format = "text"
		} else {
			c.Log.Format = "json"
		}
	}
}
```

- [ ] **Step 4: Add YAML tags to `DatabaseConfig` in `internal/config/database.go`**

Change the `DatabaseConfig` struct definition (line 13-22) to add YAML tags:

```go
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Username        string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Database        string        `yaml:"name"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	MaxOpenConns    int           `yaml:"max_connections"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}
```

- [ ] **Step 5: Do NOT modify `internal/config/redis.go`**

`RedisConfig` is only constructed via `RedisYAMLConfig.ToRedisConfig()` and `DefaultRedisConfig()`. It is never deserialized from YAML directly, so no YAML tags are needed. Adding them would create confusion.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-31 && GOPROXY=https://goproxy.cn,direct go test ./internal/config/ -v`
Expected: ALL PASS (including existing database_test.go and redis_test.go)

- [ ] **Step 7: Run full test suite to check for regressions**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-31 && GOPROXY=https://goproxy.cn,direct go test ./... 2>&1 | tail -20`
Expected: ALL PASS

- [ ] **Step 8: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go internal/config/database.go
git commit -m "feat(config): add unified AppConfig with env expansion and YAML loading

- Add ExpandEnv() for ${VAR:default} syntax
- Add AppConfig struct with environment detection (APP_ENV / env field)
- Add RedisYAMLConfig with host/port → Addr conversion
- Add YAML tags to DatabaseConfig for config file mapping
- Add ApplyDefaults() with environment-specific defaults (local vs production)"
```

Note: `RedisConfig` does NOT get YAML tags — it is constructed via `RedisYAMLConfig.ToRedisConfig()`, never deserialized from YAML directly.

---

## Task 2: Local Development Config Files

**Files:**
- Create: `configs/config.local.yaml`
- Modify: `configs/config.yaml`

- [ ] **Step 1: Create `configs/config.local.yaml`**

```yaml
# Local Development Configuration
# Usage: APP_ENV=local
# No external dependencies needed — mock services fallback automatically

env: local

server:
  port: 8080
  mode: debug

database:
  host: localhost
  port: 3306
  name: agent_infra
  user: root
  password: ""

redis:
  host: localhost
  port: 6379
  db: 0

log:
  level: debug
  format: text
```

- [ ] **Step 2: Update `configs/config.yaml`**

Replace the existing content with the production template that uses `${VAR:default}` syntax and matches the new `AppConfig` structure. Note: SLS env var names use `ALIBABA_CLOUD_ACCESS_KEY_ID` / `ALIBABA_CLOUD_ACCESS_KEY_SECRET` to match the existing `main.go` convention:

```yaml
# Production Configuration Template
# Override with environment variables for deployment

env: ${APP_ENV:production}

server:
  port: ${SERVER_PORT:8080}
  mode: ${GIN_MODE:release}

database:
  host: ${DB_HOST:localhost}
  port: ${DB_PORT:2881}
  name: ${DB_NAME:agent_infra}
  user: ${DB_USER:root}
  password: ${DB_PASSWORD:}
  max_connections: ${DB_MAX_CONN:100}

redis:
  host: ${REDIS_HOST:localhost}
  port: ${REDIS_PORT:6379}
  db: ${REDIS_DB:0}
  password: ${REDIS_PASSWORD:}

k8s:
  kubeconfig: ${KUBECONFIG:}
  namespace:
    control_plane: ${CONTROL_PLANE_NS:control-plane}
    sandbox: ${SANDBOX_NS:sandbox}

sls:
  endpoint: ${SLS_ENDPOINT:}
  project: ${SLS_PROJECT:}
  logstore: ${SLS_LOGSTORE:execution-logs}
  access_key: ${ALIBABA_CLOUD_ACCESS_KEY_ID:}
  access_secret: ${ALIBABA_CLOUD_ACCESS_KEY_SECRET:}

log:
  level: ${LOG_LEVEL:info}
  format: json
```

- [ ] **Step 3: Commit**

```bash
git add configs/config.yaml configs/config.local.yaml
git commit -m "feat(config): add local dev config and update production template"
```

---

## Task 3: Update main.go to Use Unified Config

**Files:**
- Modify: `cmd/control-plane/main.go`

This is the integration step. Replace the ad-hoc `Config` struct with `config.AppConfig`. Key changes:
- Remove old `Config` struct, `ToDatabaseConfig()`, `loadConfig()`
- Use `config.Load("configs/config.yaml")`
- `cfg.Database` goes directly to `config.NewDatabase()`
- SLS config from `cfg.SLS`
- Startup log shows environment name

- [ ] **Step 1: Update main.go**

Edit `cmd/control-plane/main.go`:

1. Remove imports: `"os"`, `"gopkg.in/yaml.v3"` (no longer needed)
2. Remove the entire `Config` struct (lines 20-32)
3. Remove the `ToDatabaseConfig()` method (lines 35-43)
4. Replace `main()` function body:

```go
func main() {
	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	cfg.ApplyDefaults()

	env := cfg.GetEnvironment()
	log.Printf("Starting control-plane in %q environment", env)

	// Set gin mode
	gin.SetMode(cfg.Server.Mode)

	// Initialize database (optional - will use mock if not available)
	var tenantSvc service.TenantService
	var templateSvc service.TemplateService
	var taskSvc service.TaskService
	var providerSvc service.ProviderService
	var capabilitySvc service.CapabilityService
	var interventionSvc service.InterventionService
	var monitoringSvc service.MonitoringService
	var db *config.Database
	db, err = config.NewDatabase(cfg.Database)
	if err != nil {
		log.Printf("Warning: failed to connect to database, using mock service: %v", err)
		tenantSvc = &mockTenantService{}
		templateSvc = &mockTemplateService{}
		taskSvc = &mockTaskService{}
		providerSvc = &mockProviderService{}
		capabilitySvc = &mockCapabilityService{}
		interventionSvc = &mockInterventionService{}
		monitoringSvc = &mockMonitoringService{}
	} else {
		// Auto-migrate models
		if err := db.AutoMigrate(&model.Tenant{}, &model.Template{}, &model.Task{}, &model.Provider{}, &model.Capability{}, &model.Intervention{}); err != nil {
			log.Printf("Warning: failed to auto-migrate: %v", err)
		}
		// Create real services with repositories
		tenantRepo := repository.NewTenantRepository(db.DB)
		tenantSvc = service.NewTenantService(tenantRepo)

		templateRepo := repository.NewTemplateRepository(db.DB)
		templateSvc = service.NewTemplateService(templateRepo)

		taskRepo := repository.NewTaskRepository(db.DB)
		taskSvc = service.NewTaskService(taskRepo)

		providerRepo := repository.NewProviderRepository(db.DB)
		providerSvc = service.NewProviderService(providerRepo)

		capabilityRepo := repository.NewCapabilityRepository(db.DB)
		capabilitySvc = service.NewCapabilityService(capabilityRepo)

		interventionRepo := repository.NewInterventionRepository(db.DB)
		interventionSvc = service.NewInterventionService(taskRepo, interventionRepo)
	}

	// Initialize monitoring
	monitoringHub := monitoring.NewHub()
	slsClient := monitoring.NewSLSClient(sls.Config{
		Endpoint:        cfg.SLS.Endpoint,
		AccessKeyID:     cfg.SLS.AccessKey,
		AccessKeySecret: cfg.SLS.AccessSecret,
		Project:         cfg.SLS.Project,
		LogStore:        cfg.SLS.LogStore,
	})
	monitoringSvc = service.NewMonitoringService(monitoringHub, slsClient)

	// Setup router (pass db for health checks - can be nil if not available)
	r := router.Setup(tenantSvc, templateSvc, taskSvc, providerSvc, capabilitySvc, monitoringSvc, monitoringHub, interventionSvc, db)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting control-plane server on %s (env=%s)", addr, env)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

5. Remove the `loadConfig()` function (lines 309-321)
6. Keep all mock service types unchanged
7. Delete the old `cmd/control-plane/config.yaml` file (replaced by `configs/config.yaml`)

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-31 && GOPROXY=https://goproxy.cn,direct go build ./cmd/control-plane/`
Expected: Builds successfully

- [ ] **Step 3: Run all tests**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-31 && GOPROXY=https://goproxy.cn,direct make test`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/control-plane/main.go
git rm cmd/control-plane/config.yaml
git commit -m "feat(config): integrate AppConfig into main.go entry point

- Replace ad-hoc Config struct with unified config.AppConfig
- SLS config from YAML instead of direct os.Getenv
- Startup log shows current environment"
```

---

## Task 4: Final Verification

- [ ] **Step 1: Run full test suite**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-31 && GOPROXY=https://goproxy.cn,direct make test`
Expected: ALL PASS

- [ ] **Step 2: Run lint**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-31 && make lint`
Expected: No errors

- [ ] **Step 3: Verify local mode config loading**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-31 && APP_ENV=local GOPROXY=https://goproxy.cn,direct go build -o /tmp/test-control-plane ./cmd/control-plane/`
Expected: Binary builds successfully

- [ ] **Step 4: Verify startup log output (quick manual check)**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-31 && APP_ENV=local timeout 3 /tmp/test-control-plane 2>&1 | head -5 || true`
Expected: Output contains `Starting control-plane in "local" environment`

- [ ] **Step 5: Update issue summary**

Update `docs/current/issues/issue-31-summary.md` scope checkboxes to mark all items as completed.

- [ ] **Step 6: Commit docs update**

```bash
git add docs/current/issues/issue-31-summary.md
git commit -m "docs: update issue-31 summary with completed scope"
```
