# Issue #10: Capability Management System - Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the Capability Management System for MVP Phase 6, providing CRUD API for managing tools, skills, and agent runtimes.

**Architecture:** Three-layer architecture following existing patterns: Repository (data access) → Service (business logic) → Handler (HTTP). Uses existing Capability model with JSON config/schema storage and permission levels.

**Tech Stack:** Go 1.22, Gin 1.9, GORM 1.25, MySQL (OceanBase compatible)

---

## 🔄 Execution Progress

> **IMPORTANT:** 执行每一步后，必须立即更新此进度表。如果 Agent 异常退出，重启后可通过此表追踪执行进度。

| Task | Status | Started | Completed | Notes |
|------|--------|---------|-----------|-------|
| Task 0: Pull Latest Code | ⬜ Not Started | - | - | |
| Task 1: Capability Repository | ⬜ Not Started | - | - | |
| Task 2: Capability Service | ⬜ Not Started | - | - | |
| Task 3: Capability Handler | ⬜ Not Started | - | - | |
| Task 4: Router Integration | ⬜ Not Started | - | - | |
| Task 5: Update Main Entry Point | ⬜ Not Started | - | - | |
| Task 6: Update Knowledge Documentation | ⬜ Not Started | - | - | |
| Task 7: Final Verification | ⬜ Not Started | - | - | |
| Task 8: Code Review | ⬜ Not Started | - | - | |
| Task 9: Create Pull Request | ⬜ Not Started | - | - | |
| Task 10: Wait for PR Merge | ⬜ Not Started | - | - | ⏳ Human Required |
| Task 11: Close Issue | ⬜ Not Started | - | - | |
| Task 12: Cleanup Environment | ⬜ Not Started | - | - | |

**Status Legend:** ⬜ Not Started | 🔄 In Progress | ✅ Completed | ❌ Failed | ⏸️ Blocked

**Last Updated:** 2026-03-24 (Plan Created)

---

## 📝 Execution Log

> 执行过程中在此记录关键信息，便于追踪和恢复。

```
[2026-03-24] Plan created and worktree set up
```

---

## Files to Create/Modify

| File | Action | Purpose |
|------|--------|---------|
| `internal/repository/capability_repo.go` | Create | CapabilityRepository interface and implementation |
| `internal/repository/capability_repo_test.go` | Create | Repository unit tests |
| `internal/service/capability_service.go` | Create | CapabilityService interface and implementation |
| `internal/service/capability_service_test.go` | Create | Service unit tests |
| `internal/api/handler/capability.go` | Create | HTTP handlers for capability endpoints |
| `internal/api/handler/capability_test.go` | Create | Handler unit tests |
| `internal/api/router/router.go` | Modify | Add capability routes |
| `docs/knowledge/capability.md` | Modify | Update Change History |

---

## Task 1: Capability Repository

**Files:**
- Create: `internal/repository/capability_repo.go`
- Create: `internal/repository/capability_repo_test.go`

- [ ] **Step 1: Write the failing test for repository interface**

```go
// internal/repository/capability_repo_test.go
package repository

import (
	"context"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/google/uuid"
)

func TestCapabilityRepository_Create(t *testing.T) {
	// Will use GORM with test database or mock
}

func TestCapabilityRepository_GetByID(t *testing.T) {
	// Test cases: success, not found
}

func TestCapabilityRepository_List(t *testing.T) {
	// Test cases: with filters, pagination
}

func TestCapabilityRepository_Update(t *testing.T) {
	// Test cases: success, not found
}

func TestCapabilityRepository_Delete(t *testing.T) {
	// Test cases: success, not found
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-10-capability-management && go test ./internal/repository/... -run Capability -v`
Expected: FAIL with "no test files" or undefined

- [ ] **Step 3: Write CapabilityRepository interface and implementation**

```go
// internal/repository/capability_repo.go
package repository

import (
	"context"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CapabilityFilter represents filtering options for listing capabilities.
type CapabilityFilter struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	TenantID  string `form:"tenant_id"`
	Type      string `form:"type"`
	Status    string `form:"status"`
	Search    string `form:"search"`
}

// SetDefaults sets default values for the filter.
func (f *CapabilityFilter) SetDefaults() {
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

// Offset returns the offset for pagination.
func (f *CapabilityFilter) Offset() int {
	return (f.Page - 1) * f.PageSize
}

// CapabilityRepository defines the interface for capability data access operations.
type CapabilityRepository interface {
	Create(ctx context.Context, capability *model.Capability) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Capability, error)
	List(ctx context.Context, filter CapabilityFilter) ([]*model.Capability, int64, error)
	Update(ctx context.Context, capability *model.Capability) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// capabilityRepository implements CapabilityRepository using GORM.
type capabilityRepository struct {
	db *gorm.DB
}

// NewCapabilityRepository creates a new CapabilityRepository instance.
func NewCapabilityRepository(db *gorm.DB) CapabilityRepository {
	return &capabilityRepository{db: db}
}

// Create inserts a new capability into the database.
func (r *capabilityRepository) Create(ctx context.Context, capability *model.Capability) error {
	if err := r.db.WithContext(ctx).Create(capability).Error; err != nil {
		return errors.NewInternalError("failed to create capability: " + err.Error())
	}
	return nil
}

// GetByID retrieves a capability by its ID.
func (r *capabilityRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
	var capability model.Capability
	if err := r.db.WithContext(ctx).First(&capability, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("capability not found")
		}
		return nil, errors.NewInternalError("failed to get capability: " + err.Error())
	}
	return &capability, nil
}

// List retrieves capabilities based on filter criteria.
func (r *capabilityRepository) List(ctx context.Context, filter CapabilityFilter) ([]*model.Capability, int64, error) {
	filter.SetDefaults()

	var capabilities []*model.Capability
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Capability{})

	// Apply filters
	if filter.TenantID != "" {
		if filter.TenantID == "global" {
			query = query.Where("tenant_id IS NULL")
		} else {
			query = query.Where("tenant_id = ?", filter.TenantID)
		}
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", search, search)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to count capabilities: " + err.Error())
	}

	// Get paginated results
	if err := query.Offset(filter.Offset()).Limit(filter.PageSize).Find(&capabilities).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to list capabilities: " + err.Error())
	}

	return capabilities, total, nil
}

// Update updates an existing capability.
func (r *capabilityRepository) Update(ctx context.Context, capability *model.Capability) error {
	result := r.db.WithContext(ctx).Save(capability)
	if result.Error != nil {
		return errors.NewInternalError("failed to update capability: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("capability not found")
	}
	return nil
}

// Delete performs a soft delete on a capability.
func (r *capabilityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&model.Capability{}, "id = ?", id)
	if result.Error != nil {
		return errors.NewInternalError("failed to delete capability: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("capability not found")
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-10-capability-management && go test ./internal/repository/... -v -cover`
Expected: PASS with coverage data

- [ ] **Step 5: Commit repository implementation**

```bash
git add internal/repository/capability_repo.go internal/repository/capability_repo_test.go
git commit -m "feat(repository): add CapabilityRepository for capability data access"
```

---

## Task 2: Capability Service

**Files:**
- Create: `internal/service/capability_service.go`
- Create: `internal/service/capability_service_test.go`

- [ ] **Step 1: Write the failing test for service interface**

```go
// internal/service/capability_service_test.go
package service

import (
	"context"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/google/uuid"
)

func TestCapabilityService_Create(t *testing.T) {
	// Test cases: success, validation errors
}

func TestCapabilityService_GetByID(t *testing.T) {
	// Test cases: success, not found, invalid ID
}

func TestCapabilityService_List(t *testing.T) {
	// Test cases: with filters, pagination
}

func TestCapabilityService_Update(t *testing.T) {
	// Test cases: success, validation errors
}

func TestCapabilityService_Delete(t *testing.T) {
	// Test cases: success, not found
}

func TestCapabilityService_Activate(t *testing.T) {
	// Test cases: success, not found
}

func TestCapabilityService_Deactivate(t *testing.T) {
	// Test cases: success, not found
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-10-capability-management && go test ./internal/service/... -run Capability -v`
Expected: FAIL with "no test files" or undefined

- [ ] **Step 3: Write CapabilityService interface and implementation**

```go
// internal/service/capability_service.go
package service

import (
	"context"
	"encoding/json"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
)

// Valid capability types
var validCapabilityTypes = map[string]bool{
	string(model.CapabilityTypeTool):         true,
	string(model.CapabilityTypeSkill):        true,
	string(model.CapabilityTypeAgentRuntime): true,
}

// Valid permission levels
var validPermissionLevels = map[string]bool{
	string(model.PermissionLevelPublic):     true,
	string(model.PermissionLevelRestricted): true,
	string(model.PermissionLevelAdminOnly):  true,
}

// CreateCapabilityRequest represents the request to create a new capability.
type CreateCapabilityRequest struct {
	TenantID        *string                 `json:"tenant_id"` // NULL for global capabilities
	Type            string                  `json:"type" binding:"required"`
	Name            string                  `json:"name" binding:"required"`
	Description     string                  `json:"description"`
	Version         string                  `json:"version"`
	Config          map[string]interface{}  `json:"config"`
	Schema          map[string]interface{}  `json:"schema"`
	PermissionLevel string                  `json:"permission_level"`
}

// UpdateCapabilityRequest represents the request to update an existing capability.
type UpdateCapabilityRequest struct {
	Name            *string                 `json:"name"`
	Description     *string                 `json:"description"`
	Version         *string                 `json:"version"`
	Config          map[string]interface{}  `json:"config"`
	Schema          map[string]interface{}  `json:"schema"`
	PermissionLevel *string                 `json:"permission_level"`
	Status          *string                 `json:"status"`
}

// CapabilityFilter represents filtering options for listing capabilities.
type CapabilityFilter struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	TenantID  string `form:"tenant_id"`
	Type      string `form:"type"`
	Status    string `form:"status"`
	Search    string `form:"search"`
}

// CapabilityService defines the interface for capability business operations.
type CapabilityService interface {
	Create(ctx context.Context, req *CreateCapabilityRequest) (*model.Capability, error)
	GetByID(ctx context.Context, id string) (*model.Capability, error)
	List(ctx context.Context, filter *CapabilityFilter) ([]*model.Capability, int64, error)
	Update(ctx context.Context, id string, req *UpdateCapabilityRequest) error
	Delete(ctx context.Context, id string) error
	Activate(ctx context.Context, id string) error
	Deactivate(ctx context.Context, id string) error
}

// capabilityService implements CapabilityService.
type capabilityService struct {
	repo repository.CapabilityRepository
}

// NewCapabilityService creates a new CapabilityService instance.
func NewCapabilityService(repo repository.CapabilityRepository) CapabilityService {
	return &capabilityService{repo: repo}
}

// Create creates a new capability with validation.
func (s *capabilityService) Create(ctx context.Context, req *CreateCapabilityRequest) (*model.Capability, error) {
	// Validate required fields
	if req.Name == "" {
		return nil, errors.NewBadRequestError("capability name is required")
	}

	if req.Type == "" {
		return nil, errors.NewBadRequestError("capability type is required")
	}

	// Validate type
	if !validCapabilityTypes[req.Type] {
		return nil, errors.NewBadRequestError("invalid capability type, must be one of: tool, skill, agent_runtime")
	}

	// Validate permission level
	permissionLevel := req.PermissionLevel
	if permissionLevel == "" {
		permissionLevel = string(model.PermissionLevelPublic)
	}
	if !validPermissionLevels[permissionLevel] {
		return nil, errors.NewBadRequestError("invalid permission level, must be one of: public, restricted, admin_only")
	}

	// Set default version
	version := req.Version
	if version == "" {
		version = "1.0.0"
	}

	// Convert config and schema to JSON
	var configJSON, schemaJSON []byte
	var err error

	if req.Config != nil {
		configJSON, err = json.Marshal(req.Config)
		if err != nil {
			return nil, errors.NewBadRequestError("invalid config format: " + err.Error())
		}
	}

	if req.Schema != nil {
		schemaJSON, err = json.Marshal(req.Schema)
		if err != nil {
			return nil, errors.NewBadRequestError("invalid schema format: " + err.Error())
		}
	}

	// Create capability model
	capability := &model.Capability{
		TenantID:        req.TenantID,
		Type:            model.CapabilityType(req.Type),
		Name:            req.Name,
		Description:     req.Description,
		Version:         version,
		Config:          configJSON,
		Schema:          schemaJSON,
		PermissionLevel: model.PermissionLevel(permissionLevel),
		Status:          model.CapabilityStatusActive,
	}

	// Call repository
	if err := s.repo.Create(ctx, capability); err != nil {
		return nil, err
	}

	return capability, nil
}

// GetByID retrieves a capability by its ID.
func (s *capabilityService) GetByID(ctx context.Context, id string) (*model.Capability, error) {
	// Parse and validate ID
	capabilityID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid capability ID format")
	}

	// Call repository
	capability, err := s.repo.GetByID(ctx, capabilityID)
	if err != nil {
		return nil, err
	}

	return capability, nil
}

// List retrieves capabilities based on filter criteria.
func (s *capabilityService) List(ctx context.Context, filter *CapabilityFilter) ([]*model.Capability, int64, error) {
	// Convert service filter to repository filter
	repoFilter := repository.CapabilityFilter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		TenantID: filter.TenantID,
		Type:     filter.Type,
		Status:   filter.Status,
		Search:   filter.Search,
	}

	// Call repository
	capabilities, total, err := s.repo.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, err
	}

	return capabilities, total, nil
}

// Update updates an existing capability with partial update support.
func (s *capabilityService) Update(ctx context.Context, id string, req *UpdateCapabilityRequest) error {
	// Parse and validate ID
	capabilityID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid capability ID format")
	}

	// Get existing capability
	capability, err := s.repo.GetByID(ctx, capabilityID)
	if err != nil {
		return err
	}

	// Validate and apply updates
	if req.Name != nil {
		if *req.Name == "" {
			return errors.NewBadRequestError("capability name cannot be empty")
		}
		capability.Name = *req.Name
	}

	if req.Description != nil {
		capability.Description = *req.Description
	}

	if req.Version != nil {
		capability.Version = *req.Version
	}

	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			return errors.NewBadRequestError("invalid config format: " + err.Error())
		}
		capability.Config = configJSON
	}

	if req.Schema != nil {
		schemaJSON, err := json.Marshal(req.Schema)
		if err != nil {
			return errors.NewBadRequestError("invalid schema format: " + err.Error())
		}
		capability.Schema = schemaJSON
	}

	if req.PermissionLevel != nil {
		if !validPermissionLevels[*req.PermissionLevel] {
			return errors.NewBadRequestError("invalid permission level, must be one of: public, restricted, admin_only")
		}
		capability.PermissionLevel = model.PermissionLevel(*req.PermissionLevel)
	}

	if req.Status != nil {
		if *req.Status != string(model.CapabilityStatusActive) && *req.Status != string(model.CapabilityStatusInactive) {
			return errors.NewBadRequestError("invalid status, must be one of: active, inactive")
		}
		capability.Status = model.CapabilityStatus(*req.Status)
	}

	// Call repository
	return s.repo.Update(ctx, capability)
}

// Delete performs a soft delete on a capability.
func (s *capabilityService) Delete(ctx context.Context, id string) error {
	// Parse and validate ID
	capabilityID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid capability ID format")
	}

	// Call repository
	return s.repo.Delete(ctx, capabilityID)
}

// Activate activates a capability.
func (s *capabilityService) Activate(ctx context.Context, id string) error {
	// Parse and validate ID
	capabilityID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid capability ID format")
	}

	// Get existing capability
	capability, err := s.repo.GetByID(ctx, capabilityID)
	if err != nil {
		return err
	}

	capability.Status = model.CapabilityStatusActive

	// Call repository
	return s.repo.Update(ctx, capability)
}

// Deactivate deactivates a capability.
func (s *capabilityService) Deactivate(ctx context.Context, id string) error {
	// Parse and validate ID
	capabilityID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid capability ID format")
	}

	// Get existing capability
	capability, err := s.repo.GetByID(ctx, capabilityID)
	if err != nil {
		return err
	}

	capability.Status = model.CapabilityStatusInactive

	// Call repository
	return s.repo.Update(ctx, capability)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-10-capability-management && go test ./internal/service/... -v -cover`
Expected: PASS with coverage data

- [ ] **Step 5: Commit service implementation**

```bash
git add internal/service/capability_service.go internal/service/capability_service_test.go
git commit -m "feat(service): add CapabilityService for capability business logic"
```

---

## Task 3: Capability Handler

**Files:**
- Create: `internal/api/handler/capability.go`
- Create: `internal/api/handler/capability_test.go`

- [ ] **Step 1: Write the failing test for handler**

```go
// internal/api/handler/capability_test.go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCapabilityHandler_Create(t *testing.T) {
	// Test cases: success, validation errors
}

func TestCapabilityHandler_GetByID(t *testing.T) {
	// Test cases: success, not found, invalid ID
}

func TestCapabilityHandler_List(t *testing.T) {
	// Test cases: with filters, pagination
}

func TestCapabilityHandler_Update(t *testing.T) {
	// Test cases: success, validation errors
}

func TestCapabilityHandler_Delete(t *testing.T) {
	// Test cases: success, not found
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-10-capability-management && go test ./internal/api/handler/... -run Capability -v`
Expected: FAIL with "no test files" or undefined

- [ ] **Step 3: Write CapabilityHandler implementation**

```go
// internal/api/handler/capability.go
package handler

import (
	"github.com/example/agent-infra/internal/api/response"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// CapabilityHandler handles HTTP requests for capability operations.
type CapabilityHandler struct {
	service service.CapabilityService
}

// NewCapabilityHandler creates a new CapabilityHandler instance.
func NewCapabilityHandler(svc service.CapabilityService) *CapabilityHandler {
	return &CapabilityHandler{
		service: svc,
	}
}

// Create handles POST /api/v1/capabilities - Create a new capability.
func (h *CapabilityHandler) Create(c *gin.Context) {
	var req service.CreateCapabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	capability, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, capability)
}

// GetByID handles GET /api/v1/capabilities/:id - Get a single capability.
func (h *CapabilityHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	capability, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, capability)
}

// List handles GET /api/v1/capabilities - List capabilities with pagination.
func (h *CapabilityHandler) List(c *gin.Context) {
	var filter service.CapabilityFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.BadRequest(c, "invalid query parameters: "+err.Error())
		return
	}

	// Set default values if not provided
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 10
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	capabilities, total, err := h.service.List(c.Request.Context(), &filter)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Paginated(c, capabilities, total, filter.Page, filter.PageSize)
}

// Update handles PUT /api/v1/capabilities/:id - Update a capability.
func (h *CapabilityHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req service.UpdateCapabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "capability updated successfully"})
}

// Delete handles DELETE /api/v1/capabilities/:id - Soft delete a capability.
func (h *CapabilityHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.service.Delete(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "capability deleted successfully"})
}

// Activate handles POST /api/v1/capabilities/:id/activate - Activate a capability.
func (h *CapabilityHandler) Activate(c *gin.Context) {
	id := c.Param("id")

	err := h.service.Activate(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "capability activated successfully"})
}

// Deactivate handles POST /api/v1/capabilities/:id/deactivate - Deactivate a capability.
func (h *CapabilityHandler) Deactivate(c *gin.Context) {
	id := c.Param("id")

	err := h.service.Deactivate(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "capability deactivated successfully"})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-10-capability-management && go test ./internal/api/handler/... -v -cover`
Expected: PASS with coverage data

- [ ] **Step 5: Commit handler implementation**

```bash
git add internal/api/handler/capability.go internal/api/handler/capability_test.go
git commit -m "feat(handler): add CapabilityHandler for capability HTTP endpoints"
```

---

## Task 4: Router Integration

**Files:**
- Modify: `internal/api/router/router.go`
- Modify: `internal/api/router/router_test.go`

- [ ] **Step 1: Write the failing test for router**

Add capability route tests to `router_test.go`

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-10-capability-management && go test ./internal/api/router/... -v`
Expected: FAIL with missing capability routes

- [ ] **Step 3: Update router to include capability routes**

Modify `internal/api/router/router.go`:

```go
// Update the Setup function signature to include capabilitySvc
func Setup(tenantSvc service.TenantService, templateSvc service.TemplateService, taskSvc service.TaskService, capabilitySvc service.CapabilityService, db DBChecker) *gin.Engine {
	// ... existing code ...

	// Capability routes (add after task routes)
	capabilityHandler := handler.NewCapabilityHandler(capabilitySvc)
	capabilities := v1.Group("/capabilities")
	{
		capabilities.POST("", capabilityHandler.Create)
		capabilities.GET("", capabilityHandler.List)
		capabilities.GET("/:id", capabilityHandler.GetByID)
		capabilities.PUT("/:id", capabilityHandler.Update)
		capabilities.DELETE("/:id", capabilityHandler.Delete)
		capabilities.POST("/:id/activate", capabilityHandler.Activate)
		capabilities.POST("/:id/deactivate", capabilityHandler.Deactivate)
	}

	return r
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-10-capability-management && go test ./internal/api/router/... -v -cover`
Expected: PASS

- [ ] **Step 5: Commit router changes**

```bash
git add internal/api/router/router.go internal/api/router/router_test.go
git commit -m "feat(router): add capability routes to API router"
```

---

## Task 5: Update Main Entry Point

**Files:**
- Modify: `cmd/control-plane/main.go`

- [ ] **Step 1: Update main.go to initialize capability service**

Update the main entry point to:
1. Initialize CapabilityRepository
2. Initialize CapabilityService
3. Pass CapabilityService to router.Setup()

- [ ] **Step 2: Verify application builds**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-10-capability-management && go build ./cmd/control-plane/...`
Expected: Build successful

- [ ] **Step 3: Commit main changes**

```bash
git add cmd/control-plane/main.go
git commit -m "feat(main): integrate CapabilityService into application"
```

---

## Task 6: Update Knowledge Documentation

**Files:**
- Modify: `docs/knowledge/capability.md`

- [ ] **Step 1: Update Change History in capability.md**

Add implementation notes and update change history:

```markdown
## 4. Implementation Notes

> **Implemented in Issue #10** - See `internal/service/capability_service.go` for source code

### 4.1 实际实现架构

```
internal/
├── repository/
│   └── capability_repo.go      # CapabilityRepository (GORM)
├── service/
│   └── capability_service.go   # CapabilityService (业务逻辑)
├── api/handler/
│   └── capability.go           # CapabilityHandler (HTTP)
└── model/
    └── capability.go           # Capability model (已存在)
```

### 4.2 API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/capabilities | 注册能力 |
| GET | /api/v1/capabilities | 能力列表 |
| GET | /api/v1/capabilities/:id | 能力详情 |
| PUT | /api/v1/capabilities/:id | 更新能力 |
| DELETE | /api/v1/capabilities/:id | 删除能力 |
| POST | /api/v1/capabilities/:id/activate | 激活能力 |
| POST | /api/v1/capabilities/:id/deactivate | 停用能力 |

## 5. Change History

| Date | Version | Issue | PRD Ref | TRD Ref | Changes |
|------|---------|-------|---------|---------|---------|
| 2026-03-24 | v1.1 | #10 | §4.3 | §4.1, §6.2.8 | 实现能力管理 CRUD API |
| 2026-03-23 | v1.0 | - | §4.3 | §4.1, §6.2.8 | 初始定义：能力注册与管理 |
```

- [ ] **Step 2: Commit documentation update**

```bash
git add docs/knowledge/capability.md
git commit -m "docs: update capability knowledge with implementation notes"
```

---

## Task 7: Final Verification

- [ ] **Step 1: Run all tests**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-10-capability-management && go test ./... -v -cover`
Expected: All tests pass

- [ ] **Step 2: Verify coverage > 80%**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-10-capability-management && go test ./internal/repository/... ./internal/service/... ./internal/api/handler/... -coverprofile=coverage.out && go tool cover -func=coverage.out`
Expected: Coverage > 80%

- [ ] **Step 3: Run linter**

Run: `cd /Users/yang/workspace/learning/agent-infra/.claude/worktrees/issue-10-capability-management && golangci-lint run ./...`
Expected: No linting errors

- [ ] **Step 4: Commit final verification**

```bash
git add .
git commit -m "test: verify all tests pass with >80% coverage"
```

---

## Task 8: Code Review

**Purpose:** Use test cases document to review the implementation for completeness and correctness.

- [ ] **Step 1: Read test cases document**

Read: `docs/v1.0-mvp/test-cases.md`
Focus on Section 8: Capability Management Tests (TC-CAP-001 to TC-CAP-005)

- [ ] **Step 2: Review implementation against test cases**

Verify each test case is covered:
| Test Case | Description | Status |
|-----------|-------------|--------|
| TC-CAP-001 | 创建能力 (POST /api/v1/capabilities) | [ ] |
| TC-CAP-002 | 获取能力列表 (GET /api/v1/capabilities) | [ ] |
| TC-CAP-003 | 获取能力详情 (GET /api/v1/capabilities/:id) | [ ] |
| TC-CAP-004 | 更新能力 (PUT /api/v1/capabilities/:id) | [ ] |
| TC-CAP-005 | 删除能力 (DELETE /api/v1/capabilities/:id) | [ ] |

- [ ] **Step 3: Fix any gaps found**

If any test case is not covered, add the missing implementation or tests.

- [ ] **Step 4: Commit review fixes (if any)**

```bash
git add .
git commit -m "fix: address code review findings"
```

---

## Task 9: Create Pull Request

- [ ] **Step 1: Push branch to remote**

```bash
git push -u origin feature/issue-10-capability-management
```

- [ ] **Step 2: Create Pull Request**

```bash
gh pr create --base main --title "feat(capability): implement Capability Management System (Issue #10)" --body "$(cat <<'EOF'
## Summary

Implements the Capability Management System for MVP Phase 6, providing CRUD API for managing tools, skills, and agent runtimes.

## Changes

- Add `CapabilityRepository` for data access layer
- Add `CapabilityService` for business logic layer
- Add `CapabilityHandler` for HTTP endpoints
- Update router to include capability routes
- Update main entry point to initialize capability service

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/capabilities | Register capability |
| GET | /api/v1/capabilities | List capabilities |
| GET | /api/v1/capabilities/:id | Get capability |
| PUT | /api/v1/capabilities/:id | Update capability |
| DELETE | /api/v1/capabilities/:id | Delete capability |
| POST | /api/v1/capabilities/:id/activate | Activate capability |
| POST | /api/v1/capabilities/:id/deactivate | Deactivate capability |

## Test Coverage

- Repository tests: >80%
- Service tests: >80%
- Handler tests: >80%

## Related

- Closes #10
- Dependencies: #5 (completed), #6 (completed)

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 3: Record PR number**

PR Number: ________ (fill in after creation)

---

## Task 10: Wait for PR Merge (Human Confirmation Required)

> ⚠️ **This step requires human input.** Wait for PR review and merge confirmation.

- [ ] **Step 1: Wait for human to confirm PR merge**

**Human Action Required:**
1. Review the PR at GitHub
2. Approve and merge the PR
3. Confirm merge by typing "merged" or the PR number

**Waiting for confirmation...**

- [ ] **Step 2: Pull latest main after merge**

After human confirms merge:
```bash
git checkout main
git pull origin main
```

---

## Task 11: Close Issue

- [ ] **Step 1: Close GitHub issue**

```bash
gh issue close 10 --repo 78188869/agent-infra --comment "Implemented in PR #<PR_NUMBER>

## Summary
- Capability Management System with full CRUD API
- Support for tool/skill/agent_runtime types
- Permission levels: public/restricted/admin_only
- Test coverage >80%

## Files Created
- internal/repository/capability_repo.go
- internal/service/capability_service.go
- internal/api/handler/capability.go

## Files Modified
- internal/api/router/router.go
- cmd/control-plane/main.go
- docs/knowledge/capability.md"
```

- [ ] **Step 2: Update Issue Summary**

Update `docs/v1.0-mvp/issues/issue-10-summary.md`:
- Change status to ✅ Completed
- Add Resolution section
- Add PR number

---

## Task 12: Cleanup Environment

- [ ] **Step 1: Remove worktree**

```bash
# Switch back to main repo
cd /Users/yang/workspace/learning/agent-infra

# Remove the worktree
git worktree remove .claude/worktrees/issue-10-capability-management

# Delete the branch (optional, keeps remote)
git branch -d feature/issue-10-capability-management
```

- [ ] **Step 2: Verify cleanup**

```bash
git worktree list
```

Expected: Only main worktree listed

---

## Verification Criteria

| Criteria | Verification Method |
|----------|---------------------|
| 能力 CRUD API 正常工作 | Unit tests for Create/Get/List/Update/Delete |
| 能力类型正确区分 | Unit tests for tool/skill/agent_runtime types |
| 参数 Schema 验证有效 | Unit tests for JSON schema validation |
| 权限控制正确 | Unit tests for permission level validation |
| 能力与模板正确关联 | (Future: template-capability association) |
| 单元测试覆盖率 > 80% | `go test -cover ./internal/...` |

---

## Dependencies

| Issue | Title | Status |
|-------|-------|--------|
| #5 | Backend Core API Development | ✅ Completed |
| #6 | Database Models and Migrations | ✅ Completed |

---

## Estimated Effort

**Total: 2-3 days**

| Task | Estimated Time |
|------|----------------|
| Task 1: Repository | 0.5 day |
| Task 2: Service | 0.5 day |
| Task 3: Handler | 0.5 day |
| Task 4: Router Integration | 0.25 day |
| Task 5: Main Entry Point | 0.25 day |
| Task 6: Documentation | 0.25 day |
| Task 7: Final Verification | 0.25 day |

---

## ⚙️ Execution Instructions

### Progress Tracking Rules

执行过程中，**每完成一个步骤**，必须立即更新此计划文件：

1. **更新进度表** - 将对应 Task 的 Status 更新为 `🔄 In Progress` 或 `✅ Completed`
2. **记录时间** - 填写 Started 和 Completed 时间
3. **添加备注** - 在 Notes 列记录关键信息（如 commit hash、文件名等）
4. **更新执行日志** - 在 Execution Log 部分添加执行记录

### Progress Update Template

每完成一个 Task 后，更新进度表：

```markdown
| Task X: Task Name | ✅ Completed | 2026-03-24 10:00 | 2026-03-24 10:30 | commit: abc123 |
```

并在 Execution Log 添加：

```
[2026-03-24 10:30] Task X completed: <brief description>
```

### Recovery from Abnormal Exit

如果 Agent 异常退出，重启后：

1. 读取此计划文件的 **🔄 Execution Progress** 部分
2. 找到最后一个 `🔄 In Progress` 或 `✅ Completed` 的 Task
3. 从下一个未完成的 Task 继续执行
4. 如果某个 Task 状态是 `🔄 In Progress`，检查该 Task 的详细步骤，从第一个未勾选的 Step 继续

### Commit Strategy

建议在以下时机提交代码：

- 每个 Task 完成后
- 每个 Step 完成后（可选）
- 提交信息格式：`feat(capability): <description>`

```bash
# 示例
git add . && git commit -m "feat(capability): add CapabilityRepository with tests"
```
