package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestLogger(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		path         string
		routePath    string
		expectedCode int
	}{
		{
			name:         "logs GET request",
			method:       http.MethodGet,
			path:         "/test",
			routePath:    "/test",
			expectedCode: http.StatusOK,
		},
		{
			name:         "logs POST request",
			method:       http.MethodPost,
			path:         "/api/v1/tenants",
			routePath:    "/api/v1/tenants",
			expectedCode: http.StatusCreated,
		},
		{
			name:         "logs request with query params",
			method:       http.MethodGet,
			path:         "/api/v1/tenants?page=1&limit=10",
			routePath:    "/api/v1/tenants",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			w := httptest.NewRecorder()
			r := gin.New()
			r.Use(Logger())
			r.Handle(tt.method, tt.routePath, func(c *gin.Context) {
				c.Status(tt.expectedCode)
			})

			// Execute
			req := httptest.NewRequest(tt.method, tt.path, nil)
			r.ServeHTTP(w, req)

			// Verify
			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}
