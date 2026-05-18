package handler

import (
	"github.com/example/agent-infra/internal/api/middleware"
	"github.com/example/agent-infra/internal/api/response"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// APIKeyHandler handles HTTP requests for API key operations.
type APIKeyHandler struct {
	service service.APIKeyService
}

// NewAPIKeyHandler creates a new APIKeyHandler instance.
func NewAPIKeyHandler(svc service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{service: svc}
}

// Create handles POST /api/v1/api-keys - Create a new API key.
func (h *APIKeyHandler) Create(c *gin.Context) {
	var req service.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)

	apiKey, rawKey, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, gin.H{
		"id":         apiKey.ID,
		"name":       apiKey.Name,
		"key_prefix": apiKey.KeyPrefix,
		"secret":     rawKey,
		"expires_at": apiKey.ExpiresAt,
		"created_at": apiKey.CreatedAt,
	})
}

// List handles GET /api/v1/api-keys - List API keys for current user.
func (h *APIKeyHandler) List(c *gin.Context) {
	var filter service.APIKeyFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.BadRequest(c, "invalid query parameters: "+err.Error())
		return
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 10
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	userID := c.GetString(middleware.ContextKeyUserID)

	keys, total, err := h.service.List(c.Request.Context(), userID, &filter)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Paginated(c, keys, total, filter.Page, filter.PageSize)
}

// Revoke handles DELETE /api/v1/api-keys/:id - Revoke an API key.
func (h *APIKeyHandler) Revoke(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString(middleware.ContextKeyUserID)

	err := h.service.Revoke(c.Request.Context(), userID, id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "api key revoked successfully"})
}
