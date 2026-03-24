package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
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

// mockTemplateService implements service.TemplateService for testing
type mockTemplateService struct{}

func (m *mockTemplateService) Create(ctx context.Context, req *service.CreateTemplateRequest) (*model.Template, error) {
	return &model.Template{}, nil
}

func (m *mockTemplateService) GetByID(ctx context.Context, id string) (*model.Template, error) {
	return &model.Template{}, nil
}

func (m *mockTemplateService) List(ctx context.Context, filter *service.TemplateFilter) ([]*model.Template, int64, error) {
	return []*model.Template{}, 0, nil
}

func (m *mockTemplateService) Update(ctx context.Context, id string, req *service.UpdateTemplateRequest) error {
	return nil
}

func (m *mockTemplateService) Delete(ctx context.Context, id string) error {
	return nil
}

// mockTaskService implements service.TaskService for testing
type mockTaskService struct{}

func (m *mockTaskService) Create(ctx context.Context, req *service.CreateTaskRequest) (*model.Task, error) {
	return &model.Task{}, nil
}

func (m *mockTaskService) GetByID(ctx context.Context, id string) (*model.Task, error) {
	return &model.Task{}, nil
}

func (m *mockTaskService) List(ctx context.Context, filter *service.TaskFilter) ([]*model.Task, int64, error) {
	return []*model.Task{}, 0, nil
}

func (m *mockTaskService) Update(ctx context.Context, id string, req *service.UpdateTaskRequest) error {
	return nil
}

func (m *mockTaskService) Delete(ctx context.Context, id string) error {
	return nil
}

// mockProviderService implements service.ProviderService for testing
type mockProviderService struct{}

func (m *mockProviderService) Create(ctx context.Context, req *service.CreateProviderRequest) (*model.Provider, error) {
	return &model.Provider{}, nil
}

func (m *mockProviderService) GetByID(ctx context.Context, id string) (*model.Provider, error) {
	return &model.Provider{}, nil
}

func (m *mockProviderService) List(ctx context.Context, filter *repository.ProviderFilter) ([]*model.Provider, int64, error) {
	return []*model.Provider{}, 0, nil
}

func (m *mockProviderService) Update(ctx context.Context, id string, req *service.UpdateProviderRequest) error {
	return nil
}

func (m *mockProviderService) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockProviderService) TestConnection(ctx context.Context, id string) (*service.ConnectionTestResult, error) {
	return &service.ConnectionTestResult{}, nil
}

func (m *mockProviderService) GetAvailableProviders(ctx context.Context, tenantID, userID string) ([]*model.Provider, error) {
	return []*model.Provider{}, nil
}

func (m *mockProviderService) ResolveProvider(ctx context.Context, specifiedProviderID, tenantID, userID string) (*model.Provider, error) {
	return &model.Provider{}, nil
}

func (m *mockProviderService) SetDefaultProvider(ctx context.Context, userID, providerID string) error {
	return nil
}

// mockDBChecker implements DBChecker for testing
type mockDBChecker struct{}

func (m *mockDBChecker) Ping() error {
	return nil
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSetup_Routes(t *testing.T) {
	mockTenantSvc := &mockTenantService{}
	mockTemplateSvc := &mockTemplateService{}
	mockTaskSvc := &mockTaskService{}
	mockProviderSvc := &mockProviderService{}
	mockDB := &mockDBChecker{}
	router := Setup(mockTenantSvc, mockTemplateSvc, mockTaskSvc, mockProviderSvc, mockDB)

	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"health check", http.MethodGet, "/health", http.StatusOK},
		{"ready check", http.MethodGet, "/ready", http.StatusOK},
		{"list tenants", http.MethodGet, "/api/v1/tenants", http.StatusOK},
		{"create tenant", http.MethodPost, "/api/v1/tenants", http.StatusBadRequest}, // 400 because no body
		{"list templates", http.MethodGet, "/api/v1/templates", http.StatusOK},
		{"create template", http.MethodPost, "/api/v1/templates", http.StatusBadRequest}, // 400 because no body
		{"list tasks", http.MethodGet, "/api/v1/tasks", http.StatusOK},
		{"create task", http.MethodPost, "/api/v1/tasks", http.StatusBadRequest}, // 400 because no body
		{"list providers", http.MethodGet, "/api/v1/providers", http.StatusOK},
		{"create provider", http.MethodPost, "/api/v1/providers", http.StatusBadRequest}, // 400 because no body
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
	mockTenantSvc := &mockTenantService{}
	mockTemplateSvc := &mockTemplateService{}
	mockTaskSvc := &mockTaskService{}
	mockProviderSvc := &mockProviderService{}
	mockDB := &mockDBChecker{}
	router := Setup(mockTenantSvc, mockTemplateSvc, mockTaskSvc, mockProviderSvc, mockDB)

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

func TestSetup_TaskRoutes(t *testing.T) {
	mockTenantSvc := &mockTenantService{}
	mockTemplateSvc := &mockTemplateService{}
	mockTaskSvc := &mockTaskService{}
	mockProviderSvc := &mockProviderService{}
	mockDB := &mockDBChecker{}
	router := Setup(mockTenantSvc, mockTemplateSvc, mockTaskSvc, mockProviderSvc, mockDB)

	// Verify all task routes are registered
	routes := router.Routes()
	routeMap := make(map[string]bool)
	for _, route := range routes {
		key := route.Method + " " + route.Path
		routeMap[key] = true
	}

	expectedRoutes := []string{
		"POST /api/v1/tasks",
		"GET /api/v1/tasks",
		"GET /api/v1/tasks/:id",
		"PUT /api/v1/tasks/:id",
		"DELETE /api/v1/tasks/:id",
	}

	for _, expected := range expectedRoutes {
		if !routeMap[expected] {
			t.Errorf("Expected route %s not found", expected)
		}
	}
}

func TestSetup_ProviderRoutes(t *testing.T) {
	mockTenantSvc := &mockTenantService{}
	mockTemplateSvc := &mockTemplateService{}
	mockTaskSvc := &mockTaskService{}
	mockProviderSvc := &mockProviderService{}
	mockDB := &mockDBChecker{}
	router := Setup(mockTenantSvc, mockTemplateSvc, mockTaskSvc, mockProviderSvc, mockDB)

	// Verify all provider routes are registered
	routes := router.Routes()
	routeMap := make(map[string]bool)
	for _, route := range routes {
		key := route.Method + " " + route.Path
		routeMap[key] = true
	}

	expectedRoutes := []string{
		"POST /api/v1/providers",
		"GET /api/v1/providers",
		"GET /api/v1/providers/available",
		"GET /api/v1/providers/:id",
		"PUT /api/v1/providers/:id",
		"DELETE /api/v1/providers/:id",
		"POST /api/v1/providers/:id/test",
		"PUT /api/v1/providers/:id/set-default",
	}

	for _, expected := range expectedRoutes {
		if !routeMap[expected] {
			t.Errorf("Expected route %s not found", expected)
		}
	}
}

func TestSetup_TaskListWithParams(t *testing.T) {
	mockTenantSvc := &mockTenantService{}
	mockTemplateSvc := &mockTemplateService{}
	mockTaskSvc := &mockTaskService{}
	mockProviderSvc := &mockProviderService{}
	mockDB := &mockDBChecker{}
	router := Setup(mockTenantSvc, mockTemplateSvc, mockTaskSvc, mockProviderSvc, mockDB)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?page=1&page_size=10&status=pending", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["code"].(float64) != 0 {
		t.Errorf("Expected code 0, got %v", response["code"])
	}
}
