// Package model provides database models for the application.
package model

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
)

// APIKeyStatus represents the status of an API key.
type APIKeyStatus string

const (
	APIKeyStatusActive  APIKeyStatus = "active"
	APIKeyStatusRevoked APIKeyStatus = "revoked"
)

// APIKey represents an API key for authentication.
// API keys are used for programmatic access to the API.
type APIKey struct {
	ID          string       `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID      string       `gorm:"type:varchar(36);not null;index" json:"user_id"`
	KeyHash     string       `gorm:"type:varchar(128);not null" json:"-"` // SHA256 hash, not exposed in JSON
	KeyPrefix   string       `gorm:"type:varchar(8);not null;index" json:"key_prefix"`
	Name        string       `gorm:"type:varchar(64)" json:"name"`
	Description string       `gorm:"type:varchar(256)" json:"description"`

	// Validity and Usage
	ExpiresAt   *time.Time `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	UsageCount  int64      `gorm:"default:0" json:"usage_count"`

	// Status
	Status APIKeyStatus `gorm:"type:varchar(20);default:'active';index" json:"status"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName returns the table name for APIKey.
func (APIKey) TableName() string {
	return "api_keys"
}

// BeforeCreate is a GORM hook that generates a UUID before creating a record.
func (k *APIKey) BeforeCreate(tx *gorm.DB) error {
	if k.ID == "" {
		k.ID = generateUUID()
	}
	return nil
}

// IsActive checks if the API key is active and not expired.
func (k *APIKey) IsActive() bool {
	if k.Status != APIKeyStatusActive {
		return false
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return false
	}
	return true
}

// IsExpired checks if the API key is expired.
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// HashKey generates a SHA256 hash of the given API key.
func HashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// ExtractPrefix extracts the prefix from an API key (first 8 characters).
func ExtractPrefix(key string) string {
	if len(key) < 8 {
		return key
	}
	return key[:8]
}
