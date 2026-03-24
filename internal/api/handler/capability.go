// Package handler provides HTTP handlers for the API.
package handler

import (
	"github.com/example/agent-infra/internal/api/response"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// CapabilityHandler handles HTTP requests for capability operations.
type CapabilityHandler struct {
	service service.CapabilityService
}

// NewCapabilityHandler creates a new CapabilityHandler instance.
func NewCapabilityHandler(svc service.CapabilityService) *CapabilityHandler {
	return &CapabilityHandler{
		service: svc,
	}
}

// Create handles POST /api/v1/capabilities - Create a new capability.
func (h *CapabilityHandler) Create(c *gin.Context) {
	var req service.CreateCapabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	capability, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, capability)
}

// GetByID handles GET /api/v1/capabilities/:id - Get a single capability.
func (h *CapabilityHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	capability, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, capability)
}

// List handles GET /api/v1/capabilities - List capabilities with pagination.
func (h *CapabilityHandler) List(c *gin.Context) {
	var filter service.CapabilityFilter
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

	capabilities, total, err := h.service.List(c.Request.Context(), &filter)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Paginated(c, capabilities, total, filter.Page, filter.PageSize)
}

// Update handles PUT /api/v1/capabilities/:id - Update a capability.
func (h *CapabilityHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req service.UpdateCapabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "capability updated successfully"})
}

// Delete handles DELETE /api/v1/capabilities/:id - Soft delete a capability.
func (h *CapabilityHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.service.Delete(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "capability deleted successfully"})
}

// Activate handles POST /api/v1/capabilities/:id/activate - Activate a capability.
func (h *CapabilityHandler) Activate(c *gin.Context) {
	id := c.Param("id")

	err := h.service.Activate(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "capability activated successfully"})
}

// Deactivate handles POST /api/v1/capabilities/:id/deactivate - Deactivate a capability.
func (h *CapabilityHandler) Deactivate(c *gin.Context) {
	id := c.Param("id")

	err := h.service.Deactivate(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "capability deactivated successfully"})
}
