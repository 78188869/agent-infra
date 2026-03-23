// Package model provides database models for the application.
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel contains common fields for all database models.
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:char(36);primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate is a GORM hook that generates a UUID before creating a record.
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// generateUUID generates a new UUID string.
func generateUUID() string {
	return uuid.New().String()
}

// AllModels returns all models for auto migration.
func AllModels() []interface{} {
	return []interface{}{
		&Tenant{},
		&User{},
		&APIKey{},
		&Template{},
		&Task{},
		&ExecutionLog{},
		&Intervention{},
		&Capability{},
		&Provider{},
		&UserProviderDefault{},
	}
}
