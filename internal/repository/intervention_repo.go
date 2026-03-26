package repository

import (
	"context"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InterventionFilter represents filtering options for listing interventions.
type InterventionFilter struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	Action     string `form:"action"`
	Status     string `form:"status"`
	OperatorID string `form:"operator_id"`
}

// SetDefaults sets default values for the filter.
func (f *InterventionFilter) SetDefaults() {
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
func (f *InterventionFilter) Offset() int {
	return (f.Page - 1) * f.PageSize
}

// InterventionRepository defines the interface for intervention data access operations.
type InterventionRepository interface {
	Create(ctx context.Context, intervention *model.Intervention) error
	GetByID(ctx context.Context, id string) (*model.Intervention, error)
	ListByTask(ctx context.Context, taskID string, filter InterventionFilter) ([]*model.Intervention, int64, error)
	Update(ctx context.Context, intervention *model.Intervention) error
}

// interventionRepository implements InterventionRepository using GORM.
type interventionRepository struct {
	db *gorm.DB
}

// NewInterventionRepository creates a new InterventionRepository instance.
func NewInterventionRepository(db *gorm.DB) InterventionRepository {
	return &interventionRepository{db: db}
}

// Create inserts a new intervention into the database.
func (r *interventionRepository) Create(ctx context.Context, intervention *model.Intervention) error {
	if err := r.db.WithContext(ctx).Create(intervention).Error; err != nil {
		return errors.NewInternalError("failed to create intervention: " + err.Error())
	}
	return nil
}

// GetByID retrieves an intervention by its ID.
func (r *interventionRepository) GetByID(ctx context.Context, id string) (*model.Intervention, error) {
	var intervention model.Intervention
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid intervention ID format")
	}
	if err := r.db.WithContext(ctx).First(&intervention, "id = ?", parsedID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("intervention not found")
		}
		return nil, errors.NewInternalError("failed to get intervention: " + err.Error())
	}
	return &intervention, nil
}

// ListByTask retrieves interventions for a specific task based on filter criteria.
func (r *interventionRepository) ListByTask(ctx context.Context, taskID string, filter InterventionFilter) ([]*model.Intervention, int64, error) {
	filter.SetDefaults()

	var interventions []*model.Intervention
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Intervention{}).Where("task_id = ?", taskID)

	// Apply filters
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.OperatorID != "" {
		query = query.Where("operator_id = ?", filter.OperatorID)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to count interventions: " + err.Error())
	}

	// Get paginated results
	if err := query.Offset(filter.Offset()).Limit(filter.PageSize).Order("created_at DESC").Find(&interventions).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to list interventions: " + err.Error())
	}

	return interventions, total, nil
}

// Update updates an existing intervention.
func (r *interventionRepository) Update(ctx context.Context, intervention *model.Intervention) error {
	result := r.db.WithContext(ctx).Save(intervention)
	if result.Error != nil {
		return errors.NewInternalError("failed to update intervention: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("intervention not found")
	}
	return nil
}
