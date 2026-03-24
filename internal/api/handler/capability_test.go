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

// mockCapabilityService implements service.CapabilityService for testing
type mockCapabilityService struct {
	createFunc     func(ctx context.Context, req *service.CreateCapabilityRequest) (*model.Capability, error)
	getByIDFunc    func(ctx context.Context, id string) (*model.Capability, error)
	listFunc       func(ctx context.Context, filter *service.CapabilityFilter) ([]*model.Capability, int64, error)
	updateFunc     func(ctx context.Context, id string, req *service.UpdateCapabilityRequest) error
	deleteFunc     func(ctx context.Context, id string) error
	activateFunc   func(ctx context.Context, id string) error
	deactivateFunc func(ctx context.Context, id string) error
}

func (m *mockCapabilityService) Create(ctx context.Context, req *service.CreateCapabilityRequest) (*model.Capability, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockCapabilityService) GetByID(ctx context.Context, id string) (*model.Capability, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockCapabilityService) List(ctx context.Context, filter *service.CapabilityFilter) ([]*model.Capability, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockCapabilityService) Update(ctx context.Context, id string, req *service.UpdateCapabilityRequest) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, req)
	}
	return nil
}

func (m *mockCapabilityService) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockCapabilityService) Activate(ctx context.Context, id string) error {
	if m.activateFunc != nil {
		return m.activateFunc(ctx, id)
	}
	return nil
}

func (m *mockCapabilityService) Deactivate(ctx context.Context, id string) error {
	if m.deactivateFunc != nil {
		return m.deactivateFunc(ctx, id)
	}
	return nil
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCapabilityHandler_Create(t *testing.T) {
	existingID := uuid.New()
	tenantID := uuid.New()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*mockCapabilityService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful create",
			requestBody: map[string]interface{}{
				"type":             "tool",
				"name":             "Test Capability",
				"permission_level": "public",
				"tenant_id":        tenantID.String(),
			},
			mockSetup: func(m *mockCapabilityService) {
				m.createFunc = func(ctx context.Context, req *service.CreateCapabilityRequest) (*model.Capability, error) {
					return &model.Capability{
						ID:             existingID.String(),
						Type:           req.Type,
						Name:           req.Name,
						Description:    req.Description,
						Version:        req.Version,
						TenantID:       &req.TenantID,
						PermissionLevel: req.PermissionLevel,
						Config:         req.Config,
						Schema:         req.Schema,
						Status:         model.CapabilityStatusActive,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["name"] != "Test Capability" {
					t.Errorf("Expected name 'Test Capability', got %v", data["name"])
				}
			},
		},
		{
			name: "missing type",
			requestBody: map[string]interface{}{
				"name":             "Test Capability",
				"permission_level": "public",
			},
			mockSetup:      func(m *mockCapabilityService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 400 {
					t.Errorf("Expected code 400, got %v", body["code"])
				}
			},
		},
		{
			name: "missing name",
			requestBody: map[string]interface{}{
				"type":             "tool",
				"permission_level": "public",
			},
			mockSetup:      func(m *mockCapabilityService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 400 {
					t.Errorf("Expected code 400, got %v", body["code"])
				}
			},
		},
		{
			name: "missing permission_level",
			requestBody: map[string]interface{}{
				"type": "tool",
				"name": "Test Capability",
			},
			mockSetup:      func(m *mockCapabilityService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 400 {
					t.Errorf("Expected code 400, got %v", body["code"])
				}
			},
		},
		{
			name: "invalid type",
			requestBody: map[string]interface{}{
				"type":             "invalid-type",
				"name":             "Test Capability",
				"permission_level": "public",
			},
			mockSetup: func(m *mockCapabilityService) {
				m.createFunc = func(ctx context.Context, req *service.CreateCapabilityRequest) (*model.Capability, error) {
					return nil, errors.NewBadRequestError("invalid type, must be one of: tool, skill, agent_runtime")
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
			name: "service validation error",
			requestBody: map[string]interface{}{
				"type":             "tool",
				"name":             "",
				"permission_level": "public",
			},
			mockSetup: func(m *mockCapabilityService) {
				m.createFunc = func(ctx context.Context, req *service.CreateCapabilityRequest) (*model.Capability, error) {
					return nil, errors.NewBadRequestError("capability name is required")
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
				"type":             "tool",
				"name":             "Test Capability",
				"permission_level": "public",
			},
			mockSetup: func(m *mockCapabilityService) {
				m.createFunc = func(ctx context.Context, req *service.CreateCapabilityRequest) (*model.Capability, error) {
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
			mockSvc := &mockCapabilityService{}
			tt.mockSetup(mockSvc)

			handler := NewCapabilityHandler(mockSvc)
			router := gin.New()
			router.POST("/api/v1/capabilities", handler.Create)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/capabilities", bytes.NewReader(body))
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

func TestCapabilityHandler_GetByID(t *testing.T) {
	existingID := uuid.New()
	tenantID := uuid.New()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockCapabilityService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful get",
			id:   existingID.String(),
			mockSetup: func(m *mockCapabilityService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Capability, error) {
					return &model.Capability{
						ID:             existingID.String(),
						TenantID:       &[]string{tenantID.String()}[0],
						Type:           model.CapabilityTypeTool,
						Name:           "Test Capability",
						Description:    "A test capability",
						Version:        "1.0.0",
						PermissionLevel: model.PermissionLevelPublic,
						Status:         model.CapabilityStatusActive,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["name"] != "Test Capability" {
					t.Errorf("Expected name 'Test Capability', got %v", data["name"])
				}
			},
		},
		{
			name: "invalid id format",
			id:   "invalid-uuid",
			mockSetup: func(m *mockCapabilityService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Capability, error) {
					return nil, errors.NewBadRequestError("invalid capability ID format")
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
			mockSetup: func(m *mockCapabilityService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Capability, error) {
					return nil, errors.NewNotFoundError("capability not found")
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
			mockSetup: func(m *mockCapabilityService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Capability, error) {
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
			mockSvc := &mockCapabilityService{}
			tt.mockSetup(mockSvc)

			handler := NewCapabilityHandler(mockSvc)
			router := gin.New()
			router.GET("/api/v1/capabilities/:id", handler.GetByID)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/capabilities/"+tt.id, nil)
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

func TestCapabilityHandler_List(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()
	tenantID := uuid.New()

	tests := []struct {
		name           string
		queryParams    string
		mockSetup      func(*mockCapabilityService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:        "successful list with defaults",
			queryParams: "",
			mockSetup: func(m *mockCapabilityService) {
				m.listFunc = func(ctx context.Context, filter *service.CapabilityFilter) ([]*model.Capability, int64, error) {
					return []*model.Capability{
						{ID: id1.String(), TenantID: &[]string{tenantID.String()}[0], Type: model.CapabilityTypeTool, Name: "Capability 1", Status: model.CapabilityStatusActive},
						{ID: id2.String(), TenantID: &[]string{tenantID.String()}[0], Type: model.CapabilityTypeSkill, Name: "Capability 2", Status: model.CapabilityStatusActive},
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
			mockSetup: func(m *mockCapabilityService) {
				m.listFunc = func(ctx context.Context, filter *service.CapabilityFilter) ([]*model.Capability, int64, error) {
					if filter.Page != 2 {
						t.Errorf("Expected page 2, got %d", filter.Page)
					}
					if filter.PageSize != 5 {
						t.Errorf("Expected page_size 5, got %d", filter.PageSize)
					}
					return []*model.Capability{}, 0, nil
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
			name:        "list with tenant_id filter",
			queryParams: "?tenant_id=" + tenantID.String(),
			mockSetup: func(m *mockCapabilityService) {
				m.listFunc = func(ctx context.Context, filter *service.CapabilityFilter) ([]*model.Capability, int64, error) {
					if filter.TenantID != tenantID.String() {
						t.Errorf("Expected tenant_id '%s', got '%s'", tenantID.String(), filter.TenantID)
					}
					return []*model.Capability{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "list with type filter",
			queryParams: "?type=tool",
			mockSetup: func(m *mockCapabilityService) {
				m.listFunc = func(ctx context.Context, filter *service.CapabilityFilter) ([]*model.Capability, int64, error) {
					if filter.Type != "tool" {
						t.Errorf("Expected type 'tool', got '%s'", filter.Type)
					}
					return []*model.Capability{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "list with status filter",
			queryParams: "?status=active",
			mockSetup: func(m *mockCapabilityService) {
				m.listFunc = func(ctx context.Context, filter *service.CapabilityFilter) ([]*model.Capability, int64, error) {
					if filter.Status != "active" {
						t.Errorf("Expected status 'active', got '%s'", filter.Status)
					}
					return []*model.Capability{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "list with search filter",
			queryParams: "?search=test",
			mockSetup: func(m *mockCapabilityService) {
				m.listFunc = func(ctx context.Context, filter *service.CapabilityFilter) ([]*model.Capability, int64, error) {
					if filter.Search != "test" {
						t.Errorf("Expected search 'test', got '%s'", filter.Search)
					}
					return []*model.Capability{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty list",
			queryParams: "",
			mockSetup: func(m *mockCapabilityService) {
				m.listFunc = func(ctx context.Context, filter *service.CapabilityFilter) ([]*model.Capability, int64, error) {
					return []*model.Capability{}, 0, nil
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
			mockSetup: func(m *mockCapabilityService) {
				m.listFunc = func(ctx context.Context, filter *service.CapabilityFilter) ([]*model.Capability, int64, error) {
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
		{
			name:        "invalid query params",
			queryParams: "?page=invalid",
			mockSetup:   func(m *mockCapabilityService) {},
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
			mockSvc := &mockCapabilityService{}
			tt.mockSetup(mockSvc)

			handler := NewCapabilityHandler(mockSvc)
			router := gin.New()
			router.GET("/api/v1/capabilities", handler.List)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/capabilities"+tt.queryParams, nil)
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

func TestCapabilityHandler_Update(t *testing.T) {
	existingID := uuid.New()
	name := "Updated Name"
	description := "Updated Description"

	tests := []struct {
		name           string
		id             string
		requestBody    interface{}
		mockSetup      func(*mockCapabilityService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful update name",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"name": name,
			},
			mockSetup: func(m *mockCapabilityService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateCapabilityRequest) error {
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
			name: "successful update description",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"description": description,
			},
			mockSetup: func(m *mockCapabilityService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateCapabilityRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful update version",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"version": "2.0.0",
			},
			mockSetup: func(m *mockCapabilityService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateCapabilityRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful update permission_level",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"permission_level": "admin_only",
			},
			mockSetup: func(m *mockCapabilityService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateCapabilityRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty update",
			id:          existingID.String(),
			requestBody: map[string]interface{}{},
			mockSetup: func(m *mockCapabilityService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateCapabilityRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid id format",
			id:   "invalid-uuid",
			requestBody: map[string]interface{}{
				"name": name,
			},
			mockSetup: func(m *mockCapabilityService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateCapabilityRequest) error {
					return errors.NewBadRequestError("invalid capability ID format")
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
				"name": name,
			},
			mockSetup: func(m *mockCapabilityService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateCapabilityRequest) error {
					return errors.NewNotFoundError("capability not found")
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
			requestBody: map[string]interface{}{
				"name": name,
			},
			mockSetup: func(m *mockCapabilityService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateCapabilityRequest) error {
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
			mockSvc := &mockCapabilityService{}
			tt.mockSetup(mockSvc)

			handler := NewCapabilityHandler(mockSvc)
			router := gin.New()
			router.PUT("/api/v1/capabilities/:id", handler.Update)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/capabilities/"+tt.id, bytes.NewReader(body))
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

func TestCapabilityHandler_Delete(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockCapabilityService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful delete",
			id:   existingID.String(),
			mockSetup: func(m *mockCapabilityService) {
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
			mockSetup: func(m *mockCapabilityService) {
				m.deleteFunc = func(ctx context.Context, id string) error {
					return errors.NewBadRequestError("invalid capability ID format")
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
			mockSetup: func(m *mockCapabilityService) {
				m.deleteFunc = func(ctx context.Context, id string) error {
					return errors.NewNotFoundError("capability not found")
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
			mockSetup: func(m *mockCapabilityService) {
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
			mockSvc := &mockCapabilityService{}
			tt.mockSetup(mockSvc)

			handler := NewCapabilityHandler(mockSvc)
			router := gin.New()
			router.DELETE("/api/v1/capabilities/:id", handler.Delete)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/capabilities/"+tt.id, nil)
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

func TestCapabilityHandler_Activate(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockCapabilityService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful activate",
			id:   existingID.String(),
			mockSetup: func(m *mockCapabilityService) {
				m.activateFunc = func(ctx context.Context, id string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["message"] != "capability activated successfully" {
					t.Errorf("Expected message 'capability activated successfully', got %v", data["message"])
				}
			},
		},
		{
			name: "invalid id format",
			id:   "invalid-uuid",
			mockSetup: func(m *mockCapabilityService) {
				m.activateFunc = func(ctx context.Context, id string) error {
					return errors.NewBadRequestError("invalid capability ID format")
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
			mockSetup: func(m *mockCapabilityService) {
				m.activateFunc = func(ctx context.Context, id string) error {
					return errors.NewNotFoundError("capability not found")
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
			name: "already active",
			id:   existingID.String(),
			mockSetup: func(m *mockCapabilityService) {
				m.activateFunc = func(ctx context.Context, id string) error {
					return errors.NewBadRequestError("capability is already active")
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
			id:   existingID.String(),
			mockSetup: func(m *mockCapabilityService) {
				m.activateFunc = func(ctx context.Context, id string) error {
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
			mockSvc := &mockCapabilityService{}
			tt.mockSetup(mockSvc)

			handler := NewCapabilityHandler(mockSvc)
			router := gin.New()
			router.POST("/api/v1/capabilities/:id/activate", handler.Activate)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/capabilities/"+tt.id+"/activate", nil)
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

func TestCapabilityHandler_Deactivate(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockCapabilityService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful deactivate",
			id:   existingID.String(),
			mockSetup: func(m *mockCapabilityService) {
				m.deactivateFunc = func(ctx context.Context, id string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["message"] != "capability deactivated successfully" {
					t.Errorf("Expected message 'capability deactivated successfully', got %v", data["message"])
				}
			},
		},
		{
			name: "invalid id format",
			id:   "invalid-uuid",
			mockSetup: func(m *mockCapabilityService) {
				m.deactivateFunc = func(ctx context.Context, id string) error {
					return errors.NewBadRequestError("invalid capability ID format")
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
			mockSetup: func(m *mockCapabilityService) {
				m.deactivateFunc = func(ctx context.Context, id string) error {
					return errors.NewNotFoundError("capability not found")
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
			name: "already inactive",
			id:   existingID.String(),
			mockSetup: func(m *mockCapabilityService) {
				m.deactivateFunc = func(ctx context.Context, id string) error {
					return errors.NewBadRequestError("capability is already inactive")
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
			id:   existingID.String(),
			mockSetup: func(m *mockCapabilityService) {
				m.deactivateFunc = func(ctx context.Context, id string) error {
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
			mockSvc := &mockCapabilityService{}
			tt.mockSetup(mockSvc)

			handler := NewCapabilityHandler(mockSvc)
			router := gin.New()
			router.POST("/api/v1/capabilities/:id/deactivate", handler.Deactivate)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/capabilities/"+tt.id+"/deactivate", nil)
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
