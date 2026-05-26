package service

import (
	"context"
	"testing"

	"github.com/example/agent-infra/internal/config"
	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCredentialServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.Credential{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestEncryptor(t *testing.T) config.Encryptor {
	t.Helper()
	enc, err := config.NewAESEncryptor("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}
	return enc
}

func newTestCredentialService(t *testing.T) CredentialService {
	t.Helper()
	db := setupCredentialServiceTestDB(t)
	repo := repository.NewCredentialRepository(db)
	enc := newTestEncryptor(t)
	return NewCredentialService(repo, enc)
}

func TestCredentialService_Interface(t *testing.T) {
	var _ CredentialService = NewCredentialService(nil, nil)
}

func TestCredentialService_Store(t *testing.T) {
	svc := newTestCredentialService(t)
	ctx := context.Background()

	t.Run("valid store", func(t *testing.T) {
		info, err := svc.Store(ctx, "user-1", &StoreCredentialRequest{
			Type:  model.CredentialTypeGitToken,
			Value: "ghp_secret123",
		})
		if err != nil {
			t.Fatalf("Store() error = %v", err)
		}
		if info.Type != model.CredentialTypeGitToken {
			t.Errorf("Type = %q, want %q", info.Type, model.CredentialTypeGitToken)
		}
		if info.ID == "" {
			t.Error("ID should not be empty")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		_, err := svc.Store(ctx, "user-1", &StoreCredentialRequest{
			Type:  "invalid",
			Value: "secret",
		})
		if err == nil {
			t.Error("expected error for invalid type")
		}
	})

	t.Run("empty value", func(t *testing.T) {
		_, err := svc.Store(ctx, "user-1", &StoreCredentialRequest{
			Type:  model.CredentialTypeGitToken,
			Value: "",
		})
		if err == nil {
			t.Error("expected error for empty value")
		}
	})
}

func TestCredentialService_Get(t *testing.T) {
	svc := newTestCredentialService(t)
	ctx := context.Background()

	svc.Store(ctx, "user-1", &StoreCredentialRequest{
		Type:  model.CredentialTypeGitToken,
		Value: "ghp_secret123",
	})

	t.Run("get existing credential", func(t *testing.T) {
		value, err := svc.Get(ctx, "user-1", model.CredentialTypeGitToken)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if value != "ghp_secret123" {
			t.Errorf("Get() = %q, want %q", value, "ghp_secret123")
		}
	})

	t.Run("get non-existent credential", func(t *testing.T) {
		_, err := svc.Get(ctx, "user-1", model.CredentialTypeDevOpsToken)
		if err == nil {
			t.Error("expected error for non-existent credential")
		}
	})
}

func TestCredentialService_Delete(t *testing.T) {
	svc := newTestCredentialService(t)
	ctx := context.Background()

	svc.Store(ctx, "user-1", &StoreCredentialRequest{
		Type:  model.CredentialTypeGitToken,
		Value: "secret",
	})

	t.Run("delete existing", func(t *testing.T) {
		if err := svc.Delete(ctx, "user-1", model.CredentialTypeGitToken); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
	})

	t.Run("delete non-existent", func(t *testing.T) {
		err := svc.Delete(ctx, "user-1", model.CredentialTypeGitToken)
		if err == nil {
			t.Error("expected error for deleting non-existent credential")
		}
	})
}

func TestCredentialService_List(t *testing.T) {
	svc := newTestCredentialService(t)
	ctx := context.Background()

	svc.Store(ctx, "user-1", &StoreCredentialRequest{Type: model.CredentialTypeGitToken, Value: "a"})
	svc.Store(ctx, "user-1", &StoreCredentialRequest{Type: model.CredentialTypeDevOpsToken, Value: "b"})

	infos, err := svc.List(ctx, "user-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(infos) != 2 {
		t.Errorf("List() returned %d items, want 2", len(infos))
	}

	for _, info := range infos {
		if info.ID == "" {
			t.Error("credential ID should not be empty")
		}
		if info.Type != model.CredentialTypeGitToken && info.Type != model.CredentialTypeDevOpsToken {
			t.Errorf("unexpected type: %s", info.Type)
		}
	}
}

func TestCredentialService_BuildSandboxEnv(t *testing.T) {
	svc := newTestCredentialService(t)
	ctx := context.Background()

	svc.Store(ctx, "user-1", &StoreCredentialRequest{Type: model.CredentialTypeGitToken, Value: "ghp_token"})
	svc.Store(ctx, "user-1", &StoreCredentialRequest{Type: model.CredentialTypeDevOpsToken, Value: "devops_token"})

	env, err := svc.BuildSandboxEnv(ctx, "user-1")
	if err != nil {
		t.Fatalf("BuildSandboxEnv() error = %v", err)
	}

	if env["GIT_TOKEN"] != "ghp_token" {
		t.Errorf("GIT_TOKEN = %q, want %q", env["GIT_TOKEN"], "ghp_token")
	}
	if env["DEVOPS_TOKEN"] != "devops_token" {
		t.Errorf("DEVOPS_TOKEN = %q, want %q", env["DEVOPS_TOKEN"], "devops_token")
	}
}

func TestCredentialService_BuildSandboxEnv_Partial(t *testing.T) {
	svc := newTestCredentialService(t)
	ctx := context.Background()

	svc.Store(ctx, "user-1", &StoreCredentialRequest{Type: model.CredentialTypeGitToken, Value: "ghp_token"})

	env, err := svc.BuildSandboxEnv(ctx, "user-1")
	if err != nil {
		t.Fatalf("BuildSandboxEnv() error = %v", err)
	}

	if env["GIT_TOKEN"] != "ghp_token" {
		t.Errorf("GIT_TOKEN = %q, want %q", env["GIT_TOKEN"], "ghp_token")
	}
	if _, ok := env["DEVOPS_TOKEN"]; ok {
		t.Error("DEVOPS_TOKEN should not be present when not stored")
	}
}

func TestCredentialService_Upsert(t *testing.T) {
	svc := newTestCredentialService(t)
	ctx := context.Background()

	svc.Store(ctx, "user-1", &StoreCredentialRequest{Type: model.CredentialTypeGitToken, Value: "old-token"})
	svc.Store(ctx, "user-1", &StoreCredentialRequest{Type: model.CredentialTypeGitToken, Value: "new-token"})

	value, err := svc.Get(ctx, "user-1", model.CredentialTypeGitToken)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if value != "new-token" {
		t.Errorf("after upsert, Get() = %q, want %q", value, "new-token")
	}

	infos, _ := svc.List(ctx, "user-1")
	if len(infos) != 1 {
		t.Errorf("after upsert, List() returned %d items, want 1", len(infos))
	}
}
