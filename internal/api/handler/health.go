package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck returns the service health status
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "control-plane",
		"version": "0.1.0",
	})
}

// ReadyCheck returns the service readiness status
func ReadyCheck(c *gin.Context) {
	// TODO: Add actual readiness checks (DB, Redis, etc.)
	c.JSON(http.StatusOK, gin.H{
		"ready": true,
	})
}
