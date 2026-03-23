package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/service"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// mockTenantService implements service.TenantService for testing
type mockTenantService struct {
	createFunc  func(ctx context.Context, req *service.CreateTenantRequest) (*model.Tenant, error)
	getByIDFunc func(ctx context.Context, id string) (*model.Tenant, error)
	listFunc    func(ctx context.Context, filter *service.TenantFilter) ([]*model.Tenant, int64, error)
	updateFunc  func(ctx context.Context, id string, req *service.UpdateTenantRequest) error
	deleteFunc  func(ctx context.Context, id string) error
}

func (m *mockTenantService) Create(ctx context.Context, req *service.CreateTenantRequest) (*model.Tenant, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockTenantService) GetByID(ctx context.Context, id string) (*model.Tenant, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockTenantService) List(ctx context.Context, filter *service.TenantFilter) ([]*model.Tenant, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockTenantService) Update(ctx context.Context, id string, req *service.UpdateTenantRequest) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, req)
	}
	return nil
}

func (m *mockTenantService) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter() *gin.Engine {
	return gin.New()
}

func TestTenantHandler_Create(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*mockTenantService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful create",
			requestBody: map[string]interface{}{
				"name":              "Test Tenant",
				"quota_cpu":         4,
				"quota_memory":      16,
				"quota_concurrency": 10,
				"quota_daily_tasks": 100,
			},
			mockSetup: func(m *mockTenantService) {
				m.createFunc = func(ctx context.Context, req *service.CreateTenantRequest) (*model.Tenant, error) {
					return &model.Tenant{
						BaseModel:        model.BaseModel{ID: existingID},
						Name:             req.Name,
						QuotaCPU:         req.QuotaCPU,
						QuotaMemory:      req.QuotaMemory,
						QuotaConcurrency: req.QuotaConcurrency,
						QuotaDailyTasks:  req.QuotaDailyTasks,
						Status:           model.TenantStatusActive,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["name"] != "Test Tenant" {
					t.Errorf("Expected name 'Test Tenant', got %v", data["name"])
				}
			},
		},
		{
			name: "missing name",
			requestBody: map[string]interface{}{
				"quota_cpu": 4,
			},
			mockSetup:      func(m *mockTenantService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 400 {
					t.Errorf("Expected code 400, got %v", body["code"])
				}
			},
		},
		{
			name: "service validation error",
			requestBody: map[string]interface{}{
				"name":      "Test Tenant",
				"quota_cpu": -1,
			},
			mockSetup: func(m *mockTenantService) {
				m.createFunc = func(ctx context.Context, req *service.CreateTenantRequest) (*model.Tenant, error) {
					return nil, errors.NewBadRequestError("quota_cpu must be a positive value")
				}
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 400 {
					t.Errorf("Expected code 400, got %v", body["code"])
				}
			},
		},
		{
			name: "internal error",
			requestBody: map[string]interface{}{
				"name": "Test Tenant",
			},
			mockSetup: func(m *mockTenantService) {
				m.createFunc = func(ctx context.Context, req *service.CreateTenantRequest) (*model.Tenant, error) {
					return nil, errors.NewInternalError("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 500 {
					t.Errorf("Expected code 500, got %v", body["code"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockTenantService{}
			tt.mockSetup(mockSvc)

			handler := NewTenantHandler(mockSvc)
			router := setupTestRouter()
			router.POST("/api/v1/tenants", handler.Create)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
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

func TestTenantHandler_GetByID(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockTenantService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful get",
			id:   existingID.String(),
			mockSetup: func(m *mockTenantService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Tenant, error) {
					return &model.Tenant{
						BaseModel: model.BaseModel{ID: existingID},
						Name:      "Test Tenant",
						Status:    model.TenantStatusActive,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["name"] != "Test Tenant" {
					t.Errorf("Expected name 'Test Tenant', got %v", data["name"])
				}
			},
		},
		{
			name: "invalid id format",
			id:   "invalid-uuid",
			mockSetup: func(m *mockTenantService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Tenant, error) {
					return nil, errors.NewBadRequestError("invalid tenant ID format")
				}
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 400 {
					t.Errorf("Expected code 400, got %v", body["code"])
				}
			},
		},
		{
			name: "not found",
			id:   uuid.New().String(),
			mockSetup: func(m *mockTenantService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Tenant, error) {
					return nil, errors.NewNotFoundError("tenant not found")
				}
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 404 {
					t.Errorf("Expected code 404, got %v", body["code"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockTenantService{}
			tt.mockSetup(mockSvc)

			handler := NewTenantHandler(mockSvc)
			router := setupTestRouter()
			router.GET("/api/v1/tenants/:id", handler.GetByID)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/"+tt.id, nil)
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

func TestTenantHandler_List(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()

	tests := []struct {
		name           string
		queryParams    string
		mockSetup      func(*mockTenantService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:        "successful list with defaults",
			queryParams: "",
			mockSetup: func(m *mockTenantService) {
				m.listFunc = func(ctx context.Context, filter *service.TenantFilter) ([]*model.Tenant, int64, error) {
					return []*model.Tenant{
						{BaseModel: model.BaseModel{ID: id1}, Name: "Tenant 1", Status: model.TenantStatusActive},
						{BaseModel: model.BaseModel{ID: id2}, Name: "Tenant 2", Status: model.TenantStatusActive},
					}, 2, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				items := data["items"].([]interface{})
				if len(items) != 2 {
					t.Errorf("Expected 2 items, got %d", len(items))
				}
				if data["total"].(float64) != 2 {
					t.Errorf("Expected total 2, got %v", data["total"])
				}
			},
		},
		{
			name:        "list with pagination params",
			queryParams: "?page=2&page_size=5",
			mockSetup: func(m *mockTenantService) {
				m.listFunc = func(ctx context.Context, filter *service.TenantFilter) ([]*model.Tenant, int64, error) {
					if filter.Page != 2 {
						t.Errorf("Expected page 2, got %d", filter.Page)
					}
					if filter.PageSize != 5 {
						t.Errorf("Expected page_size 5, got %d", filter.PageSize)
					}
					return []*model.Tenant{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].(map[string]interface{})
				if data["page"].(float64) != 2 {
					t.Errorf("Expected page 2, got %v", data["page"])
				}
				if data["page_size"].(float64) != 5 {
					t.Errorf("Expected page_size 5, got %v", data["page_size"])
				}
			},
		},
		{
			name:        "list with status filter",
			queryParams: "?status=active",
			mockSetup: func(m *mockTenantService) {
				m.listFunc = func(ctx context.Context, filter *service.TenantFilter) ([]*model.Tenant, int64, error) {
					if filter.Status != "active" {
						t.Errorf("Expected status 'active', got '%s'", filter.Status)
					}
					return []*model.Tenant{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty list",
			queryParams: "",
			mockSetup: func(m *mockTenantService) {
				m.listFunc = func(ctx context.Context, filter *service.TenantFilter) ([]*model.Tenant, int64, error) {
					return []*model.Tenant{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].(map[string]interface{})
				items := data["items"].([]interface{})
				if len(items) != 0 {
					t.Errorf("Expected 0 items, got %d", len(items))
				}
				if data["total"].(float64) != 0 {
					t.Errorf("Expected total 0, got %v", data["total"])
				}
			},
		},
		{
			name:        "internal error",
			queryParams: "",
			mockSetup: func(m *mockTenantService) {
				m.listFunc = func(ctx context.Context, filter *service.TenantFilter) ([]*model.Tenant, int64, error) {
					return nil, 0, errors.NewInternalError("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 500 {
					t.Errorf("Expected code 500, got %v", body["code"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockTenantService{}
			tt.mockSetup(mockSvc)

			handler := NewTenantHandler(mockSvc)
			router := setupTestRouter()
			router.GET("/api/v1/tenants", handler.List)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants"+tt.queryParams, nil)
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

func TestTenantHandler_Update(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		requestBody    interface{}
		mockSetup      func(*mockTenantService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful update name",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"name": "Updated Name",
			},
			mockSetup: func(m *mockTenantService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTenantRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
			},
		},
		{
			name: "successful update quotas",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"quota_cpu":         8,
				"quota_memory":      32,
				"quota_concurrency": 20,
				"quota_daily_tasks": 200,
			},
			mockSetup: func(m *mockTenantService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTenantRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful update status",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"status": "suspended",
			},
			mockSetup: func(m *mockTenantService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTenantRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty update",
			id:          existingID.String(),
			requestBody: map[string]interface{}{},
			mockSetup: func(m *mockTenantService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTenantRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid id format",
			id:   "invalid-uuid",
			requestBody: map[string]interface{}{
				"name": "Updated Name",
			},
			mockSetup: func(m *mockTenantService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTenantRequest) error {
					return errors.NewBadRequestError("invalid tenant ID format")
				}
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 400 {
					t.Errorf("Expected code 400, got %v", body["code"])
				}
			},
		},
		{
			name: "not found",
			id:   uuid.New().String(),
			requestBody: map[string]interface{}{
				"name": "Updated Name",
			},
			mockSetup: func(m *mockTenantService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTenantRequest) error {
					return errors.NewNotFoundError("tenant not found")
				}
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 404 {
					t.Errorf("Expected code 404, got %v", body["code"])
				}
			},
		},
		{
			name: "validation error",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"quota_cpu": -1,
			},
			mockSetup: func(m *mockTenantService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTenantRequest) error {
					return errors.NewBadRequestError("quota_cpu must be a positive value")
				}
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 400 {
					t.Errorf("Expected code 400, got %v", body["code"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockTenantService{}
			tt.mockSetup(mockSvc)

			handler := NewTenantHandler(mockSvc)
			router := setupTestRouter()
			router.PUT("/api/v1/tenants/:id", handler.Update)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/tenants/"+tt.id, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
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

func TestTenantHandler_Delete(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockTenantService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful delete",
			id:   existingID.String(),
			mockSetup: func(m *mockTenantService) {
				m.deleteFunc = func(ctx context.Context, id string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
			},
		},
		{
			name: "invalid id format",
			id:   "invalid-uuid",
			mockSetup: func(m *mockTenantService) {
				m.deleteFunc = func(ctx context.Context, id string) error {
					return errors.NewBadRequestError("invalid tenant ID format")
				}
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 400 {
					t.Errorf("Expected code 400, got %v", body["code"])
				}
			},
		},
		{
			name: "not found",
			id:   uuid.New().String(),
			mockSetup: func(m *mockTenantService) {
				m.deleteFunc = func(ctx context.Context, id string) error {
					return errors.NewNotFoundError("tenant not found")
				}
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 404 {
					t.Errorf("Expected code 404, got %v", body["code"])
				}
			},
		},
		{
			name: "internal error",
			id:   existingID.String(),
			mockSetup: func(m *mockTenantService) {
				m.deleteFunc = func(ctx context.Context, id string) error {
					return errors.NewInternalError("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 500 {
					t.Errorf("Expected code 500, got %v", body["code"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockTenantService{}
			tt.mockSetup(mockSvc)

			handler := NewTenantHandler(mockSvc)
			router := setupTestRouter()
			router.DELETE("/api/v1/tenants/:id", handler.Delete)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/tenants/"+tt.id, nil)
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
