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

// mockInterventionService implements service.InterventionService for testing
type mockInterventionService struct {
	pauseFunc            func(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error)
	resumeFunc           func(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error)
	cancelFunc           func(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error)
	injectFunc           func(ctx context.Context, req *service.InjectInterventionRequest) (*model.Intervention, error)
	listInterventionsFunc func(ctx context.Context, taskID string, filter *service.InterventionFilter) ([]*model.Intervention, int64, error)
}

func (m *mockInterventionService) Pause(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error) {
	if m.pauseFunc != nil {
		return m.pauseFunc(ctx, taskID, operatorID, reason)
	}
	return nil, nil
}

func (m *mockInterventionService) Resume(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error) {
	if m.resumeFunc != nil {
		return m.resumeFunc(ctx, taskID, operatorID, reason)
	}
	return nil, nil
}

func (m *mockInterventionService) Cancel(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error) {
	if m.cancelFunc != nil {
		return m.cancelFunc(ctx, taskID, operatorID, reason)
	}
	return nil, nil
}

func (m *mockInterventionService) Inject(ctx context.Context, req *service.InjectInterventionRequest) (*model.Intervention, error) {
	if m.injectFunc != nil {
		return m.injectFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockInterventionService) ListInterventions(ctx context.Context, taskID string, filter *service.InterventionFilter) ([]*model.Intervention, int64, error) {
	if m.listInterventionsFunc != nil {
		return m.listInterventionsFunc(ctx, taskID, filter)
	}
	return nil, 0, nil
}

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestRouterWithAuth creates a test router with user_id in context for auth simulation
func setupTestRouterWithAuth(userID string) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(UserIDKey, userID)
		c.Next()
	})
	return router
}

// setupTestRouterWithoutAuth creates a test router without user_id in context
func setupTestRouterWithoutAuth() *gin.Engine {
	return gin.New()
}

func TestInterventionHandler_Pause(t *testing.T) {
	taskID := uuid.New()
	operatorID := uuid.New()
	interventionID := uuid.New()

	tests := []struct {
		name           string
		taskID         string
		requestBody    interface{}
		mockSetup      func(*mockInterventionService)
		expectedStatus int
		useAuth        bool
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:   "successful pause",
			taskID: taskID.String(),
			requestBody: map[string]interface{}{
				"reason": "Need to review task progress",
			},
			mockSetup: func(m *mockInterventionService) {
				m.pauseFunc = func(ctx context.Context, tid, oid, reason string) (*model.Intervention, error) {
					return &model.Intervention{
						BaseModel:  model.BaseModel{ID: interventionID},
						TaskID:     tid,
						OperatorID: oid,
						Action:     model.InterventionActionPause,
						Reason:     reason,
						Status:     model.InterventionStatusApplied,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			useAuth:        true,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["action"] != "pause" {
					t.Errorf("Expected action 'pause', got %v", data["action"])
				}
			},
		},
		{
			name:   "pause without reason",
			taskID: taskID.String(),
			requestBody: map[string]interface{}{
				"reason": "",
			},
			mockSetup: func(m *mockInterventionService) {
				m.pauseFunc = func(ctx context.Context, tid, oid, reason string) (*model.Intervention, error) {
					return &model.Intervention{
						BaseModel:  model.BaseModel{ID: interventionID},
						TaskID:     tid,
						OperatorID: oid,
						Action:     model.InterventionActionPause,
						Status:     model.InterventionStatusApplied,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			useAuth:        true,
		},
		{
			name:        "unauthorized - no user_id in context",
			taskID:      taskID.String(),
			requestBody: map[string]interface{}{"reason": "test"},
			mockSetup: func(m *mockInterventionService) {
				// This won't be called because auth check happens first
			},
			expectedStatus: http.StatusUnauthorized,
			useAuth:        false,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 401 {
					t.Errorf("Expected code 401, got %v", body["code"])
				}
			},
		},
		{
			name:   "task not found",
			taskID: uuid.New().String(),
			requestBody: map[string]interface{}{
				"reason": "test",
			},
			mockSetup: func(m *mockInterventionService) {
				m.pauseFunc = func(ctx context.Context, tid, oid, reason string) (*model.Intervention, error) {
					return nil, errors.NewNotFoundError("task not found")
				}
			},
			expectedStatus: http.StatusNotFound,
			useAuth:        true,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 404 {
					t.Errorf("Expected code 404, got %v", body["code"])
				}
			},
		},
		{
			name:   "invalid task state",
			taskID: taskID.String(),
			requestBody: map[string]interface{}{
				"reason": "test",
			},
			useAuth: true,
			mockSetup: func(m *mockInterventionService) {
				m.pauseFunc = func(ctx context.Context, tid, oid, reason string) (*model.Intervention, error) {
					return nil, errors.NewBadRequestError("cannot pause task in 'pending' state")
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
			name:   "invalid task ID format",
			taskID: "invalid-uuid",
			requestBody: map[string]interface{}{
				"reason": "test",
			},
			useAuth: true,
			mockSetup: func(m *mockInterventionService) {
				m.pauseFunc = func(ctx context.Context, tid, oid, reason string) (*model.Intervention, error) {
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
			name:   "internal error",
			taskID: taskID.String(),
			requestBody: map[string]interface{}{
				"reason": "test",
			},
			useAuth: true,
			mockSetup: func(m *mockInterventionService) {
				m.pauseFunc = func(ctx context.Context, tid, oid, reason string) (*model.Intervention, error) {
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
			mockSvc := &mockInterventionService{}
			tt.mockSetup(mockSvc)

			handler := NewInterventionHandler(mockSvc)
			var router *gin.Engine
			if tt.useAuth {
				router = setupTestRouterWithAuth(operatorID.String())
			} else {
				router = setupTestRouterWithoutAuth()
			}
			router.POST("/api/v1/tasks/:id/pause", handler.Pause)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+tt.taskID+"/pause", bytes.NewReader(body))
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

func TestInterventionHandler_Resume(t *testing.T) {
	taskID := uuid.New()
	operatorID := uuid.New()
	interventionID := uuid.New()

	tests := []struct {
		name           string
		taskID         string
		requestBody    interface{}
		mockSetup      func(*mockInterventionService)
		expectedStatus int
		useAuth        bool
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:   "successful resume",
			taskID: taskID.String(),
			requestBody: map[string]interface{}{
				"reason": "Ready to continue",
			},
			mockSetup: func(m *mockInterventionService) {
				m.resumeFunc = func(ctx context.Context, tid, oid, reason string) (*model.Intervention, error) {
					return &model.Intervention{
						BaseModel:  model.BaseModel{ID: interventionID},
						TaskID:     tid,
						OperatorID: oid,
						Action:     model.InterventionActionResume,
						Reason:     reason,
						Status:     model.InterventionStatusApplied,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			useAuth:        true,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["action"] != "resume" {
					t.Errorf("Expected action 'resume', got %v", data["action"])
				}
			},
		},
		{
			name:   "task not paused",
			taskID: taskID.String(),
			requestBody: map[string]interface{}{
				"reason": "test",
			},
			useAuth: true,
			mockSetup: func(m *mockInterventionService) {
				m.resumeFunc = func(ctx context.Context, tid, oid, reason string) (*model.Intervention, error) {
					return nil, errors.NewBadRequestError("cannot resume task in 'running' state")
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
			name:           "unauthorized - no user_id in context",
			taskID:         taskID.String(),
			requestBody:    map[string]interface{}{"reason": "test"},
			mockSetup:      func(m *mockInterventionService) {},
			expectedStatus: http.StatusUnauthorized,
			useAuth:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockInterventionService{}
			tt.mockSetup(mockSvc)

			handler := NewInterventionHandler(mockSvc)
			var router *gin.Engine
			if tt.useAuth {
				router = setupTestRouterWithAuth(operatorID.String())
			} else {
				router = setupTestRouterWithoutAuth()
			}
			router.POST("/api/v1/tasks/:id/resume", handler.Resume)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+tt.taskID+"/resume", bytes.NewReader(body))
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

func TestInterventionHandler_Cancel(t *testing.T) {
	taskID := uuid.New()
	operatorID := uuid.New()
	interventionID := uuid.New()

	tests := []struct {
		name           string
		taskID         string
		requestBody    interface{}
		mockSetup      func(*mockInterventionService)
		expectedStatus int
		useAuth        bool
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:   "successful cancel",
			taskID: taskID.String(),
			requestBody: map[string]interface{}{
				"reason": "Task no longer needed",
			},
			mockSetup: func(m *mockInterventionService) {
				m.cancelFunc = func(ctx context.Context, tid, oid, reason string) (*model.Intervention, error) {
					return &model.Intervention{
						BaseModel:  model.BaseModel{ID: interventionID},
						TaskID:     tid,
						OperatorID: oid,
						Action:     model.InterventionActionCancel,
						Reason:     reason,
						Status:     model.InterventionStatusApplied,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			useAuth:        true,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["action"] != "cancel" {
					t.Errorf("Expected action 'cancel', got %v", data["action"])
				}
			},
		},
		{
			name:   "cannot cancel completed task",
			useAuth: true,
			taskID: taskID.String(),
			requestBody: map[string]interface{}{
				"reason": "test",
			},
			mockSetup: func(m *mockInterventionService) {
				m.cancelFunc = func(ctx context.Context, tid, oid, reason string) (*model.Intervention, error) {
					return nil, errors.NewBadRequestError("cannot cancel task in 'succeeded' state")
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
			name:           "unauthorized - no user_id in context",
			taskID:         taskID.String(),
			requestBody:    map[string]interface{}{"reason": "test"},
			mockSetup:      func(m *mockInterventionService) {},
			expectedStatus: http.StatusUnauthorized,
			useAuth:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockInterventionService{}
			tt.mockSetup(mockSvc)

			handler := NewInterventionHandler(mockSvc)
			var router *gin.Engine
			if tt.useAuth {
				router = setupTestRouterWithAuth(operatorID.String())
			} else {
				router = setupTestRouterWithoutAuth()
			}
			router.POST("/api/v1/tasks/:id/cancel", handler.Cancel)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+tt.taskID+"/cancel", bytes.NewReader(body))
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

func TestInterventionHandler_Inject(t *testing.T) {
	taskID := uuid.New()
	operatorID := uuid.New()
	interventionID := uuid.New()

	tests := []struct {
		name           string
		taskID         string
		requestBody    interface{}
		mockSetup      func(*mockInterventionService)
		expectedStatus int
		useAuth        bool
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:   "successful inject",
			taskID: taskID.String(),
			requestBody: map[string]interface{}{
				"content": "Please review the output and proceed",
				"reason":  "Quality check needed",
			},
			mockSetup: func(m *mockInterventionService) {
				m.injectFunc = func(ctx context.Context, req *service.InjectInterventionRequest) (*model.Intervention, error) {
					return &model.Intervention{
						BaseModel:  model.BaseModel{ID: interventionID},
						TaskID:     req.TaskID,
						OperatorID: req.OperatorID,
						Action:     model.InterventionActionInject,
						Status:     model.InterventionStatusApplied,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			useAuth:        true,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 0 {
					t.Errorf("Expected code 0, got %v", body["code"])
				}
				data := body["data"].(map[string]interface{})
				if data["action"] != "inject" {
					t.Errorf("Expected action 'inject', got %v", data["action"])
				}
			},
		},
		{
			name:   "missing required content field",
			taskID: taskID.String(),
			requestBody: map[string]interface{}{
				"reason": "test",
			},
			mockSetup:      func(m *mockInterventionService) {},
			expectedStatus: http.StatusBadRequest,
			useAuth:        true,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["code"].(float64) != 400 {
					t.Errorf("Expected code 400, got %v", body["code"])
				}
			},
		},
		{
			name:   "cannot inject into paused task",
			taskID: taskID.String(),
			requestBody: map[string]interface{}{
				"content": "test instruction",
			},
			useAuth: true,
			mockSetup: func(m *mockInterventionService) {
				m.injectFunc = func(ctx context.Context, req *service.InjectInterventionRequest) (*model.Intervention, error) {
					return nil, errors.NewBadRequestError("cannot inject into task in 'paused' state")
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
			name:           "unauthorized - no user_id in context",
			taskID:         taskID.String(),
			requestBody:    map[string]interface{}{"content": "test"},
			mockSetup:      func(m *mockInterventionService) {},
			expectedStatus: http.StatusUnauthorized,
			useAuth:        false,
		},
		{
			name:   "task not found",
			taskID: uuid.New().String(),
			requestBody: map[string]interface{}{
				"content": "test instruction",
			},
			useAuth: true,
			mockSetup: func(m *mockInterventionService) {
				m.injectFunc = func(ctx context.Context, req *service.InjectInterventionRequest) (*model.Intervention, error) {
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
			mockSvc := &mockInterventionService{}
			tt.mockSetup(mockSvc)

			handler := NewInterventionHandler(mockSvc)
			var router *gin.Engine
			if tt.useAuth {
				router = setupTestRouterWithAuth(operatorID.String())
			} else {
				router = setupTestRouterWithoutAuth()
			}
			router.POST("/api/v1/tasks/:id/inject", handler.Inject)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+tt.taskID+"/inject", bytes.NewReader(body))
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

func TestInterventionHandler_ListInterventions(t *testing.T) {
	taskID := uuid.New()
	interventionID1 := uuid.New()
	interventionID2 := uuid.New()

	tests := []struct {
		name           string
		taskID         string
		queryParams    string
		mockSetup      func(*mockInterventionService)
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:        "successful list with defaults",
			taskID:      taskID.String(),
			queryParams: "",
			mockSetup: func(m *mockInterventionService) {
				m.listInterventionsFunc = func(ctx context.Context, tid string, filter *service.InterventionFilter) ([]*model.Intervention, int64, error) {
					return []*model.Intervention{
						{
							BaseModel:  model.BaseModel{ID: interventionID1},
							TaskID:     tid,
							Action:     model.InterventionActionPause,
							Status:     model.InterventionStatusApplied,
						},
						{
							BaseModel:  model.BaseModel{ID: interventionID2},
							TaskID:     tid,
							Action:     model.InterventionActionResume,
							Status:     model.InterventionStatusApplied,
						},
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
			taskID:      taskID.String(),
			queryParams: "?page=2&page_size=5",
			mockSetup: func(m *mockInterventionService) {
				m.listInterventionsFunc = func(ctx context.Context, tid string, filter *service.InterventionFilter) ([]*model.Intervention, int64, error) {
					if filter.Page != 2 {
						t.Errorf("Expected page 2, got %d", filter.Page)
					}
					if filter.PageSize != 5 {
						t.Errorf("Expected page_size 5, got %d", filter.PageSize)
					}
					return []*model.Intervention{}, 0, nil
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
			name:        "list with action filter",
			taskID:      taskID.String(),
			queryParams: "?action=pause",
			mockSetup: func(m *mockInterventionService) {
				m.listInterventionsFunc = func(ctx context.Context, tid string, filter *service.InterventionFilter) ([]*model.Intervention, int64, error) {
					if filter.Action != "pause" {
						t.Errorf("Expected action 'pause', got '%s'", filter.Action)
					}
					return []*model.Intervention{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "list with status filter",
			taskID:      taskID.String(),
			queryParams: "?status=applied",
			mockSetup: func(m *mockInterventionService) {
				m.listInterventionsFunc = func(ctx context.Context, tid string, filter *service.InterventionFilter) ([]*model.Intervention, int64, error) {
					if filter.Status != "applied" {
						t.Errorf("Expected status 'applied', got '%s'", filter.Status)
					}
					return []*model.Intervention{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty list",
			taskID:      taskID.String(),
			queryParams: "",
			mockSetup: func(m *mockInterventionService) {
				m.listInterventionsFunc = func(ctx context.Context, tid string, filter *service.InterventionFilter) ([]*model.Intervention, int64, error) {
					return []*model.Intervention{}, 0, nil
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
			name:        "invalid task ID",
			taskID:      "invalid-uuid",
			queryParams: "",
			mockSetup: func(m *mockInterventionService) {
				m.listInterventionsFunc = func(ctx context.Context, tid string, filter *service.InterventionFilter) ([]*model.Intervention, int64, error) {
					return nil, 0, errors.NewBadRequestError("invalid task ID format")
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
			name:        "internal error",
			taskID:      taskID.String(),
			queryParams: "",
			mockSetup: func(m *mockInterventionService) {
				m.listInterventionsFunc = func(ctx context.Context, tid string, filter *service.InterventionFilter) ([]*model.Intervention, int64, error) {
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
			mockSvc := &mockInterventionService{}
			tt.mockSetup(mockSvc)

			handler := NewInterventionHandler(mockSvc)
			router := setupTestRouter()
			router.GET("/api/v1/tasks/:id/interventions", handler.ListInterventions)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+tt.taskID+"/interventions"+tt.queryParams, nil)
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
