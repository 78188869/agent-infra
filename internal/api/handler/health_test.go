package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// mockDBChecker implements DBChecker interface for testing
type mockDBChecker struct {
	pingFunc func() error
}

func (m *mockDBChecker) Ping() error {
	if m.pingFunc != nil {
		return m.pingFunc()
	}
	return nil
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "returns healthy status",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["status"] != "healthy" {
					t.Errorf("Expected status 'healthy', got %v", body["status"])
				}
				if body["service"] != "control-plane" {
					t.Errorf("Expected service 'control-plane', got %v", body["service"])
				}
				if body["version"] != "0.1.0" {
					t.Errorf("Expected version '0.1.0', got %v", body["version"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/health", HealthCheck)

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to parse response: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestReadyCheck_WithDatabase(t *testing.T) {
	tests := []struct {
		name           string
		mockSetup      func(*mockDBChecker)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "all checks healthy",
			mockSetup: func(m *mockDBChecker) {
				m.pingFunc = func() error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["ready"] != true {
					t.Errorf("Expected ready true, got %v", body["ready"])
				}
				checks, ok := body["checks"].(map[string]interface{})
				if !ok {
					t.Errorf("Expected checks to be a map, got %v", body["checks"])
					return
				}
				if checks["database"] != "healthy" {
					t.Errorf("Expected database 'healthy', got %v", checks["database"])
				}
				if checks["redis"] != "not_configured" {
					t.Errorf("Expected redis 'not_configured', got %v", checks["redis"])
				}
			},
		},
		{
			name: "database unhealthy",
			mockSetup: func(m *mockDBChecker) {
				m.pingFunc = func() error {
					return errors.New("connection refused")
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["ready"] != false {
					t.Errorf("Expected ready false, got %v", body["ready"])
				}
				checks, ok := body["checks"].(map[string]interface{})
				if !ok {
					t.Errorf("Expected checks to be a map, got %v", body["checks"])
					return
				}
				if checks["database"] != "unhealthy" {
					t.Errorf("Expected database 'unhealthy', got %v", checks["database"])
				}
				if checks["redis"] != "not_configured" {
					t.Errorf("Expected redis 'not_configured', got %v", checks["redis"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDBChecker{}
			tt.mockSetup(mockDB)

			handler := NewReadyCheckHandler(mockDB)
			router := gin.New()
			router.GET("/ready", handler.ReadyCheck)

			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to parse response: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestReadyCheck_NoDatabase(t *testing.T) {
	// Test the case when no database is configured
	handler := NewReadyCheckHandler(nil)
	router := gin.New()
	router.GET("/ready", handler.ReadyCheck)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["ready"] != true {
		t.Errorf("Expected ready true when no DB, got %v", response["ready"])
	}

	checks, ok := response["checks"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected checks to be a map, got %v", response["checks"])
		return
	}

	if checks["database"] != "not_configured" {
		t.Errorf("Expected database 'not_configured', got %v", checks["database"])
	}
}
