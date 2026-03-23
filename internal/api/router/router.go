package router

import (
	"github.com/example/agent-infra/internal/api/handler"
	"github.com/example/agent-infra/internal/api/middleware"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// Setup initializes the gin router with all routes.
func Setup(tenantSvc service.TenantService) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())

	// Health check endpoints
	r.GET("/health", handler.HealthCheck)
	r.GET("/ready", handler.ReadyCheck)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Tenant routes
		tenantHandler := handler.NewTenantHandler(tenantSvc)
		tenants := v1.Group("/tenants")
		{
			tenants.POST("", tenantHandler.Create)
			tenants.GET("", tenantHandler.List)
			tenants.GET("/:id", tenantHandler.GetByID)
			tenants.PUT("/:id", tenantHandler.Update)
			tenants.DELETE("/:id", tenantHandler.Delete)
		}
	}

	return r
}
