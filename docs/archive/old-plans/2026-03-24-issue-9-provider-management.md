# Plan: MVP Phase 5 - Provider Management System

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Provider management system with three-tier scope (system/tenant/user), provider selection priority, and connection testing.

**Architecture:** Layered architecture following existing patterns (Handler → Service → Repository). Provider model already exists, need to implement full CRUD API, scope-based filtering, default provider selection chain, and connection testing.

**Tech Stack:** Go 1.22 + Gin + GORM, following existing service/repository patterns

---

## Context

**Issue**: #9 - MVP Phase 5 - Provider Management System
**Status**: Not Started
**Dependencies**:
- Issue #5 (Backend Core API) - ✅ Completed
- Issue #6 (Database Models) - ✅ Completed

**背景**：根据 TRD 第4.1.1节，实现 Provider 管理系统，支持多模型/Agent运行时切换（类似 cc switch）。

**已完成的基础设施**：
- `internal/model/provider.go` - Provider 模型已定义完整
- `internal/model/user_provider_default.go` - 用户默认 Provider 设置
- 数据库表结构已设计（providers, user_provider_defaults）

**需要实现**：
1. Provider CRUD API（7个端点）
2. 三层作用域过滤
3. Provider 选择优先级链
4. 连接测试功能
5. 系统预置 Provider 初始化

## Objectives

1. 实现 Provider CRUD API（GET/POST/PUT/DELETE）
2. 实现三层作用域管理（system/tenant/user）
3. 实现 Provider 选择优先级（任务指定 > 用户默认 > 租户默认 > 系统默认）
4. 实现连接测试功能
5. 实现系统预置 Provider 初始化
6. 单元测试覆盖率 > 80%

## Knowledge Required

- [x] docs/knowledge/provider.md - Provider 管理机制
- [x] docs/v1.0-mvp/TRD.md §4.1.1, §6.2.9 - Provider 技术设计
- [x] internal/model/provider.go - Provider 模型定义
- [x] internal/service/tenant_service.go - Service 层模式参考

## File Structure

```
internal/
├── model/
│   ├── provider.go              # 已存在
│   └── user_provider_default.go # 已存在
├── repository/
│   ├── provider_repo.go         # 新建
│   └── provider_repo_test.go    # 新建
├── service/
│   ├── provider_service.go      # 新建
│   └── provider_service_test.go # 新建
├── api/handler/
│   ├── provider.go              # 新建
│   └── provider_test.go         # 新建
└── api/router/
    └── router.go                # 修改：添加 Provider 路由

internal/seed/
└── providers.go                 # 新建：系统预置 Provider
```

## Tasks

### Task 1: Provider Repository

**Files:**
- Create: `internal/repository/provider_repo.go`
- Test: `internal/repository/provider_repo_test.go`

- [ ] **Step 1: Write the failing test for ProviderRepository interface**

```go
// internal/repository/provider_repo_test.go
package repository

import (
    "context"
    "testing"

    "github.com/example/agent-infra/internal/model"
    "github.com/google/uuid"
)

func TestProviderRepository_Create(t *testing.T) {
    // Test implementation
}

func TestProviderRepository_GetByID(t *testing.T) {
    // Test implementation
}

func TestProviderRepository_List(t *testing.T) {
    // Test implementation with scope filtering
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/repository/... -run TestProviderRepository -v`
Expected: FAIL with "undefined: ProviderRepository"

- [ ] **Step 3: Write ProviderRepository interface and implementation**

```go
// internal/repository/provider_repo.go
package repository

import (
    "context"

    "github.com/example/agent-infra/internal/model"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// ProviderFilter represents filtering options for listing providers.
type ProviderFilter struct {
    Page     int    `form:"page"`
    PageSize int    `form:"page_size"`
    Scope    string `form:"scope"`    // system, tenant, user
    TenantID string `form:"tenant_id"`
    UserID   string `form:"user_id"`
    Type     string `form:"type"`
    Status   string `form:"status"`
    Search   string `form:"search"`
}

func (f *ProviderFilter) SetDefaults() {
    if f.Page <= 0 {
        f.Page = 1
    }
    if f.PageSize <= 0 {
        f.PageSize = 10
    }
    if f.PageSize > 100 {
        f.PageSize = 100
    }
}

func (f *ProviderFilter) Offset() int {
    return (f.Page - 1) * f.PageSize
}

// ProviderRepository defines the interface for provider data access operations.
type ProviderRepository interface {
    Create(ctx context.Context, provider *model.Provider) error
    GetByID(ctx context.Context, id uuid.UUID) (*model.Provider, error)
    List(ctx context.Context, filter ProviderFilter) ([]*model.Provider, int64, error)
    Update(ctx context.Context, provider *model.Provider) error
    Delete(ctx context.Context, id uuid.UUID) error

    // Scope-specific queries
    GetByScopeAndName(ctx context.Context, scope model.ProviderScope, tenantID, userID *string, name string) (*model.Provider, error)
    GetDefaultProvider(ctx context.Context, scope model.ProviderScope, tenantID, userID *string) (*model.Provider, error)

    // User default provider
    SetUserDefaultProvider(ctx context.Context, userID, providerID string) error
    GetUserDefaultProvider(ctx context.Context, userID string) (*model.Provider, error)
}

type providerRepository struct {
    db *gorm.DB
}

func NewProviderRepository(db *gorm.DB) ProviderRepository {
    return &providerRepository{db: db}
}

// Implement all interface methods...
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/repository/... -run TestProviderRepository -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/repository/provider_repo.go internal/repository/provider_repo_test.go
git commit -m "feat(repository): add ProviderRepository with scope filtering"
```

---

### Task 2: Provider Service

**Files:**
- Create: `internal/service/provider_service.go`
- Test: `internal/service/provider_service_test.go`

- [ ] **Step 1: Write the failing test for ProviderService interface**

```go
// internal/service/provider_service_test.go
package service

import (
    "context"
    "testing"

    "github.com/example/agent-infra/internal/model"
    "github.com/google/uuid"
)

func TestProviderService_Create(t *testing.T) {
    // Test create provider with scope validation
}

func TestProviderService_GetProviderSelectionChain(t *testing.T) {
    // Test priority: specified > user_default > tenant_default > system_default
}

func TestProviderService_TestConnection(t *testing.T) {
    // Test connection testing functionality
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/... -run TestProviderService -v`
Expected: FAIL

- [ ] **Step 3: Write ProviderService interface and implementation**

```go
// internal/service/provider_service.go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/example/agent-infra/internal/model"
    "github.com/example/agent-infra/internal/repository"
    "github.com/example/agent-infra/pkg/errors"
    "github.com/google/uuid"
)

// CreateProviderRequest represents the request to create a new provider.
type CreateProviderRequest struct {
    Scope       model.ProviderScope `json:"scope" binding:"required,oneof=system tenant user"`
    TenantID    *string             `json:"tenant_id"`
    UserID      *string             `json:"user_id"`
    Name        string              `json:"name" binding:"required"`
    Type        model.ProviderType  `json:"type" binding:"required"`
    Description string              `json:"description"`

    // API Configuration
    APIEndpoint string          `json:"api_endpoint"`
    APIKeyRef   string          `json:"api_key_ref"`
    ModelMapping json.RawMessage `json:"model_mapping"`

    // Runtime Configuration
    RuntimeType    model.RuntimeType `json:"runtime_type"`
    RuntimeImage   string            `json:"runtime_image"`
    RuntimeCommand json.RawMessage   `json:"runtime_command"`

    // cc switch compatible
    EnvVars        json.RawMessage `json:"env_vars"`
    Permissions    json.RawMessage `json:"permissions"`
    EnabledPlugins json.RawMessage `json:"enabled_plugins"`
    ExtraParams    json.RawMessage `json:"extra_params"`
}

// UpdateProviderRequest represents the request to update a provider.
type UpdateProviderRequest struct {
    Name           *string            `json:"name"`
    Type           *model.ProviderType `json:"type"`
    Description    *string            `json:"description"`
    APIEndpoint    *string            `json:"api_endpoint"`
    APIKeyRef      *string            `json:"api_key_ref"`
    ModelMapping   json.RawMessage    `json:"model_mapping"`
    RuntimeType    *model.RuntimeType `json:"runtime_type"`
    RuntimeImage   *string            `json:"runtime_image"`
    RuntimeCommand json.RawMessage    `json:"runtime_command"`
    EnvVars        json.RawMessage    `json:"env_vars"`
    Permissions    json.RawMessage    `json:"permissions"`
    EnabledPlugins json.RawMessage    `json:"enabled_plugins"`
    ExtraParams    json.RawMessage    `json:"extra_params"`
    Status         *model.ProviderStatus `json:"status"`
}

// ProviderFilter represents filtering options.
type ProviderFilter struct {
    Page     int    `form:"page"`
    PageSize int    `form:"page_size"`
    Scope    string `form:"scope"`
    TenantID string `form:"tenant_id"`
    UserID   string `form:"user_id"`
    Type     string `form:"type"`
    Status   string `form:"status"`
    Search   string `form:"search"`
}

// ConnectionTestResult represents the result of a connection test.
type ConnectionTestResult struct {
    Success      bool     `json:"success"`
    Message      string   `json:"message"`
    ResponseTime int64    `json:"response_time_ms"`
    Models       []string `json:"available_models,omitempty"`
}

// ProviderService defines the interface for provider business operations.
type ProviderService interface {
    Create(ctx context.Context, req *CreateProviderRequest) (*model.Provider, error)
    GetByID(ctx context.Context, id string) (*model.Provider, error)
    List(ctx context.Context, filter *ProviderFilter) ([]*model.Provider, int64, error)
    Update(ctx context.Context, id string, req *UpdateProviderRequest) error
    Delete(ctx context.Context, id string) error
    TestConnection(ctx context.Context, id string) (*ConnectionTestResult, error)

    // Provider selection
    GetAvailableProviders(ctx context.Context, tenantID, userID string) ([]*model.Provider, error)
    ResolveProvider(ctx context.Context, specifiedProviderID, tenantID, userID string) (*model.Provider, error)
    SetDefaultProvider(ctx context.Context, providerID, userID string) error
}

type providerService struct {
    repo repository.ProviderRepository
}

func NewProviderService(repo repository.ProviderRepository) ProviderService {
    return &providerService{repo: repo}
}

// Implement all interface methods with business logic...
```

- [ ] **Step 4: Implement provider selection priority chain**

```go
// ResolveProvider resolves the provider based on priority chain.
// Priority: specified > user_default > tenant_default > system_default
func (s *providerService) ResolveProvider(ctx context.Context, specifiedProviderID, tenantID, userID string) (*model.Provider, error) {
    // 1. If specified, use it
    if specifiedProviderID != "" {
        provider, err := s.repo.GetByID(ctx, uuid.MustParse(specifiedProviderID))
        if err == nil && provider.IsActive() {
            return provider, nil
        }
    }

    // 2. Try user default
    if userID != "" {
        provider, err := s.repo.GetUserDefaultProvider(ctx, userID)
        if err == nil && provider != nil && provider.IsActive() {
            return provider, nil
        }
    }

    // 3. Try tenant default
    if tenantID != "" {
        tid := tenantID
        provider, err := s.repo.GetDefaultProvider(ctx, model.ProviderScopeTenant, &tid, nil)
        if err == nil && provider != nil && provider.IsActive() {
            return provider, nil
        }
    }

    // 4. Fall back to system default
    provider, err := s.repo.GetDefaultProvider(ctx, model.ProviderScopeSystem, nil, nil)
    if err != nil {
        return nil, errors.NewNotFoundError("no available provider found")
    }
    return provider, nil
}
```

- [ ] **Step 5: Implement connection testing**

```go
// TestConnection tests the provider connection by making a simple API call.
func (s *providerService) TestConnection(ctx context.Context, id string) (*ConnectionTestResult, error) {
    providerID, err := uuid.Parse(id)
    if err != nil {
        return nil, errors.NewBadRequestError("invalid provider ID format")
    }

    provider, err := s.repo.GetByID(ctx, providerID)
    if err != nil {
        return nil, err
    }

    if provider.APIEndpoint == "" {
        return &ConnectionTestResult{
            Success: false,
            Message: "API endpoint not configured",
        }, nil
    }

    // Make HTTP request to test connection
    start := time.Now()
    client := &http.Client{Timeout: 10 * time.Second}

    req, err := http.NewRequestWithContext(ctx, "GET", provider.APIEndpoint+"/v1/models", nil)
    if err != nil {
        return &ConnectionTestResult{
            Success: false,
            Message: fmt.Sprintf("Failed to create request: %v", err),
        }, nil
    }

    resp, err := client.Do(req)
    elapsed := time.Since(start).Milliseconds()

    if err != nil {
        return &ConnectionTestResult{
            Success:      false,
            Message:      fmt.Sprintf("Connection failed: %v", err),
            ResponseTime: elapsed,
        }, nil
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        return &ConnectionTestResult{
            Success:      true,
            Message:      "Connection successful",
            ResponseTime: elapsed,
        }, nil
    }

    return &ConnectionTestResult{
        Success:      false,
        Message:      fmt.Sprintf("API returned status %d", resp.StatusCode),
        ResponseTime: elapsed,
    }, nil
}
```

- [ ] **Step 6: Run tests to verify**

Run: `go test ./internal/service/... -run TestProviderService -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/service/provider_service.go internal/service/provider_service_test.go
git commit -m "feat(service): add ProviderService with selection priority and connection testing"
```

---

### Task 3: Provider Handler

**Files:**
- Create: `internal/api/handler/provider.go`
- Test: `internal/api/handler/provider_test.go`

- [ ] **Step 1: Write the failing test for ProviderHandler**

```go
// internal/api/handler/provider_test.go
package handler

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestProviderHandler_Create(t *testing.T) {
    // Test POST /api/v1/providers
}

func TestProviderHandler_List(t *testing.T) {
    // Test GET /api/v1/providers
}

func TestProviderHandler_TestConnection(t *testing.T) {
    // Test POST /api/v1/providers/:id/test
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api/handler/... -run TestProviderHandler -v`
Expected: FAIL

- [ ] **Step 3: Write ProviderHandler implementation**

```go
// internal/api/handler/provider.go
package handler

import (
    "github.com/example/agent-infra/internal/api/response"
    "github.com/example/agent-infra/internal/service"
    "github.com/gin-gonic/gin"
)

type ProviderHandler struct {
    service service.ProviderService
}

func NewProviderHandler(svc service.ProviderService) *ProviderHandler {
    return &ProviderHandler{service: svc}
}

// Create handles POST /api/v1/providers
func (h *ProviderHandler) Create(c *gin.Context) {
    var req service.CreateProviderRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "invalid request body: "+err.Error())
        return
    }

    provider, err := h.service.Create(c.Request.Context(), &req)
    if err != nil {
        handleError(c, err)
        return
    }

    response.Created(c, provider)
}

// GetByID handles GET /api/v1/providers/:id
func (h *ProviderHandler) GetByID(c *gin.Context) {
    id := c.Param("id")

    provider, err := h.service.GetByID(c.Request.Context(), id)
    if err != nil {
        handleError(c, err)
        return
    }

    response.Success(c, provider)
}

// List handles GET /api/v1/providers
func (h *ProviderHandler) List(c *gin.Context) {
    var filter service.ProviderFilter
    if err := c.ShouldBindQuery(&filter); err != nil {
        response.BadRequest(c, "invalid query parameters: "+err.Error())
        return
    }

    if filter.Page <= 0 {
        filter.Page = 1
    }
    if filter.PageSize <= 0 {
        filter.PageSize = 10
    }
    if filter.PageSize > 100 {
        filter.PageSize = 100
    }

    providers, total, err := h.service.List(c.Request.Context(), &filter)
    if err != nil {
        handleError(c, err)
        return
    }

    response.Paginated(c, providers, total, filter.Page, filter.PageSize)
}

// Update handles PUT /api/v1/providers/:id
func (h *ProviderHandler) Update(c *gin.Context) {
    id := c.Param("id")

    var req service.UpdateProviderRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "invalid request body: "+err.Error())
        return
    }

    err := h.service.Update(c.Request.Context(), id, &req)
    if err != nil {
        handleError(c, err)
        return
    }

    response.Success(c, gin.H{"message": "provider updated successfully"})
}

// Delete handles DELETE /api/v1/providers/:id
func (h *ProviderHandler) Delete(c *gin.Context) {
    id := c.Param("id")

    err := h.service.Delete(c.Request.Context(), id)
    if err != nil {
        handleError(c, err)
        return
    }

    response.Success(c, gin.H{"message": "provider deleted successfully"})
}

// TestConnection handles POST /api/v1/providers/:id/test
func (h *ProviderHandler) TestConnection(c *gin.Context) {
    id := c.Param("id")

    result, err := h.service.TestConnection(c.Request.Context(), id)
    if err != nil {
        handleError(c, err)
        return
    }

    response.Success(c, result)
}

// GetAvailable handles GET /api/v1/providers/available
func (h *ProviderHandler) GetAvailable(c *gin.Context) {
    // Get tenant_id and user_id from context (set by auth middleware)
    tenantID, _ := c.Get("tenant_id")
    userID, _ := c.Get("user_id")

    var tid, uid string
    if tenantID != nil {
        tid = tenantID.(string)
    }
    if userID != nil {
        uid = userID.(string)
    }

    providers, err := h.service.GetAvailableProviders(c.Request.Context(), tid, uid)
    if err != nil {
        handleError(c, err)
        return
    }

    response.Success(c, providers)
}

// SetDefault handles PUT /api/v1/providers/:id/set-default
func (h *ProviderHandler) SetDefault(c *gin.Context) {
    id := c.Param("id")

    // Get user_id from context
    userID, exists := c.Get("user_id")
    if !exists {
        response.Unauthorized(c, "user not authenticated")
        return
    }

    err := h.service.SetDefaultProvider(c.Request.Context(), id, userID.(string))
    if err != nil {
        handleError(c, err)
        return
    }

    response.Success(c, gin.H{"message": "default provider set successfully"})
}
```

- [ ] **Step 4: Run tests to verify**

Run: `go test ./internal/api/handler/... -run TestProviderHandler -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/handler/provider.go internal/api/handler/provider_test.go
git commit -m "feat(handler): add ProviderHandler with CRUD and connection testing"
```

---

### Task 4: Router Integration

**Files:**
- Modify: `internal/api/router/router.go`

- [ ] **Step 1: Write test for router with provider routes**

```go
// In router_test.go, add test for provider routes
func TestSetup_ProviderRoutes(t *testing.T) {
    // Test that provider routes are properly registered
}
```

- [ ] **Step 2: Update router.go to add Provider routes**

```go
// In internal/api/router/router.go, update Setup function:

func Setup(
    tenantSvc service.TenantService,
    templateSvc service.TemplateService,
    taskSvc service.TaskService,
    providerSvc service.ProviderService, // Add
    db DBChecker,
) *gin.Engine {
    // ... existing code ...

    // Provider routes
    providerHandler := handler.NewProviderHandler(providerSvc)
    providers := v1.Group("/providers")
    {
        providers.POST("", providerHandler.Create)
        providers.GET("", providerHandler.List)
        providers.GET("/available", providerHandler.GetAvailable)
        providers.GET("/:id", providerHandler.GetByID)
        providers.PUT("/:id", providerHandler.Update)
        providers.DELETE("/:id", providerHandler.Delete)
        providers.POST("/:id/test", providerHandler.TestConnection)
        providers.PUT("/:id/set-default", providerHandler.SetDefault)
    }

    return r
}
```

- [ ] **Step 3: Run tests to verify**

Run: `go test ./internal/api/router/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/api/router/router.go internal/api/router/router_test.go
git commit -m "feat(router): add Provider routes to API router"
```

---

### Task 5: System Preset Providers

**Files:**
- Create: `internal/seed/providers.go`

- [ ] **Step 1: Write seed data initialization**

```go
// internal/seed/providers.go
package seed

import (
    "context"
    "log"

    "github.com/example/agent-infra/internal/model"
    "gorm.io/gorm"
    "gorm.io/datatypes"
)

// SystemProviders defines the system preset providers.
var SystemProviders = []model.Provider{
    {
        Scope:        model.ProviderScopeSystem,
        Name:         "claude-code",
        Type:         model.ProviderTypeClaudeCode,
        Description:  "Official Claude Code CLI from Anthropic",
        APIEndpoint:  "https://api.anthropic.com",
        RuntimeType:  model.RuntimeTypeCLI,
        Status:       model.ProviderStatusActive,
    },
    {
        Scope:        model.ProviderScopeSystem,
        Name:         "zhipu-glm",
        Type:         model.ProviderTypeAnthropicCompat,
        Description:  "Zhipu GLM via Anthropic compatible API",
        APIEndpoint:  "https://open.bigmodel.cn/api/anthropic",
        APIKeyRef:    "zhipu-api-key",
        ModelMapping: datatypes.JSON(`{"default":"glm-5","opus":"glm-5","sonnet":"glm-4.7","haiku":"glm-4.5-air"}`),
        RuntimeType:  model.RuntimeTypeCLI,
        Status:       model.ProviderStatusActive,
    },
    {
        Scope:        model.ProviderScopeSystem,
        Name:         "deepseek",
        Type:         model.ProviderTypeAnthropicCompat,
        Description:  "DeepSeek via Anthropic compatible API",
        APIEndpoint:  "https://api.deepseek.com",
        APIKeyRef:    "deepseek-api-key",
        ModelMapping: datatypes.JSON(`{"default":"deepseek-chat","opus":"deepseek-reasoner","sonnet":"deepseek-chat","haiku":"deepseek-chat"}`),
        RuntimeType:  model.RuntimeTypeCLI,
        Status:       model.ProviderStatusActive,
    },
}

// SeedProviders initializes system providers if they don't exist.
func SeedProviders(db *gorm.DB) error {
    ctx := context.Background()

    for _, provider := range SystemProviders {
        var existing model.Provider
        err := db.WithContext(ctx).
            Where("scope = ? AND name = ?", model.ProviderScopeSystem, provider.Name).
            First(&existing).Error

        if err == gorm.ErrRecordNotFound {
            if err := db.WithContext(ctx).Create(&provider).Error; err != nil {
                return err
            }
            log.Printf("Seeded provider: %s", provider.Name)
        } else if err != nil {
            return err
        }
    }

    return nil
}
```

- [ ] **Step 2: Write test for seed function**

```go
// internal/seed/providers_test.go
package seed

import (
    "testing"

    "github.com/example/agent-infra/internal/model"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func TestSeedProviders(t *testing.T) {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        t.Fatalf("Failed to connect to database: %v", err)
    }

    // Migrate schema
    db.AutoMigrate(&model.Provider{})

    // Run seed
    err = SeedProviders(db)
    if err != nil {
        t.Fatalf("SeedProviders failed: %v", err)
    }

    // Verify providers were created
    var count int64
    db.Model(&model.Provider{}).Where("scope = ?", model.ProviderScopeSystem).Count(&count)
    if count != int64(len(SystemProviders)) {
        t.Errorf("Expected %d system providers, got %d", len(SystemProviders), count)
    }

    // Test idempotency
    err = SeedProviders(db)
    if err != nil {
        t.Fatalf("Second SeedProviders failed: %v", err)
    }

    db.Model(&model.Provider{}).Where("scope = ?", model.ProviderScopeSystem).Count(&count)
    if count != int64(len(SystemProviders)) {
        t.Errorf("Expected %d system providers after re-seed, got %d", len(SystemProviders), count)
    }
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/seed/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/seed/providers.go internal/seed/providers_test.go
git commit -m "feat(seed): add system preset providers (Claude Code, GLM, DeepSeek)"
```

---

### Task 6: Update Migration

**Files:**
- Modify: `internal/migration/migrator.go`

- [ ] **Step 1: Ensure providers and user_provider_defaults tables are in migration**

```go
// In internal/migration/migrator.go, verify models are included:
func Migrate(db *gorm.DB) error {
    return db.AutoMigrate(
        // ... existing models ...
        &model.Provider{},
        &model.UserProviderDefault{},
    )
}
```

- [ ] **Step 2: Run migration test**

Run: `go test ./internal/migration/... -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/migration/migrator.go
git commit -m "feat(migration): ensure Provider tables are migrated"
```

---

### Task 7: Update Knowledge Documentation

**Files:**
- Modify: `docs/knowledge/provider.md`

- [ ] **Step 1: Update Change History in provider.md**

Add entry to Change History:
```markdown
| Date | Version | Issue | PRD Ref | TRD Ref | Changes |
|------|---------|-------|---------|---------|---------|
| 2026-03-24 | v1.1 | #9 | §4.11 | §4.1.1, §6.2.9 | 实现 Provider CRUD API、选择优先级、连接测试 |
```

- [ ] **Step 2: Commit**

```bash
git add docs/knowledge/provider.md
git commit -m "docs: update provider.md with Issue #9 changes"
```

---

### Task 8: Create Issue Summary

**Files:**
- Create: `docs/v1.0-mvp/issues/issue-9-summary.md`

- [ ] **Step 1: Create issue summary document**

```markdown
# Issue #9: MVP Phase 5 - Provider Management System

## Summary
实现 Provider 管理系统，支持三层作用域（system/tenant/user）、Provider 选择优先级链、连接测试功能。

## Impact
- 新增 Provider CRUD API（7个端点）
- 实现 Provider 选择优先级：任务指定 > 用户默认 > 租户默认 > 系统默认
- 系统预置 Provider：Claude Code、智谱 GLM、DeepSeek

## Status
Completed

## Related
- PRD: US-D06 Provider 选择与配置, US-A04 Provider 管理
- TRD: §4.1.1, §6.2.9, §7.1.6
- Knowledge: docs/knowledge/provider.md

## Resolution
- 实现了 ProviderRepository、ProviderService、ProviderHandler
- 实现了三层作用域过滤和 Provider 选择优先级链
- 实现了连接测试功能
- 添加了系统预置 Provider 种子数据

## Change History
| 日期 | 变更内容 |
|------|---------|
| 2026-03-24 | 创建 Issue Summary |
| 2026-03-24 | 完成实现 |
```

- [ ] **Step 2: Commit**

```bash
git add docs/v1.0-mvp/issues/issue-9-summary.md
git commit -m "docs: add Issue #9 summary"
```

---

## Dependencies

| 依赖 | 状态 | 说明 |
|------|------|------|
| Issue #5 (Backend Core API) | ✅ Completed | API 模式已建立 |
| Issue #6 (Database Models) | ✅ Completed | Provider 模型已定义 |

**解除阻塞**：所有依赖项已完成，可立即开始开发。

## Risks

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| API Key 安全 | 密钥泄露 | 使用 K8s Secret 引用，不直接存储 |
| 连接测试超时 | 用户体验 | 设置合理超时（10秒），提供清晰错误信息 |
| 作用域隔离不当 | 数据泄露 | 严格验证 scope + tenant_id + user_id 组合 |

## Technical Details

### API Endpoints

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/providers | Provider 列表 |
| POST | /api/v1/providers | 创建 Provider |
| GET | /api/v1/providers/:id | Provider 详情 |
| PUT | /api/v1/providers/:id | 更新 Provider |
| DELETE | /api/v1/providers/:id | 删除 Provider |
| POST | /api/v1/providers/:id/test | 测试连接 |
| GET | /api/v1/providers/available | 任务创建时可用的 Provider |
| PUT | /api/v1/providers/:id/set-default | 设置默认 |

### Provider Selection Priority

```
任务创建时指定 provider_id?
    │
    ├── 是 ──▶ 使用指定的 Provider
    │
    └── 否 ──▶ 用户有默认 Provider?
                    │
                    ├── 是 ──▶ 使用用户默认 Provider
                    │
                    └── 否 ──▶ 租户有默认 Provider?
                                    │
                                    ├── 是 ──▶ 使用租户默认 Provider
                                    │
                                    └── 否 ──▶ 使用系统默认 Provider (Claude Code)
```

### System Preset Providers

| Provider | Type | Endpoint |
|----------|------|----------|
| claude-code | claude_code | https://api.anthropic.com |
| zhipu-glm | anthropic_compatible | https://open.bigmodel.cn/api/anthropic |
| deepseek | anthropic_compatible | https://api.deepseek.com |

## Status

Not Started
