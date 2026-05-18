# Issue #53: Auth Middleware + API Key Management

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement API Key authentication middleware and CRUD endpoints, protecting all /api/v1/* routes with user/tenant context injection.

**Architecture:** Follow Handler → Service → Repository → Model layering. Authenticator interface abstracts auth for future SSO. SHA256 hash-only storage for API keys. Middleware extracts Bearer token, validates against DB, injects user_id/tenant_id into gin.Context.

**Tech Stack:** Go 1.21, Gin 1.9, GORM 1.25, crypto/sha256

---

## File Structure

| Action | File | Responsibility |
|--------|------|---------------|
| Create | `internal/repository/api_key_repo.go` | API Key DB queries (create, getByHash, listByUser, update) |
| Create | `internal/repository/api_key_repo_test.go` | Repository unit tests |
| Create | `internal/repository/user_repo.go` | User lookup by ID (for middleware to load user after key match) |
| Create | `internal/repository/user_repo_test.go` | Repository unit tests |
| Create | `internal/service/api_key_service.go` | API Key business logic (generate, validate, revoke, list) |
| Create | `internal/service/api_key_service_test.go` | Service unit tests |
| Create | `internal/api/middleware/auth.go` | Authenticator interface + APIKeyAuth middleware |
| Create | `internal/api/middleware/auth_test.go` | Middleware unit tests |
| Create | `internal/api/handler/api_key.go` | HTTP handlers for POST/GET/DELETE /api/v1/api-keys |
| Create | `internal/api/handler/api_key_test.go` | Handler unit tests |
| Modify | `internal/api/router/router.go` | Apply auth middleware to /api/v1/*, add api-keys routes |
| Modify | `internal/api/router/router_test.go` | Update router tests for auth middleware |
| Modify | `cmd/control-plane/main.go` | Wire up new repo → service → handler |

---

## Task 1: API Key Repository

**Files:**
- Create: `internal/repository/api_key_repo.go`
- Create: `internal/repository/api_key_repo_test.go`

- [ ] **Step 1: Write APIKeyRepository interface and implementation**

```go
// internal/repository/api_key_repo.go
package repository

import (
	"context"
	"time"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"gorm.io/gorm"
)

// APIKeyFilter represents filtering options for listing API keys.
type APIKeyFilter struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   string `form:"status"`
}

// SetDefaults sets default values for the filter.
func (f *APIKeyFilter) SetDefaults() {
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

// Offset returns the calculated offset for pagination.
func (f *APIKeyFilter) Offset() int {
	return (f.Page - 1) * f.PageSize
}

// APIKeyRepository defines the interface for API key data access.
type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *model.APIKey) error
	GetByHash(ctx context.Context, keyHash string) (*model.APIKey, error)
	GetByID(ctx context.Context, id string) (*model.APIKey, error)
	ListByUser(ctx context.Context, userID string, filter APIKeyFilter) ([]*model.APIKey, int64, error)
	Update(ctx context.Context, apiKey *model.APIKey) error
	UpdateLastUsed(ctx context.Context, id string) error
}

type apiKeyRepository struct {
	db *gorm.DB
}

// NewAPIKeyRepository creates a new APIKeyRepository instance.
func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) Create(ctx context.Context, apiKey *model.APIKey) error {
	if err := r.db.WithContext(ctx).Create(apiKey).Error; err != nil {
		return errors.NewInternalError("failed to create api key: " + err.Error())
	}
	return nil
}

func (r *apiKeyRepository) GetByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	var apiKey model.APIKey
	if err := r.db.WithContext(ctx).Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.NewInternalError("failed to get api key by hash: " + err.Error())
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) GetByID(ctx context.Context, id string) (*model.APIKey, error) {
	var apiKey model.APIKey
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("api key not found")
		}
		return nil, errors.NewInternalError("failed to get api key: " + err.Error())
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) ListByUser(ctx context.Context, userID string, filter APIKeyFilter) ([]*model.APIKey, int64, error) {
	filter.SetDefaults()

	var keys []*model.APIKey
	var total int64

	query := r.db.WithContext(ctx).Model(&model.APIKey{}).Where("user_id = ?", userID)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to count api keys: " + err.Error())
	}

	if err := query.Order("created_at DESC").Offset(filter.Offset()).Limit(filter.PageSize).Find(&keys).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to list api keys: " + err.Error())
	}

	return keys, total, nil
}

func (r *apiKeyRepository) Update(ctx context.Context, apiKey *model.APIKey) error {
	result := r.db.WithContext(ctx).Save(apiKey)
	if result.Error != nil {
		return errors.NewInternalError("failed to update api key: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("api key not found")
	}
	return nil
}

func (r *apiKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&model.APIKey{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_used_at": now,
			"usage_count":  gorm.Expr("usage_count + 1"),
		})
	if result.Error != nil {
		return errors.NewInternalError("failed to update api key usage: " + result.Error.Error())
	}
	return nil
}
```

- [ ] **Step 2: Write repository tests**

```go
// internal/repository/api_key_repo_test.go
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/example/agent-infra/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAPIKeyTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.APIKey{}, &model.User{}, &model.Tenant{}))
	return db
}

func TestAPIKeyRepository_Create(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	apiKey := &model.APIKey{
		UserID:    "user-123",
		KeyHash:   model.HashKey("test-secret-key"),
		KeyPrefix: model.ExtractPrefix("test-secret-key"),
		Name:      "Test Key",
		Status:    model.APIKeyStatusActive,
	}

	err := repo.Create(ctx, apiKey)
	assert.NoError(t, err)
	assert.NotEmpty(t, apiKey.ID)
}

func TestAPIKeyRepository_GetByHash(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	secret := "ak-test-secret-key-123"
	apiKey := &model.APIKey{
		UserID:    "user-123",
		KeyHash:   model.HashKey(secret),
		KeyPrefix: model.ExtractPrefix(secret),
		Name:      "Test Key",
		Status:    model.APIKeyStatusActive,
	}
	require.NoError(t, repo.Create(ctx, apiKey))

	found, err := repo.GetByHash(ctx, model.HashKey(secret))
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, apiKey.ID, found.ID)

	notFound, err := repo.GetByHash(ctx, model.HashKey("wrong-key"))
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestAPIKeyRepository_GetByID(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	apiKey := &model.APIKey{
		UserID:    "user-123",
		KeyHash:   model.HashKey("secret"),
		KeyPrefix: "ak-test",
		Name:      "Test Key",
		Status:    model.APIKeyStatusActive,
	}
	require.NoError(t, repo.Create(ctx, apiKey))

	found, err := repo.GetByID(ctx, apiKey.ID)
	assert.NoError(t, err)
	assert.Equal(t, apiKey.Name, found.Name)

	_, err = repo.GetByID(ctx, "non-existent-id")
	assert.Error(t, err)
}

func TestAPIKeyRepository_ListByUser(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, &model.APIKey{
			UserID:    "user-123",
			KeyHash:   model.HashKey("secret-" + string(rune(i))),
			KeyPrefix: "ak-test",
			Name:      "Key " + string(rune(i)),
			Status:    model.APIKeyStatusActive,
		}))
	}
	require.NoError(t, repo.Create(ctx, &model.APIKey{
		UserID:    "user-123",
		KeyHash:   model.HashKey("revoked-secret"),
		KeyPrefix: "ak-rev",
		Name:      "Revoked Key",
		Status:    model.APIKeyStatusRevoked,
	}))

	keys, total, err := repo.ListByUser(ctx, "user-123", APIKeyFilter{PageSize: 10})
	assert.NoError(t, err)
	assert.Equal(t, int64(4), total)
	assert.Len(t, keys, 4)

	activeKeys, total, err := repo.ListByUser(ctx, "user-123", APIKeyFilter{Status: "active", PageSize: 10})
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, activeKeys, 3)
}

func TestAPIKeyRepository_UpdateLastUsed(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	apiKey := &model.APIKey{
		UserID:     "user-123",
		KeyHash:    model.HashKey("secret"),
		KeyPrefix:  "ak-test",
		Name:       "Test Key",
		Status:     model.APIKeyStatusActive,
		UsageCount: 0,
	}
	require.NoError(t, repo.Create(ctx, apiKey))
	before := time.Now()

	err := repo.UpdateLastUsed(ctx, apiKey.ID)
	assert.NoError(t, err)

	updated, err := repo.GetByID(ctx, apiKey.ID)
	assert.NoError(t, err)
	assert.True(t, updated.LastUsedAt.After(before))
	assert.Equal(t, int64(1), updated.UsageCount)
}
```

- [ ] **Step 3: Run tests to verify**

Run: `cd .claude/worktrees/issue-53 && go test ./internal/repository/ -run TestAPIKey -v`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
cd .claude/worktrees/issue-53
git add internal/repository/api_key_repo.go internal/repository/api_key_repo_test.go
git commit -m "feat(repository): add API key repository with CRUD operations"
```

---

## Task 2: User Repository

**Files:**
- Create: `internal/repository/user_repo.go`
- Create: `internal/repository/user_repo_test.go`

- [ ] **Step 1: Write UserRepository interface and implementation**

```go
// internal/repository/user_repo.go
package repository

import (
	"context"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"gorm.io/gorm"
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	GetByID(ctx context.Context, id string) (*model.User, error)
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("user not found")
		}
		return nil, errors.NewInternalError("failed to get user: " + err.Error())
	}
	return &user, nil
}
```

- [ ] **Step 2: Write user repository tests**

```go
// internal/repository/user_repo_test.go
package repository

import (
	"context"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUserTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Tenant{}, &model.User{}))
	return db
}

func TestUserRepository_GetByID(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create tenant first (FK constraint)
	tenant := &model.Tenant{Name: "test-tenant", Status: "active"}
	require.NoError(t, db.Create(tenant).Error)

	user := &model.User{
		TenantID: tenant.ID,
		Username: "testuser",
		Role:     model.UserRoleDeveloper,
		Status:   model.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)

	found, err := repo.GetByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Equal(t, user.Username, found.Username)
	assert.Equal(t, tenant.ID, found.TenantID)

	_, err = repo.GetByID(ctx, "non-existent-id")
	assert.Error(t, err)
}
```

- [ ] **Step 3: Run tests to verify**

Run: `cd .claude/worktrees/issue-53 && go test ./internal/repository/ -run TestUser -v`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
cd .claude/worktrees/issue-53
git add internal/repository/user_repo.go internal/repository/user_repo_test.go
git commit -m "feat(repository): add user repository for auth middleware lookup"
```

---

## Task 3: API Key Service

**Files:**
- Create: `internal/service/api_key_service.go`
- Create: `internal/service/api_key_service_test.go`

- [ ] **Step 1: Write APIKeyService interface and implementation**

```go
// internal/service/api_key_service.go
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
)

const (
	apiKeyPrefix = "ak"
	secretLength = 32
)

// CreateAPIKeyRequest represents the request to create a new API key.
type CreateAPIKeyRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ExpiresIn   *int   `json:"expires_in"` // hours until expiration, nil = no expiry
}

// APIKeyFilter represents filtering options for listing API keys.
type APIKeyFilter struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   string `form:"status"`
}

// APIKeyService defines the interface for API key business operations.
type APIKeyService interface {
	Create(ctx context.Context, userID string, req *CreateAPIKeyRequest) (*model.APIKey, string, error)
	GetByID(ctx context.Context, userID string, keyID string) (*model.APIKey, error)
	List(ctx context.Context, userID string, filter *APIKeyFilter) ([]*model.APIKey, int64, error)
	Revoke(ctx context.Context, userID string, keyID string) error
	Validate(ctx context.Context, rawKey string) (*model.APIKey, error)
}

type apiKeyService struct {
	repo repository.APIKeyRepository
}

// NewAPIKeyService creates a new APIKeyService instance.
func NewAPIKeyService(repo repository.APIKeyRepository) APIKeyService {
	return &apiKeyService{repo: repo}
}

func (s *apiKeyService) Create(ctx context.Context, userID string, req *CreateAPIKeyRequest) (*model.APIKey, string, error) {
	if req.Name == "" {
		return nil, "", errors.NewBadRequestError("api key name is required")
	}

	secret, err := generateSecret()
	if err != nil {
		return nil, "", errors.NewInternalError("failed to generate api key: " + err.Error())
	}

	rawKey := fmt.Sprintf("%s_%s", apiKeyPrefix, secret)
	keyHash := model.HashKey(rawKey)
	keyPrefix := model.ExtractPrefix(rawKey)

	apiKey := &model.APIKey{
		UserID:      userID,
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		Name:        req.Name,
		Description: req.Description,
		Status:      model.APIKeyStatusActive,
	}

	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Hour)
		apiKey.ExpiresAt = &expiresAt
	}

	if err := s.repo.Create(ctx, apiKey); err != nil {
		return nil, "", err
	}

	return apiKey, rawKey, nil
}

func (s *apiKeyService) GetByID(ctx context.Context, userID string, keyID string) (*model.APIKey, error) {
	apiKey, err := s.repo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if apiKey.UserID != userID {
		return nil, errors.NewNotFoundError("api key not found")
	}
	return apiKey, nil
}

func (s *apiKeyService) List(ctx context.Context, userID string, filter *APIKeyFilter) ([]*model.APIKey, int64, error) {
	repoFilter := repository.APIKeyFilter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Status:   filter.Status,
	}
	return s.repo.ListByUser(ctx, userID, repoFilter)
}

func (s *apiKeyService) Revoke(ctx context.Context, userID string, keyID string) error {
	apiKey, err := s.repo.GetByID(ctx, keyID)
	if err != nil {
		return err
	}
	if apiKey.UserID != userID {
		return errors.NewNotFoundError("api key not found")
	}
	if apiKey.Status == model.APIKeyStatusRevoked {
		return errors.NewBadRequestError("api key is already revoked")
	}

	apiKey.Status = model.APIKeyStatusRevoked
	return s.repo.Update(ctx, apiKey)
}

func (s *apiKeyService) Validate(ctx context.Context, rawKey string) (*model.APIKey, error) {
	keyHash := model.HashKey(rawKey)

	apiKey, err := s.repo.GetByHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, errors.NewBadRequestError("invalid api key")
	}
	if !apiKey.IsActive() {
		return nil, errors.NewBadRequestError("api key is inactive or expired")
	}

	// Update last used asynchronously (don't block the request)
	go func() {
		_ = s.repo.UpdateLastUsed(context.Background(), apiKey.ID)
	}()

	return apiKey, nil
}

func generateSecret() (string, error) {
	bytes := make([]byte, secretLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
```

- [ ] **Step 2: Write service tests**

```go
// internal/service/api_key_service_test.go
package service

import (
	"context"
	"testing"
	"time"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAPIKeyRepository is a mock for APIKeyRepository.
type MockAPIKeyRepository struct {
	mock.Mock
}

func (m *MockAPIKeyRepository) Create(ctx context.Context, apiKey *model.APIKey) error {
	args := m.Called(ctx, apiKey)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	args := m.Called(ctx, keyHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) GetByID(ctx context.Context, id string) (*model.APIKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) ListByUser(ctx context.Context, userID string, filter interface{}) ([]*model.APIKey, int64, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).([]*model.APIKey), args.Get(1).(int64), args.Error(2)
}

func (m *MockAPIKeyRepository) Update(ctx context.Context, apiKey *model.APIKey) error {
	args := m.Called(ctx, apiKey)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestAPIKeyService_Create(t *testing.T) {
	mockRepo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(mockRepo)
	ctx := context.Background()

	mockRepo.On("Create", ctx, mock.AnythingOfType("*model.APIKey")).Return(nil)

	apiKey, rawKey, err := svc.Create(ctx, "user-123", &CreateAPIKeyRequest{
		Name: "Test Key",
	})

	assert.NoError(t, err)
	assert.NotNil(t, apiKey)
	assert.NotEmpty(t, rawKey)
	assert.Contains(t, rawKey, "ak_")
	assert.Equal(t, "user-123", apiKey.UserID)
	assert.Equal(t, "Test Key", apiKey.Name)
	assert.Equal(t, model.APIKeyStatusActive, apiKey.Status)
	mockRepo.AssertExpectations(t)
}

func TestAPIKeyService_Create_WithExpiry(t *testing.T) {
	mockRepo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(mockRepo)
	ctx := context.Background()

	expiresIn := 24
	mockRepo.On("Create", ctx, mock.AnythingOfType("*model.APIKey")).Return(nil)

	apiKey, _, err := svc.Create(ctx, "user-123", &CreateAPIKeyRequest{
		Name:      "Expiring Key",
		ExpiresIn: &expiresIn,
	})

	assert.NoError(t, err)
	assert.NotNil(t, apiKey.ExpiresAt)
	assert.True(t, apiKey.ExpiresAt.After(time.Now()))
}

func TestAPIKeyService_Create_EmptyName(t *testing.T) {
	mockRepo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(mockRepo)
	ctx := context.Background()

	_, _, err := svc.Create(ctx, "user-123", &CreateAPIKeyRequest{Name: ""})
	assert.Error(t, err)
	assert.Equal(t, 400, err.(*errors.AppError).HTTPStatus)
}

func TestAPIKeyService_Revoke(t *testing.T) {
	mockRepo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(mockRepo)
	ctx := context.Background()

	apiKey := &model.APIKey{
		ID:     "key-123",
		UserID: "user-123",
		Status: model.APIKeyStatusActive,
	}
	mockRepo.On("GetByID", ctx, "key-123").Return(apiKey, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*model.APIKey")).Return(nil)

	err := svc.Revoke(ctx, "user-123", "key-123")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAPIKeyService_Revoke_WrongUser(t *testing.T) {
	mockRepo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(mockRepo)
	ctx := context.Background()

	apiKey := &model.APIKey{
		ID:     "key-123",
		UserID: "user-456",
		Status: model.APIKeyStatusActive,
	}
	mockRepo.On("GetByID", ctx, "key-123").Return(apiKey, nil)

	err := svc.Revoke(ctx, "user-123", "key-123")
	assert.Error(t, err)
	assert.Equal(t, 404, err.(*errors.AppError).HTTPStatus)
}

func TestAPIKeyService_Validate_Success(t *testing.T) {
	mockRepo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(mockRepo)
	ctx := context.Background()

	secret := "ak_testsecret123"
	apiKey := &model.APIKey{
		ID:     "key-123",
		UserID: "user-123",
		Status: model.APIKeyStatusActive,
	}
	mockRepo.On("GetByHash", ctx, model.HashKey(secret)).Return(apiKey, nil)
	mockRepo.On("UpdateLastUsed", mock.Anything, "key-123").Return(nil)

	validated, err := svc.Validate(ctx, secret)
	assert.NoError(t, err)
	assert.Equal(t, "key-123", validated.ID)
}

func TestAPIKeyService_Validate_InvalidKey(t *testing.T) {
	mockRepo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(mockRepo)
	ctx := context.Background()

	mockRepo.On("GetByHash", ctx, model.HashKey("wrong-key")).Return(nil, nil)

	_, err := svc.Validate(ctx, "wrong-key")
	assert.Error(t, err)
	assert.Equal(t, 400, err.(*errors.AppError).HTTPStatus)
}

func TestAPIKeyService_Validate_ExpiredKey(t *testing.T) {
	mockRepo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(mockRepo)
	ctx := context.Background()

	past := time.Now().Add(-24 * time.Hour)
	apiKey := &model.APIKey{
		ID:        "key-123",
		UserID:    "user-123",
		Status:    model.APIKeyStatusActive,
		ExpiresAt: &past,
	}
	mockRepo.On("GetByHash", ctx, model.HashKey("expired-key")).Return(apiKey, nil)

	_, err := svc.Validate(ctx, "expired-key")
	assert.Error(t, err)
}
```

- [ ] **Step 3: Run tests to verify**

Run: `cd .claude/worktrees/issue-53 && go test ./internal/service/ -run TestAPIKey -v`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
cd .claude/worktrees/issue-53
git add internal/service/api_key_service.go internal/service/api_key_service_test.go
git commit -m "feat(service): add API key service with create/revoke/validate"
```

---

## Task 4: Auth Middleware

**Files:**
- Create: `internal/api/middleware/auth.go`
- Create: `internal/api/middleware/auth_test.go`

- [ ] **Step 1: Write Authenticator interface and APIKeyAuth middleware**

```go
// internal/api/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/example/agent-infra/internal/api/response"
	"github.com/example/agent-infra/internal/repository"
	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyUserID is the gin context key for authenticated user ID.
	ContextKeyUserID = "user_id"
	// ContextKeyTenantID is the gin context key for authenticated user's tenant ID.
	ContextKeyTenantID = "tenant_id"
	// ContextKeyUserRole is the gin context key for authenticated user's role.
	ContextKeyUserRole = "user_role"
)

// APIKeyAuth creates a middleware that validates API Key authentication.
// It extracts the Bearer token from the Authorization header, validates it
// against the database, and injects user_id/tenant_id into the context.
func APIKeyAuth(apiKeyRepo repository.APIKeyRepository, userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			response.Unauthorized(c, "missing or invalid authorization header")
			c.Abort()
			return
		}

		// Look up API key by hash
		keyHash := modelHashKey(token)
		apiKey, err := apiKeyRepo.GetByHash(c.Request.Context(), keyHash)
		if err != nil {
			response.Unauthorized(c, "invalid api key")
			c.Abort()
			return
		}
		if apiKey == nil || !apiKey.IsActive() {
			response.Unauthorized(c, "invalid or expired api key")
			c.Abort()
			return
		}

		// Load user to get tenant_id
		user, err := userRepo.GetByID(c.Request.Context(), apiKey.UserID)
		if err != nil {
			response.Unauthorized(c, "user not found")
			c.Abort()
			return
		}
		if !user.IsActive() {
			response.Unauthorized(c, "user account is disabled")
			c.Abort()
			return
		}

		// Inject into context
		c.Set(ContextKeyUserID, user.ID)
		c.Set(ContextKeyTenantID, user.TenantID)
		c.Set(ContextKeyUserRole, string(user.Role))

		// Update last used asynchronously
		go func() {
			_ = apiKeyRepo.UpdateLastUsed(c.Request.Context(), apiKey.ID)
		}()

		c.Next()
	}
}

func extractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return ""
	}

	scheme := strings.ToLower(parts[0])
	if scheme != "bearer" {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

func modelHashKey(key string) string {
	return model.HashKey(key)
}
```

Wait — I shouldn't import model from middleware. Let me fix the approach: the middleware should accept a validate function or use the APIKeyRepository which already handles hashing. Actually, the HashKey function is in the model package, which is fine to import. Let me re-examine the project structure - the middleware already imports from `internal/api/response` and can import from `internal/model` and `internal/repository`.

Actually, looking more carefully at the architecture, the middleware should NOT directly call the repository. It should use the service layer. Let me fix this to follow Handler → Service → Repository pattern.

**Revised approach**: The middleware takes an `APIKeyService` (or just a validator function) instead of repositories directly. This keeps the layering clean.

```go
// internal/api/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/example/agent-infra/internal/api/response"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyUserID is the gin context key for authenticated user ID.
	ContextKeyUserID = "user_id"
	// ContextKeyTenantID is the gin context key for authenticated user's tenant ID.
	ContextKeyTenantID = "tenant_id"
	// ContextKeyUserRole is the gin context key for authenticated user's role.
	ContextKeyUserRole = "user_role"
)

// APIKeyAuth creates a middleware that validates API Key authentication.
func APIKeyAuth(apiKeySvc service.APIKeyService, userSvc service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			response.Unauthorized(c, "missing or invalid authorization header")
			c.Abort()
			return
		}

		// Validate API key
		apiKey, err := apiKeySvc.Validate(c.Request.Context(), token)
		if err != nil {
			response.Unauthorized(c, "invalid or expired api key")
			c.Abort()
			return
		}

		// Load user to get tenant_id
		user, err := userSvc.GetByID(c.Request.Context(), apiKey.UserID)
		if err != nil {
			response.Unauthorized(c, "user not found")
			c.Abort()
			return
		}

		// Inject into context
		c.Set(ContextKeyUserID, user.ID)
		c.Set(ContextKeyTenantID, user.TenantID)
		c.Set(ContextKeyUserRole, string(user.Role))

		c.Next()
	}
}

func extractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
```

This is cleaner but requires a `UserService` interface too. Let me add that in Task 3 as part of the service layer.

- [ ] **Step 2: Write middleware tests**

```go
// internal/api/middleware/auth_test.go
package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAPIKeyService struct {
	mock.Mock
}

func (m *MockAPIKeyService) Create(ctx context.Context, userID string, req *service.CreateAPIKeyRequest) (*model.APIKey, string, error) {
	return nil, "", nil
}

func (m *MockAPIKeyService) Validate(ctx context.Context, rawKey string) (*model.APIKey, error) {
	args := m.Called(ctx, rawKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.APIKey), args.Error(1)
}

// ... other methods stubbed

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetByID(ctx context.Context, id string) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func TestAPIKeyAuth_MissingHeader(t *testing.T) {
	mockAPIKeySvc := new(MockAPIKeyService)
	mockUserSvc := new(MockUserService)

	r := gin.New()
	r.Use(APIKeyAuth(mockAPIKeySvc, mockUserSvc))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	mockAPIKeySvc := new(MockAPIKeyService)
	mockUserSvc := new(MockUserService)

	apiKey := &model.APIKey{ID: "key-1", UserID: "user-1"}
	user := &model.User{ID: "user-1", TenantID: "tenant-1", Role: model.UserRoleDeveloper, Status: model.UserStatusActive}

	mockAPIKeySvc.On("Validate", mock.Anything, "ak_valid_token").Return(apiKey, nil)
	mockUserSvc.On("GetByID", mock.Anything, "user-1").Return(user, nil)

	r := gin.New()
	r.Use(APIKeyAuth(mockAPIKeySvc, mockUserSvc))
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get(ContextKeyUserID)
		tenantID, _ := c.Get(ContextKeyTenantID)
		c.JSON(200, gin.H{"user_id": userID, "tenant_id": tenantID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ak_valid_token")
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	mockAPIKeySvc := new(MockAPIKeyService)
	mockUserSvc := new(MockUserService)

	mockAPIKeySvc.On("Validate", mock.Anything, "bad_token").Return(nil, errors.New("invalid"))

	r := gin.New()
	r.Use(APIKeyAuth(mockAPIKeySvc, mockUserSvc))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer bad_token")
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}
```

- [ ] **Step 3: Run tests to verify**

Run: `cd .claude/worktrees/issue-53 && go test ./internal/api/middleware/ -run TestAPIKey -v`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
cd .claude/worktrees/issue-53
git add internal/api/middleware/auth.go internal/api/middleware/auth_test.go
git commit -m "feat(middleware): add API key auth middleware with context injection"
```

---

## Task 5: API Key Handler

**Files:**
- Create: `internal/api/handler/api_key.go`
- Create: `internal/api/handler/api_key_test.go`

- [ ] **Step 1: Write APIKeyHandler**

```go
// internal/api/handler/api_key.go
package handler

import (
	"github.com/example/agent-infra/internal/api/middleware"
	"github.com/example/agent-infra/internal/api/response"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// APIKeyHandler handles HTTP requests for API key operations.
type APIKeyHandler struct {
	service service.APIKeyService
}

// NewAPIKeyHandler creates a new APIKeyHandler instance.
func NewAPIKeyHandler(svc service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{service: svc}
}

// Create handles POST /api/v1/api-keys - Create a new API key.
// Returns the full API key in the response (only shown once).
func (h *APIKeyHandler) Create(c *gin.Context) {
	var req service.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)

	apiKey, rawKey, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, gin.H{
		"id":          apiKey.ID,
		"name":        apiKey.Name,
		"key_prefix":  apiKey.KeyPrefix,
		"secret":      rawKey,
		"expires_at":  apiKey.ExpiresAt,
		"created_at":  apiKey.CreatedAt,
	})
}

// List handles GET /api/v1/api-keys - List API keys for current user.
func (h *APIKeyHandler) List(c *gin.Context) {
	var filter service.APIKeyFilter
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

	userID := c.GetString(middleware.ContextKeyUserID)

	keys, total, err := h.service.List(c.Request.Context(), userID, &filter)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Paginated(c, keys, total, filter.Page, filter.PageSize)
}

// Revoke handles DELETE /api/v1/api-keys/:id - Revoke an API key.
func (h *APIKeyHandler) Revoke(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString(middleware.ContextKeyUserID)

	err := h.service.Revoke(c.Request.Context(), userID, id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "api key revoked successfully"})
}
```

- [ ] **Step 2: Write handler tests**

```go
// internal/api/handler/api_key_test.go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/agent-infra/internal/api/middleware"
	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAPIKeyServiceForHandler struct {
	mock.Mock
}

func (m *MockAPIKeyServiceForHandler) Create(ctx context.Context, userID string, req *service.CreateAPIKeyRequest) (*model.APIKey, string, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*model.APIKey), args.Get(1).(string), args.Error(2)
}

func (m *MockAPIKeyServiceForHandler) GetByID(ctx context.Context, userID string, keyID string) (*model.APIKey, error) {
	args := m.Called(ctx, userID, keyID)
	if args.Get(0) == nil {
		return nil, args.Error(2)
	}
	return args.Get(0).(*model.APIKey), args.Error(2)
}

func (m *MockAPIKeyServiceForHandler) List(ctx context.Context, userID string, filter *service.APIKeyFilter) ([]*model.APIKey, int64, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).([]*model.APIKey), args.Get(1).(int64), args.Error(2)
}

func (m *MockAPIKeyServiceForHandler) Revoke(ctx context.Context, userID string, keyID string) error {
	args := m.Called(ctx, userID, keyID)
	return args.Error(0)
}

func (m *MockAPIKeyServiceForHandler) Validate(ctx context.Context, rawKey string) (*model.APIKey, error) {
	args := m.Called(ctx, rawKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.APIKey), args.Error(1)
}

func setupAPIKeyRouter(mockSvc *MockAPIKeyServiceForHandler) *gin.Engine {
	r := gin.New()
	handler := NewAPIKeyHandler(mockSvc)

	// Simulate auth middleware setting user context
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, "user-123")
		c.Set(middleware.ContextKeyTenantID, "tenant-123")
		c.Next()
	})

	keys := r.Group("/api/v1/api-keys")
	{
		keys.POST("", handler.Create)
		keys.GET("", handler.List)
		keys.DELETE("/:id", handler.Revoke)
	}
	return r
}

func TestAPIKeyHandler_Create(t *testing.T) {
	mockSvc := new(MockAPIKeyServiceForHandler)
	r := setupAPIKeyRouter(mockSvc)

	apiKey := &model.APIKey{
		ID:        "key-123",
		Name:      "Test Key",
		KeyPrefix: "ak_abcd",
	}
	mockSvc.On("Create", mock.Anything, "user-123", mock.AnythingOfType("*service.CreateAPIKeyRequest")).
		Return(apiKey, "ak_abcd1234secret", nil)

	body, _ := json.Marshal(map[string]string{"name": "Test Key"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/api-keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "ak_abcd1234secret", data["secret"])
}

func TestAPIKeyHandler_Create_MissingName(t *testing.T) {
	mockSvc := new(MockAPIKeyServiceForHandler)
	r := setupAPIKeyRouter(mockSvc)

	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/api-keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

func TestAPIKeyHandler_List(t *testing.T) {
	mockSvc := new(MockAPIKeyServiceForHandler)
	r := setupAPIKeyRouter(mockSvc)

	mockSvc.On("List", mock.Anything, "user-123", mock.AnythingOfType("*service.APIKeyFilter")).
		Return([]*model.APIKey{}, int64(0), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/api-keys", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestAPIKeyHandler_Revoke(t *testing.T) {
	mockSvc := new(MockAPIKeyServiceForHandler)
	r := setupAPIKeyRouter(mockSvc)

	mockSvc.On("Revoke", mock.Anything, "user-123", "key-123").Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/api-keys/key-123", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}
```

- [ ] **Step 3: Run tests to verify**

Run: `cd .claude/worktrees/issue-53 && go test ./internal/api/handler/ -run TestAPIKey -v`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
cd .claude/worktrees/issue-53
git add internal/api/handler/api_key.go internal/api/handler/api_key_test.go
git commit -m "feat(handler): add API key CRUD endpoints"
```

---

## Task 6: Router Wiring + UserService

**Files:**
- Create: `internal/service/user_service.go` (minimal UserService interface + impl)
- Modify: `internal/api/router/router.go`
- Modify: `internal/api/router/router_test.go`

- [ ] **Step 1: Create minimal UserService**

The auth middleware needs a UserService to load user by ID. Create a minimal service:

```go
// internal/service/user_service.go
package service

import (
	"context"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
)

// UserService defines the interface for user business operations.
type UserService interface {
	GetByID(ctx context.Context, id string) (*model.User, error)
}

type userService struct {
	repo repository.UserRepository
}

// NewUserService creates a new UserService instance.
func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) GetByID(ctx context.Context, id string) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !user.IsActive() {
		return nil, errors.NewBadRequestError("user account is disabled")
	}
	return user, nil
}
```

- [ ] **Step 2: Update router to apply auth middleware and add api-keys routes**

Modify `internal/api/router/router.go`:

1. Add `apiKeySvc` and `userSvc` parameters to `Setup`
2. Create auth middleware instance
3. Apply to `/api/v1` group
4. Add api-keys routes

The updated `Setup` function signature becomes:

```go
func Setup(
	tenantSvc service.TenantService,
	templateSvc service.TemplateService,
	taskSvc service.TaskService,
	providerSvc service.ProviderService,
	capabilitySvc service.CapabilityService,
	monitorSvc service.MonitoringService,
	hub *monitoring.Hub,
	interventionSvc service.InterventionService,
	apiKeySvc service.APIKeyService,
	userSvc service.UserService,
	db DBChecker,
) *gin.Engine {
```

Apply auth middleware to v1 group:

```go
v1 := r.Group("/api/v1")
v1.Use(middleware.APIKeyAuth(apiKeySvc, userSvc))
```

Add api-keys routes:

```go
// API Key routes
apiKeyHandler := handler.NewAPIKeyHandler(apiKeySvc)
apiKeys := v1.Group("/api-keys")
{
	apiKeys.POST("", apiKeyHandler.Create)
	apiKeys.GET("", apiKeyHandler.List)
	apiKeys.DELETE("/:id", apiKeyHandler.Revoke)
}
```

- [ ] **Step 3: Update router_test.go to include new parameters**

Add nil/zero params for apiKeySvc and userSvc in test setup.

- [ ] **Step 4: Run router tests**

Run: `cd .claude/worktrees/issue-53 && go test ./internal/api/router/ -v`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
cd .claude/worktrees/issue-53
git add internal/service/user_service.go internal/api/router/router.go internal/api/router/router_test.go
git commit -m "feat(router): wire auth middleware to all /api/v1/* routes, add api-keys endpoints"
```

---

## Task 7: Main Wiring + Integration

**Files:**
- Modify: `cmd/control-plane/main.go`

- [ ] **Step 1: Wire new components in main.go**

In the `// 6. Repositories` section, add:

```go
apiKeyRepo := repository.NewAPIKeyRepository(db.DB)
userRepo := repository.NewUserRepository(db.DB)
```

In the `// 7. Services` section, add:

```go
apiKeySvc := service.NewAPIKeyService(apiKeyRepo)
userSvc := service.NewUserService(userRepo)
```

Update the `router.Setup` call to pass the new services:

```go
r := router.Setup(tenantSvc, templateSvc, taskSvc, providerSvc, capabilitySvc, monitoringSvc, monitoringHub, interventionSvc, apiKeySvc, userSvc, db)
```

- [ ] **Step 2: Run full test suite**

Run: `cd .claude/worktrees/issue-53 && go test ./... -v 2>&1 | tail -50`
Expected: All tests PASS, no compilation errors

- [ ] **Step 3: Commit**

```bash
cd .claude/worktrees/issue-53
git add cmd/control-plane/main.go
git commit -m "feat(main): wire auth middleware and API key service into startup"
```

---

## Task 8: Update Issue Summary + Final Verification

**Files:**
- Modify: `docs/v1.0-mvp/issues/issue-53-auth-middleware.md`

- [ ] **Step 1: Update issue summary checkboxes**

Mark completed scope items and acceptance criteria in the issue summary file.

- [ ] **Step 2: Run full test suite with coverage**

Run: `cd .claude/worktrees/issue-53 && go test -cover ./internal/...`
Expected: Coverage > 80% for new code

- [ ] **Step 3: Run linter**

Run: `cd .claude/worktrees/issue-53 && make lint`
Expected: No errors

- [ ] **Step 4: Final commit**

```bash
cd .claude/worktrees/issue-53
git add docs/v1.0-mvp/issues/issue-53-auth-middleware.md
git commit -m "docs(issue-53): update issue summary with completion status"
```
