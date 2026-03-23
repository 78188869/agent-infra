// Package handler provides HTTP handlers for the API.
package handler

import (
	"github.com/example/agent-infra/internal/api/response"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// TemplateHandler handles HTTP requests for template operations.
type TemplateHandler struct {
	service service.TemplateService
}

// NewTemplateHandler creates a new TemplateHandler instance.
func NewTemplateHandler(svc service.TemplateService) *TemplateHandler {
	return &TemplateHandler{
		service: svc,
	}
}

// Create handles POST /api/v1/templates - Create a new template.
func (h *TemplateHandler) Create(c *gin.Context) {
	var req service.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	template, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, template)
}

// GetByID handles GET /api/v1/templates/:id - Get a single template.
func (h *TemplateHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	template, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, template)
}

// List handles GET /api/v1/templates - List templates with pagination.
func (h *TemplateHandler) List(c *gin.Context) {
	var filter service.TemplateFilter
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

	templates, total, err := h.service.List(c.Request.Context(), &filter)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Paginated(c, templates, total, filter.Page, filter.PageSize)
}

// Update handles PUT /api/v1/templates/:id - Update a template.
func (h *TemplateHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req service.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "template updated successfully"})
}

// Delete handles DELETE /api/v1/templates/:id - Soft delete a template.
func (h *TemplateHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.service.Delete(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "template deleted successfully"})
}
