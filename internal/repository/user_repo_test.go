package repository

import (
	"context"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
)

func TestNewUserRepository(t *testing.T) {
	repo := NewUserRepository(nil)
	if repo == nil {
		t.Error("NewUserRepository should return non-nil")
	}
}

func TestUserRepository_Interface(t *testing.T) {
	var _ UserRepository = NewUserRepository(nil)
}

func TestUserModel_Methods(t *testing.T) {
	t.Run("IsAdmin", func(t *testing.T) {
		admin := &model.User{Role: model.UserRoleAdmin}
		if !admin.IsAdmin() {
			t.Error("Admin user should be admin")
		}
		dev := &model.User{Role: model.UserRoleDeveloper}
		if dev.IsAdmin() {
			t.Error("Developer should not be admin")
		}
	})

	t.Run("IsOperator", func(t *testing.T) {
		op := &model.User{Role: model.UserRoleOperator}
		if !op.IsOperator() {
			t.Error("Operator should be operator")
		}
	})

	t.Run("IsActive", func(t *testing.T) {
		active := &model.User{Status: model.UserStatusActive}
		if !active.IsActive() {
			t.Error("Active user should be active")
		}
		disabled := &model.User{Status: model.UserStatusDisabled}
		if disabled.IsActive() {
			t.Error("Disabled user should not be active")
		}
	})
}

func TestUserRepository_ErrorTypes(t *testing.T) {
	err := errors.NewNotFoundError("user not found")
	if err.HTTPStatus != 404 {
		t.Errorf("Expected 404, got %d", err.HTTPStatus)
	}
}

func TestUserRepository_Context(t *testing.T) {
	ctx := context.Background()
	_ = ctx // verify context is available
}
