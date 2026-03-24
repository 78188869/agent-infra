// Package handler provides HTTP handlers for the API.
package handler

import (
	"github.com/example/agent-infra/internal/api/response"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// ProviderHandler handles HTTP requests for provider operations.
type ProviderHandler struct {
	service service.ProviderService
}

// NewProviderHandler creates a new ProviderHandler instance.
func NewProviderHandler(svc service.ProviderService) *ProviderHandler {
	return &ProviderHandler{
		service: svc,
	}
}

// Create handles POST /api/v1/providers - Create a new provider.
func (h *ProviderHandler) Create(c *gin.Context) {
	var req service.CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	provider, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, provider)
}

// GetByID handles GET /api/v1/providers/:id - Get a single provider.
func (h *ProviderHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	provider, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, provider)
}

// List handles GET /api/v1/providers - List providers with pagination.
func (h *ProviderHandler) List(c *gin.Context) {
	var filter service.ProviderFilter
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

	providers, total, err := h.service.List(c.Request.Context(), &filter)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Paginated(c, providers, total, filter.Page, filter.PageSize)
}

// Update handles PUT /api/v1/providers/:id - Update a provider.
func (h *ProviderHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req service.UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "provider updated successfully"})
}

// Delete handles DELETE /api/v1/providers/:id - Soft delete a provider.
func (h *ProviderHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.service.Delete(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "provider deleted successfully"})
}

// TestConnection handles POST /api/v1/providers/:id/test - Test provider connection.
func (h *ProviderHandler) TestConnection(c *gin.Context) {
	id := c.Param("id")

	result, err := h.service.TestConnection(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, result)
}

// GetAvailable handles GET /api/v1/providers/available - Get available providers for user.
func (h *ProviderHandler) GetAvailable(c *gin.Context) {
	// Get tenant_id and user_id from context (set by auth middleware)
	tenantID, _ := c.Get("tenant_id")
	userID, _ := c.Get("user_id")

	var tid, uid string
	if tenantID != nil {
		tid = tenantID.(string)
	}
	if userID != nil {
		uid = userID.(string)
	}

	providers, err := h.service.GetAvailableProviders(c.Request.Context(), tid, uid)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, providers)
}

// SetDefault handles PUT /api/v1/providers/:id/set-default - Set default provider for user.
func (h *ProviderHandler) SetDefault(c *gin.Context) {
	id := c.Param("id")

	// Get user_id from context
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	err := h.service.SetDefaultProvider(c.Request.Context(), userID.(string), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "default provider set successfully"})
}
