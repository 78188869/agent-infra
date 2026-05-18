package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
)

const (
	apiKeyPrefix = "ak"
	secretLength = 32
)

// CreateAPIKeyRequest represents the request to create a new API key.
type CreateAPIKeyRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ExpiresIn   *int   `json:"expires_in"`
}

// APIKeyFilter represents filtering options for listing API keys.
type APIKeyFilter struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   string `form:"status"`
}

// APIKeyService defines the interface for API key business operations.
type APIKeyService interface {
	Create(ctx context.Context, userID string, req *CreateAPIKeyRequest) (*model.APIKey, string, error)
	GetByID(ctx context.Context, userID string, keyID string) (*model.APIKey, error)
	List(ctx context.Context, userID string, filter *APIKeyFilter) ([]*model.APIKey, int64, error)
	Revoke(ctx context.Context, userID string, keyID string) error
	Validate(ctx context.Context, rawKey string) (*model.APIKey, error)
}

type apiKeyService struct {
	repo repository.APIKeyRepository
}

// NewAPIKeyService creates a new APIKeyService instance.
func NewAPIKeyService(repo repository.APIKeyRepository) APIKeyService {
	return &apiKeyService{repo: repo}
}

func (s *apiKeyService) Create(ctx context.Context, userID string, req *CreateAPIKeyRequest) (*model.APIKey, string, error) {
	if req.Name == "" {
		return nil, "", errors.NewBadRequestError("api key name is required")
	}

	secret, err := generateSecret()
	if err != nil {
		return nil, "", errors.NewInternalError("failed to generate api key: " + err.Error())
	}

	rawKey := fmt.Sprintf("%s_%s", apiKeyPrefix, secret)
	keyHash := model.HashKey(rawKey)
	keyPrefix := model.ExtractPrefix(rawKey)

	apiKey := &model.APIKey{
		UserID:      userID,
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		Name:        req.Name,
		Description: req.Description,
		Status:      model.APIKeyStatusActive,
	}

	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Hour)
		apiKey.ExpiresAt = &expiresAt
	}

	if err := s.repo.Create(ctx, apiKey); err != nil {
		return nil, "", err
	}

	return apiKey, rawKey, nil
}

func (s *apiKeyService) GetByID(ctx context.Context, userID string, keyID string) (*model.APIKey, error) {
	apiKey, err := s.repo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if apiKey.UserID != userID {
		return nil, errors.NewNotFoundError("api key not found")
	}
	return apiKey, nil
}

func (s *apiKeyService) List(ctx context.Context, userID string, filter *APIKeyFilter) ([]*model.APIKey, int64, error) {
	repoFilter := repository.APIKeyFilter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Status:   filter.Status,
	}
	return s.repo.ListByUser(ctx, userID, repoFilter)
}

func (s *apiKeyService) Revoke(ctx context.Context, userID string, keyID string) error {
	apiKey, err := s.repo.GetByID(ctx, keyID)
	if err != nil {
		return err
	}
	if apiKey.UserID != userID {
		return errors.NewNotFoundError("api key not found")
	}
	if apiKey.Status == model.APIKeyStatusRevoked {
		return errors.NewBadRequestError("api key is already revoked")
	}

	apiKey.Status = model.APIKeyStatusRevoked
	return s.repo.Update(ctx, apiKey)
}

func (s *apiKeyService) Validate(ctx context.Context, rawKey string) (*model.APIKey, error) {
	keyHash := model.HashKey(rawKey)

	apiKey, err := s.repo.GetByHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, errors.NewBadRequestError("invalid api key")
	}
	if !apiKey.IsActive() {
		return nil, errors.NewBadRequestError("api key is inactive or expired")
	}

	go func() {
		_ = s.repo.UpdateLastUsed(context.Background(), apiKey.ID)
	}()

	return apiKey, nil
}

func generateSecret() (string, error) {
	bytes := make([]byte, secretLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
