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
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	TenantID string `form:"tenant_id"`
	Type     string `form:"type"`
	Status   string `form:"status"`
	Search   string `form:"search"`
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
