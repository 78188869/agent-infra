package repository

import (
	"context"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TemplateRepository defines the interface for template data access operations.
type TemplateRepository interface {
	Create(ctx context.Context, template *model.Template) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error)
	List(ctx context.Context, filter TemplateFilter) ([]*model.Template, int64, error)
	Update(ctx context.Context, template *model.Template) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// templateRepository implements TemplateRepository using GORM.
type templateRepository struct {
	db *gorm.DB
}

// NewTemplateRepository creates a new TemplateRepository instance.
func NewTemplateRepository(db *gorm.DB) TemplateRepository {
	return &templateRepository{db: db}
}

// Create inserts a new template into the database.
func (r *templateRepository) Create(ctx context.Context, template *model.Template) error {
	if err := r.db.WithContext(ctx).Create(template).Error; err != nil {
		return errors.NewInternalError("failed to create template: " + err.Error())
	}
	return nil
}

// GetByID retrieves a template by its ID.
func (r *templateRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	var template model.Template
	if err := r.db.WithContext(ctx).First(&template, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("template not found")
		}
		return nil, errors.NewInternalError("failed to get template: " + err.Error())
	}
	return &template, nil
}

// List retrieves templates based on filter criteria.
func (r *templateRepository) List(ctx context.Context, filter TemplateFilter) ([]*model.Template, int64, error) {
	filter.SetDefaults()

	var templates []*model.Template
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Template{})

	// Apply filters
	if filter.TenantID != "" {
		query = query.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.SceneType != "" {
		query = query.Where("scene_type = ?", filter.SceneType)
	}
	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("name LIKE ?", search)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to count templates: " + err.Error())
	}

	// Get paginated results
	if err := query.Offset(filter.Offset()).Limit(filter.PageSize).Find(&templates).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to list templates: " + err.Error())
	}

	return templates, total, nil
}

// Update updates an existing template.
func (r *templateRepository) Update(ctx context.Context, template *model.Template) error {
	result := r.db.WithContext(ctx).Save(template)
	if result.Error != nil {
		return errors.NewInternalError("failed to update template: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("template not found")
	}
	return nil
}

// Delete performs a soft delete on a template.
func (r *templateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&model.Template{}, "id = ?", id)
	if result.Error != nil {
		return errors.NewInternalError("failed to delete template: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("template not found")
	}
	return nil
}
