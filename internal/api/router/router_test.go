package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/example/agent-infra/internal/monitoring"
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

func (m *mockTaskService) UpdateStatus(ctx context.Context, id string, status string, message string) error {
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

// mockCapabilityService implements service.CapabilityService for testing
type mockCapabilityService struct{}

func (m *mockCapabilityService) Create(ctx context.Context, req *service.CreateCapabilityRequest) (*model.Capability, error) {
	return &model.Capability{}, nil
}

func (m *mockCapabilityService) GetByID(ctx context.Context, id string) (*model.Capability, error) {
	return &model.Capability{}, nil
}

func (m *mockCapabilityService) List(ctx context.Context, filter *service.CapabilityFilter) ([]*model.Capability, int64, error) {
	return []*model.Capability{}, 0, nil
}

func (m *mockCapabilityService) Update(ctx context.Context, id string, req *service.UpdateCapabilityRequest) error {
	return nil
}

func (m *mockCapabilityService) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockCapabilityService) Activate(ctx context.Context, id string) error {
	return nil
}

func (m *mockCapabilityService) Deactivate(ctx context.Context, id string) error {
	return nil
}

// mockInterventionService implements service.InterventionService for testing
type mockInterventionService struct{}

func (m *mockInterventionService) Pause(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error) {
	return &model.Intervention{}, nil
}

func (m *mockInterventionService) Resume(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error) {
	return &model.Intervention{}, nil
}

func (m *mockInterventionService) Cancel(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error) {
	return &model.Intervention{}, nil
}

func (m *mockInterventionService) Inject(ctx context.Context, req *service.InjectInterventionRequest) (*model.Intervention, error) {
	return &model.Intervention{}, nil
}

func (m *mockInterventionService) ListInterventions(ctx context.Context, taskID string, filter *service.InterventionFilter) ([]*model.Intervention, int64, error) {
	return []*model.Intervention{}, 0, nil
}

func (m *mockInterventionService) HandleWrapperEvent(ctx context.Context, taskID string, eventType string, payload map[string]interface{}) error {
	return nil
}

// mockDBChecker implements DBChecker for testing
type mockDBChecker struct{}

func (m *mockDBChecker) Ping() error {
	return nil
}

// mockAPIKeyServiceForRouter implements service.APIKeyService for router tests
type mockAPIKeyServiceForRouter struct{}

func (m *mockAPIKeyServiceForRouter) Create(ctx context.Context, userID string, req *service.CreateAPIKeyRequest) (*model.APIKey, string, error) {
	return nil, "", nil
}

func (m *mockAPIKeyServiceForRouter) GetByID(ctx context.Context, userID string, keyID string) (*model.APIKey, error) {
	return nil, nil
}

func (m *mockAPIKeyServiceForRouter) List(ctx context.Context, userID string, filter *service.APIKeyFilter) ([]*model.APIKey, int64, error) {
	return nil, 0, nil
}

func (m *mockAPIKeyServiceForRouter) Revoke(ctx context.Context, userID string, keyID string) error {
	return nil
}

func (m *mockAPIKeyServiceForRouter) Validate(ctx context.Context, rawKey string) (*model.APIKey, error) {
	return &model.APIKey{ID: "test-key", UserID: "test-user"}, nil
}

// mockUserServiceForRouter implements service.UserService for router tests
type mockUserServiceForRouter struct{}

func (m *mockUserServiceForRouter) GetByID(ctx context.Context, id string) (*model.User, error) {
	return &model.User{
		ID:       "test-user",
		TenantID: "test-tenant",
		Role:     model.UserRoleDeveloper,
		Status:   model.UserStatusActive,
	}, nil
}

	// mockCredentialServiceForRouter implements service.CredentialService for router tests
	type mockCredentialServiceForRouter struct{}

	func (m *mockCredentialServiceForRouter) Store(ctx context.Context, userID string, req *service.StoreCredentialRequest) (*service.CredentialInfo, error) {
		return &service.CredentialInfo{ID: "test-cred", Type: req.Type}, nil
	}

	func (m *mockCredentialServiceForRouter) Get(ctx context.Context, userID, credType string) (string, error) {
		return "", nil
	}

	func (m *mockCredentialServiceForRouter) Delete(ctx context.Context, userID, credType string) error {
		return nil
	}

	func (m *mockCredentialServiceForRouter) List(ctx context.Context, userID string) ([]*service.CredentialInfo, error) {
		return nil, nil
	}

	func (m *mockCredentialServiceForRouter) BuildSandboxEnv(ctx context.Context, userID string) (map[string]string, error) {
		return nil, nil
	}

// mockMonitoringService implements service.MonitoringService for testing
type mockMonitoringService struct{}

func (m *mockMonitoringService) RecordTaskStatusChange(ctx context.Context, taskID, tenantID, oldStatus, newStatus string) error {
	return nil
}

func (m *mockMonitoringService) RecordLogEntry(ctx context.Context, taskID, tenantID string, eventType model.EventType, eventName string, content interface{}) error {
	return nil
}

func (m *mockMonitoringService) RecordTaskProgress(ctx context.Context, taskID, tenantID string, progress int64, tokensUsed int64, elapsedSecs int64) error {
	return nil
}

func (m *mockMonitoringService) BroadcastTaskCompletion(ctx context.Context, taskID, tenantID string) error {
	return nil
}

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter() *gin.Engine {
	return Setup(
		&mockTenantService{},
		&mockTemplateService{},
		&mockTaskService{},
		&mockProviderService{},
		&mockCapabilityService{},
		&mockMonitoringService{},
		monitoring.NewHub(),
		&mockInterventionService{},
		&mockAPIKeyServiceForRouter{},
		&mockUserServiceForRouter{},
			&mockCredentialServiceForRouter{},
		&mockDBChecker{},
	)
}

func TestSetup_Routes(t *testing.T) {
	router := setupTestRouter()

	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"health check", http.MethodGet, "/health", http.StatusOK},
		{"ready check", http.MethodGet, "/ready", http.StatusOK},
		// API v1 routes require auth - without token returns 401
		{"list tenants no auth", http.MethodGet, "/api/v1/tenants", http.StatusUnauthorized},
		{"create tenant no auth", http.MethodPost, "/api/v1/tenants", http.StatusUnauthorized},
		{"list tasks no auth", http.MethodGet, "/api/v1/tasks", http.StatusUnauthorized},
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

func TestSetup_AuthenticatedRoutes(t *testing.T) {
	router := setupTestRouter()

	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"list tenants", http.MethodGet, "/api/v1/tenants", http.StatusOK},
		{"create tenant no body", http.MethodPost, "/api/v1/tenants", http.StatusBadRequest},
		{"list templates", http.MethodGet, "/api/v1/templates", http.StatusOK},
		{"list tasks", http.MethodGet, "/api/v1/tasks", http.StatusOK},
		{"list providers", http.MethodGet, "/api/v1/providers", http.StatusOK},
		{"list capabilities", http.MethodGet, "/api/v1/capabilities", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("Route %s %s: expected status %d, got %d", tt.method, tt.path, tt.status, w.Code)
			}
		})
	}
}

func TestSetup_APIKeyRoutes(t *testing.T) {
	router := setupTestRouter()

	routes := router.Routes()
	routeMap := make(map[string]bool)
	for _, route := range routes {
		key := route.Method + " " + route.Path
		routeMap[key] = true
	}

	expectedRoutes := []string{
		"POST /api/v1/api-keys",
		"GET /api/v1/api-keys",
		"DELETE /api/v1/api-keys/:id",
	}

	for _, expected := range expectedRoutes {
		if !routeMap[expected] {
			t.Errorf("Expected route %s not found", expected)
		}
	}
}

func TestSetup_TenantRoutes(t *testing.T) {
	router := setupTestRouter()

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
	router := setupTestRouter()

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
	router := setupTestRouter()

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

func TestSetup_TaskListWithAuth(t *testing.T) {
	router := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?page=1&page_size=10&status=pending", nil)
	req.Header.Set("Authorization", "Bearer test-token")
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

func TestSetup_CapabilityRoutes(t *testing.T) {
	router := setupTestRouter()

	routes := router.Routes()
	routeMap := make(map[string]bool)
	for _, route := range routes {
		key := route.Method + " " + route.Path
		routeMap[key] = true
	}

	expectedRoutes := []string{
		"POST /api/v1/capabilities",
		"GET /api/v1/capabilities",
		"GET /api/v1/capabilities/:id",
		"PUT /api/v1/capabilities/:id",
		"DELETE /api/v1/capabilities/:id",
		"POST /api/v1/capabilities/:id/activate",
		"POST /api/v1/capabilities/:id/deactivate",
	}

	for _, expected := range expectedRoutes {
		if !routeMap[expected] {
			t.Errorf("Expected route %s not found", expected)
		}
	}
}

func TestSetup_InterventionRoutes(t *testing.T) {
	router := setupTestRouter()

	routes := router.Routes()
	routeMap := make(map[string]bool)
	for _, route := range routes {
		key := route.Method + " " + route.Path
		routeMap[key] = true
	}

	expectedRoutes := []string{
		"POST /api/v1/tasks/:id/pause",
		"POST /api/v1/tasks/:id/resume",
		"POST /api/v1/tasks/:id/cancel",
		"POST /api/v1/tasks/:id/inject",
		"GET /api/v1/tasks/:id/interventions",
	}

	for _, expected := range expectedRoutes {
		if !routeMap[expected] {
			t.Errorf("Expected route %s not found", expected)
		}
	}
}

func TestSetup_InternalRoutes(t *testing.T) {
	router := setupTestRouter()

	routes := router.Routes()
	routeMap := make(map[string]bool)
	for _, route := range routes {
		key := route.Method + " " + route.Path
		routeMap[key] = true
	}

	expectedRoutes := []string{
		"POST /internal/tasks/:id/events",
	}

	for _, expected := range expectedRoutes {
		if !routeMap[expected] {
			t.Errorf("Expected route %s not found", expected)
		}
	}
}

func TestSetup_InternalWrapperEvent(t *testing.T) {
	testToken := "test-internal-token-12345"
	os.Setenv("INTERNAL_TOKEN", testToken)
	defer os.Unsetenv("INTERNAL_TOKEN")

	router := setupTestRouter()

	taskID := "00000000-0000-0000-0000-000000000001"
	body := strings.NewReader(`{"event_type":"heartbeat","payload":{"status":"running","progress":50}}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/tasks/"+taskID+"/events", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", testToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestSetup_InternalWrapperEvent_Unauthorized(t *testing.T) {
	testToken := "secret-token"
	os.Setenv("INTERNAL_TOKEN", testToken)
	defer os.Unsetenv("INTERNAL_TOKEN")

	router := setupTestRouter()

	taskID := "00000000-0000-0000-0000-000000000001"
	body := strings.NewReader(`{"event_type":"heartbeat","payload":{"status":"running","progress":50}}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/tasks/"+taskID+"/events", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d, body: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}
