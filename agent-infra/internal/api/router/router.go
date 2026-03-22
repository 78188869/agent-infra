package router

import (
	"github.com/example/agent-infra/internal/api/handler"
	"github.com/example/agent-infra/internal/api/middleware"
	"github.com/gin-gonic/gin"
)

// Setup initializes the gin router with all routes
func Setup() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())

	// Health check endpoints
	r.GET("/health", handler.HealthCheck)
	r.GET("/ready", handler.ReadyCheck)

	return r
}
