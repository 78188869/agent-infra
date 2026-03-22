package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a gin middleware for request logging
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		// Simple structured logging
		// TODO: replace with proper structured logger (zap/zerolog)
		logMsg := fmt.Sprintf("[GIN] %s | %s %s | %v | %d\n",
			start.Format(time.RFC3339),
			method,
			path,
			latency,
			status,
		)
		_, _ = gin.DefaultWriter.Write([]byte(logMsg))
	}
}
