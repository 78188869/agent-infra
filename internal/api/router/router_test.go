package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// mockTenantService implements service.TenantService for testing
type mockTenantService struct{}

func (m *mockTenantService) Create(ctx context.Context, req *service.CreateTenantRequest) (*model.Tenant, error) {
	return &model.Tenant{}, nil
}

func (m *mockTenantService) GetByID(ctx context.Context, id string) (*model.Tenant, error) {
	return &model.Tenant{}, nil
}

func (m *mockTenantService) List(ctx context.Context, filter *service.TenantFilter) ([]*model.Tenant, int64, error) {
	return []*model.Tenant{}, 0, nil
}

func (m *mockTenantService) Update(ctx context.Context, id string, req *service.UpdateTenantRequest) error {
	return nil
}

func (m *mockTenantService) Delete(ctx context.Context, id string) error {
	return nil
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSetup_Routes(t *testing.T) {
	mockSvc := &mockTenantService{}
	router := Setup(mockSvc)

	tests := []struct {
		name    string
		method  string
		path    string
		status  int
	}{
		{"health check", http.MethodGet, "/health", http.StatusOK},
		{"ready check", http.MethodGet, "/ready", http.StatusOK},
		{"list tenants", http.MethodGet, "/api/v1/tenants", http.StatusOK},
		{"create tenant", http.MethodPost, "/api/v1/tenants", http.StatusBadRequest}, // 400 because no body
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("Route %s %s: expected status %d, got %d", tt.method, tt.path, tt.status, w.Code)
			}
		})
	}
}

func TestSetup_TenantRoutes(t *testing.T) {
	mockSvc := &mockTenantService{}
	router := Setup(mockSvc)

	// Verify all tenant routes are registered
	routes := router.Routes()
	routeMap := make(map[string]bool)
	for _, route := range routes {
		key := route.Method + " " + route.Path
		routeMap[key] = true
	}

	expectedRoutes := []string{
		"POST /api/v1/tenants",
		"GET /api/v1/tenants",
		"GET /api/v1/tenants/:id",
		"PUT /api/v1/tenants/:id",
		"DELETE /api/v1/tenants/:id",
	}

	for _, expected := range expectedRoutes {
		if !routeMap[expected] {
			t.Errorf("Expected route %s not found", expected)
		}
	}
}
