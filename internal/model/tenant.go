// Package model provides database models for the application.
package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
)

// TenantStatus represents the status of a tenant.
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
)

// Tenant represents a tenant in the system.
// Tenants are the basic unit of resource isolation.
type Tenant struct {
	ID          string `gorm:"type:varchar(36);primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(128);not null" json:"name"`
	Description string `gorm:"type:varchar(512)" json:"description"`

	// Resource Quotas
	QuotaCPU               int   `gorm:"default:100" json:"quota_cpu"`                // CPU core limit
	QuotaMemory            int64 `gorm:"default:200" json:"quota_memory"`             // Memory limit in GB
	QuotaConcurrency       int   `gorm:"default:50" json:"quota_concurrency"`         // Max concurrent tasks
	QuotaDailyTasks        int   `gorm:"default:1000" json:"quota_daily_tasks"`       // Daily task limit
	QuotaMaxTokenPerTask   int64 `gorm:"default:500000" json:"quota_max_token_per_task"` // Max tokens per task

	// Status
	Status TenantStatus `gorm:"type:enum('active','suspended');default:'active'" json:"status"`

	// Timestamps
	CreatedAt int64          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt int64          `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Users      []User      `gorm:"foreignKey:TenantID" json:"users,omitempty"`
	Templates  []Template  `gorm:"foreignKey:TenantID" json:"templates,omitempty"`
	Tasks      []Task      `gorm:"foreignKey:TenantID" json:"tasks,omitempty"`
}

// TableName returns the table name for Tenant.
func (Tenant) TableName() string {
	return "tenants"
}

// BeforeCreate is a GORM hook that generates a UUID before creating a record.
func (t *Tenant) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = generateUUID()
	}
	return nil
}

// ResourceQuota represents tenant resource quota configuration.
type ResourceQuota struct {
	CPU               int   `json:"cpu"`
	Memory            int64 `json:"memory"`
	Concurrency       int   `json:"concurrency"`
	DailyTasks        int   `json:"daily_tasks"`
	MaxTokenPerTask   int64 `json:"max_token_per_task"`
}

// Value implements driver.Valuer for ResourceQuota.
func (r ResourceQuota) Value() (driver.Value, error) {
	return json.Marshal(r)
}

// Scan implements sql.Scanner for ResourceQuota.
func (r *ResourceQuota) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, r)
}

// QuotaUsage represents current quota usage for a tenant.
type QuotaUsage struct {
	CurrentConcurrency int   `json:"current_concurrency"`
	TodayTasks         int   `json:"today_tasks"`
	UsedMemory        int64 `json:"used_memory"`
	UsedCPU           int   `json:"used_cpu"`
}
