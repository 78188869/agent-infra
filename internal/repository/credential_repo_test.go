package repository

import (
	"context"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCredentialTestDB(t *testing.T) *gorm.DB {
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

func TestCredentialRepository_Interface(t *testing.T) {
	var _ CredentialRepository = NewCredentialRepository(nil)
}

func TestCredentialRepository_Store(t *testing.T) {
	db := setupCredentialTestDB(t)
	repo := NewCredentialRepository(db)
	ctx := context.Background()

	cred := &model.Credential{
		UserID:    "user-1",
		Type:      model.CredentialTypeGitToken,
		Encrypted: "enc-value-1",
	}

	if err := repo.Store(ctx, cred); err != nil {
		t.Fatalf("Store() error = %v", err)
	}
}

func TestCredentialRepository_Store_Upsert(t *testing.T) {
	db := setupCredentialTestDB(t)
	repo := NewCredentialRepository(db)
	ctx := context.Background()

	cred := &model.Credential{
		UserID:    "user-1",
		Type:      model.CredentialTypeGitToken,
		Encrypted: "enc-value-1",
	}
	if err := repo.Store(ctx, cred); err != nil {
		t.Fatalf("first Store() error = %v", err)
	}

	cred2 := &model.Credential{
		UserID:    "user-1",
		Type:      model.CredentialTypeGitToken,
		Encrypted: "enc-value-2",
	}
	if err := repo.Store(ctx, cred2); err != nil {
		t.Fatalf("upsert Store() error = %v", err)
	}

	result, err := repo.GetByUserAndType(ctx, "user-1", model.CredentialTypeGitToken)
	if err != nil {
		t.Fatalf("GetByUserAndType() error = %v", err)
	}
	if result.Encrypted != "enc-value-2" {
		t.Errorf("after upsert, Encrypted = %q, want %q", result.Encrypted, "enc-value-2")
	}
}

func TestCredentialRepository_GetByUserAndType(t *testing.T) {
	db := setupCredentialTestDB(t)
	repo := NewCredentialRepository(db)
	ctx := context.Background()

	t.Run("not found returns nil", func(t *testing.T) {
		result, err := repo.GetByUserAndType(ctx, "user-1", model.CredentialTypeGitToken)
		if err != nil {
			t.Fatalf("GetByUserAndType() error = %v", err)
		}
		if result != nil {
			t.Error("expected nil for not found")
		}
	})

	t.Run("found returns credential", func(t *testing.T) {
		cred := &model.Credential{
			UserID:    "user-1",
			Type:      model.CredentialTypeGitToken,
			Encrypted: "enc-value",
		}
		repo.Store(ctx, cred)

		result, err := repo.GetByUserAndType(ctx, "user-1", model.CredentialTypeGitToken)
		if err != nil {
			t.Fatalf("GetByUserAndType() error = %v", err)
		}
		if result == nil {
			t.Fatal("expected credential, got nil")
		}
		if result.Encrypted != "enc-value" {
			t.Errorf("Encrypted = %q, want %q", result.Encrypted, "enc-value")
		}
	})
}

func TestCredentialRepository_ListByUser(t *testing.T) {
	db := setupCredentialTestDB(t)
	repo := NewCredentialRepository(db)
	ctx := context.Background()

	repo.Store(ctx, &model.Credential{UserID: "user-1", Type: model.CredentialTypeGitToken, Encrypted: "a"})
	repo.Store(ctx, &model.Credential{UserID: "user-1", Type: model.CredentialTypeDevOpsToken, Encrypted: "b"})
	repo.Store(ctx, &model.Credential{UserID: "user-2", Type: model.CredentialTypeGitToken, Encrypted: "c"})

	creds, err := repo.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(creds) != 2 {
		t.Errorf("ListByUser() returned %d creds, want 2", len(creds))
	}
}

func TestCredentialRepository_Delete(t *testing.T) {
	db := setupCredentialTestDB(t)
	repo := NewCredentialRepository(db)
	ctx := context.Background()

	repo.Store(ctx, &model.Credential{UserID: "user-1", Type: model.CredentialTypeGitToken, Encrypted: "a"})

	t.Run("successful delete", func(t *testing.T) {
		if err := repo.Delete(ctx, "user-1", model.CredentialTypeGitToken); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		result, _ := repo.GetByUserAndType(ctx, "user-1", model.CredentialTypeGitToken)
		if result != nil {
			t.Error("credential should be deleted")
		}
	})

	t.Run("delete non-existent returns not found", func(t *testing.T) {
		err := repo.Delete(ctx, "user-1", model.CredentialTypeGitToken)
		if err == nil {
			t.Error("expected error for deleting non-existent credential")
		}
	})
}
