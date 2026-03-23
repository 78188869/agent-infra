// Package handler provides HTTP handlers for the API.
package handler

import (
	"github.com/example/agent-infra/internal/api/response"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// TaskHandler handles HTTP requests for task operations.
type TaskHandler struct {
	service service.TaskService
}

// NewTaskHandler creates a new TaskHandler instance.
func NewTaskHandler(svc service.TaskService) *TaskHandler {
	return &TaskHandler{
		service: svc,
	}
}

// Create handles POST /api/v1/tasks - Create a new task.
func (h *TaskHandler) Create(c *gin.Context) {
	var req service.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	task, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, task)
}

// GetByID handles GET /api/v1/tasks/:id - Get a single task.
func (h *TaskHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	task, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, task)
}

// List handles GET /api/v1/tasks - List tasks with pagination.
func (h *TaskHandler) List(c *gin.Context) {
	var filter service.TaskFilter
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

	tasks, total, err := h.service.List(c.Request.Context(), &filter)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Paginated(c, tasks, total, filter.Page, filter.PageSize)
}

// Update handles PUT /api/v1/tasks/:id - Update a task.
func (h *TaskHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req service.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "task updated successfully"})
}

// Delete handles DELETE /api/v1/tasks/:id - Soft delete a task.
func (h *TaskHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.service.Delete(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "task deleted successfully"})
}
