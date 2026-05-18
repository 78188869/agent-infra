package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/agent-infra/internal/api/middleware"
	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// mockAPIKeyServiceForHandler implements service.APIKeyService for handler tests
type mockAPIKeyServiceForHandler struct {
	createFunc  func(ctx context.Context, userID string, req *service.CreateAPIKeyRequest) (*model.APIKey, string, error)
	getByIDFunc func(ctx context.Context, userID string, keyID string) (*model.APIKey, error)
	listFunc    func(ctx context.Context, userID string, filter *service.APIKeyFilter) ([]*model.APIKey, int64, error)
	revokeFunc  func(ctx context.Context, userID string, keyID string) error
	validateFunc func(ctx context.Context, rawKey string) (*model.APIKey, error)
}

func (m *mockAPIKeyServiceForHandler) Create(ctx context.Context, userID string, req *service.CreateAPIKeyRequest) (*model.APIKey, string, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, userID, req)
	}
	return nil, "", nil
}

func (m *mockAPIKeyServiceForHandler) GetByID(ctx context.Context, userID string, keyID string) (*model.APIKey, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, userID, keyID)
	}
	return nil, nil
}

func (m *mockAPIKeyServiceForHandler) List(ctx context.Context, userID string, filter *service.APIKeyFilter) ([]*model.APIKey, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, userID, filter)
	}
	return []*model.APIKey{}, 0, nil
}

func (m *mockAPIKeyServiceForHandler) Revoke(ctx context.Context, userID string, keyID string) error {
	if m.revokeFunc != nil {
		return m.revokeFunc(ctx, userID, keyID)
	}
	return nil
}

func (m *mockAPIKeyServiceForHandler) Validate(ctx context.Context, rawKey string) (*model.APIKey, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, rawKey)
	}
	return nil, nil
}

func setupAPIKeyRouter(mockSvc *mockAPIKeyServiceForHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAPIKeyHandler(mockSvc)

	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, "user-123")
		c.Set(middleware.ContextKeyTenantID, "tenant-123")
		c.Next()
	})

	keys := r.Group("/api/v1/api-keys")
	{
		keys.POST("", h.Create)
		keys.GET("", h.List)
		keys.DELETE("/:id", h.Revoke)
	}
	return r
}

func TestAPIKeyHandler_Create(t *testing.T) {
	now := time.Now()
	mockSvc := &mockAPIKeyServiceForHandler{
		createFunc: func(ctx context.Context, userID string, req *service.CreateAPIKeyRequest) (*model.APIKey, string, error) {
			return &model.APIKey{
				ID:        "key-123",
				Name:      req.Name,
				KeyPrefix: "ak_abcd",
				CreatedAt: now,
			}, "ak_abcd1234secret", nil
		},
	}
	r := setupAPIKeyRouter(mockSvc)

	body, _ := json.Marshal(map[string]string{"name": "Test Key"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/api-keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("Expected 201, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["secret"] != "ak_abcd1234secret" {
		t.Errorf("Expected secret in response, got %v", data["secret"])
	}
}

func TestAPIKeyHandler_Create_MissingName(t *testing.T) {
	mockSvc := &mockAPIKeyServiceForHandler{}
	r := setupAPIKeyRouter(mockSvc)

	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/api-keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for missing name, got %d", w.Code)
	}
}

func TestAPIKeyHandler_Create_InvalidJSON(t *testing.T) {
	mockSvc := &mockAPIKeyServiceForHandler{}
	r := setupAPIKeyRouter(mockSvc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/api-keys", bytes.NewBufferString("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestAPIKeyHandler_List(t *testing.T) {
	mockSvc := &mockAPIKeyServiceForHandler{
		listFunc: func(ctx context.Context, userID string, filter *service.APIKeyFilter) ([]*model.APIKey, int64, error) {
			return []*model.APIKey{
				{ID: "key-1", Name: "Key 1", Status: model.APIKeyStatusActive},
			}, 1, nil
		},
	}
	r := setupAPIKeyRouter(mockSvc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/api-keys", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["total"].(float64) != 1 {
		t.Errorf("Expected total=1, got %v", data["total"])
	}
}

func TestAPIKeyHandler_List_WithPagination(t *testing.T) {
	mockSvc := &mockAPIKeyServiceForHandler{
		listFunc: func(ctx context.Context, userID string, filter *service.APIKeyFilter) ([]*model.APIKey, int64, error) {
			if filter.Page != 2 || filter.PageSize != 5 {
				t.Errorf("Expected page=2, pageSize=5, got page=%d pageSize=%d", filter.Page, filter.PageSize)
			}
			return []*model.APIKey{}, int64(15), nil
		},
	}
	r := setupAPIKeyRouter(mockSvc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/api-keys?page=2&page_size=5", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestAPIKeyHandler_Revoke(t *testing.T) {
	mockSvc := &mockAPIKeyServiceForHandler{
		revokeFunc: func(ctx context.Context, userID string, keyID string) error {
			if userID != "user-123" {
				t.Errorf("Expected userID=user-123, got %s", userID)
			}
			if keyID != "key-123" {
				t.Errorf("Expected keyID=key-123, got %s", keyID)
			}
			return nil
		},
	}
	r := setupAPIKeyRouter(mockSvc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/api-keys/key-123", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}
