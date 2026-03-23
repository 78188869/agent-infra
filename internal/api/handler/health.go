package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DBChecker defines the interface for database health checking.
type DBChecker interface {
	Ping() error
}

// ReadyCheckHandler handles readiness check requests with dependency checks.
type ReadyCheckHandler struct {
	db DBChecker
}

// NewReadyCheckHandler creates a new readiness check handler with the given database checker.
// If db is nil, the database check will report "not_configured".
func NewReadyCheckHandler(db DBChecker) *ReadyCheckHandler {
	return &ReadyCheckHandler{
		db: db,
	}
}

// HealthCheck returns the service health status.
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "control-plane",
		"version": "0.1.0",
	})
}

// ReadyCheck returns the service readiness status with detailed component checks.
func (h *ReadyCheckHandler) ReadyCheck(c *gin.Context) {
	checks := make(map[string]string)
	allHealthy := true

	// Check database connectivity
	if h.db == nil {
		checks["database"] = "not_configured"
	} else {
		if err := h.db.Ping(); err != nil {
			checks["database"] = "unhealthy"
			allHealthy = false
		} else {
			checks["database"] = "healthy"
		}
	}

	// Redis is not configured in the current implementation
	checks["redis"] = "not_configured"

	status := http.StatusOK
	if !allHealthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"ready":  allHealthy,
		"checks": checks,
	})
}
