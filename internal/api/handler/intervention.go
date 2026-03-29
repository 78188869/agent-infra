// Package handler provides HTTP handlers for the API.
package handler

import (
	"github.com/example/agent-infra/internal/api/response"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// Context key for user_id stored in context by auth middleware.
const UserIDKey = "user_id"

// InterventionHandler handles HTTP requests for intervention operations.
type InterventionHandler struct {
	service service.InterventionService
}

// NewInterventionHandler creates a new InterventionHandler instance.
func NewInterventionHandler(svc service.InterventionService) *InterventionHandler {
	return &InterventionHandler{
		service: svc,
	}
}

// PauseRequest represents the request body for pausing a task.
type PauseRequest struct {
	Reason string `json:"reason"`
}

// ResumeRequest represents the request body for resuming a task.
type ResumeRequest struct {
	Reason string `json:"reason"`
}

// CancelRequest represents the request body for canceling a task.
type CancelRequest struct {
	Reason string `json:"reason"`
}

// InjectRequest represents the request body for injecting a command to a task.
type InjectRequest struct {
	Content string `json:"content" binding:"required"`
	Reason  string `json:"reason"`
}

// Pause handles POST /api/v1/tasks/:id/pause - Pause a running task.
// TODO: Extract user_id from context for operator_id after auth middleware is implemented.
func (h *InterventionHandler) Pause(c *gin.Context) {
	taskID := c.Param("id")

	var req PauseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// PauseRequest has no required fields, so binding should always succeed
		// This check is kept for consistency with other handlers
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// TODO: Extract user_id from context after auth middleware is implemented
	// For now, use a placeholder or extract from context if available
	operatorID, exists := c.Get(UserIDKey)
	if !exists {
		// TODO: Replace with proper auth middleware check
		// For now, return unauthorized if no user_id in context
		response.Unauthorized(c, "user not authenticated")
		return
	}

	intervention, err := h.service.Pause(c.Request.Context(), taskID, operatorID.(string), req.Reason)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, intervention)
}

// Resume handles POST /api/v1/tasks/:id/resume - Resume a paused task.
// TODO: Extract user_id from context for operator_id after auth middleware is implemented.
func (h *InterventionHandler) Resume(c *gin.Context) {
	taskID := c.Param("id")

	var req ResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// TODO: Extract user_id from context after auth middleware is implemented
	operatorID, exists := c.Get(UserIDKey)
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	intervention, err := h.service.Resume(c.Request.Context(), taskID, operatorID.(string), req.Reason)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, intervention)
}

// Cancel handles POST /api/v1/tasks/:id/cancel - Cancel a task.
// TODO: Extract user_id from context for operator_id after auth middleware is implemented.
func (h *InterventionHandler) Cancel(c *gin.Context) {
	taskID := c.Param("id")

	var req CancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// TODO: Extract user_id from context after auth middleware is implemented
	operatorID, exists := c.Get(UserIDKey)
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	intervention, err := h.service.Cancel(c.Request.Context(), taskID, operatorID.(string), req.Reason)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, intervention)
}

// Inject handles POST /api/v1/tasks/:id/inject - Inject command to running task.
// TODO: Extract user_id from context for operator_id after auth middleware is implemented.
func (h *InterventionHandler) Inject(c *gin.Context) {
	taskID := c.Param("id")

	var req InjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// TODO: Extract user_id from context after auth middleware is implemented
	operatorID, exists := c.Get(UserIDKey)
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	// Convert handler request to service request
	serviceReq := &service.InjectInterventionRequest{
		TaskID:      taskID,
		OperatorID:  operatorID.(string),
		Instruction: req.Content,
		Context:     req.Reason, // Using reason as context for inject
	}

	intervention, err := h.service.Inject(c.Request.Context(), serviceReq)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, intervention)
}

// ListInterventions handles GET /api/v1/tasks/:id/interventions - Get intervention history.
func (h *InterventionHandler) ListInterventions(c *gin.Context) {
	taskID := c.Param("id")

	var filter service.InterventionFilter
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

	interventions, total, err := h.service.ListInterventions(c.Request.Context(), taskID, &filter)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Paginated(c, interventions, total, filter.Page, filter.PageSize)
}

// WrapperEventRequest represents the request body for wrapper event push.
type WrapperEventRequest struct {
	EventType string                 `json:"event_type" binding:"required"`
	Payload   map[string]interface{} `json:"payload"`
}

// HandleWrapperEvent handles POST /internal/tasks/:id/events - Receive events from wrapper sidecar.
func (h *InterventionHandler) HandleWrapperEvent(c *gin.Context) {
	taskID := c.Param("id")

	var req WrapperEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.HandleWrapperEvent(c.Request.Context(), taskID, req.EventType, req.Payload); err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, nil)
}
