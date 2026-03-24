package repository

import (
	"context"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
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

// SetDefaults sets default values for the filter.
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

// Offset returns the calculated offset for pagination.
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

// providerRepository implements ProviderRepository using GORM.
type providerRepository struct {
	db *gorm.DB
}

// NewProviderRepository creates a new ProviderRepository instance.
func NewProviderRepository(db *gorm.DB) ProviderRepository {
	return &providerRepository{db: db}
}

// Create inserts a new provider into the database.
func (r *providerRepository) Create(ctx context.Context, provider *model.Provider) error {
	if err := r.db.WithContext(ctx).Create(provider).Error; err != nil {
		return errors.NewInternalError("failed to create provider: " + err.Error())
	}
	return nil
}

// GetByID retrieves a provider by its ID.
func (r *providerRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
	var provider model.Provider
	if err := r.db.WithContext(ctx).First(&provider, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("provider not found")
		}
		return nil, errors.NewInternalError("failed to get provider: " + err.Error())
	}
	return &provider, nil
}

// List retrieves providers based on filter criteria.
func (r *providerRepository) List(ctx context.Context, filter ProviderFilter) ([]*model.Provider, int64, error) {
	filter.SetDefaults()

	var providers []*model.Provider
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Provider{})

	// Apply scope filtering
	if filter.Scope != "" {
		query = query.Where("scope = ?", filter.Scope)
	}
	if filter.TenantID != "" {
		query = query.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
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
		return nil, 0, errors.NewInternalError("failed to count providers: " + err.Error())
	}

	// Get paginated results
	if err := query.Offset(filter.Offset()).Limit(filter.PageSize).Find(&providers).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to list providers: " + err.Error())
	}

	return providers, total, nil
}

// Update updates an existing provider.
func (r *providerRepository) Update(ctx context.Context, provider *model.Provider) error {
	// First check if the provider exists
	var existing model.Provider
	if err := r.db.WithContext(ctx).Where("id = ?", provider.ID).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError("provider not found")
		}
		return errors.NewInternalError("failed to check provider existence: " + err.Error())
	}

	// Now perform the update
	result := r.db.WithContext(ctx).Save(provider)
	if result.Error != nil {
		return errors.NewInternalError("failed to update provider: " + result.Error.Error())
	}
	return nil
}

// Delete performs a soft delete on a provider.
func (r *providerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&model.Provider{}, "id = ?", id)
	if result.Error != nil {
		return errors.NewInternalError("failed to delete provider: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("provider not found")
	}
	return nil
}

// GetByScopeAndName retrieves a provider by scope, tenant/user context, and name.
// This is used for unique name lookups within a specific scope.
func (r *providerRepository) GetByScopeAndName(ctx context.Context, scope model.ProviderScope, tenantID, userID *string, name string) (*model.Provider, error) {
	var provider model.Provider

	query := r.db.WithContext(ctx).Where("scope = ? AND name = ?", scope, name)

	// Add scope-specific conditions
	switch scope {
	case model.ProviderScopeSystem:
		query = query.Where("tenant_id IS NULL AND user_id IS NULL")
	case model.ProviderScopeTenant:
		if tenantID != nil {
			query = query.Where("tenant_id = ?", *tenantID)
		}
		query = query.Where("user_id IS NULL")
	case model.ProviderScopeUser:
		if userID != nil {
			query = query.Where("user_id = ?", *userID)
		}
	}

	if err := query.First(&provider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("provider not found")
		}
		return nil, errors.NewInternalError("failed to get provider: " + err.Error())
	}
	return &provider, nil
}

// GetDefaultProvider retrieves the default provider for a given scope and context.
// This returns the first active provider in the specified scope.
func (r *providerRepository) GetDefaultProvider(ctx context.Context, scope model.ProviderScope, tenantID, userID *string) (*model.Provider, error) {
	var provider model.Provider

	query := r.db.WithContext(ctx).
		Where("scope = ? AND status = ?", scope, model.ProviderStatusActive)

	// Add scope-specific conditions
	switch scope {
	case model.ProviderScopeSystem:
		query = query.Where("tenant_id IS NULL AND user_id IS NULL")
	case model.ProviderScopeTenant:
		if tenantID != nil {
			query = query.Where("tenant_id = ?", *tenantID)
		}
		query = query.Where("user_id IS NULL")
	case model.ProviderScopeUser:
		if userID != nil {
			query = query.Where("user_id = ?", *userID)
		}
	}

	// Order by created_at to get the oldest (first created) as default
	if err := query.Order("created_at ASC").First(&provider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("provider not found")
		}
		return nil, errors.NewInternalError("failed to get default provider: " + err.Error())
	}
	return &provider, nil
}

// SetUserDefaultProvider sets or updates the default provider for a user.
// This uses upsert logic - if a default exists, it updates; otherwise, it creates.
func (r *providerRepository) SetUserDefaultProvider(ctx context.Context, userID, providerID string) error {
	// Check if user default already exists
	var existing model.UserProviderDefault
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Create new default
		userDefault := &model.UserProviderDefault{
			UserID:     userID,
			ProviderID: providerID,
		}
		if err := r.db.WithContext(ctx).Create(userDefault).Error; err != nil {
			return errors.NewInternalError("failed to set user default provider: " + err.Error())
		}
		return nil
	}

	if err != nil {
		return errors.NewInternalError("failed to check user default provider: " + err.Error())
	}

	// Update existing default
	if err := r.db.WithContext(ctx).
		Model(&existing).
		Update("provider_id", providerID).Error; err != nil {
		return errors.NewInternalError("failed to update user default provider: " + err.Error())
	}

	return nil
}

// GetUserDefaultProvider retrieves the default provider for a user.
func (r *providerRepository) GetUserDefaultProvider(ctx context.Context, userID string) (*model.Provider, error) {
	var userDefault model.UserProviderDefault
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&userDefault).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("user default provider not found")
		}
		return nil, errors.NewInternalError("failed to get user default: " + err.Error())
	}

	// Get the provider
	var provider model.Provider
	if err := r.db.WithContext(ctx).
		First(&provider, "id = ?", userDefault.ProviderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("provider not found")
		}
		return nil, errors.NewInternalError("failed to get provider: " + err.Error())
	}

	return &provider, nil
}
