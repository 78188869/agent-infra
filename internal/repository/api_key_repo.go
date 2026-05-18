package repository

import (
	"context"
	"time"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"gorm.io/gorm"
)

// APIKeyFilter represents filtering options for listing API keys.
type APIKeyFilter struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   string `form:"status"`
}

// SetDefaults sets default values for the filter.
func (f *APIKeyFilter) SetDefaults() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 10
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
}

// Offset returns the calculated offset for pagination.
func (f *APIKeyFilter) Offset() int {
	return (f.Page - 1) * f.PageSize
}

// APIKeyRepository defines the interface for API key data access.
type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *model.APIKey) error
	GetByHash(ctx context.Context, keyHash string) (*model.APIKey, error)
	GetByID(ctx context.Context, id string) (*model.APIKey, error)
	ListByUser(ctx context.Context, userID string, filter APIKeyFilter) ([]*model.APIKey, int64, error)
	Update(ctx context.Context, apiKey *model.APIKey) error
	UpdateLastUsed(ctx context.Context, id string) error
}

type apiKeyRepository struct {
	db *gorm.DB
}

// NewAPIKeyRepository creates a new APIKeyRepository instance.
func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) Create(ctx context.Context, apiKey *model.APIKey) error {
	if err := r.db.WithContext(ctx).Create(apiKey).Error; err != nil {
		return errors.NewInternalError("failed to create api key: " + err.Error())
	}
	return nil
}

func (r *apiKeyRepository) GetByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	var apiKey model.APIKey
	if err := r.db.WithContext(ctx).Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.NewInternalError("failed to get api key by hash: " + err.Error())
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) GetByID(ctx context.Context, id string) (*model.APIKey, error) {
	var apiKey model.APIKey
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("api key not found")
		}
		return nil, errors.NewInternalError("failed to get api key: " + err.Error())
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) ListByUser(ctx context.Context, userID string, filter APIKeyFilter) ([]*model.APIKey, int64, error) {
	filter.SetDefaults()

	var keys []*model.APIKey
	var total int64

	query := r.db.WithContext(ctx).Model(&model.APIKey{}).Where("user_id = ?", userID)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to count api keys: " + err.Error())
	}

	if err := query.Order("created_at DESC").Offset(filter.Offset()).Limit(filter.PageSize).Find(&keys).Error; err != nil {
		return nil, 0, errors.NewInternalError("failed to list api keys: " + err.Error())
	}

	return keys, total, nil
}

func (r *apiKeyRepository) Update(ctx context.Context, apiKey *model.APIKey) error {
	result := r.db.WithContext(ctx).Save(apiKey)
	if result.Error != nil {
		return errors.NewInternalError("failed to update api key: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("api key not found")
	}
	return nil
}

func (r *apiKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&model.APIKey{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_used_at": now,
			"usage_count":  gorm.Expr("usage_count + 1"),
		})
	if result.Error != nil {
		return errors.NewInternalError("failed to update api key usage: " + result.Error.Error())
	}
	return nil
}
