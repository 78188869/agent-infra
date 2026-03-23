package repository

import (
	"context"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskRepository defines the interface for task data access operations.
type TaskRepository interface {
	Create(ctx context.Context, task *model.Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Task, error)
	List(ctx context.Context, filter TaskFilter) ([]*model.Task, int64, error)
	Update(ctx context.Context, task *model.Task) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByStatus(ctx context.Context, status string, limit int) ([]*model.Task, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, reason string) error
}

// taskRepository implements TaskRepository using GORM.
type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository creates a new TaskRepository instance.
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

// Create inserts a new task into the database.
func (r *taskRepository) Create(ctx context.Context, task *model.Task) error {
	if err := r.db.WithContext(ctx).Create(task).Error; err != nil {
		return errors.NewInternalError("failed to create task: " + err.Error())
	}
	return nil
}

// GetByID retrieves a task by its ID.
func (r *taskRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Task, error) {
	var task model.Task
	if err := r.db.WithContext(ctx).First(&task, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("task not found")
		}
		return nil, errors.NewInternalError("failed to get task: " + err.Error())
	}
	return &task, nil
}

// List retrieves tasks based on filter criteria.
func (r *taskRepository) List(ctx context.Context, filter TaskFilter) ([]*model.Task, int64, error) {
	filter.SetDefaults()

	var tasks []*model.Task
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Task{})

	// Apply filters
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.TenantID != "" {
		query = query.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("name LIKE ?", search)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to count tasks: " + err.Error())
	}

	// Get paginated results
	if err := query.Offset(filter.Offset()).Limit(filter.PageSize).Find(&tasks).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to list tasks: " + err.Error())
	}

	return tasks, total, nil
}

// Update updates an existing task.
func (r *taskRepository) Update(ctx context.Context, task *model.Task) error {
	result := r.db.WithContext(ctx).Save(task)
	if result.Error != nil {
		return errors.NewInternalError("failed to update task: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("task not found")
	}
	return nil
}

// Delete performs a soft delete on a task.
func (r *taskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&model.Task{}, "id = ?", id)
	if result.Error != nil {
		return errors.NewInternalError("failed to delete task: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("task not found")
	}
	return nil
}

// ListByStatus retrieves tasks with a specific status.
func (r *taskRepository) ListByStatus(ctx context.Context, status string, limit int) ([]*model.Task, error) {
	var tasks []*model.Task

	query := r.db.WithContext(ctx).Model(&model.Task{}).Where("status = ?", status)
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&tasks).Error; err != nil {
		return nil, errors.NewInternalError("failed to list tasks by status: " + err.Error())
	}

	return tasks, nil
}

// UpdateStatus updates the status of a task.
func (r *taskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, reason string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	// Set error message if status is failed
	if status == model.TaskStatusFailed && reason != "" {
		updates["error_message"] = reason
	}

	result := r.db.WithContext(ctx).Model(&model.Task{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return errors.NewInternalError("failed to update task status: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("task not found")
	}

	return nil
}
