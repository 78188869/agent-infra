// Package model provides database models for the application.
package model

import (
	"time"

	"gorm.io/gorm"
)

// UserProviderDefault represents a user's default provider setting.
// Users can set their own default provider for task execution.
type UserProviderDefault struct {
	ID         string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID     string    `gorm:"type:varchar(36);not null;uniqueIndex:uk_user" json:"user_id"`
	ProviderID string    `gorm:"type:varchar(36);not null" json:"provider_id"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	User     *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Provider *Provider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

// TableName returns the table name for UserProviderDefault.
func (UserProviderDefault) TableName() string {
	return "user_provider_defaults"
}

// BeforeCreate is a GORM hook that generates a UUID before creating a record.
func (u *UserProviderDefault) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = generateUUID()
	}
	return nil
}
