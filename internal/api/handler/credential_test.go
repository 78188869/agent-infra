package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/agent-infra/internal/api/middleware"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// mockCredentialService implements service.CredentialService for testing.
type mockCredentialService struct {
	storeFn       func(ctx context.Context, userID string, req *service.StoreCredentialRequest) (*service.CredentialInfo, error)
	listFn        func(ctx context.Context, userID string) ([]*service.CredentialInfo, error)
	deleteFn      func(ctx context.Context, userID, credType string) error
}

func (m *mockCredentialService) Store(ctx context.Context, userID string, req *service.StoreCredentialRequest) (*service.CredentialInfo, error) {
	return m.storeFn(ctx, userID, req)
}

func (m *mockCredentialService) Get(ctx context.Context, userID, credType string) (string, error) {
	return "", nil
}

func (m *mockCredentialService) Delete(ctx context.Context, userID, credType string) error {
	return m.deleteFn(ctx, userID, credType)
}

func (m *mockCredentialService) List(ctx context.Context, userID string) ([]*service.CredentialInfo, error) {
	return m.listFn(ctx, userID)
}

func (m *mockCredentialService) BuildSandboxEnv(ctx context.Context, userID string) (map[string]string, error) {
	return nil, nil
}

func setupCredentialRouter(svc service.CredentialService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCredentialHandler(svc)

	// Simulate auth middleware setting userID
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, "test-user")
		c.Next()
	})

	credentials := r.Group("/api/v1/credentials")
	{
		credentials.POST("", h.Store)
		credentials.GET("", h.List)
		credentials.DELETE("/:type", h.Delete)
	}

	return r
}

func TestCredentialHandler_Store(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		svc := &mockCredentialService{
			storeFn: func(ctx context.Context, userID string, req *service.StoreCredentialRequest) (*service.CredentialInfo, error) {
				return &service.CredentialInfo{ID: "cred-1", Type: req.Type}, nil
			},
		}
		r := setupCredentialRouter(svc)

		body, _ := json.Marshal(map[string]string{"type": "git_token", "value": "secret"})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/credentials", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Store() status = %d, want %d", w.Code, http.StatusCreated)
		}
	})

	t.Run("missing type", func(t *testing.T) {
		svc := &mockCredentialService{}
		r := setupCredentialRouter(svc)

		body, _ := json.Marshal(map[string]string{"value": "secret"})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/credentials", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Store() status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestCredentialHandler_List(t *testing.T) {
	svc := &mockCredentialService{
		listFn: func(ctx context.Context, userID string) ([]*service.CredentialInfo, error) {
			return []*service.CredentialInfo{
				{ID: "cred-1", Type: "git_token"},
			}, nil
		},
	}
	r := setupCredentialRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("List() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCredentialHandler_Delete(t *testing.T) {
	svc := &mockCredentialService{
		deleteFn: func(ctx context.Context, userID, credType string) error {
			return nil
		},
	}
	r := setupCredentialRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/credentials/git_token", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Delete() status = %d, want %d", w.Code, http.StatusOK)
	}
}
