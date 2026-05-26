package repository

import (
	"context"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CredentialRepository defines the interface for credential data access.
type CredentialRepository interface {
	Store(ctx context.Context, cred *model.Credential) error
	GetByUserAndType(ctx context.Context, userID, credType string) (*model.Credential, error)
	ListByUser(ctx context.Context, userID string) ([]*model.Credential, error)
	Delete(ctx context.Context, userID, credType string) error
}

type credentialRepository struct {
	db *gorm.DB
}

// NewCredentialRepository creates a new CredentialRepository instance.
func NewCredentialRepository(db *gorm.DB) CredentialRepository {
	return &credentialRepository{db: db}
}

func (r *credentialRepository) Store(ctx context.Context, cred *model.Credential) error {
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "type"}},
		DoUpdates: clause.AssignmentColumns([]string{"encrypted", "updated_at"}),
	}).Create(cred)

	if result.Error != nil {
		return errors.NewInternalError("failed to store credential: " + result.Error.Error())
	}
	return nil
}

func (r *credentialRepository) GetByUserAndType(ctx context.Context, userID, credType string) (*model.Credential, error) {
	var cred model.Credential
	if err := r.db.WithContext(ctx).Where("user_id = ? AND type = ?", userID, credType).First(&cred).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.NewInternalError("failed to get credential: " + err.Error())
	}
	return &cred, nil
}

func (r *credentialRepository) ListByUser(ctx context.Context, userID string) ([]*model.Credential, error) {
	var creds []*model.Credential
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&creds).Error; err != nil {
		return nil, errors.NewInternalError("failed to list credentials: " + err.Error())
	}
	return creds, nil
}

func (r *credentialRepository) Delete(ctx context.Context, userID, credType string) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND type = ?", userID, credType).
		Delete(&model.Credential{})

	if result.Error != nil {
		return errors.NewInternalError("failed to delete credential: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("credential not found")
	}
	return nil
}
