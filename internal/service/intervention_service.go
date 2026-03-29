// Package service provides business logic implementations for the application.
package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// InjectInterventionRequest represents the request to inject an intervention.
type InjectInterventionRequest struct {
	TaskID      string          `json:"task_id" binding:"required"`
	OperatorID  string          `json:"operator_id" binding:"required"`
	Instruction string          `json:"instruction" binding:"required"`
	Context     string          `json:"context"`
}

// InterventionFilter represents filtering options for listing interventions.
type InterventionFilter struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	Action     string `form:"action"`
	Status     string `form:"status"`
	OperatorID string `form:"operator_id"`
}

// InterventionService defines the interface for intervention business operations.
type InterventionService interface {
	Pause(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error)
	Resume(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error)
	Cancel(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error)
	Inject(ctx context.Context, req *InjectInterventionRequest) (*model.Intervention, error)
	ListInterventions(ctx context.Context, taskID string, filter *InterventionFilter) ([]*model.Intervention, int64, error)
	HandleWrapperEvent(ctx context.Context, taskID string, eventType string, payload map[string]interface{}) error
}

// TaskEventHandler defines the interface for handling task events from the executor.
// This decouples the service layer from the executor package.
type TaskEventHandler interface {
	HandleTaskEvent(ctx context.Context, taskID string, eventType string, payload map[string]interface{}) error
}

// interventionService implements InterventionService.
type interventionService struct {
	taskRepo         repository.TaskRepository
	interventionRepo repository.InterventionRepository
	eventHandler     TaskEventHandler
}

// NewInterventionService creates a new InterventionService instance.
func NewInterventionService(
	taskRepo repository.TaskRepository,
	interventionRepo repository.InterventionRepository,
) InterventionService {
	return &interventionService{
		taskRepo:         taskRepo,
		interventionRepo: interventionRepo,
	}
}

// SetEventHandler sets the task event handler for the service.
// This is used to break circular dependencies between service and executor packages.
func (s *interventionService) SetEventHandler(handler TaskEventHandler) {
	s.eventHandler = handler
}

// canPause checks if a task can be paused based on its current status.
func canPause(status string) bool {
	return status == model.TaskStatusRunning
}

// canResume checks if a task can be resumed based on its current status.
func canResume(status string) bool {
	return status == model.TaskStatusPaused
}

// canCancel checks if a task can be cancelled based on its current status.
func canCancel(status string) bool {
	validStates := map[string]bool{
		model.TaskStatusPending:         true,
		model.TaskStatusScheduled:       true,
		model.TaskStatusRunning:         true,
		model.TaskStatusPaused:          true,
		model.TaskStatusWaitingApproval: true,
		model.TaskStatusRetrying:        true,
	}
	return validStates[status]
}

// canInject checks if a task can receive an injection based on its current status.
func canInject(status string) bool {
	return status == model.TaskStatusRunning || status == model.TaskStatusWaitingApproval
}

// Pause pauses a running task.
func (s *interventionService) Pause(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error) {
	// Validate IDs
	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid task ID format")
	}

	_, err = uuid.Parse(operatorID)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid operator ID format")
	}

	// Get task
	task, err := s.taskRepo.GetByID(ctx, taskUUID)
	if err != nil {
		return nil, err
	}

	// Validate task state
	if !canPause(task.Status) {
		return nil, errors.NewBadRequestError("cannot pause task in '" + task.Status + "' state")
	}

	// Create intervention record
	intervention := &model.Intervention{
		TaskID:     taskID,
		OperatorID: operatorID,
		Action:     model.InterventionActionPause,
		Reason:     reason,
		Status:     model.InterventionStatusPending,
	}

	if err := s.interventionRepo.Create(ctx, intervention); err != nil {
		return nil, err
	}

	// Update task status
	task.Status = model.TaskStatusPaused
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Update intervention status to applied
	intervention.Status = model.InterventionStatusApplied
	result := model.InterventionResult{
		Success:   true,
		Message:   "Task paused successfully",
		Timestamp: time.Now().Unix(),
	}
	resultJSON, _ := json.Marshal(result)
	intervention.Result = datatypes.JSON(resultJSON)

	if err := s.interventionRepo.Update(ctx, intervention); err != nil {
		return nil, err
	}

	return intervention, nil
}

// Resume resumes a paused task.
func (s *interventionService) Resume(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error) {
	// Validate IDs
	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid task ID format")
	}

	_, err = uuid.Parse(operatorID)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid operator ID format")
	}

	// Get task
	task, err := s.taskRepo.GetByID(ctx, taskUUID)
	if err != nil {
		return nil, err
	}

	// Validate task state
	if !canResume(task.Status) {
		return nil, errors.NewBadRequestError("cannot resume task in '" + task.Status + "' state")
	}

	// Create intervention record
	intervention := &model.Intervention{
		TaskID:     taskID,
		OperatorID: operatorID,
		Action:     model.InterventionActionResume,
		Reason:     reason,
		Status:     model.InterventionStatusPending,
	}

	if err := s.interventionRepo.Create(ctx, intervention); err != nil {
		return nil, err
	}

	// Update task status
	task.Status = model.TaskStatusRunning
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Update intervention status to applied
	intervention.Status = model.InterventionStatusApplied
	result := model.InterventionResult{
		Success:   true,
		Message:   "Task resumed successfully",
		Timestamp: time.Now().Unix(),
	}
	resultJSON, _ := json.Marshal(result)
	intervention.Result = datatypes.JSON(resultJSON)

	if err := s.interventionRepo.Update(ctx, intervention); err != nil {
		return nil, err
	}

	return intervention, nil
}

// Cancel cancels a task.
func (s *interventionService) Cancel(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error) {
	// Validate IDs
	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid task ID format")
	}

	_, err = uuid.Parse(operatorID)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid operator ID format")
	}

	// Get task
	task, err := s.taskRepo.GetByID(ctx, taskUUID)
	if err != nil {
		return nil, err
	}

	// Validate task state
	if !canCancel(task.Status) {
		return nil, errors.NewBadRequestError("cannot cancel task in '" + task.Status + "' state")
	}

	// Create intervention record
	intervention := &model.Intervention{
		TaskID:     taskID,
		OperatorID: operatorID,
		Action:     model.InterventionActionCancel,
		Reason:     reason,
		Status:     model.InterventionStatusPending,
	}

	if err := s.interventionRepo.Create(ctx, intervention); err != nil {
		return nil, err
	}

	// Update task status
	task.Status = model.TaskStatusCancelled
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Update intervention status to applied
	intervention.Status = model.InterventionStatusApplied
	result := model.InterventionResult{
		Success:   true,
		Message:   "Task cancelled successfully",
		Timestamp: time.Now().Unix(),
	}
	resultJSON, _ := json.Marshal(result)
	intervention.Result = datatypes.JSON(resultJSON)

	if err := s.interventionRepo.Update(ctx, intervention); err != nil {
		return nil, err
	}

	return intervention, nil
}

// Inject injects an instruction into a running task.
func (s *interventionService) Inject(ctx context.Context, req *InjectInterventionRequest) (*model.Intervention, error) {
	// Validate IDs
	taskUUID, err := uuid.Parse(req.TaskID)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid task ID format")
	}

	_, err = uuid.Parse(req.OperatorID)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid operator ID format")
	}

	// Validate instruction
	if req.Instruction == "" {
		return nil, errors.NewBadRequestError("instruction is required")
	}

	// Get task
	task, err := s.taskRepo.GetByID(ctx, taskUUID)
	if err != nil {
		return nil, err
	}

	// Validate task state
	if !canInject(task.Status) {
		return nil, errors.NewBadRequestError("cannot inject into task in '" + task.Status + "' state")
	}

	// Create intervention content
	content := model.InterventionContent{
		Instruction: req.Instruction,
		Context:     req.Context,
	}
	contentJSON, _ := json.Marshal(content)

	// Create intervention record
	intervention := &model.Intervention{
		TaskID:     req.TaskID,
		OperatorID: req.OperatorID,
		Action:     model.InterventionActionInject,
		Content:    datatypes.JSON(contentJSON),
		Status:     model.InterventionStatusPending,
	}

	if err := s.interventionRepo.Create(ctx, intervention); err != nil {
		return nil, err
	}

	// Forward instruction to the task executor via the event handler
	if s.eventHandler != nil {
		eventPayload := map[string]interface{}{
			"instruction": req.Instruction,
			"context":     req.Context,
			"task_id":     req.TaskID,
		}
		if err := s.eventHandler.HandleTaskEvent(ctx, req.TaskID, "inject_instruction", eventPayload); err != nil {
			// Update intervention status to failed
			intervention.Status = model.InterventionStatusFailed
			failResult := model.InterventionResult{
				Success:   false,
				Message:   "Failed to inject instruction: " + err.Error(),
				Timestamp: time.Now().Unix(),
			}
			failResultJSON, _ := json.Marshal(failResult)
			intervention.Result = datatypes.JSON(failResultJSON)
			_ = s.interventionRepo.Update(ctx, intervention)
			return nil, err
		}
	}

	// Update intervention status to applied
	intervention.Status = model.InterventionStatusApplied
	result := model.InterventionResult{
		Success:   true,
		Message:   "Instruction injected successfully",
		Timestamp: time.Now().Unix(),
	}
	resultJSON, _ := json.Marshal(result)
	intervention.Result = datatypes.JSON(resultJSON)

	if err := s.interventionRepo.Update(ctx, intervention); err != nil {
		return nil, err
	}

	return intervention, nil
}

// ListInterventions retrieves interventions for a specific task.
func (s *interventionService) ListInterventions(ctx context.Context, taskID string, filter *InterventionFilter) ([]*model.Intervention, int64, error) {
	// Validate task ID
	_, err := uuid.Parse(taskID)
	if err != nil {
		return nil, 0, errors.NewBadRequestError("invalid task ID format")
	}

	// Convert service filter to repository filter
	repoFilter := repository.InterventionFilter{
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		Action:     filter.Action,
		Status:     filter.Status,
		OperatorID: filter.OperatorID,
	}

	// Call repository
	interventions, total, err := s.interventionRepo.ListByTask(ctx, taskID, repoFilter)
	if err != nil {
		return nil, 0, err
	}

	return interventions, total, nil
}

// HandleWrapperEvent handles an event pushed from the wrapper sidecar.
// It delegates to the executor's HandleTaskEvent via the TaskEventHandler interface.
func (s *interventionService) HandleWrapperEvent(ctx context.Context, taskID string, eventType string, payload map[string]interface{}) error {
	// Validate task ID
	_, err := uuid.Parse(taskID)
	if err != nil {
		return errors.NewBadRequestError("invalid task ID format")
	}

	if eventType == "" {
		return errors.NewBadRequestError("event_type is required")
	}

	if s.eventHandler == nil {
		return errors.NewInternalError("event handler not configured")
	}

	return s.eventHandler.HandleTaskEvent(ctx, taskID, eventType, payload)
}
