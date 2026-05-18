package service

import (
	"context"
	"testing"
	"time"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
)

// mockAPIKeyRepository implements repository.APIKeyRepository for testing
type mockAPIKeyRepository struct {
	createFunc        func(ctx context.Context, apiKey *model.APIKey) error
	getByHashFunc     func(ctx context.Context, keyHash string) (*model.APIKey, error)
	getByIDFunc       func(ctx context.Context, id string) (*model.APIKey, error)
	listByUserFunc    func(ctx context.Context, userID string, filter repository.APIKeyFilter) ([]*model.APIKey, int64, error)
	updateFunc        func(ctx context.Context, apiKey *model.APIKey) error
	updateLastUsedFunc func(ctx context.Context, id string) error
}

func (m *mockAPIKeyRepository) Create(ctx context.Context, apiKey *model.APIKey) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, apiKey)
	}
	return nil
}

func (m *mockAPIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	if m.getByHashFunc != nil {
		return m.getByHashFunc(ctx, keyHash)
	}
	return nil, nil
}

func (m *mockAPIKeyRepository) GetByID(ctx context.Context, id string) (*model.APIKey, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockAPIKeyRepository) ListByUser(ctx context.Context, userID string, filter repository.APIKeyFilter) ([]*model.APIKey, int64, error) {
	if m.listByUserFunc != nil {
		return m.listByUserFunc(ctx, userID, filter)
	}
	return []*model.APIKey{}, 0, nil
}

func (m *mockAPIKeyRepository) Update(ctx context.Context, apiKey *model.APIKey) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, apiKey)
	}
	return nil
}

func (m *mockAPIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	if m.updateLastUsedFunc != nil {
		return m.updateLastUsedFunc(ctx, id)
	}
	return nil
}

func TestAPIKeyService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		repo := &mockAPIKeyRepository{}
		svc := NewAPIKeyService(repo)

		apiKey, rawKey, err := svc.Create(ctx, "user-123", &CreateAPIKeyRequest{
			Name: "Test Key",
		})

		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if apiKey == nil {
			t.Fatal("Expected apiKey, got nil")
		}
		if rawKey == "" {
			t.Error("Expected raw key to be returned")
		}
		if len(rawKey) < 10 {
			t.Errorf("Raw key too short: %s", rawKey)
		}
		if apiKey.UserID != "user-123" {
			t.Errorf("Expected UserID='user-123', got '%s'", apiKey.UserID)
		}
		if apiKey.Name != "Test Key" {
			t.Errorf("Expected Name='Test Key', got '%s'", apiKey.Name)
		}
		if apiKey.Status != model.APIKeyStatusActive {
			t.Errorf("Expected Status=active, got '%s'", apiKey.Status)
		}
	})

	t.Run("create with expiry", func(t *testing.T) {
		repo := &mockAPIKeyRepository{}
		svc := NewAPIKeyService(repo)
		expiresIn := 24

		apiKey, _, err := svc.Create(ctx, "user-123", &CreateAPIKeyRequest{
			Name:      "Expiring Key",
			ExpiresIn: &expiresIn,
		})

		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if apiKey.ExpiresAt == nil {
			t.Error("Expected ExpiresAt to be set")
		}
		if !apiKey.ExpiresAt.After(time.Now()) {
			t.Error("ExpiresAt should be in the future")
		}
	})

	t.Run("create with empty name", func(t *testing.T) {
		repo := &mockAPIKeyRepository{}
		svc := NewAPIKeyService(repo)

		_, _, err := svc.Create(ctx, "user-123", &CreateAPIKeyRequest{Name: ""})
		if err == nil {
			t.Error("Expected error for empty name")
		}
		if appErr, ok := err.(*errors.AppError); ok {
			if appErr.HTTPStatus != 400 {
				t.Errorf("Expected 400, got %d", appErr.HTTPStatus)
			}
		}
	})

	t.Run("create repo error", func(t *testing.T) {
		repo := &mockAPIKeyRepository{
			createFunc: func(ctx context.Context, apiKey *model.APIKey) error {
				return errors.NewInternalError("db error")
			},
		}
		svc := NewAPIKeyService(repo)

		_, _, err := svc.Create(ctx, "user-123", &CreateAPIKeyRequest{Name: "Key"})
		if err == nil {
			t.Error("Expected error from repo")
		}
	})
}

func TestAPIKeyService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := &mockAPIKeyRepository{
			getByIDFunc: func(ctx context.Context, id string) (*model.APIKey, error) {
				return &model.APIKey{ID: "key-123", UserID: "user-123", Name: "Test"}, nil
			},
		}
		svc := NewAPIKeyService(repo)

		apiKey, err := svc.GetByID(ctx, "user-123", "key-123")
		if err != nil {
			t.Errorf("GetByID returned error: %v", err)
		}
		if apiKey.Name != "Test" {
			t.Errorf("Expected Name='Test', got '%s'", apiKey.Name)
		}
	})

	t.Run("wrong user", func(t *testing.T) {
		repo := &mockAPIKeyRepository{
			getByIDFunc: func(ctx context.Context, id string) (*model.APIKey, error) {
				return &model.APIKey{ID: "key-123", UserID: "user-456"}, nil
			},
		}
		svc := NewAPIKeyService(repo)

		_, err := svc.GetByID(ctx, "user-123", "key-123")
		if err == nil {
			t.Error("Expected error for wrong user")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockAPIKeyRepository{
			getByIDFunc: func(ctx context.Context, id string) (*model.APIKey, error) {
				return nil, errors.NewNotFoundError("not found")
			},
		}
		svc := NewAPIKeyService(repo)

		_, err := svc.GetByID(ctx, "user-123", "nonexistent")
		if err == nil {
			t.Error("Expected error for not found")
		}
	})
}

func TestAPIKeyService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns keys", func(t *testing.T) {
		repo := &mockAPIKeyRepository{
			listByUserFunc: func(ctx context.Context, userID string, filter repository.APIKeyFilter) ([]*model.APIKey, int64, error) {
				return []*model.APIKey{{ID: "key-1"}}, 1, nil
			},
		}
		svc := NewAPIKeyService(repo)

		keys, total, err := svc.List(ctx, "user-123", &APIKeyFilter{PageSize: 10})
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if total != 1 {
			t.Errorf("Expected total=1, got %d", total)
		}
		if len(keys) != 1 {
			t.Errorf("Expected 1 key, got %d", len(keys))
		}
	})
}

func TestAPIKeyService_Revoke(t *testing.T) {
	ctx := context.Background()

	t.Run("successful revoke", func(t *testing.T) {
		repo := &mockAPIKeyRepository{
			getByIDFunc: func(ctx context.Context, id string) (*model.APIKey, error) {
				return &model.APIKey{ID: "key-123", UserID: "user-123", Status: model.APIKeyStatusActive}, nil
			},
			updateFunc: func(ctx context.Context, apiKey *model.APIKey) error {
				if apiKey.Status != model.APIKeyStatusRevoked {
					t.Errorf("Expected status revoked, got %s", apiKey.Status)
				}
				return nil
			},
		}
		svc := NewAPIKeyService(repo)

		err := svc.Revoke(ctx, "user-123", "key-123")
		if err != nil {
			t.Errorf("Revoke returned error: %v", err)
		}
	})

	t.Run("wrong user", func(t *testing.T) {
		repo := &mockAPIKeyRepository{
			getByIDFunc: func(ctx context.Context, id string) (*model.APIKey, error) {
				return &model.APIKey{ID: "key-123", UserID: "user-456"}, nil
			},
		}
		svc := NewAPIKeyService(repo)

		err := svc.Revoke(ctx, "user-123", "key-123")
		if err == nil {
			t.Error("Expected error for wrong user")
		}
	})

	t.Run("already revoked", func(t *testing.T) {
		repo := &mockAPIKeyRepository{
			getByIDFunc: func(ctx context.Context, id string) (*model.APIKey, error) {
				return &model.APIKey{ID: "key-123", UserID: "user-123", Status: model.APIKeyStatusRevoked}, nil
			},
		}
		svc := NewAPIKeyService(repo)

		err := svc.Revoke(ctx, "user-123", "key-123")
		if err == nil {
			t.Error("Expected error for already revoked key")
		}
	})
}

func TestAPIKeyService_Validate(t *testing.T) {
	ctx := context.Background()

	t.Run("valid key", func(t *testing.T) {
		repo := &mockAPIKeyRepository{
			getByHashFunc: func(ctx context.Context, keyHash string) (*model.APIKey, error) {
				return &model.APIKey{ID: "key-123", UserID: "user-123", Status: model.APIKeyStatusActive}, nil
			},
			updateLastUsedFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}
		svc := NewAPIKeyService(repo)

		apiKey, err := svc.Validate(ctx, "ak_valid_token")
		if err != nil {
			t.Errorf("Validate returned error: %v", err)
		}
		if apiKey.ID != "key-123" {
			t.Errorf("Expected ID='key-123', got '%s'", apiKey.ID)
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		repo := &mockAPIKeyRepository{
			getByHashFunc: func(ctx context.Context, keyHash string) (*model.APIKey, error) {
				return nil, nil
			},
		}
		svc := NewAPIKeyService(repo)

		_, err := svc.Validate(ctx, "bad_token")
		if err == nil {
			t.Error("Expected error for invalid key")
		}
	})

	t.Run("expired key", func(t *testing.T) {
		past := time.Now().Add(-24 * time.Hour)
		repo := &mockAPIKeyRepository{
			getByHashFunc: func(ctx context.Context, keyHash string) (*model.APIKey, error) {
				return &model.APIKey{ID: "key-123", Status: model.APIKeyStatusActive, ExpiresAt: &past}, nil
			},
		}
		svc := NewAPIKeyService(repo)

		_, err := svc.Validate(ctx, "expired_token")
		if err == nil {
			t.Error("Expected error for expired key")
		}
	})

	t.Run("revoked key", func(t *testing.T) {
		repo := &mockAPIKeyRepository{
			getByHashFunc: func(ctx context.Context, keyHash string) (*model.APIKey, error) {
				return &model.APIKey{ID: "key-123", Status: model.APIKeyStatusRevoked}, nil
			},
		}
		svc := NewAPIKeyService(repo)

		_, err := svc.Validate(ctx, "revoked_token")
		if err == nil {
			t.Error("Expected error for revoked key")
		}
	})
}

func TestGenerateSecret(t *testing.T) {
	secret, err := generateSecret()
	if err != nil {
		t.Errorf("generateSecret returned error: %v", err)
	}
	if len(secret) != 64 {
		t.Errorf("Expected 64-char hex secret, got %d chars", len(secret))
	}

	secret2, _ := generateSecret()
	if secret == secret2 {
		t.Error("Two generated secrets should be different")
	}
}
