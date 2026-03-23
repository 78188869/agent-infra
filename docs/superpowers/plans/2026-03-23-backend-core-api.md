# Backend Core API Development Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the core RESTful API framework for the control-plane service, including health check, tenant management, template management, and task execution APIs.

**Architecture:** Layered architecture with Handler → Service → Repository pattern. Handlers handle HTTP requests/responses, Services contain business logic, and Repository handles data access via GORM.

**Tech Stack:** Go 1.22 + Gin 1.9 + GORM 1.25 + validator v10

**Related Issue:** #5 - MVP Phase 1 - Backend Core API Development

**Reference Documents:**
- TRD: `docs/v1.0-mvp/plans/2026-03-22-mvp-trd.md`
- Knowledge: `docs/knowledge/core-api.md`, `docs/knowledge/database.md`

---

## Verification Criteria

Based on Issue #5 Definition of Done:

| Criteria | Verification Method |
|----------|---------------------|
| All API endpoints implemented | `make test` + manual API testing |
| Unit test coverage > 80% | `go test -cover ./...` |
| API documentation updated | Swagger/OpenAPI spec generated |
| Code review passed | PR review approved |

**Acceptance Test Checklist:**
- [ ] `GET /health` returns 200 with `{"status": "healthy"}`
- [ ] `GET /ready` returns 200 with readiness status including DB/Redis checks
- [ ] `POST /api/v1/tenants` creates tenant with valid input
- [ ] `GET /api/v1/tenants` returns paginated tenant list
- [ ] `GET /api/v1/tenants/{id}` returns single tenant
- [ ] `PUT /api/v1/tenants/{id}` updates tenant
- [ ] `DELETE /api/v1/tenants/{id}` soft-deletes tenant
- [ ] `POST /api/v1/templates` creates template with valid input
- [ ] `GET /api/v1/templates` returns paginated template list
- [ ] `GET /api/v1/templates/{id}` returns single template
- [ ] `PUT /api/v1/templates/{id}` updates template
- [ ] `DELETE /api/v1/templates/{id}` soft-deletes template
- [ ] `POST /api/v1/tasks` creates task with valid input
- [ ] `GET /api/v1/tasks` returns paginated task list
- [ ] `GET /api/v1/tasks/{id}` returns single task
- [ ] `PUT /api/v1/tasks/{id}` updates task status
- [ ] `DELETE /api/v1/tasks/{id}` soft-deletes task
- [ ] Test coverage > 80% verified by `go test -cover ./...`

---

## File Structure

### New Files to Create

```
control-plane/
├── internal/
│   ├── model/                      # Data models
│   │   ├── base.go                 # Base model with soft delete
│   │   ├── tenant.go               # Tenant model
│   │   ├── user.go                 # User model (stub for MVP)
│   │   ├── task.go                 # Task model
│   │   └── template.go             # Template model
│   │
│   ├── repository/                 # Repository layer
│   │   ├── repository.go           # Repository interfaces
│   │   ├── tenant_repo.go          # Tenant repository
│   │   ├── task_repo.go            # Task repository
│   │   └── template_repo.go        # Template repository
│   │
│   ├── service/                    # Service layer
│   │   ├── tenant_service.go       # Tenant business logic
│   │   ├── task_service.go         # Task business logic
│   │   └── template_service.go     # Template business logic
│   │
│   ├── api/
│   │   ├── handler/
│   │   │   ├── tenant.go           # Tenant HTTP handlers
│   │   │   ├── template.go         # Template HTTP handlers
│   │   │   └── task.go             # Task HTTP handlers
│   │   └── response/
│   │       └── response.go         # Unified response utilities
│   │
│   └── config/
│       └── database.go             # Database connection
│
├── pkg/
│   └── errors/
│       └── errors.go               # Custom error types
│
└── tests/
    └── api/
        └── handler/
            ├── tenant_test.go      # Tenant handler tests
            ├── template_test.go    # Template handler tests
            └── task_test.go        # Task handler tests
```

### Files to Modify

- `go.mod` - Add GORM and validator dependencies
- `internal/api/router/router.go` - Add new API routes
- `internal/api/handler/health.go` - Enhance readiness check
- `cmd/control-plane/main.go` - Initialize database connection

---

## Task 1: Infrastructure Setup

**Files:**
- Modify: `go.mod`
- Create: `internal/config/database.go`
- Create: `pkg/errors/errors.go`
- Create: `internal/api/response/response.go`
- Create: `internal/model/base.go`

- [ ] **Step 1: Add dependencies to go.mod**

Add required dependencies for GORM and UUID generation.

- [ ] **Step 2: Write failing test for database connection**

Create test file that verifies database connection can be established.

- [ ] **Step 3: Implement database configuration**

Create `internal/config/database.go` with GORM connection setup using configuration from TRD:
- Connection pool: max_idle_conns=10, max_open_conns=100
- Support for MySQL-compatible DSN (OceanBase)

- [ ] **Step 4: Create error types**

Create `pkg/errors/errors.go` with custom error types:
- `AppError` struct with code, message, and HTTP status
- Predefined errors: `ErrNotFound`, `ErrBadRequest`, `ErrInternal`

- [ ] **Step 5: Create unified response utilities**

Create `internal/api/response/response.go` with:
- `Success(c, data)` - 200 response
- `Created(c, data)` - 201 response
- `BadRequest(c, message)` - 400 response
- `NotFound(c, message)` - 404 response
- `InternalError(c, message)` - 500 response
- Response format: `{"code": 0, "message": "success", "data": {...}}`

- [ ] **Step 6: Create base model**

Create `internal/model/base.go` with:
- `BaseModel` struct with ID, CreatedAt, UpdatedAt, DeletedAt
- UUID generation hook

- [ ] **Step 7: Commit infrastructure setup**

```bash
git add go.mod go.sum internal/config/ pkg/errors/ internal/api/response/ internal/model/base.go
git commit -m "feat(infra): add database config, error types, and response utilities"
```

---

## Task 2: Tenant Model and Repository

**Files:**
- Create: `internal/model/tenant.go`
- Create: `internal/repository/repository.go`
- Create: `internal/repository/tenant_repo.go`

- [ ] **Step 1: Write failing test for Tenant model**

Create test that verifies Tenant struct fields and GORM tags.

- [ ] **Step 2: Implement Tenant model**

Create `internal/model/tenant.go` based on TRD §6.2.1:
```go
type Tenant struct {
    BaseModel
    Name             string     `gorm:"type:varchar(128);not null"`
    QuotaCPU         int        `gorm:"default:4"`
    QuotaMemory      int64      `gorm:"default:16"`  // GB
    QuotaConcurrency int        `gorm:"default:10"`
    QuotaDailyTasks  int        `gorm:"default:100"`
    Status           string     `gorm:"type:enum('active','suspended');default:'active'"`
}
```

- [ ] **Step 3: Write failing test for TenantRepository**

Create test for CRUD operations.

- [ ] **Step 4: Define Repository interface**

Create `internal/repository/repository.go` with generic Repository interface.

- [ ] **Step 5: Implement TenantRepository**

Create `internal/repository/tenant_repo.go` with:
- `Create(ctx, tenant) error`
- `GetByID(ctx, id) (*model.Tenant, error)`
- `List(ctx, filter) ([]*model.Tenant, int64, error)`
- `Update(ctx, tenant) error`
- `Delete(ctx, id) error` (soft delete)

- [ ] **Step 6: Run tests and verify they pass**

Run: `go test ./internal/repository/... -v`

- [ ] **Step 7: Commit tenant model and repository**

```bash
git add internal/model/tenant.go internal/repository/
git commit -m "feat(model): add Tenant model and repository"
```

---

## Task 3: Tenant Service

**Files:**
- Create: `internal/service/tenant_service.go`

- [ ] **Step 1: Write failing test for TenantService**

Create test for business logic methods.

- [ ] **Step 2: Implement TenantService interface**

Define interface based on TRD §4.1.1:
```go
type TenantService interface {
    Create(ctx, req *CreateTenantRequest) (*Tenant, error)
    GetByID(ctx, id string) (*Tenant, error)
    List(ctx, filter *TenantFilter) ([]*Tenant, int64, error)
    Update(ctx, id string, req *UpdateTenantRequest) error
    Delete(ctx, id string) error
}
```

- [ ] **Step 3: Implement TenantService methods**

Implement business logic:
- Validate tenant name uniqueness
- Validate quota limits
- Map repository errors to service errors

- [ ] **Step 4: Run tests and verify they pass**

Run: `go test ./internal/service/... -v`

- [ ] **Step 5: Commit tenant service**

```bash
git add internal/service/tenant_service.go internal/service/tenant_service_test.go
git commit -m "feat(service): add TenantService with business logic"
```

---

## Task 4: Tenant API Handlers

**Files:**
- Create: `internal/api/handler/tenant.go`
- Modify: `internal/api/router/router.go`

- [ ] **Step 1: Write failing test for TenantHandler**

Create `tests/api/handler/tenant_test.go` with tests for all CRUD endpoints.

- [ ] **Step 2: Implement TenantHandler struct**

Create handler with service dependency injection.

- [ ] **Step 3: Implement POST /api/v1/tenants**

Create tenant endpoint with request validation:
- Request: `{"name": "tenant-1", "quota_cpu": 4, ...}`
- Response: `{"code": 0, "data": {"id": "...", "name": "...", ...}}`

- [ ] **Step 4: Implement GET /api/v1/tenants**

List tenants with pagination support:
- Query params: `page`, `page_size`, `status`
- Response: `{"code": 0, "data": {"items": [...], "total": 100}}`

- [ ] **Step 5: Implement GET /api/v1/tenants/:id**

Get single tenant by ID.

- [ ] **Step 6: Implement PUT /api/v1/tenants/:id**

Update tenant with validation.

- [ ] **Step 7: Implement DELETE /api/v1/tenants/:id**

Soft delete tenant.

- [ ] **Step 8: Register routes in router**

Modify `internal/api/router/router.go`:
```go
v1 := r.Group("/api/v1")
tenants := v1.Group("/tenants")
{
    tenants.POST("", tenantHandler.Create)
    tenants.GET("", tenantHandler.List)
    tenants.GET("/:id", tenantHandler.GetByID)
    tenants.PUT("/:id", tenantHandler.Update)
    tenants.DELETE("/:id", tenantHandler.Delete)
}
```

- [ ] **Step 9: Run tests and verify they pass**

Run: `go test ./tests/api/handler/... -v`

- [ ] **Step 10: Commit tenant handlers**

```bash
git add internal/api/handler/tenant.go internal/api/router/router.go tests/api/handler/tenant_test.go
git commit -m "feat(api): add Tenant CRUD endpoints"
```

---

## Task 5: Template Model and Repository

**Files:**
- Create: `internal/model/template.go`
- Create: `internal/repository/template_repo.go`

- [ ] **Step 1: Write failing test for Template model**

Create test that verifies Template struct fields.

- [ ] **Step 2: Implement Template model**

Create `internal/model/template.go` based on TRD §6.2.4:
```go
type Template struct {
    BaseModel
    TenantID    string          `gorm:"type:varchar(36);index"`
    Name        string          `gorm:"type:varchar(128);not null"`
    Version     string          `gorm:"type:varchar(32);default:'1.0.0'"`
    Spec        string          `gorm:"type:mediumtext"`
    SceneType   string          `gorm:"type:enum('coding','ops','analysis','content','custom');default:'custom'"`
    Status      string          `gorm:"type:enum('draft','published','deprecated');default:'draft'"`
    ProviderID  *string         `gorm:"type:varchar(36)"`
}
```

- [ ] **Step 3: Write failing test for TemplateRepository**

Create test for CRUD operations.

- [ ] **Step 4: Implement TemplateRepository**

Same pattern as TenantRepository with tenant-scoped queries.

- [ ] **Step 5: Run tests and verify they pass**

Run: `go test ./internal/repository/... -v`

- [ ] **Step 6: Commit template model and repository**

```bash
git add internal/model/template.go internal/repository/template_repo.go
git commit -m "feat(model): add Template model and repository"
```

---

## Task 6: Template Service and Handlers

**Files:**
- Create: `internal/service/template_service.go`
- Create: `internal/api/handler/template.go`
- Modify: `internal/api/router/router.go`

- [ ] **Step 1: Write failing test for TemplateService**

Create test for business logic.

- [ ] **Step 2: Implement TemplateService**

Based on TRD knowledge file:
- Validate YAML spec format
- Validate scene_type enum
- Business rule: only draft templates can be deleted

- [ ] **Step 3: Write failing test for TemplateHandler**

Create test for all template endpoints.

- [ ] **Step 4: Implement TemplateHandler**

Implement all CRUD endpoints following the same pattern as TenantHandler.

- [ ] **Step 5: Register template routes**

Add routes to router.go:
```go
templates := v1.Group("/templates")
{
    templates.POST("", templateHandler.Create)
    templates.GET("", templateHandler.List)
    templates.GET("/:id", templateHandler.GetByID)
    templates.PUT("/:id", templateHandler.Update)
    templates.DELETE("/:id", templateHandler.Delete)
}
```

- [ ] **Step 6: Run tests and verify they pass**

Run: `go test ./... -v`

- [ ] **Step 7: Commit template service and handlers**

```bash
git add internal/service/template_service.go internal/api/handler/template.go
git commit -m "feat(api): add Template CRUD endpoints"
```

---

## Task 7: Task Model and Repository

**Files:**
- Create: `internal/model/task.go`
- Create: `internal/repository/task_repo.go`

- [ ] **Step 1: Write failing test for Task model**

Create test that verifies Task struct fields.

- [ ] **Step 2: Implement Task model**

Create `internal/model/task.go` based on TRD §6.2.3:
```go
type Task struct {
    BaseModel
    TenantID      string          `gorm:"type:varchar(36);index:idx_tenant_status"`
    TemplateID    *string         `gorm:"type:varchar(36);index"`
    CreatorID     string          `gorm:"type:varchar(36);index"`
    ProviderID    string          `gorm:"type:varchar(36);not null"`
    Name          string          `gorm:"type:varchar(256)"`
    Status        string          `gorm:"type:enum(...);default:'pending';index:idx_tenant_status"`
    Priority      string          `gorm:"type:enum('high','normal','low');default:'normal'"`
    Params        datatypes.JSON  `gorm:"type:json"`
    // ... other fields from TRD
}
```

- [ ] **Step 3: Write failing test for TaskRepository**

Create test for CRUD and status update operations.

- [ ] **Step 4: Implement TaskRepository**

Include status-specific queries:
- `ListByStatus(ctx, status, limit)`
- `UpdateStatus(ctx, id, status, reason)`

- [ ] **Step 5: Run tests and verify they pass**

Run: `go test ./internal/repository/... -v`

- [ ] **Step 6: Commit task model and repository**

```bash
git add internal/model/task.go internal/repository/task_repo.go
git commit -m "feat(model): add Task model and repository"
```

---

## Task 8: Task Service and Handlers

**Files:**
- Create: `internal/service/task_service.go`
- Create: `internal/api/handler/task.go`
- Modify: `internal/api/router/router.go`

- [ ] **Step 1: Write failing test for TaskService**

Create test for business logic including status transitions.

- [ ] **Step 2: Implement TaskService**

Based on TRD knowledge file:
- Validate template exists
- Validate params against template spec
- Check tenant quota
- Status transition validation

- [ ] **Step 3: Write failing test for TaskHandler**

Create test for all task endpoints.

- [ ] **Step 4: Implement TaskHandler**

Implement all CRUD endpoints with proper validation.

- [ ] **Step 5: Register task routes**

Add routes to router.go:
```go
tasks := v1.Group("/tasks")
{
    tasks.POST("", taskHandler.Create)
    tasks.GET("", taskHandler.List)
    tasks.GET("/:id", taskHandler.GetByID)
    tasks.PUT("/:id", taskHandler.Update)
    tasks.DELETE("/:id", taskHandler.Delete)
}
```

- [ ] **Step 6: Run tests and verify they pass**

Run: `go test ./... -v`

- [ ] **Step 7: Commit task service and handlers**

```bash
git add internal/service/task_service.go internal/api/handler/task.go
git commit -m "feat(api): add Task CRUD endpoints"
```

---

## Task 9: Enhance Health Check

**Files:**
- Modify: `internal/api/handler/health.go`

- [ ] **Step 1: Write failing test for enhanced readiness check**

Test that readiness includes DB/Redis connectivity status.

- [ ] **Step 2: Implement enhanced readiness check**

Modify `ReadyCheck` to:
- Check database connectivity with `db.Raw("SELECT 1")`
- Return detailed status for each component

```go
{
    "ready": true,
    "checks": {
        "database": "healthy",
        "redis": "not_configured"  // MVP phase
    }
}
```

- [ ] **Step 3: Run tests and verify they pass**

Run: `go test ./internal/api/handler/... -v`

- [ ] **Step 4: Commit health check enhancement**

```bash
git add internal/api/handler/health.go
git commit -m "feat(api): enhance readiness check with DB connectivity"
```

---

## Task 10: Integration and Main Update

**Files:**
- Modify: `cmd/control-plane/main.go`
- Modify: `cmd/control-plane/config.yaml`

- [ ] **Step 1: Update main.go to initialize database**

Add database initialization before router setup.

- [ ] **Step 2: Update config.yaml with database configuration**

Add database DSN and connection pool settings.

- [ ] **Step 3: Run full integration test**

Start server and test all endpoints manually or with integration test.

- [ ] **Step 4: Run all tests and verify coverage**

Run: `go test -cover ./...`
Verify coverage > 80%

- [ ] **Step 5: Commit integration changes**

```bash
git add cmd/control-plane/main.go cmd/control-plane/config.yaml
git commit -m "feat: integrate database and all API handlers"
```

---

## Task 11: Final Verification and Documentation

**Files:**
- Create: `docs/api/openapi.yaml` (or similar)
- Update: README or API documentation

- [ ] **Step 1: Generate API documentation**

Create OpenAPI/Swagger spec for all endpoints.

- [ ] **Step 2: Verify all acceptance criteria**

Run through the acceptance test checklist at the top of this plan.

- [ ] **Step 3: Final test coverage check**

Run: `go test -cover ./...`
Ensure coverage > 80%

- [ ] **Step 4: Commit documentation**

```bash
git add docs/api/
git commit -m "docs: add API documentation"
```

---

## Dependency Graph

```
Task 1 (Infrastructure)
    ├── Task 2 (Tenant Model/Repo) ──→ Task 3 (Tenant Service) ──→ Task 4 (Tenant Handlers)
    │
    ├── Task 5 (Template Model/Repo)
    │       └── Task 6 (Template Service/Handlers) [depends on Task 2 for tenant validation]
    │
    ├── Task 7 (Task Model/Repo)
    │       └── Task 8 (Task Service/Handlers) [depends on Task 5 for template validation]
    │
    └── Task 9 (Health Check Enhancement)
            └── Task 10 (Integration)
                    └── Task 11 (Final Verification)
```

## Estimated Effort

| Task | Estimated Time |
|------|---------------|
| Task 1: Infrastructure | 0.5 day |
| Task 2-4: Tenant Module | 1 day |
| Task 5-6: Template Module | 1 day |
| Task 7-8: Task Module | 1 day |
| Task 9: Health Check | 0.5 day |
| Task 10-11: Integration & Docs | 0.5-1 day |
| **Total** | **4-5 days** |

---

## Notes for Implementer

1. **Use TDD**: Write failing tests first, then implement
2. **Follow existing patterns**: Check existing handler and middleware code for style
3. **Keep it simple**: MVP scope - no authentication middleware yet (use stub for now)
4. **Database**: Use GORM AutoMigrate for development, real migrations for production later
5. **Error handling**: Use the unified error types from pkg/errors
6. **Response format**: Always use response.Success/Created/Error helpers
