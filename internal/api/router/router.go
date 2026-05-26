package router

import (
	"os"

	"github.com/example/agent-infra/internal/api/handler"
	"github.com/example/agent-infra/internal/api/middleware"
	"github.com/example/agent-infra/internal/monitoring"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// DBChecker defines the interface for database health checking.
type DBChecker interface {
	Ping() error
}

// Setup initializes the gin router with all routes.
func Setup(tenantSvc service.TenantService, templateSvc service.TemplateService, taskSvc service.TaskService, providerSvc service.ProviderService, capabilitySvc service.CapabilityService, monitorSvc service.MonitoringService, hub *monitoring.Hub, interventionSvc service.InterventionService, apiKeySvc service.APIKeyService, userSvc service.UserService, credentialSvc service.CredentialService, db DBChecker) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())

	// Health check endpoints (no auth required)
	r.GET("/health", handler.HealthCheck)
	readyHandler := handler.NewReadyCheckHandler(db)
	r.GET("/ready", readyHandler.ReadyCheck)

	// API v1 routes (all protected with API Key auth)
	v1 := r.Group("/api/v1")
	v1.Use(middleware.APIKeyAuth(apiKeySvc, userSvc))

	// Create handlers that are shared between route groups
	interventionHandler := handler.NewInterventionHandler(interventionSvc)

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

		// Intervention routes (nested under tasks)
		tasks.POST("/:id/pause", interventionHandler.Pause)
		tasks.POST("/:id/resume", interventionHandler.Resume)
		tasks.POST("/:id/cancel", interventionHandler.Cancel)
		tasks.POST("/:id/inject", interventionHandler.Inject)
		tasks.GET("/:id/interventions", interventionHandler.ListInterventions)

		// API Key routes
		apiKeyHandler := handler.NewAPIKeyHandler(apiKeySvc)
		apiKeys := v1.Group("/api-keys")
		{
			apiKeys.POST("", apiKeyHandler.Create)
			apiKeys.GET("", apiKeyHandler.List)
			apiKeys.DELETE("/:id", apiKeyHandler.Revoke)
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
			// Credential routes
			credentialHandler := handler.NewCredentialHandler(credentialSvc)
			credentials := v1.Group("/credentials")
			{
				credentials.POST("", credentialHandler.Store)
				credentials.GET("", credentialHandler.List)
				credentials.DELETE("/:type", credentialHandler.Delete)
			}

		// Capability routes
		capabilityHandler := handler.NewCapabilityHandler(capabilitySvc)
		capabilities := v1.Group("/capabilities")
		{
			capabilities.POST("", capabilityHandler.Create)
			capabilities.GET("", capabilityHandler.List)
			capabilities.GET("/:id", capabilityHandler.GetByID)
			capabilities.PUT("/:id", capabilityHandler.Update)
			capabilities.DELETE("/:id", capabilityHandler.Delete)
			capabilities.POST("/:id/activate", capabilityHandler.Activate)
			capabilities.POST("/:id/deactivate", capabilityHandler.Deactivate)
		}

		// Monitoring routes
		wsHandler := handler.NewWSHandler(hub)
		metricsHandler := handler.NewMetricsHandler(monitorSvc)
		v1.GET("/ws", wsHandler.HandleWebSocket)

		metrics := v1.Group("/metrics")
		{
			metrics.GET("/dashboard", metricsHandler.GetDashboard)
			metrics.GET("/tasks", metricsHandler.GetTaskStats)
			metrics.GET("/resources", metricsHandler.GetResourceUsage)
			metrics.GET("/tenants", metricsHandler.GetTenantStats)
		}

		// Task log routes
		tasks.GET("/:id/logs", metricsHandler.GetTaskLogs)
	}

	// Internal routes for wrapper event push (protected with shared-secret auth)
	internalToken := os.Getenv("INTERNAL_TOKEN")
	internal := r.Group("/internal", middleware.InternalAuth(internalToken))
	{
		internal.POST("/tasks/:id/events", interventionHandler.HandleWrapperEvent)
	}

	return r
}
