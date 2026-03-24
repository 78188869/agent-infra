package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// DBChecker defines the interface for database health checking.
type DBChecker interface {
	Ping() error
}

// RedisChecker defines the interface for Redis health checking.
type RedisChecker interface {
	Ping(ctx interface{}) error
}

// ReadyCheckHandler handles readiness check requests with dependency checks.
type ReadyCheckHandler struct {
	db    DBChecker
	redis *redis.Client
}

// NewReadyCheckHandler creates a new readiness check handler with the given database checker.
// If db is nil, the database check will report "not_configured".
// Deprecated: Use NewReadyCheckHandlerWithRedis for full functionality.
func NewReadyCheckHandler(db DBChecker) *ReadyCheckHandler {
	return &ReadyCheckHandler{
		db: db,
	}
}

// NewReadyCheckHandlerWithRedis creates a new readiness check handler with database and Redis checkers.
func NewReadyCheckHandlerWithRedis(db DBChecker, redisClient *redis.Client) *ReadyCheckHandler {
	return &ReadyCheckHandler{
		db:    db,
		redis: redisClient,
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

	// Check Redis connectivity
	if h.redis == nil {
		checks["redis"] = "not_configured"
	} else {
		if err := h.redis.Ping(c.Request.Context()).Err(); err != nil {
			checks["redis"] = "unhealthy"
			allHealthy = false
		} else {
			checks["redis"] = "healthy"
		}
	}

	status := http.StatusOK
	if !allHealthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"ready":  allHealthy,
		"checks": checks,
	})
}
