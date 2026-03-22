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

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// TODO: Add actual API routes
		v1.GET("/hello", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "Hello, Agentic Coding Platform!",
			})
		})
	}

	return r
}
