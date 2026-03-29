package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// InternalAuth validates shared secret for internal routes.
// Wrapper containers must send the X-Internal-Token header matching
// the configured token value.
func InternalAuth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqToken := c.GetHeader("X-Internal-Token")
		if reqToken == "" || reqToken != token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
