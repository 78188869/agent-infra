package middleware

import (
	"strings"

	"github.com/example/agent-infra/internal/api/response"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyUserID is the gin context key for authenticated user ID.
	ContextKeyUserID = "user_id"
	// ContextKeyTenantID is the gin context key for authenticated user's tenant ID.
	ContextKeyTenantID = "tenant_id"
	// ContextKeyUserRole is the gin context key for authenticated user's role.
	ContextKeyUserRole = "user_role"
)

// APIKeyAuth creates a middleware that validates API Key authentication.
// It extracts the Bearer token from the Authorization header, validates it
// via APIKeyService, loads the user via UserService, and injects
// user_id/tenant_id/role into the gin context.
func APIKeyAuth(apiKeySvc service.APIKeyService, userSvc service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			response.Unauthorized(c, "missing or invalid authorization header")
			c.Abort()
			return
		}

		apiKey, err := apiKeySvc.Validate(c.Request.Context(), token)
		if err != nil {
			response.Unauthorized(c, "invalid or expired api key")
			c.Abort()
			return
		}

		user, err := userSvc.GetByID(c.Request.Context(), apiKey.UserID)
		if err != nil {
			response.Unauthorized(c, "user not found or disabled")
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, user.ID)
		c.Set(ContextKeyTenantID, user.TenantID)
		c.Set(ContextKeyUserRole, string(user.Role))

		c.Next()
	}
}

func extractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return strings.TrimSpace(parts[1])
}
