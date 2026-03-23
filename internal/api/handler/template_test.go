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
	"github.com/google/uuid"
)

// mockTemplateService implements service.TemplateService for testing
type mockTemplateService struct {
	createFunc  func(ctx context.Context, req *service.CreateTemplateRequest) (*model.Template, error)
	getByIDFunc func(ctx context.Context, id string) (*model.Template, error)
	listFunc    func(ctx context.Context, filter *service.TemplateFilter) ([]*model.Template, int64, error)
	updateFunc  func(ctx context.Context, id string, req *service.UpdateTemplateRequest) error
	deleteFunc  func(ctx context.Context, id string) error
}

func (m *mockTemplateService) Create(ctx context.Context, req *service.CreateTemplateRequest) (*model.Template, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockTemplateService) GetByID(ctx context.Context, id string) (*model.Template, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockTemplateService) List(ctx context.Context, filter *service.TemplateFilter) ([]*model.Template, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockTemplateService) Update(ctx context.Context, id string, req *service.UpdateTemplateRequest) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, req)
	}
	return nil
}

func (m *mockTemplateService) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestTemplateHandler_Create(t *testing.T) {
	existingID := uuid.New()
	tenantID := uuid.New()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*mockTemplateService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful create",
			requestBody: map[string]interface{}{
				"tenant_id":  tenantID.String(),
				"name":       "Test Template",
				"version":    "1.0.0",
				"spec":       "name: test\nversion: '1.0'",
				"scene_type": "coding",
			},
			mockSetup: func(m *mockTemplateService) {
				m.createFunc = func(ctx context.Context, req *service.CreateTemplateRequest) (*model.Template, error) {
					return &model.Template{
						BaseModel: model.BaseModel{ID: existingID},
						TenantID:  req.TenantID,
						Name:      req.Name,
						Version:   req.Version,
						Spec:      req.Spec,
						SceneType: req.SceneType,
						Status:    model.TemplateStatusDraft,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["name"] != "Test Template" {
					t.Errorf("Expected name 'Test Template', got %v", data["name"])
				}
			},
		},
		{
			name: "missing name",
			requestBody: map[string]interface{}{
				"tenant_id":  tenantID.String(),
				"scene_type": "coding",
			},
			mockSetup:      func(m *mockTemplateService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 400 {
					t.Errorf("Expected code 400, got %v", body["code"])
				}
			},
		},
		{
			name: "missing tenant_id",
			requestBody: map[string]interface{}{
				"name":       "Test Template",
				"scene_type": "coding",
			},
			mockSetup:      func(m *mockTemplateService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 400 {
					t.Errorf("Expected code 400, got %v", body["code"])
				}
			},
		},
		{
			name: "service validation error - invalid scene_type",
			requestBody: map[string]interface{}{
				"tenant_id":  tenantID.String(),
				"name":       "Test Template",
				"scene_type": "invalid-type",
			},
			mockSetup: func(m *mockTemplateService) {
				m.createFunc = func(ctx context.Context, req *service.CreateTemplateRequest) (*model.Template, error) {
					return nil, errors.NewBadRequestError("invalid scene_type")
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
			name: "service validation error - invalid yaml",
			requestBody: map[string]interface{}{
				"tenant_id":  tenantID.String(),
				"name":       "Test Template",
				"spec":       "invalid: [yaml",
				"scene_type": "coding",
			},
			mockSetup: func(m *mockTemplateService) {
				m.createFunc = func(ctx context.Context, req *service.CreateTemplateRequest) (*model.Template, error) {
					return nil, errors.NewBadRequestError("invalid YAML spec format")
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
				"tenant_id":  tenantID.String(),
				"name":       "Test Template",
				"scene_type": "coding",
			},
			mockSetup: func(m *mockTemplateService) {
				m.createFunc = func(ctx context.Context, req *service.CreateTemplateRequest) (*model.Template, error) {
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
			mockSvc := &mockTemplateService{}
			tt.mockSetup(mockSvc)

			handler := NewTemplateHandler(mockSvc)
			router := setupTestRouter()
			router.POST("/api/v1/templates", handler.Create)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewReader(body))
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

func TestTemplateHandler_GetByID(t *testing.T) {
	existingID := uuid.New()
	tenantID := uuid.New()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockTemplateService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful get",
			id:   existingID.String(),
			mockSetup: func(m *mockTemplateService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Template, error) {
					return &model.Template{
						BaseModel: model.BaseModel{ID: existingID},
						TenantID:  tenantID.String(),
						Name:      "Test Template",
						SceneType: model.TemplateSceneTypeCoding,
						Status:    model.TemplateStatusDraft,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["name"] != "Test Template" {
					t.Errorf("Expected name 'Test Template', got %v", data["name"])
				}
			},
		},
		{
			name: "invalid id format",
			id:   "invalid-uuid",
			mockSetup: func(m *mockTemplateService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Template, error) {
					return nil, errors.NewBadRequestError("invalid template ID format")
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
			mockSetup: func(m *mockTemplateService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Template, error) {
					return nil, errors.NewNotFoundError("template not found")
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
			mockSvc := &mockTemplateService{}
			tt.mockSetup(mockSvc)

			handler := NewTemplateHandler(mockSvc)
			router := setupTestRouter()
			router.GET("/api/v1/templates/:id", handler.GetByID)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/"+tt.id, nil)
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

func TestTemplateHandler_List(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()
	tenantID := uuid.New()

	tests := []struct {
		name           string
		queryParams    string
		mockSetup      func(*mockTemplateService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:        "successful list with defaults",
			queryParams: "",
			mockSetup: func(m *mockTemplateService) {
				m.listFunc = func(ctx context.Context, filter *service.TemplateFilter) ([]*model.Template, int64, error) {
					return []*model.Template{
						{BaseModel: model.BaseModel{ID: id1}, TenantID: tenantID.String(), Name: "Template 1", Status: model.TemplateStatusDraft},
						{BaseModel: model.BaseModel{ID: id2}, TenantID: tenantID.String(), Name: "Template 2", Status: model.TemplateStatusDraft},
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
			mockSetup: func(m *mockTemplateService) {
				m.listFunc = func(ctx context.Context, filter *service.TemplateFilter) ([]*model.Template, int64, error) {
					if filter.Page != 2 {
						t.Errorf("Expected page 2, got %d", filter.Page)
					}
					if filter.PageSize != 5 {
						t.Errorf("Expected page_size 5, got %d", filter.PageSize)
					}
					return []*model.Template{}, 0, nil
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
			mockSetup: func(m *mockTemplateService) {
				m.listFunc = func(ctx context.Context, filter *service.TemplateFilter) ([]*model.Template, int64, error) {
					if filter.TenantID != tenantID.String() {
						t.Errorf("Expected tenant_id '%s', got '%s'", tenantID.String(), filter.TenantID)
					}
					return []*model.Template{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "list with scene_type filter",
			queryParams: "?scene_type=coding",
			mockSetup: func(m *mockTemplateService) {
				m.listFunc = func(ctx context.Context, filter *service.TemplateFilter) ([]*model.Template, int64, error) {
					if filter.SceneType != "coding" {
						t.Errorf("Expected scene_type 'coding', got '%s'", filter.SceneType)
					}
					return []*model.Template{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "list with status filter",
			queryParams: "?status=draft",
			mockSetup: func(m *mockTemplateService) {
				m.listFunc = func(ctx context.Context, filter *service.TemplateFilter) ([]*model.Template, int64, error) {
					if filter.Status != "draft" {
						t.Errorf("Expected status 'draft', got '%s'", filter.Status)
					}
					return []*model.Template{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty list",
			queryParams: "",
			mockSetup: func(m *mockTemplateService) {
				m.listFunc = func(ctx context.Context, filter *service.TemplateFilter) ([]*model.Template, int64, error) {
					return []*model.Template{}, 0, nil
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
			mockSetup: func(m *mockTemplateService) {
				m.listFunc = func(ctx context.Context, filter *service.TemplateFilter) ([]*model.Template, int64, error) {
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
			mockSvc := &mockTemplateService{}
			tt.mockSetup(mockSvc)

			handler := NewTemplateHandler(mockSvc)
			router := setupTestRouter()
			router.GET("/api/v1/templates", handler.List)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/templates"+tt.queryParams, nil)
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

func TestTemplateHandler_Update(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		requestBody    interface{}
		mockSetup      func(*mockTemplateService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful update name",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"name": "Updated Name",
			},
			mockSetup: func(m *mockTemplateService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTemplateRequest) error {
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
			name: "successful update scene_type",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"scene_type": "ops",
			},
			mockSetup: func(m *mockTemplateService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTemplateRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful update status",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"status": "published",
			},
			mockSetup: func(m *mockTemplateService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTemplateRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty update",
			id:          existingID.String(),
			requestBody: map[string]interface{}{},
			mockSetup: func(m *mockTemplateService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTemplateRequest) error {
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
			mockSetup: func(m *mockTemplateService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTemplateRequest) error {
					return errors.NewBadRequestError("invalid template ID format")
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
			mockSetup: func(m *mockTemplateService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTemplateRequest) error {
					return errors.NewNotFoundError("template not found")
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
			name: "validation error - invalid scene_type",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"scene_type": "invalid-type",
			},
			mockSetup: func(m *mockTemplateService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTemplateRequest) error {
					return errors.NewBadRequestError("invalid scene_type")
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
			name: "validation error - invalid yaml",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"spec": "invalid: [yaml",
			},
			mockSetup: func(m *mockTemplateService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTemplateRequest) error {
					return errors.NewBadRequestError("invalid YAML spec format")
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
			mockSvc := &mockTemplateService{}
			tt.mockSetup(mockSvc)

			handler := NewTemplateHandler(mockSvc)
			router := setupTestRouter()
			router.PUT("/api/v1/templates/:id", handler.Update)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/templates/"+tt.id, bytes.NewReader(body))
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

func TestTemplateHandler_Delete(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockTemplateService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful delete draft template",
			id:   existingID.String(),
			mockSetup: func(m *mockTemplateService) {
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
			name: "cannot delete published template",
			id:   existingID.String(),
			mockSetup: func(m *mockTemplateService) {
				m.deleteFunc = func(ctx context.Context, id string) error {
					return errors.NewBadRequestError("only draft templates can be deleted")
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
			name: "invalid id format",
			id:   "invalid-uuid",
			mockSetup: func(m *mockTemplateService) {
				m.deleteFunc = func(ctx context.Context, id string) error {
					return errors.NewBadRequestError("invalid template ID format")
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
			mockSetup: func(m *mockTemplateService) {
				m.deleteFunc = func(ctx context.Context, id string) error {
					return errors.NewNotFoundError("template not found")
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
			mockSetup: func(m *mockTemplateService) {
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
			mockSvc := &mockTemplateService{}
			tt.mockSetup(mockSvc)

			handler := NewTemplateHandler(mockSvc)
			router := setupTestRouter()
			router.DELETE("/api/v1/templates/:id", handler.Delete)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/templates/"+tt.id, nil)
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
