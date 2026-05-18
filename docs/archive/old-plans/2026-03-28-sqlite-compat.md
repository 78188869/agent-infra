# SQLite Compatibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the application support SQLite as a database backend, eliminating the dependency on external database services for local development.

**Architecture:** Add a `Driver` field to `DatabaseConfig` that selects between `mysql` and `sqlite`. For SQLite, use a file-based DSN. Replace all MySQL-specific GORM tags (`type:enum(...)`, `type:mediumtext`, `type:timestamp(3)`) with cross-database compatible alternatives — use `type:varchar(N)` for all enum fields (with application-layer validation via constants) and `type:text` for large text fields. The `gorm.io/driver/sqlite` package is already in `go.mod`.

**Tech Stack:** Go 1.21, GORM 1.25, gorm.io/driver/sqlite (already in go.mod), gorm.io/driver/mysql

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/config/database.go` | Modify | Add `Driver` field, driver-based factory for opening DB |
| `internal/config/database_test.go` | Modify | Update tests for new Driver field, add SQLite DSN tests |
| `internal/config/config.go` | Modify | No change needed (DatabaseConfig embedded) |
| `internal/model/base.go` | No change | `type:char(36)` is SQLite-compatible |
| `internal/model/tenant.go` | Modify | `enum(...)` → `varchar(20)` |
| `internal/model/user.go` | Modify | `enum(...)` → `varchar(20)`, `type:timestamp` → remove type tag |
| `internal/model/task.go` | Modify | `enum(...)` → `varchar(32)` |
| `internal/model/template.go` | Modify | `enum(...)` → `varchar(20)`, `mediumtext` → `text` |
| `internal/model/provider.go` | Modify | 4x `enum(...)` → `varchar(32)` |
| `internal/model/capability.go` | Modify | 3x `enum(...)` → `varchar(20)` |
| `internal/model/intervention.go` | Modify | 2x `enum(...)` → `varchar(20)` |
| `internal/model/execution_log.go` | Modify | `enum(...)` → `varchar(32)`, `timestamp(3)` → remove precision |
| `internal/model/api_key.go` | Modify | `enum(...)` → `varchar(20)`, `type:timestamp` → remove type tag |
| `configs/config.yaml` | Modify | Add `driver` field |
| `internal/model/compat_test.go` | Create | Integration test: SQLite AutoMigrate + CRUD validation |

---

### Task 1: Add Driver Selection to DatabaseConfig

**Files:**
- Modify: `internal/config/database.go`
- Modify: `internal/config/database_test.go`

- [ ] **Step 1: Write the failing test for Driver field and SQLite DSN**

Add tests in `internal/config/database_test.go`:

```go
func TestDatabaseConfig_Driver(t *testing.T) {
	cfg := DatabaseConfig{Driver: "sqlite"}
	if cfg.Driver != "sqlite" {
		t.Errorf("Driver = %v, want sqlite", cfg.Driver)
	}
}

func TestDatabaseConfig_SQLiteDSN(t *testing.T) {
	cfg := DatabaseConfig{
		Driver:   "sqlite",
		Database: "test.db",
	}
	got := cfg.DSN()
	if got != "test.db" {
		t.Errorf("SQLite DSN() = %v, want test.db", got)
	}
}

func TestDatabaseConfig_MySQLDSN(t *testing.T) {
	cfg := DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "testdb",
	}
	got := cfg.DSN()
	expected := "root:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	if got != expected {
		t.Errorf("MySQL DSN() = %v, want %v", got, expected)
	}
}

func TestDatabaseConfig_DefaultDriver(t *testing.T) {
	cfg := DatabaseConfig{}
	if cfg.Driver != "" {
		t.Errorf("Default Driver = %v, want empty (defaults to mysql)", cfg.Driver)
	}
}

func TestDatabaseConfig_IsSQLite(t *testing.T) {
	tests := []struct {
		driver string
		want   bool
	}{
		{"sqlite", true},
		{"mysql", false},
		{"", false},
	}
	for _, tt := range tests {
		cfg := DatabaseConfig{Driver: tt.driver}
		if got := cfg.IsSQLite(); got != tt.want {
			t.Errorf("IsSQLite(%q) = %v, want %v", tt.driver, got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run "TestDatabaseConfig_Driver|TestDatabaseConfig_SQLiteDSN|TestDatabaseConfig_MySQLDSN|TestDatabaseConfig_DefaultDriver|TestDatabaseConfig_IsSQLite" -v`
Expected: FAIL — `cfg.Driver`, `cfg.DSN()` for sqlite, `cfg.IsSQLite()` do not exist yet.

- [ ] **Step 3: Implement Driver field and DSN routing**

Modify `internal/config/database.go`:

```go
package config

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	Driver          string        `yaml:"driver"`
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Username        string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Database        string        `yaml:"name"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	MaxOpenConns    int           `yaml:"max_connections"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// Database wraps GORM DB with configuration.
type Database struct {
	*gorm.DB
	Config DatabaseConfig
}

// DefaultDatabaseConfig returns the default database configuration.
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Driver:          "mysql",
		Host:            "localhost",
		Port:            3306,
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
	}
}

// IsSQLite returns true if the configured driver is SQLite.
func (c DatabaseConfig) IsSQLite() bool {
	return c.Driver == "sqlite"
}

// DSN returns the database DSN string.
// For MySQL: user:pass@tcp(host:port)/db?params
// For SQLite: file path (e.g., "agent_infra.db")
func (c DatabaseConfig) DSN() string {
	if c.IsSQLite() {
		db := c.Database
		if db == "" {
			db = "agent_infra.db"
		}
		return db
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

// NewDatabase creates a new database connection with the given configuration.
func NewDatabase(cfg DatabaseConfig) (*Database, error) {
	var dialector gorm.Dialector
	if cfg.IsSQLite() {
		dialector = sqlite.Open(cfg.DSN())
	} else {
		dialector = mysql.Open(cfg.DSN())
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool (not applicable for SQLite in-memory/file mode,
	// but harmless to set)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	return &Database{
		DB:     db,
		Config: cfg,
	}, nil
}

// Close closes the database connection.
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping verifies the database connection is still alive.
func (d *Database) Ping() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: ALL PASS

- [ ] **Step 5: Update existing DSN tests for MySQL explicit driver**

In `internal/config/database_test.go`, add `Driver: "mysql"` to existing test cases in `TestDatabaseConfig_DSN`:

```go
func TestDatabaseConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		config   DatabaseConfig
		expected string
	}{
		{
			name: "basic DSN",
			config: DatabaseConfig{
				Driver:   "mysql",
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Password: "password",
				Database: "testdb",
			},
			expected: "root:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "DSN with custom port",
			config: DatabaseConfig{
				Driver:   "mysql",
				Host:     "db.example.com",
				Port:     3307,
				Username: "admin",
				Password: "secret123",
				Database: "production",
			},
			expected: "admin:secret123@tcp(db.example.com:3307)/production?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "DSN with empty password",
			config: DatabaseConfig{
				Driver:   "mysql",
				Host:     "127.0.0.1",
				Port:     3306,
				Username: "user",
				Password: "",
				Database: "mydb",
			},
			expected: "user:@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.DSN()
			if got != tt.expected {
				t.Errorf("DSN() = %v, want %v", got, tt.expected)
			}
		})
	}
}
```

- [ ] **Step 6: Run all config tests**

Run: `go test ./internal/config/ -v`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add internal/config/database.go internal/config/database_test.go
git commit -m "feat(config): add driver selection supporting MySQL and SQLite"
```

---

### Task 2: Replace MySQL-Specific Model Tags

**Files:**
- Modify: `internal/model/tenant.go`
- Modify: `internal/model/user.go`
- Modify: `internal/model/task.go`
- Modify: `internal/model/template.go`
- Modify: `internal/model/provider.go`
- Modify: `internal/model/capability.go`
- Modify: `internal/model/intervention.go`
- Modify: `internal/model/execution_log.go`
- Modify: `internal/model/api_key.go`

- [ ] **Step 1: Write failing test — SQLite AutoMigrate for all models**

Create `internal/model/compat_test.go`:

```go
package model

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSQLiteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open SQLite in-memory db: %v", err)
	}
	return db
}

func TestSQLite_AllModelsAutoMigrate(t *testing.T) {
	db := setupSQLiteTestDB(t)

	models := AllModels()
	for _, mdl := range models {
		if err := db.AutoMigrate(mdl); err != nil {
			t.Errorf("AutoMigrate(%T) failed on SQLite: %v", mdl, err)
		}
	}
}

func TestSQLite_TenantCRUD(t *testing.T) {
	db := setupSQLiteTestDB(t)
	if err := db.AutoMigrate(&Tenant{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// Create
	tenant := &Tenant{
		Name:             "TestTenant",
		QuotaCPU:         4,
		QuotaMemory:      16,
		QuotaConcurrency: 10,
		QuotaDailyTasks:  100,
		Status:           TenantStatusActive,
	}
	tenant.ID = generateUUID()
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("Create tenant failed: %v", err)
	}

	// Read
	var found Tenant
	if err := db.First(&found, "id = ?", tenant.ID).Error; err != nil {
		t.Fatalf("Read tenant failed: %v", err)
	}
	if found.Name != "TestTenant" {
		t.Errorf("Name = %q, want TestTenant", found.Name)
	}
	if found.Status != TenantStatusActive {
		t.Errorf("Status = %q, want %q", found.Status, TenantStatusActive)
	}

	// Update
	if err := db.Model(&found).Update("status", TenantStatusSuspended).Error; err != nil {
		t.Fatalf("Update tenant failed: %v", err)
	}

	// Verify update
	var updated Tenant
	if err := db.First(&updated, "id = ?", tenant.ID).Error; err != nil {
		t.Fatalf("Read updated tenant failed: %v", err)
	}
	if updated.Status != TenantStatusSuspended {
		t.Errorf("Status after update = %q, want %q", updated.Status, TenantStatusSuspended)
	}
}

func TestSQLite_TaskCRUD(t *testing.T) {
	db := setupSQLiteTestDB(t)
	if err := db.AutoMigrate(&Task{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	task := &Task{
		TenantID:   generateUUID(),
		CreatorID:  generateUUID(),
		ProviderID: generateUUID(),
		Name:       "TestTask",
		Status:     TaskStatusPending,
		Priority:   TaskPriorityHigh,
	}
	task.ID = generateUUID()
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("Create task failed: %v", err)
	}

	var found Task
	if err := db.First(&found, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("Read task failed: %v", err)
	}
	if found.Status != TaskStatusPending {
		t.Errorf("Status = %q, want %q", found.Status, TaskStatusPending)
	}
	if found.Priority != TaskPriorityHigh {
		t.Errorf("Priority = %q, want %q", found.Priority, TaskPriorityHigh)
	}
}

func TestSQLite_ProviderCRUD(t *testing.T) {
	db := setupSQLiteTestDB(t)
	if err := db.AutoMigrate(&Provider{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	provider := &Provider{
		ID:          generateUUID(),
		Scope:       ProviderScopeSystem,
		Name:        "test-provider",
		Type:        ProviderTypeClaudeCode,
		RuntimeType: RuntimeTypeCLI,
		Status:      ProviderStatusActive,
	}
	if err := db.Create(provider).Error; err != nil {
		t.Fatalf("Create provider failed: %v", err)
	}

	var found Provider
	if err := db.First(&found, "id = ?", provider.ID).Error; err != nil {
		t.Fatalf("Read provider failed: %v", err)
	}
	if found.Scope != ProviderScopeSystem {
		t.Errorf("Scope = %q, want %q", found.Scope, ProviderScopeSystem)
	}
	if found.Type != ProviderTypeClaudeCode {
		t.Errorf("Type = %q, want %q", found.Type, ProviderTypeClaudeCode)
	}
}

func TestSQLite_TemplateCRUD(t *testing.T) {
	db := setupSQLiteTestDB(t)
	if err := db.AutoMigrate(&Template{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	longSpec := ""
	for i := 0; i < 10000; i++ {
		longSpec += "x"
	}

	tmpl := &Template{
		TenantID:  generateUUID(),
		Name:      "test-template",
		Spec:      longSpec,
		SceneType: TemplateSceneTypeCoding,
		Status:    TemplateStatusDraft,
	}
	tmpl.ID = generateUUID()
	if err := db.Create(tmpl).Error; err != nil {
		t.Fatalf("Create template with long spec failed: %v", err)
	}

	var found Template
	if err := db.First(&found, "id = ?", tmpl.ID).Error; err != nil {
		t.Fatalf("Read template failed: %v", err)
	}
	if len(found.Spec) != 10000 {
		t.Errorf("Spec length = %d, want 10000", len(found.Spec))
	}
}

func TestSQLite_ExecutionLogCRUD(t *testing.T) {
	db := setupSQLiteTestDB(t)
	if err := db.AutoMigrate(&ExecutionLog{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	log := &ExecutionLog{
		TaskID:    generateUUID(),
		EventType: EventTypeStatusChange,
		EventName: "status_changed",
	}
	if err := db.Create(log).Error; err != nil {
		t.Fatalf("Create execution log failed: %v", err)
	}

	var found ExecutionLog
	if err := db.First(&found, "task_id = ?", log.TaskID).Error; err != nil {
		t.Fatalf("Read execution log failed: %v", err)
	}
	if found.EventType != EventTypeStatusChange {
		t.Errorf("EventType = %q, want %q", found.EventType, EventTypeStatusChange)
	}
}
```

- [ ] **Step 2: Run test to verify it fails on enum/mediumtext/timestamp(3)**

Run: `go test ./internal/model/ -run "TestSQLite_" -v`
Expected: FAIL — SQLite cannot parse `enum(...)`, `mediumtext`, or `timestamp(3)` type tags.

- [ ] **Step 3: Replace all MySQL-specific tags in model files**

**tenant.go** line 17:
```go
// Before:
Status string `gorm:"type:enum('active','suspended');default:'active'" json:"status"`
// After:
Status string `gorm:"type:varchar(20);default:'active'" json:"status"`
```

**user.go** lines 38-39:
```go
// Before:
Role   UserRole   `gorm:"type:enum('developer','admin','operator','reviewer');default:'developer'" json:"role"`
Status UserStatus `gorm:"type:enum('active','disabled');default:'active'" json:"status"`
// After:
Role   UserRole   `gorm:"type:varchar(20);default:'developer'" json:"role"`
Status UserStatus `gorm:"type:varchar(20);default:'active'" json:"status"`
```

**user.go** line 42 — remove `type:timestamp` (GORM handles time.Time natively):
```go
// Before:
LastLoginAt *time.Time    `gorm:"type:timestamp" json:"last_login_at"`
// After:
LastLoginAt *time.Time    `json:"last_login_at"`
```

**task.go** lines 35-36:
```go
// Before:
Status   string         `gorm:"type:enum(...);default:'pending';..." json:"status"`
Priority string         `gorm:"type:enum('high','normal','low');default:'normal'" json:"priority"`
// After:
Status   string         `gorm:"type:varchar(32);default:'pending';index:idx_tenant_status" json:"status"`
Priority string         `gorm:"type:varchar(20);default:'normal'" json:"priority"`
```

**template.go** lines 25-27:
```go
// Before:
Spec      string  `gorm:"type:mediumtext" json:"spec"`
SceneType string  `gorm:"type:enum('coding','ops','analysis','content','custom');default:'custom'" json:"scene_type"`
Status    string  `gorm:"type:enum('draft','published','deprecated');default:'draft'" json:"status"`
// After:
Spec      string  `gorm:"type:text" json:"spec"`
SceneType string  `gorm:"type:varchar(20);default:'custom'" json:"scene_type"`
Status    string  `gorm:"type:varchar(20);default:'draft'" json:"status"`
```

**provider.go** lines 54, 60, 72, 85:
```go
// Before:
Scope        ProviderScope  `gorm:"type:enum('system','tenant','user');..." json:"scope"`
Type         ProviderType   `gorm:"type:enum('claude_code','anthropic_compatible','openai_compatible','custom');..." json:"type"`
RuntimeType  RuntimeType    `gorm:"type:enum('cli','api','sdk');..." json:"runtime_type"`
Status       ProviderStatus `gorm:"type:enum('active','inactive','deprecated');..." json:"status"`
// After:
Scope        ProviderScope  `gorm:"type:varchar(20);not null;default:'system';uniqueIndex:uk_scope_name;index:idx_scope_tenant;index:idx_scope_user" json:"scope"`
Type         ProviderType   `gorm:"type:varchar(32);not null;index:idx_type_status" json:"type"`
RuntimeType  RuntimeType    `gorm:"type:varchar(20);default:'cli'" json:"runtime_type"`
Status       ProviderStatus `gorm:"type:varchar(20);default:'active';index:idx_type_status" json:"status"`
```

**capability.go** lines 44, 54, 57:
```go
// Before:
Type            CapabilityType   `gorm:"type:enum('tool','skill','agent_runtime');..." json:"type"`
PermissionLevel PermissionLevel  `gorm:"type:enum('public','restricted','admin_only');..." json:"permission_level"`
Status          CapabilityStatus `gorm:"type:enum('active','inactive');..." json:"status"`
// After:
Type            CapabilityType   `gorm:"type:varchar(20);not null;uniqueIndex:uk_tenant_type_name;index:idx_type_status" json:"type"`
PermissionLevel PermissionLevel  `gorm:"type:varchar(20);default:'public'" json:"permission_level"`
Status          CapabilityStatus `gorm:"type:varchar(20);default:'active';index:idx_type_status" json:"status"`
```

**intervention.go** lines 36, 42:
```go
// Before:
Action  InterventionAction `gorm:"type:enum('pause','resume','cancel','inject','modify');not null" json:"action"`
Status  InterventionStatus `gorm:"type:enum('pending','applied','failed');default:'pending'" json:"status"`
// After:
Action  InterventionAction `gorm:"type:varchar(20);not null" json:"action"`
Status  InterventionStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
```

**execution_log.go** lines 34, 42:
```go
// Before:
EventType EventType `gorm:"type:enum('status_change','tool_call','tool_result','llm_input','llm_output','error','heartbeat','intervention','metric','checkpoint');not null;index:idx_task_event" json:"event_type"`
Timestamp time.Time `gorm:"type:timestamp(3);default:CURRENT_TIMESTAMP(3);index:idx_task_time" json:"timestamp"`
// After:
EventType EventType `gorm:"type:varchar(32);not null;index:idx_task_event" json:"event_type"`
Timestamp time.Time `gorm:"index:idx_task_time" json:"timestamp"`
```

**api_key.go** line 36, and lines 31-32:
```go
// Before:
Status APIKeyStatus `gorm:"type:enum('active','revoked');default:'active';index" json:"status"`
ExpiresAt   *time.Time `gorm:"type:timestamp" json:"expires_at"`
LastUsedAt  *time.Time `gorm:"type:timestamp" json:"last_used_at"`
// After:
Status APIKeyStatus `gorm:"type:varchar(20);default:'active';index" json:"status"`
ExpiresAt   *time.Time `json:"expires_at"`
LastUsedAt  *time.Time `json:"last_used_at"`
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/model/ -v`
Expected: ALL PASS (both existing tests and new SQLite tests)

- [ ] **Step 5: Commit**

```bash
git add internal/model/compat_test.go internal/model/tenant.go internal/model/user.go internal/model/task.go internal/model/template.go internal/model/provider.go internal/model/capability.go internal/model/intervention.go internal/model/execution_log.go internal/model/api_key.go
git commit -m "feat(model): replace MySQL-specific tags with cross-DB compatible types"
```

---

### Task 3: Update Config YAML and Defaults

**Files:**
- Modify: `configs/config.yaml`

- [ ] **Step 1: Add driver field to config.yaml**

Add `driver` field under `database:` section:

```yaml
database:
  driver: ${DB_DRIVER:mysql}
  host: ${DB_HOST:localhost}
  port: ${DB_PORT:2881}
  name: ${DB_NAME:agent_infra}
  user: ${DB_USER:root}
  password: ${DB_PASSWORD:}
  max_connections: ${DB_MAX_CONN:100}
```

- [ ] **Step 2: Verify config loading test still passes**

Run: `go test ./internal/config/ -v`
Expected: ALL PASS

- [ ] **Step 3: Commit**

```bash
git add configs/config.yaml
git commit -m "feat(config): add DB_DRIVER env var for database driver selection"
```

---

### Task 4: End-to-End Integration Test

**Files:**
- Create: `internal/model/integration_test.go`

- [ ] **Step 1: Write integration test — full SQLite lifecycle with all models**

Create `internal/model/integration_test.go`:

```go
package model

import (
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	models := AllModels()
	for _, m := range models {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatalf("migrate %T: %v", m, err)
		}
	}
	return db
}

func TestSQLiteIntegration_FullLifecycle(t *testing.T) {
	db := newTestSQLiteDB(t)

	// 1. Create Tenant
	tenant := &Tenant{
		Name:             "IntegrationTenant",
		QuotaCPU:         8,
		QuotaMemory:      32,
		QuotaConcurrency: 20,
		QuotaDailyTasks:  500,
		Status:           TenantStatusActive,
	}
	tenant.ID = generateUUID()
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	// 2. Create User
	user := &User{
		TenantID:    tenant.ID.String(),
		Username:    "testuser",
		DisplayName: "Test User",
		Role:        UserRoleAdmin,
		Status:      UserStatusActive,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	// 3. Create Provider
	provider := &Provider{
		ID:          generateUUID(),
		Scope:       ProviderScopeTenant,
		TenantID:    strPtr(tenant.ID.String()),
		Name:        "test-provider",
		Type:        ProviderTypeClaudeCode,
		RuntimeType: RuntimeTypeCLI,
		Status:      ProviderStatusActive,
	}
	if err := db.Create(provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}

	// 4. Create Template
	tmpl := &Template{
		TenantID:   tenant.ID.String(),
		Name:       "integration-template",
		Version:    "1.0.0",
		Spec:       "spec: value",
		SceneType:  TemplateSceneTypeCoding,
		Status:     TemplateStatusPublished,
		ProviderID: strPtr(provider.ID),
	}
	tmpl.ID = generateUUID()
	if err := db.Create(tmpl).Error; err != nil {
		t.Fatalf("create template: %v", err)
	}

	// 5. Create Task
	task := &Task{
		TenantID:   tenant.ID.String(),
		TemplateID: strPtr(tmpl.ID.String()),
		CreatorID:  user.ID,
		ProviderID: provider.ID,
		Name:       "integration-task",
		Status:     TaskStatusPending,
		Priority:   TaskPriorityNormal,
		Params:     datatypes.JSON(`{"key":"value"}`),
	}
	task.ID = generateUUID()
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	// 6. Create ExecutionLog
	execLog := &ExecutionLog{
		TaskID:    task.ID.String(),
		EventType: EventTypeStatusChange,
		EventName: "created",
		Content:   datatypes.JSON(`{"from":"","to":"pending"}`),
	}
	if err := db.Create(execLog).Error; err != nil {
		t.Fatalf("create execution log: %v", err)
	}

	// 7. Create Intervention
	intervention := &Intervention{
		TaskID:     task.ID.String(),
		OperatorID: user.ID,
		Action:     InterventionActionPause,
		Reason:     "testing",
		Status:     InterventionStatusPending,
	}
	intervention.ID = generateUUID()
	if err := db.Create(intervention).Error; err != nil {
		t.Fatalf("create intervention: %v", err)
	}

	// 8. Create Capability
	capability := &Capability{
		ID:              generateUUID(),
		Type:            CapabilityTypeTool,
		Name:            "test-tool",
		PermissionLevel: PermissionLevelPublic,
		Status:          CapabilityStatusActive,
	}
	if err := db.Create(capability).Error; err != nil {
		t.Fatalf("create capability: %v", err)
	}

	// 9. Create APIKey
	apiKey := &APIKey{
		ID:        generateUUID(),
		UserID:    user.ID,
		KeyHash:   "abc123hash",
		KeyPrefix: "ak_test",
		Name:      "test-key",
		Status:    APIKeyStatusActive,
	}
	if err := db.Create(apiKey).Error; err != nil {
		t.Fatalf("create api key: %v", err)
	}

	// 10. Verify all data is retrievable
	var count int64
	db.Model(&Tenant{}).Where("id = ?", tenant.ID).Count(&count)
	if count != 1 {
		t.Errorf("tenant count = %d, want 1", count)
	}
	db.Model(&User{}).Where("id = ?", user.ID).Count(&count)
	if count != 1 {
		t.Errorf("user count = %d, want 1", count)
	}
	db.Model(&Task{}).Where("id = ?", task.ID).Count(&count)
	if count != 1 {
		t.Errorf("task count = %d, want 1", count)
	}

	// 11. Test status update (enum validation via app layer)
	if err := db.Model(task).Update("status", TaskStatusRunning).Error; err != nil {
		t.Fatalf("update task status: %v", err)
	}
	var updatedTask Task
	db.First(&updatedTask, "id = ?", task.ID)
	if updatedTask.Status != TaskStatusRunning {
		t.Errorf("task status = %q, want %q", updatedTask.Status, TaskStatusRunning)
	}

	// 12. Test soft delete
	now := time.Now()
	user.DeletedAt = gorm.DeletedAt{Time: now, Valid: true}
	db.Save(user)
	var deletedUser User
	err := db.First(&deletedUser, "id = ?", user.ID).Error
	if err == nil {
		t.Error("expected error for soft-deleted user, got nil")
	}
}

func TestSQLite_InterventionValidation(t *testing.T) {
	db := newTestSQLiteDB(t)

	// Verify enum values are still valid at app layer
	tests := []struct {
		action  InterventionAction
		isValid bool
	}{
		{InterventionActionPause, true},
		{InterventionActionResume, true},
		{InterventionActionCancel, true},
		{InterventionActionInject, true},
		{InterventionActionModify, true},
	}
	for _, tt := range tests {
		iv := &Intervention{
			TaskID:     generateUUID(),
			OperatorID: generateUUID(),
			Action:     tt.action,
			Status:     InterventionStatusPending,
		}
		iv.ID = generateUUID()
		if err := db.Create(iv).Error; err != nil {
			t.Errorf("create intervention with action %q: %v", tt.action, err)
		}
	}
}

func TestSQLite_SoftDelete(t *testing.T) {
	db := newTestSQLiteDB(t)

	tenant := &Tenant{
		Name:   "SoftDeleteTenant",
		Status: TenantStatusActive,
	}
	tenant.ID = generateUUID()
	db.Create(tenant)

	// Soft delete
	db.Delete(tenant)

	// Should not find with normal query
	var found Tenant
	err := db.First(&found, "id = ?", tenant.ID).Error
	if err == nil {
		t.Error("expected not to find soft-deleted tenant")
	}

	// Should find with unscoped
	var unscoped Tenant
	err = db.Unscoped().First(&unscoped, "id = ?", tenant.ID).Error
	if err != nil {
		t.Errorf("expected to find soft-deleted tenant with unscoped: %v", err)
	}
	if unscoped.DeletedAt.Valid == false {
		t.Error("expected DeletedAt to be set")
	}
}

// helper
func strPtr(s string) *string { return &s }
```

- [ ] **Step 2: Run integration tests**

Run: `go test ./internal/model/ -run "TestSQLiteIntegration_|TestSQLite_Intervention|TestSQLite_SoftDelete" -v`
Expected: ALL PASS

- [ ] **Step 3: Run ALL model tests**

Run: `go test ./internal/model/ -v`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add internal/model/integration_test.go
git commit -m "test(model): add SQLite integration tests covering full lifecycle"
```

---

### Task 5: Run Full Test Suite and Lint

**Files:** No changes, verification only.

- [ ] **Step 1: Run full test suite**

Run: `make test`
Expected: ALL PASS

- [ ] **Step 2: Run lint**

Run: `make lint`
Expected: No errors

- [ ] **Step 3: Run coverage**

Run: `go test -cover ./internal/...`
Expected: No regressions in coverage

---

### Task 6: Update Knowledge Module Documentation

**Files:**
- Modify: `docs/knowledge/database.md`

- [ ] **Step 1: Update database.md to document SQLite support**

Add a section after §3.1 (技术选型):

```markdown
### 3.1.1 本地开发支持（SQLite）

| 组件 | 选型 | 说明 |
|------|------|------|
| 本地数据库 | SQLite 3 | 零进程嵌入式数据库，仅用于本地开发 |
| 配置 | `DB_DRIVER=sqlite` | 通过环境变量切换 |

**切换方式**：
- 设置环境变量 `DB_DRIVER=sqlite`，`DB_NAME` 指定文件路径（默认 `agent_infra.db`）
- Model 使用 `varchar` 替代 `enum`，在应用层通过常量校验合法性
```

Also update §4.2 to add SQLite config example:

```yaml
# SQLite (local development)
database:
  driver: sqlite
  name: agent_infra.db
```

Update Change History table with new entry.

- [ ] **Step 2: Commit**

```bash
git add docs/knowledge/database.md
git commit -m "docs(database): document SQLite support for local development"
```
