// Package service provides business logic implementations for the application.
package service

import (
	"context"
	"encoding/json"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// CreateTaskRequest represents the request to create a new task.
type CreateTaskRequest struct {
	TenantID   string          `json:"tenant_id" binding:"required"`
	TemplateID *string         `json:"template_id"`
	CreatorID  string          `json:"creator_id" binding:"required"`
	ProviderID string          `json:"provider_id" binding:"required"`
	Name       string          `json:"name" binding:"required"`
	Priority   string          `json:"priority"`
	Params     json.RawMessage `json:"params"`
}

// UpdateTaskRequest represents the request to update an existing task.
type UpdateTaskRequest struct {
	Status       *string         `json:"status"`
	ErrorMessage *string         `json:"error_message"`
	Result       json.RawMessage `json:"result"`
}

// TaskFilter represents filtering options for listing tasks.
type TaskFilter struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   string `form:"status"`
	TenantID string `form:"tenant_id"`
	Search   string `form:"search"`
}

// validStatusTransitions defines the allowed status transitions.
var validStatusTransitions = map[string][]string{
	model.TaskStatusPending:         {model.TaskStatusScheduled, model.TaskStatusCancelled},
	model.TaskStatusScheduled:       {model.TaskStatusRunning, model.TaskStatusCancelled},
	model.TaskStatusRunning:         {model.TaskStatusPaused, model.TaskStatusSucceeded, model.TaskStatusFailed, model.TaskStatusCancelled},
	model.TaskStatusPaused:          {model.TaskStatusRunning, model.TaskStatusCancelled},
	model.TaskStatusWaitingApproval: {model.TaskStatusRunning, model.TaskStatusCancelled},
	model.TaskStatusRetrying:        {model.TaskStatusRunning, model.TaskStatusFailed, model.TaskStatusCancelled},
	// Terminal states - no transitions allowed
	model.TaskStatusSucceeded: {},
	model.TaskStatusFailed:    {},
	model.TaskStatusCancelled: {},
}

// TaskService defines the interface for task business operations.
type TaskService interface {
	Create(ctx context.Context, req *CreateTaskRequest) (*model.Task, error)
	GetByID(ctx context.Context, id string) (*model.Task, error)
	List(ctx context.Context, filter *TaskFilter) ([]*model.Task, int64, error)
	Update(ctx context.Context, id string, req *UpdateTaskRequest) error
	Delete(ctx context.Context, id string) error
}

// taskService implements TaskService.
type taskService struct {
	repo repository.TaskRepository
}

// NewTaskService creates a new TaskService instance.
func NewTaskService(repo repository.TaskRepository) TaskService {
	return &taskService{repo: repo}
}

// Create creates a new task with validation.
func (s *taskService) Create(ctx context.Context, req *CreateTaskRequest) (*model.Task, error) {
	// Validate required fields
	if req.TenantID == "" {
		return nil, errors.NewBadRequestError("tenant_id is required")
	}
	if req.CreatorID == "" {
		return nil, errors.NewBadRequestError("creator_id is required")
	}
	if req.ProviderID == "" {
		return nil, errors.NewBadRequestError("provider_id is required")
	}
	if req.Name == "" {
		return nil, errors.NewBadRequestError("name is required")
	}

	// Validate tenant ID format
	if _, err := uuid.Parse(req.TenantID); err != nil {
		return nil, errors.NewBadRequestError("invalid tenant_id format")
	}

	// Validate creator ID format
	if _, err := uuid.Parse(req.CreatorID); err != nil {
		return nil, errors.NewBadRequestError("invalid creator_id format")
	}

	// Validate provider ID format
	if _, err := uuid.Parse(req.ProviderID); err != nil {
		return nil, errors.NewBadRequestError("invalid provider_id format")
	}

	// Validate template exists (stub - always returns true for now)
	if req.TemplateID != nil && *req.TemplateID != "" {
		if _, err := uuid.Parse(*req.TemplateID); err != nil {
			return nil, errors.NewBadRequestError("invalid template_id format")
		}
		// TODO: Actually validate template exists when template service is available
	}

	// Validate params as JSON
	var params datatypes.JSON
	if len(req.Params) > 0 {
		if !json.Valid(req.Params) {
			return nil, errors.NewBadRequestError("params must be valid JSON")
		}
		params = datatypes.JSON(req.Params)
	}

	// Check tenant quota (stub - always returns true for now)
	// TODO: Implement actual quota check when quota service is available

	// Set default priority if not provided
	priority := req.Priority
	if priority == "" {
		priority = model.TaskPriorityNormal
	}

	// Validate priority value
	if priority != model.TaskPriorityHigh && priority != model.TaskPriorityNormal && priority != model.TaskPriorityLow {
		return nil, errors.NewBadRequestError("invalid priority value, must be 'high', 'normal', or 'low'")
	}

	// Create task model
	task := &model.Task{
		TenantID:   req.TenantID,
		TemplateID: req.TemplateID,
		CreatorID:  req.CreatorID,
		ProviderID: req.ProviderID,
		Name:       req.Name,
		Status:     model.TaskStatusPending,
		Priority:   priority,
		Params:     params,
	}

	// Call repository
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

// GetByID retrieves a task by its ID.
func (s *taskService) GetByID(ctx context.Context, id string) (*model.Task, error) {
	// Parse and validate ID
	taskID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid task ID format")
	}

	// Call repository
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// List retrieves tasks based on filter criteria.
func (s *taskService) List(ctx context.Context, filter *TaskFilter) ([]*model.Task, int64, error) {
	// Convert service filter to repository filter
	repoFilter := repository.TaskFilter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Status:   filter.Status,
		TenantID: filter.TenantID,
		Search:   filter.Search,
	}

	// Call repository
	tasks, total, err := s.repo.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// Update updates an existing task with status transition validation.
func (s *taskService) Update(ctx context.Context, id string, req *UpdateTaskRequest) error {
	// Parse and validate ID
	taskID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid task ID format")
	}

	// Get existing task
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	// Validate and apply status updates
	if req.Status != nil {
		if !isValidStatusTransition(task.Status, *req.Status) {
			return errors.NewBadRequestError("invalid status transition from '" + task.Status + "' to '" + *req.Status + "'")
		}
		task.Status = *req.Status
	}

	// Update error message if provided
	if req.ErrorMessage != nil {
		task.ErrorMessage = *req.ErrorMessage
	}

	// Update result if provided
	if len(req.Result) > 0 {
		if !json.Valid(req.Result) {
			return errors.NewBadRequestError("result must be valid JSON")
		}
		task.Result = datatypes.JSON(req.Result)
	}

	// Call repository
	return s.repo.Update(ctx, task)
}

// Delete performs a soft delete on a task.
func (s *taskService) Delete(ctx context.Context, id string) error {
	// Parse and validate ID
	taskID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid task ID format")
	}

	// Call repository
	return s.repo.Delete(ctx, taskID)
}

// isValidStatusTransition checks if a status transition is valid.
func isValidStatusTransition(from, to string) bool {
	allowedTransitions, exists := validStatusTransitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == to {
			return true
		}
	}
	return false
}
