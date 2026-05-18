package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/service"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/gin-gonic/gin"
)

// mockAPIKeyServiceForAuth implements service.APIKeyService for testing
type mockAPIKeyServiceForAuth struct {
	validateFunc func(ctx context.Context, rawKey string) (*model.APIKey, error)
}

func (m *mockAPIKeyServiceForAuth) Create(ctx context.Context, userID string, req *service.CreateAPIKeyRequest) (*model.APIKey, string, error) {
	return nil, "", nil
}

func (m *mockAPIKeyServiceForAuth) GetByID(ctx context.Context, userID string, keyID string) (*model.APIKey, error) {
	return nil, nil
}

func (m *mockAPIKeyServiceForAuth) List(ctx context.Context, userID string, filter *service.APIKeyFilter) ([]*model.APIKey, int64, error) {
	return nil, 0, nil
}

func (m *mockAPIKeyServiceForAuth) Revoke(ctx context.Context, userID string, keyID string) error {
	return nil
}

func (m *mockAPIKeyServiceForAuth) Validate(ctx context.Context, rawKey string) (*model.APIKey, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, rawKey)
	}
	return nil, nil
}

// mockUserServiceForAuth implements service.UserService for testing
type mockUserServiceForAuth struct {
	getByIDFunc func(ctx context.Context, id string) (*model.User, error)
}

func (m *mockUserServiceForAuth) GetByID(ctx context.Context, id string) (*model.User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func TestAPIKeyAuth_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAPIKeySvc := &mockAPIKeyServiceForAuth{}
	mockUserSvc := &mockUserServiceForAuth{}

	r := gin.New()
	r.Use(APIKeyAuth(mockAPIKeySvc, mockUserSvc))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_InvalidScheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAPIKeySvc := &mockAPIKeyServiceForAuth{}
	mockUserSvc := &mockUserServiceForAuth{}

	r := gin.New()
	r.Use(APIKeyAuth(mockAPIKeySvc, mockUserSvc))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected 401 for Basic auth, got %d", w.Code)
	}
}

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAPIKeySvc := &mockAPIKeyServiceForAuth{
		validateFunc: func(ctx context.Context, rawKey string) (*model.APIKey, error) {
			return &model.APIKey{ID: "key-1", UserID: "user-1"}, nil
		},
	}
	mockUserSvc := &mockUserServiceForAuth{
		getByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			return &model.User{
				ID:       "user-1",
				TenantID: "tenant-1",
				Role:     model.UserRoleDeveloper,
				Status:   model.UserStatusActive,
			}, nil
		},
	}

	r := gin.New()
	r.Use(APIKeyAuth(mockAPIKeySvc, mockUserSvc))
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get(ContextKeyUserID)
		tenantID, _ := c.Get(ContextKeyTenantID)
		userRole, _ := c.Get(ContextKeyUserRole)
		c.JSON(200, gin.H{
			"user_id":    userID,
			"tenant_id":  tenantID,
			"user_role":  userRole,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ak_valid_token")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAPIKeySvc := &mockAPIKeyServiceForAuth{
		validateFunc: func(ctx context.Context, rawKey string) (*model.APIKey, error) {
			return nil, errors.NewBadRequestError("invalid key")
		},
	}
	mockUserSvc := &mockUserServiceForAuth{}

	r := gin.New()
	r.Use(APIKeyAuth(mockAPIKeySvc, mockUserSvc))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer bad_token")
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_UserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAPIKeySvc := &mockAPIKeyServiceForAuth{
		validateFunc: func(ctx context.Context, rawKey string) (*model.APIKey, error) {
			return &model.APIKey{ID: "key-1", UserID: "user-1"}, nil
		},
	}
	mockUserSvc := &mockUserServiceForAuth{
		getByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			return nil, errors.NewNotFoundError("user not found")
		},
	}

	r := gin.New()
	r.Use(APIKeyAuth(mockAPIKeySvc, mockUserSvc))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ak_valid_token")
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected 401 for user not found, got %d", w.Code)
	}
}

func TestExtractBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"empty header", "", ""},
		{"basic auth", "Basic dXNlcjpwYXNz", ""},
		{"bearer token", "Bearer ak_test123", "ak_test123"},
		{"bearer with extra spaces", "Bearer   ak_test123  ", "ak_test123"},
		{"lowercase bearer", "bearer ak_test123", "ak_test123"},
		{"no space", "Bearer", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			if tt.header != "" {
				c.Request, _ = http.NewRequest("GET", "/test", nil)
				c.Request.Header.Set("Authorization", tt.header)
			} else {
				c.Request, _ = http.NewRequest("GET", "/test", nil)
			}
			got := extractBearerToken(c)
			if got != tt.want {
				t.Errorf("extractBearerToken() = %q, want %q", got, tt.want)
			}
		})
	}
}
