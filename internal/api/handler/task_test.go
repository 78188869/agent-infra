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
	"gorm.io/datatypes"
)

// mockTaskService implements service.TaskService for testing
type mockTaskService struct {
	createFunc  func(ctx context.Context, req *service.CreateTaskRequest) (*model.Task, error)
	getByIDFunc func(ctx context.Context, id string) (*model.Task, error)
	listFunc    func(ctx context.Context, filter *service.TaskFilter) ([]*model.Task, int64, error)
	updateFunc  func(ctx context.Context, id string, req *service.UpdateTaskRequest) error
	deleteFunc  func(ctx context.Context, id string) error
}

func (m *mockTaskService) Create(ctx context.Context, req *service.CreateTaskRequest) (*model.Task, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockTaskService) GetByID(ctx context.Context, id string) (*model.Task, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockTaskService) List(ctx context.Context, filter *service.TaskFilter) ([]*model.Task, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockTaskService) Update(ctx context.Context, id string, req *service.UpdateTaskRequest) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, req)
	}
	return nil
}

func (m *mockTaskService) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestTaskHandler_Create(t *testing.T) {
	existingID := uuid.New()
	tenantID := uuid.New()
	creatorID := uuid.New()
	providerID := uuid.New()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*mockTaskService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful create",
			requestBody: map[string]interface{}{
				"tenant_id":   tenantID.String(),
				"creator_id":  creatorID.String(),
				"provider_id": providerID.String(),
				"name":        "Test Task",
				"priority":    "high",
				"params":      map[string]string{"key": "value"},
			},
			mockSetup: func(m *mockTaskService) {
				m.createFunc = func(ctx context.Context, req *service.CreateTaskRequest) (*model.Task, error) {
					return &model.Task{
						BaseModel:   model.BaseModel{ID: existingID},
						TenantID:    req.TenantID,
						CreatorID:   req.CreatorID,
						ProviderID:  req.ProviderID,
						Name:        req.Name,
						Status:      model.TaskStatusPending,
						Priority:    req.Priority,
						Params:      datatypes.JSON(req.Params),
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["name"] != "Test Task" {
					t.Errorf("Expected name 'Test Task', got %v", data["name"])
				}
			},
		},
		{
			name: "missing required field tenant_id",
			requestBody: map[string]interface{}{
				"creator_id":  creatorID.String(),
				"provider_id": providerID.String(),
				"name":        "Test Task",
			},
			mockSetup:      func(m *mockTaskService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 400 {
					t.Errorf("Expected code 400, got %v", body["code"])
				}
			},
		},
		{
			name: "missing required field name",
			requestBody: map[string]interface{}{
				"tenant_id":   tenantID.String(),
				"creator_id":  creatorID.String(),
				"provider_id": providerID.String(),
			},
			mockSetup:      func(m *mockTaskService) {},
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
				"tenant_id":   "invalid-uuid",
				"creator_id":  creatorID.String(),
				"provider_id": providerID.String(),
				"name":        "Test Task",
			},
			mockSetup: func(m *mockTaskService) {
				m.createFunc = func(ctx context.Context, req *service.CreateTaskRequest) (*model.Task, error) {
					return nil, errors.NewBadRequestError("invalid tenant_id format")
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
				"tenant_id":   tenantID.String(),
				"creator_id":  creatorID.String(),
				"provider_id": providerID.String(),
				"name":        "Test Task",
			},
			mockSetup: func(m *mockTaskService) {
				m.createFunc = func(ctx context.Context, req *service.CreateTaskRequest) (*model.Task, error) {
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
			mockSvc := &mockTaskService{}
			tt.mockSetup(mockSvc)

			handler := NewTaskHandler(mockSvc)
			router := setupTestRouter()
			router.POST("/api/v1/tasks", handler.Create)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
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

func TestTaskHandler_GetByID(t *testing.T) {
	existingID := uuid.New()
	tenantID := uuid.New()
	creatorID := uuid.New()
	providerID := uuid.New()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockTaskService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful get",
			id:   existingID.String(),
			mockSetup: func(m *mockTaskService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Task, error) {
					return &model.Task{
						BaseModel:   model.BaseModel{ID: existingID},
						TenantID:    tenantID.String(),
						Name:        "Test Task",
						Status:      model.TaskStatusPending,
						ProviderID:  providerID.String(),
						CreatorID:   creatorID.String(),
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["name"] != "Test Task" {
					t.Errorf("Expected name 'Test Task', got %v", data["name"])
				}
			},
		},
		{
			name: "invalid id format",
			id:   "invalid-uuid",
			mockSetup: func(m *mockTaskService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Task, error) {
					return nil, errors.NewBadRequestError("invalid task ID format")
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
			mockSetup: func(m *mockTaskService) {
				m.getByIDFunc = func(ctx context.Context, id string) (*model.Task, error) {
					return nil, errors.NewNotFoundError("task not found")
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
			mockSvc := &mockTaskService{}
			tt.mockSetup(mockSvc)

			handler := NewTaskHandler(mockSvc)
			router := setupTestRouter()
			router.GET("/api/v1/tasks/:id", handler.GetByID)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+tt.id, nil)
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

func TestTaskHandler_List(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()
	tenantID := uuid.New()

	tests := []struct {
		name           string
		queryParams    string
		mockSetup      func(*mockTaskService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:        "successful list with defaults",
			queryParams: "",
			mockSetup: func(m *mockTaskService) {
				m.listFunc = func(ctx context.Context, filter *service.TaskFilter) ([]*model.Task, int64, error) {
					return []*model.Task{
						{BaseModel: model.BaseModel{ID: id1}, TenantID: tenantID.String(), Name: "Task 1", Status: model.TaskStatusPending},
						{BaseModel: model.BaseModel{ID: id2}, TenantID: tenantID.String(), Name: "Task 2", Status: model.TaskStatusRunning},
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
			mockSetup: func(m *mockTaskService) {
				m.listFunc = func(ctx context.Context, filter *service.TaskFilter) ([]*model.Task, int64, error) {
					if filter.Page != 2 {
						t.Errorf("Expected page 2, got %d", filter.Page)
					}
					if filter.PageSize != 5 {
						t.Errorf("Expected page_size 5, got %d", filter.PageSize)
					}
					return []*model.Task{}, 0, nil
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
			queryParams: "?status=pending",
			mockSetup: func(m *mockTaskService) {
				m.listFunc = func(ctx context.Context, filter *service.TaskFilter) ([]*model.Task, int64, error) {
					if filter.Status != "pending" {
						t.Errorf("Expected status 'pending', got '%s'", filter.Status)
					}
					return []*model.Task{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "list with tenant_id filter",
			queryParams: "?tenant_id=" + tenantID.String(),
			mockSetup: func(m *mockTaskService) {
				m.listFunc = func(ctx context.Context, filter *service.TaskFilter) ([]*model.Task, int64, error) {
					if filter.TenantID != tenantID.String() {
						t.Errorf("Expected tenant_id '%s', got '%s'", tenantID.String(), filter.TenantID)
					}
					return []*model.Task{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty list",
			queryParams: "",
			mockSetup: func(m *mockTaskService) {
				m.listFunc = func(ctx context.Context, filter *service.TaskFilter) ([]*model.Task, int64, error) {
					return []*model.Task{}, 0, nil
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
			mockSetup: func(m *mockTaskService) {
				m.listFunc = func(ctx context.Context, filter *service.TaskFilter) ([]*model.Task, int64, error) {
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
			mockSvc := &mockTaskService{}
			tt.mockSetup(mockSvc)

			handler := NewTaskHandler(mockSvc)
			router := setupTestRouter()
			router.GET("/api/v1/tasks", handler.List)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks"+tt.queryParams, nil)
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

func TestTaskHandler_Update(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		requestBody    interface{}
		mockSetup      func(*mockTaskService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful status update",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"status": "scheduled",
			},
			mockSetup: func(m *mockTaskService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTaskRequest) error {
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
			name: "successful update with result",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"status": "succeeded",
				"result": map[string]string{"output": "done"},
			},
			mockSetup: func(m *mockTaskService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTaskRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful update with error message",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"status":        "failed",
				"error_message": "something went wrong",
			},
			mockSetup: func(m *mockTaskService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTaskRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty update",
			id:          existingID.String(),
			requestBody: map[string]interface{}{},
			mockSetup: func(m *mockTaskService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTaskRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid id format",
			id:   "invalid-uuid",
			requestBody: map[string]interface{}{
				"status": "running",
			},
			mockSetup: func(m *mockTaskService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTaskRequest) error {
					return errors.NewBadRequestError("invalid task ID format")
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
			name: "invalid status transition",
			id:   existingID.String(),
			requestBody: map[string]interface{}{
				"status": "running",
			},
			mockSetup: func(m *mockTaskService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTaskRequest) error {
					return errors.NewBadRequestError("invalid status transition")
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
				"status": "scheduled",
			},
			mockSetup: func(m *mockTaskService) {
				m.updateFunc = func(ctx context.Context, id string, req *service.UpdateTaskRequest) error {
					return errors.NewNotFoundError("task not found")
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
			mockSvc := &mockTaskService{}
			tt.mockSetup(mockSvc)

			handler := NewTaskHandler(mockSvc)
			router := setupTestRouter()
			router.PUT("/api/v1/tasks/:id", handler.Update)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/"+tt.id, bytes.NewReader(body))
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

func TestTaskHandler_Delete(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		id             string
		mockSetup      func(*mockTaskService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful delete",
			id:   existingID.String(),
			mockSetup: func(m *mockTaskService) {
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
			mockSetup: func(m *mockTaskService) {
				m.deleteFunc = func(ctx context.Context, id string) error {
					return errors.NewBadRequestError("invalid task ID format")
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
			mockSetup: func(m *mockTaskService) {
				m.deleteFunc = func(ctx context.Context, id string) error {
					return errors.NewNotFoundError("task not found")
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
			mockSetup: func(m *mockTaskService) {
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
			mockSvc := &mockTaskService{}
			tt.mockSetup(mockSvc)

			handler := NewTaskHandler(mockSvc)
			router := setupTestRouter()
			router.DELETE("/api/v1/tasks/:id", handler.Delete)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+tt.id, nil)
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
