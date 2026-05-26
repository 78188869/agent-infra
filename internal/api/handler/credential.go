package handler

import (
	"github.com/example/agent-infra/internal/api/middleware"
	"github.com/example/agent-infra/internal/api/response"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// CredentialHandler handles HTTP requests for credential operations.
type CredentialHandler struct {
	service service.CredentialService
}

// NewCredentialHandler creates a new CredentialHandler instance.
func NewCredentialHandler(svc service.CredentialService) *CredentialHandler {
	return &CredentialHandler{service: svc}
}

// Store handles POST /api/v1/credentials - Store a credential.
func (h *CredentialHandler) Store(c *gin.Context) {
	var req service.StoreCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)

	info, err := h.service.Store(c.Request.Context(), userID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, info)
}

// List handles GET /api/v1/credentials - List credentials for current user.
func (h *CredentialHandler) List(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)

	infos, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, infos)
}

// Delete handles DELETE /api/v1/credentials/:type - Delete a credential.
func (h *CredentialHandler) Delete(c *gin.Context) {
	credType := c.Param("type")
	userID := c.GetString(middleware.ContextKeyUserID)

	if err := h.service.Delete(c.Request.Context(), userID, credType); err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "credential deleted successfully"})
}
