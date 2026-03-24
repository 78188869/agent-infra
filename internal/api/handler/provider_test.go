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

// mockProviderService implements service.ProviderService for testing
type mockProviderService struct {
	createFunc              func(ctx context.Context, req *service.CreateProviderRequest) (*model.Provider, error)
	getByIDFunc             func(ctx context.Context, id string) (*model.Provider, error)
	listFunc                func(ctx context.Context, filter *service.ProviderFilter) ([]*model.Provider, int64, error)
	updateFunc              func(ctx context.Context, id string, req *service.UpdateProviderRequest) error
	deleteFunc              func(ctx context.Context, id string) error
	testConnectionFunc      func(ctx context.Context, id string) (*service.ConnectionTestResult, error)
	getAvailableProvidersFunc func(ctx context.Context, tenantID, userID string) ([]*model.Provider, error)
	resolveProviderFunc     func(ctx context.Context, specifiedProviderID, tenantID, userID string) (*model.Provider, error)
	setDefaultProviderFunc  func(ctx context.Context, userID, providerID string) error
}

func (m *mockProviderService) Create(ctx context.Context, req *service.CreateProviderRequest) (*model.Provider, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockProviderService) GetByID(ctx context.Context, id string) (*model.Provider, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockProviderService) List(ctx context.Context, filter *service.ProviderFilter) ([]*model.Provider, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockProviderService) Update(ctx context.Context, id string, req *service.UpdateProviderRequest) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, req)
	}
	return nil
}

func (m *mockProviderService) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockProviderService) TestConnection(ctx context.Context, id string) (*service.ConnectionTestResult, error) {
	if m.testConnectionFunc != nil {
		return m.testConnectionFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockProviderService) GetAvailableProviders(ctx context.Context, tenantID, userID string) ([]*model.Provider, error) {
	if m.getAvailableProvidersFunc != nil {
		return m.getAvailableProvidersFunc(ctx, tenantID, userID)
	}
	return nil, nil
}

func (m *mockProviderService) ResolveProvider(ctx context.Context, specifiedProviderID, tenantID, userID string) (*model.Provider, error) {
	if m.resolveProviderFunc != nil {
		return m.resolveProviderFunc(ctx, specifiedProviderID, tenantID, userID)
	}
	return nil, nil
}

func (m *mockProviderService) SetDefaultProvider(ctx context.Context, userID, providerID string) error {
	if m.setDefaultProviderFunc != nil {
		return m.setDefaultProviderFunc(ctx, userID, providerID)
	}
	return nil
}

func TestProviderHandler_Create(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*mockProviderService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful create system provider",
			requestBody: map[string]interface{}{
				"name":         "OpenAI Provider",
				"type":         "openai_compatible",
				"scope":        "system",
				"api_endpoint": "https://api.openai.com",
				"runtime_type": "cli",
			},
			mockSetup: func(m *mockProviderService) {
				m.createFunc = func(ctx context.Context, req *service.CreateProviderRequest) (*model.Provider, error) {
					return &model.Provider{
						ID:           existingID.String(),
						Name:         req.Name,
						Type:         req.Type,
						Scope:        req.Scope,
						APIEndpoint:  req.APIEndpoint,
						RuntimeType:  req.RuntimeType,
						Status:       model.ProviderStatusActive,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["name"] != "OpenAI Provider" {
					t.Errorf("Expected name 'OpenAI Provider', got %v", data["name"])
				}
			},
		},
		{
			name: "missing required field name",
			requestBody: map[string]interface{}{
				"type":  "openai_compatible",
				"scope": "system",
			},
			mockSetup:      func(m *mockProviderService) {},
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
				"name":  "Test Provider",
				"type":  "openai_compatible",
				"scope": "tenant",
				// Missing tenant_id for tenant scope
			},
			mockSetup: func(m *mockProviderService) {
				m.createFunc = func(ctx context.Context, req *service.CreateProviderRequest) (*model.Provider, error) {
					return nil, errors.NewBadRequestError("tenant_id is required for tenant scope")
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
				"name":  "Test Provider",
				"type":  "openai_compatible",
				"scope": "system",
			},
			mockSetup: func(m *mockProviderService) {
				m.createFunc = func(ctx context.Context, req *service.CreateProviderRequest) (*model.Provider, error) {
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
			mockSvc := &mockProviderService{}
			tt.mockSetup(mockSvc)

			handler := NewProviderHandler(mockSvc)
			router := setupTestRouter()
			router.POST("/api/v1/providers", handler.Create)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/providers", bytes.NewReader(body))
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

func TestProviderHandler_GetByID(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockProviderService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful get",
			id:   existingID.String(),
			mockSetup: func(m *mockProviderService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Provider, error) {
					return &model.Provider{
						ID:     existingID.String(),
						Name:   "Test Provider",
						Type:   model.ProviderTypeOpenAICompat,
						Scope:  model.ProviderScopeSystem,
						Status: model.ProviderStatusActive,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["name"] != "Test Provider" {
					t.Errorf("Expected name 'Test Provider', got %v", data["name"])
				}
			},
		},
		{
			name: "invalid id format",
			id:   "invalid-uuid",
			mockSetup: func(m *mockProviderService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Provider, error) {
					return nil, errors.NewBadRequestError("invalid provider ID format")
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
			mockSetup: func(m *mockProviderService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Provider, error) {
					return nil, errors.NewNotFoundError("provider not found")
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
			mockSvc := &mockProviderService{}
			tt.mockSetup(mockSvc)

			handler := NewProviderHandler(mockSvc)
			router := setupTestRouter()
			router.GET("/api/v1/providers/:id", handler.GetByID)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+tt.id, nil)
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

func TestProviderHandler_List(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()

	tests := []struct {
		name           string
		queryParams    string
		mockSetup      func(*mockProviderService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:        "successful list with defaults",
			queryParams: "",
			mockSetup: func(m *mockProviderService) {
				m.listFunc = func(ctx context.Context, filter *service.ProviderFilter) ([]*model.Provider, int64, error) {
					return []*model.Provider{
						{ID: id1.String(), Name: "Provider 1", Status: model.ProviderStatusActive},
						{ID: id2.String(), Name: "Provider 2", Status: model.ProviderStatusActive},
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
			mockSetup: func(m *mockProviderService) {
				m.listFunc = func(ctx context.Context, filter *service.ProviderFilter) ([]*model.Provider, int64, error) {
					if filter.Page != 2 {
						t.Errorf("Expected page 2, got %d", filter.Page)
					}
					if filter.PageSize != 5 {
						t.Errorf("Expected page_size 5, got %d", filter.PageSize)
					}
					return []*model.Provider{}, 0, nil
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
			name:        "list with scope filter",
			queryParams: "?scope=system",
			mockSetup: func(m *mockProviderService) {
				m.listFunc = func(ctx context.Context, filter *service.ProviderFilter) ([]*model.Provider, int64, error) {
					if filter.Scope != "system" {
						t.Errorf("Expected scope 'system', got '%s'", filter.Scope)
					}
					return []*model.Provider{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty list",
			queryParams: "",
			mockSetup: func(m *mockProviderService) {
				m.listFunc = func(ctx context.Context, filter *service.ProviderFilter) ([]*model.Provider, int64, error) {
					return []*model.Provider{}, 0, nil
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
			mockSetup: func(m *mockProviderService) {
				m.listFunc = func(ctx context.Context, filter *service.ProviderFilter) ([]*model.Provider, int64, error) {
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
			mockSvc := &mockProviderService{}
			tt.mockSetup(mockSvc)

			handler := NewProviderHandler(mockSvc)
			router := setupTestRouter()
			router.GET("/api/v1/providers", handler.List)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/providers"+tt.queryParams, nil)
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

func TestProviderHandler_Update(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		requestBody    interface{}
		mockSetup      func(*mockProviderService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful update name",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"name": "Updated Name",
			},
			mockSetup: func(m *mockProviderService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateProviderRequest) error {
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
			name: "successful update status",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"status": "inactive",
			},
			mockSetup: func(m *mockProviderService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateProviderRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty update",
			id:          existingID.String(),
			requestBody: map[string]interface{}{},
			mockSetup: func(m *mockProviderService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateProviderRequest) error {
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
			mockSetup: func(m *mockProviderService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateProviderRequest) error {
					return errors.NewBadRequestError("invalid provider ID format")
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
			mockSetup: func(m *mockProviderService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateProviderRequest) error {
					return errors.NewNotFoundError("provider not found")
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
				"name": "",
			},
			mockSetup: func(m *mockProviderService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateProviderRequest) error {
					return errors.NewBadRequestError("provider name cannot be empty")
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
			mockSvc := &mockProviderService{}
			tt.mockSetup(mockSvc)

			handler := NewProviderHandler(mockSvc)
			router := setupTestRouter()
			router.PUT("/api/v1/providers/:id", handler.Update)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/providers/"+tt.id, bytes.NewReader(body))
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

func TestProviderHandler_Delete(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockProviderService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful delete",
			id:   existingID.String(),
			mockSetup: func(m *mockProviderService) {
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
			mockSetup: func(m *mockProviderService) {
				m.deleteFunc = func(ctx context.Context, id string) error {
					return errors.NewBadRequestError("invalid provider ID format")
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
			mockSetup: func(m *mockProviderService) {
				m.deleteFunc = func(ctx context.Context, id string) error {
					return errors.NewNotFoundError("provider not found")
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
			mockSetup: func(m *mockProviderService) {
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
			mockSvc := &mockProviderService{}
			tt.mockSetup(mockSvc)

			handler := NewProviderHandler(mockSvc)
			router := setupTestRouter()
			router.DELETE("/api/v1/providers/:id", handler.Delete)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/providers/"+tt.id, nil)
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

func TestProviderHandler_TestConnection(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockProviderService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful connection test",
			id:   existingID.String(),
			mockSetup: func(m *mockProviderService) {
				m.testConnectionFunc = func(ctx context.Context, id string) (*service.ConnectionTestResult, error) {
					return &service.ConnectionTestResult{
						Success:      true,
						Message:      "Connection successful",
						ResponseTime: 150,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["success"].(bool) != true {
					t.Errorf("Expected success true, got %v", data["success"])
				}
			},
		},
		{
			name: "connection test failed",
			id:   existingID.String(),
			mockSetup: func(m *mockProviderService) {
				m.testConnectionFunc = func(ctx context.Context, id string) (*service.ConnectionTestResult, error) {
					return &service.ConnectionTestResult{
						Success:      false,
						Message:      "Connection refused",
						ResponseTime: 0,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].(map[string]interface{})
				if data["success"].(bool) != false {
					t.Errorf("Expected success false, got %v", data["success"])
				}
			},
		},
		{
			name: "provider not found",
			id:   uuid.New().String(),
			mockSetup: func(m *mockProviderService) {
				m.testConnectionFunc = func(ctx context.Context, id string) (*service.ConnectionTestResult, error) {
					return nil, errors.NewNotFoundError("provider not found")
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
			mockSvc := &mockProviderService{}
			tt.mockSetup(mockSvc)

			handler := NewProviderHandler(mockSvc)
			router := setupTestRouter()
			router.POST("/api/v1/providers/:id/test", handler.TestConnection)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/providers/"+tt.id+"/test", nil)
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

func TestProviderHandler_GetAvailable(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()
	tenantID := uuid.New().String()
	userID := uuid.New().String()

	tests := []struct {
		name           string
		mockSetup      func(*mockProviderService)
		setupContext   func(c *gin.Context)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful get available with auth context",
			mockSetup: func(m *mockProviderService) {
				m.getAvailableProvidersFunc = func(ctx context.Context, tid, uid string) ([]*model.Provider, error) {
					if tid != tenantID {
						t.Errorf("Expected tenantID %s, got %s", tenantID, tid)
					}
					if uid != userID {
						t.Errorf("Expected userID %s, got %s", userID, uid)
					}
					return []*model.Provider{
						{ID: id1.String(), Name: "System Provider", Scope: model.ProviderScopeSystem, Status: model.ProviderStatusActive},
						{ID: id2.String(), Name: "User Provider", Scope: model.ProviderScopeUser, Status: model.ProviderStatusActive},
					}, nil
				}
			},
			setupContext: func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
				c.Set("user_id", userID)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].([]interface{})
				if len(data) != 2 {
					t.Errorf("Expected 2 providers, got %d", len(data))
				}
			},
		},
		{
			name: "successful get available without auth context",
			mockSetup: func(m *mockProviderService) {
				m.getAvailableProvidersFunc = func(ctx context.Context, tid, uid string) ([]*model.Provider, error) {
					if tid != "" {
						t.Errorf("Expected empty tenantID, got %s", tid)
					}
					if uid != "" {
						t.Errorf("Expected empty userID, got %s", uid)
					}
					return []*model.Provider{
						{ID: id1.String(), Name: "System Provider", Scope: model.ProviderScopeSystem, Status: model.ProviderStatusActive},
					}, nil
				}
			},
			setupContext: func(c *gin.Context) {
				// No auth context set
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].([]interface{})
				if len(data) != 1 {
					t.Errorf("Expected 1 provider, got %d", len(data))
				}
			},
		},
		{
			name: "internal error",
			mockSetup: func(m *mockProviderService) {
				m.getAvailableProvidersFunc = func(ctx context.Context, tid, uid string) ([]*model.Provider, error) {
					return nil, errors.NewInternalError("database error")
				}
			},
			setupContext: func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
				c.Set("user_id", userID)
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
			mockSvc := &mockProviderService{}
			tt.mockSetup(mockSvc)

			handler := NewProviderHandler(mockSvc)
			router := setupTestRouter()
			router.GET("/api/v1/providers/available", func(c *gin.Context) {
				tt.setupContext(c)
				handler.GetAvailable(c)
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/available", nil)
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

func TestProviderHandler_SetDefault(t *testing.T) {
	existingID := uuid.New()
	userID := uuid.New().String()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockProviderService)
		setupContext   func(c *gin.Context)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful set default",
			id:   existingID.String(),
			mockSetup: func(m *mockProviderService) {
				m.setDefaultProviderFunc = func(ctx context.Context, uid, providerID string) error {
					if uid != userID {
						t.Errorf("Expected userID %s, got %s", userID, uid)
					}
					if providerID != existingID.String() {
						t.Errorf("Expected providerID %s, got %s", existingID.String(), providerID)
					}
					return nil
				}
			},
			setupContext: func(c *gin.Context) {
				c.Set("user_id", userID)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
			},
		},
		{
			name: "unauthorized - no user context",
			id:   existingID.String(),
			mockSetup: func(m *mockProviderService) {
				// Should not be called
			},
			setupContext: func(c *gin.Context) {
				// No user_id set
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 401 {
					t.Errorf("Expected code 401, got %v", body["code"])
				}
			},
		},
		{
			name: "provider not found",
			id:   uuid.New().String(),
			mockSetup: func(m *mockProviderService) {
				m.setDefaultProviderFunc = func(ctx context.Context, uid, providerID string) error {
					return errors.NewNotFoundError("provider not found")
				}
			},
			setupContext: func(c *gin.Context) {
				c.Set("user_id", userID)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 404 {
					t.Errorf("Expected code 404, got %v", body["code"])
				}
			},
		},
		{
			name: "invalid provider id",
			id:   "invalid-uuid",
			mockSetup: func(m *mockProviderService) {
				m.setDefaultProviderFunc = func(ctx context.Context, uid, providerID string) error {
					return errors.NewBadRequestError("invalid provider ID format")
				}
			},
			setupContext: func(c *gin.Context) {
				c.Set("user_id", userID)
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
			mockSvc := &mockProviderService{}
			tt.mockSetup(mockSvc)

			handler := NewProviderHandler(mockSvc)
			router := setupTestRouter()
			router.PUT("/api/v1/providers/:id/set-default", func(c *gin.Context) {
				tt.setupContext(c)
				handler.SetDefault(c)
			})

			req := httptest.NewRequest(http.MethodPut, "/api/v1/providers/"+tt.id+"/set-default", nil)
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
