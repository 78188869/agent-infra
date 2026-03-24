package router

import (
	"github.com/example/agent-infra/internal/api/handler"
	"github.com/example/agent-infra/internal/api/middleware"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// DBChecker defines the interface for database health checking.
type DBChecker interface {
	Ping() error
}

// Setup initializes the gin router with all routes.
func Setup(tenantSvc service.TenantService, templateSvc service.TemplateService, taskSvc service.TaskService, providerSvc service.ProviderService, db DBChecker) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())

	// Health check endpoints
	r.GET("/health", handler.HealthCheck)
	readyHandler := handler.NewReadyCheckHandler(db)
	r.GET("/ready", readyHandler.ReadyCheck)

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

		// Template routes
		templateHandler := handler.NewTemplateHandler(templateSvc)
		templates := v1.Group("/templates")
		{
			templates.POST("", templateHandler.Create)
			templates.GET("", templateHandler.List)
			templates.GET("/:id", templateHandler.GetByID)
			templates.PUT("/:id", templateHandler.Update)
			templates.DELETE("/:id", templateHandler.Delete)
		}

		// Task routes
		taskHandler := handler.NewTaskHandler(taskSvc)
		tasks := v1.Group("/tasks")
		{
			tasks.POST("", taskHandler.Create)
			tasks.GET("", taskHandler.List)
			tasks.GET("/:id", taskHandler.GetByID)
			tasks.PUT("/:id", taskHandler.Update)
			tasks.DELETE("/:id", taskHandler.Delete)
		}

		// Provider routes
		providerHandler := handler.NewProviderHandler(providerSvc)
		providers := v1.Group("/providers")
		{
			providers.POST("", providerHandler.Create)
			providers.GET("", providerHandler.List)
			providers.GET("/available", providerHandler.GetAvailable)
			providers.GET("/:id", providerHandler.GetByID)
			providers.PUT("/:id", providerHandler.Update)
			providers.DELETE("/:id", providerHandler.Delete)
			providers.POST("/:id/test", providerHandler.TestConnection)
			providers.PUT("/:id/set-default", providerHandler.SetDefault)
		}
	}

	return r
}
