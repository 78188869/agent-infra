package repository

import (
	"context"
	"testing"
	"time"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
)

// mockAPIKeyDB simulates a database for API key tests
type mockAPIKeyDB struct {
	keys     map[string]*model.APIKey
	createFn func(ctx context.Context, apiKey *model.APIKey) error
}

func newMockAPIKeyDB() *mockAPIKeyDB {
	return &mockAPIKeyDB{
		keys: make(map[string]*model.APIKey),
	}
}

func (m *mockAPIKeyDB) findByHash(hash string) *model.APIKey {
	for _, k := range m.keys {
		if k.KeyHash == hash {
			return k
		}
	}
	return nil
}

// Tests

func TestAPIKeyRepository_Create(t *testing.T) {
	t.Run("successful create", func(t *testing.T) {
		repo := NewAPIKeyRepository(nil)
		if repo == nil {
			t.Error("NewAPIKeyRepository should return non-nil")
		}
	})
}

func TestAPIKeyRepository_Interface(t *testing.T) {
	var _ APIKeyRepository = NewAPIKeyRepository(nil)
}

func TestAPIKeyFilter_SetDefaults(t *testing.T) {
	t.Run("sets default page and page_size", func(t *testing.T) {
		filter := APIKeyFilter{}
		filter.SetDefaults()
		if filter.Page != 1 {
			t.Errorf("Expected Page=1, got %d", filter.Page)
		}
		if filter.PageSize != 10 {
			t.Errorf("Expected PageSize=10, got %d", filter.PageSize)
		}
	})

	t.Run("respects valid values", func(t *testing.T) {
		filter := APIKeyFilter{Page: 5, PageSize: 20}
		filter.SetDefaults()
		if filter.Page != 5 {
			t.Errorf("Expected Page=5, got %d", filter.Page)
		}
		if filter.PageSize != 20 {
			t.Errorf("Expected PageSize=20, got %d", filter.PageSize)
		}
	})

	t.Run("caps page size at 100", func(t *testing.T) {
		filter := APIKeyFilter{Page: 1, PageSize: 200}
		filter.SetDefaults()
		if filter.PageSize != 100 {
			t.Errorf("Expected PageSize=100, got %d", filter.PageSize)
		}
	})
}

func TestAPIKeyFilter_Offset(t *testing.T) {
	tests := []struct {
		page     int
		pageSize int
		expected int
	}{
		{1, 10, 0},
		{2, 10, 10},
		{3, 20, 40},
	}
	for _, tt := range tests {
		filter := APIKeyFilter{Page: tt.page, PageSize: tt.pageSize}
		if got := filter.Offset(); got != tt.expected {
			t.Errorf("Offset(Page=%d, PageSize=%d) = %d, want %d",
				tt.page, tt.pageSize, got, tt.expected)
		}
	}
}

func TestAPIKeyModel_HashKey(t *testing.T) {
	hash1 := model.HashKey("test-key")
	hash2 := model.HashKey("test-key")
	if hash1 != hash2 {
		t.Error("HashKey should be deterministic")
	}
	if len(hash1) != 64 {
		t.Errorf("Expected 64-char hex hash, got %d chars", len(hash1))
	}

	hash3 := model.HashKey("different-key")
	if hash1 == hash3 {
		t.Error("Different inputs should produce different hashes")
	}
}

func TestAPIKeyModel_ExtractPrefix(t *testing.T) {
	prefix := model.ExtractPrefix("ak_abcdef123456")
	if prefix != "ak_abcde" {
		t.Errorf("Expected 'ak_abcde', got '%s'", prefix)
	}

	short := model.ExtractPrefix("ab")
	if short != "ab" {
		t.Errorf("Expected 'ab' for short input, got '%s'", short)
	}
}

func TestAPIKeyModel_IsActive(t *testing.T) {
	t.Run("active key", func(t *testing.T) {
		key := &model.APIKey{Status: model.APIKeyStatusActive}
		if !key.IsActive() {
			t.Error("Active key should be active")
		}
	})

	t.Run("revoked key", func(t *testing.T) {
		key := &model.APIKey{Status: model.APIKeyStatusRevoked}
		if key.IsActive() {
			t.Error("Revoked key should not be active")
		}
	})

	t.Run("expired key", func(t *testing.T) {
		past := time.Now().Add(-24 * time.Hour)
		key := &model.APIKey{
			Status:    model.APIKeyStatusActive,
			ExpiresAt: &past,
		}
		if key.IsActive() {
			t.Error("Expired key should not be active")
		}
	})

	t.Run("not expired key", func(t *testing.T) {
		future := time.Now().Add(24 * time.Hour)
		key := &model.APIKey{
			Status:    model.APIKeyStatusActive,
			ExpiresAt: &future,
		}
		if !key.IsActive() {
			t.Error("Non-expired key should be active")
		}
	})
}

func TestAPIKeyRepository_ErrorTypes(t *testing.T) {
	err := errors.NewNotFoundError("api key not found")
	if err.HTTPStatus != 404 {
		t.Errorf("Expected 404, got %d", err.HTTPStatus)
	}

	internalErr := errors.NewInternalError("something failed")
	if internalErr.HTTPStatus != 500 {
		t.Errorf("Expected 500, got %d", internalErr.HTTPStatus)
	}
}
