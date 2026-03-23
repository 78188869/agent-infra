// Package model provides database models for the application.
package model

import (
	"time"

	"gorm.io/gorm"
)

// UserRole represents the role of a user.
type UserRole string

const (
	UserRoleDeveloper UserRole = "developer"
	UserRoleAdmin     UserRole = "admin"
	UserRoleOperator  UserRole = "operator"
	UserRoleReviewer  UserRole = "reviewer"
)

// UserStatus represents the status of a user.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

// User represents a user in the system.
// Users belong to a tenant and have specific roles.
type User struct {
	ID          string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID    string     `gorm:"type:varchar(36);not null;index:idx_tenant;index:idx_tenant_username" json:"tenant_id"`
	Username    string     `gorm:"type:varchar(64);not null;index:idx_tenant_username" json:"username"`
	DisplayName string     `gorm:"type:varchar(128)" json:"display_name"`
	Email       string     `gorm:"type:varchar(128)" json:"email"`

	// Role and Status
	Role   UserRole   `gorm:"type:enum('developer','admin','operator','reviewer');default:'developer'" json:"role"`
	Status UserStatus `gorm:"type:enum('active','disabled');default:'active'" json:"status"`

	// Timestamps
	LastLoginAt *time.Time    `gorm:"type:timestamp" json:"last_login_at"`
	CreatedAt   time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Tenant    *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	APIKeys   []APIKey  `gorm:"foreignKey:UserID" json:"api_keys,omitempty"`
	Tasks     []Task    `gorm:"foreignKey:CreatorID" json:"tasks,omitempty"`
}

// TableName returns the table name for User.
func (User) TableName() string {
	return "users"
}

// BeforeCreate is a GORM hook that generates a UUID before creating a record.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = generateUUID()
	}
	return nil
}

// IsAdmin checks if the user has admin role.
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// IsOperator checks if the user has operator role.
func (u *User) IsOperator() bool {
	return u.Role == UserRoleOperator
}

// IsActive checks if the user is active.
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}
