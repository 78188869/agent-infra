// Package handler provides HTTP handlers for the API.
package handler

import (
	"github.com/example/agent-infra/internal/api/response"
	"github.com/example/agent-infra/internal/service"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/gin-gonic/gin"
)

// TenantHandler handles HTTP requests for tenant operations.
type TenantHandler struct {
	service service.TenantService
}

// NewTenantHandler creates a new TenantHandler instance.
func NewTenantHandler(svc service.TenantService) *TenantHandler {
	return &TenantHandler{
		service: svc,
	}
}

// Create handles POST /api/v1/tenants - Create a new tenant.
func (h *TenantHandler) Create(c *gin.Context) {
	var req service.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	tenant, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, tenant)
}

// GetByID handles GET /api/v1/tenants/:id - Get a single tenant.
func (h *TenantHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	tenant, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, tenant)
}

// List handles GET /api/v1/tenants - List tenants with pagination.
func (h *TenantHandler) List(c *gin.Context) {
	var filter service.TenantFilter
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

	tenants, total, err := h.service.List(c.Request.Context(), &filter)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Paginated(c, tenants, total, filter.Page, filter.PageSize)
}

// Update handles PUT /api/v1/tenants/:id - Update a tenant.
func (h *TenantHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req service.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "tenant updated successfully"})
}

// Delete handles DELETE /api/v1/tenants/:id - Soft delete a tenant.
func (h *TenantHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.service.Delete(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "tenant deleted successfully"})
}

// handleError maps service errors to appropriate HTTP responses.
func handleError(c *gin.Context, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		response.InternalError(c, "internal server error")
		return
	}

	switch appErr.HTTPStatus {
	case 400:
		response.BadRequest(c, appErr.Message)
	case 404:
		response.NotFound(c, appErr.Message)
	default:
		response.InternalError(c, appErr.Message)
	}
}
