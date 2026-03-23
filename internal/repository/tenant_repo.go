package repository

import (
	"context"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantRepository defines the interface for tenant data access operations.
type TenantRepository interface {
	Create(ctx context.Context, tenant *model.Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error)
	List(ctx context.Context, filter TenantFilter) ([]*model.Tenant, int64, error)
	Update(ctx context.Context, tenant *model.Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// tenantRepository implements TenantRepository using GORM.
type tenantRepository struct {
	db *gorm.DB
}

// NewTenantRepository creates a new TenantRepository instance.
func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantRepository{db: db}
}

// Create inserts a new tenant into the database.
func (r *tenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	if err := r.db.WithContext(ctx).Create(tenant).Error; err != nil {
		return errors.NewInternalError("failed to create tenant: " + err.Error())
	}
	return nil
}

// GetByID retrieves a tenant by its ID.
func (r *tenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := r.db.WithContext(ctx).First(&tenant, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("tenant not found")
		}
		return nil, errors.NewInternalError("failed to get tenant: " + err.Error())
	}
	return &tenant, nil
}

// List retrieves tenants based on filter criteria.
func (r *tenantRepository) List(ctx context.Context, filter TenantFilter) ([]*model.Tenant, int64, error) {
	filter.SetDefaults()

	var tenants []*model.Tenant
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Tenant{})

	// Apply filters
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("name LIKE ?", search)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to count tenants: " + err.Error())
	}

	// Get paginated results
	if err := query.Offset(filter.Offset()).Limit(filter.PageSize).Find(&tenants).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to list tenants: " + err.Error())
	}

	return tenants, total, nil
}

// Update updates an existing tenant.
func (r *tenantRepository) Update(ctx context.Context, tenant *model.Tenant) error {
	result := r.db.WithContext(ctx).Save(tenant)
	if result.Error != nil {
		return errors.NewInternalError("failed to update tenant: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("tenant not found")
	}
	return nil
}

// Delete performs a soft delete on a tenant.
func (r *tenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&model.Tenant{}, "id = ?", id)
	if result.Error != nil {
		return errors.NewInternalError("failed to delete tenant: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("tenant not found")
	}
	return nil
}
