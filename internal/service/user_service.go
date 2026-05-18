package service

import (
	"context"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
)

// UserService defines the interface for user business operations.
type UserService interface {
	GetByID(ctx context.Context, id string) (*model.User, error)
}

type userService struct {
	repo repository.UserRepository
}

// NewUserService creates a new UserService instance.
func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) GetByID(ctx context.Context, id string) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !user.IsActive() {
		return nil, errors.NewBadRequestError("user account is disabled")
	}
	return user, nil
}
