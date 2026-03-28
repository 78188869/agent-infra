package model

import (
	"gorm.io/datatypes"
)

// Task status constants.
const (
	TaskStatusPending         = "pending"
	TaskStatusScheduled       = "scheduled"
	TaskStatusRunning         = "running"
	TaskStatusPaused          = "paused"
	TaskStatusWaitingApproval = "waiting_approval"
	TaskStatusRetrying        = "retrying"
	TaskStatusSucceeded       = "succeeded"
	TaskStatusFailed          = "failed"
	TaskStatusCancelled       = "cancelled"
)

// Task priority constants.
const (
	TaskPriorityHigh   = "high"
	TaskPriorityNormal = "normal"
	TaskPriorityLow    = "low"
)

// Task represents a task in the system with execution parameters and results.
type Task struct {
	BaseModel
	TenantID     string         `gorm:"type:varchar(36);index:idx_tenant_status" json:"tenant_id"`
	TemplateID   *string        `gorm:"type:varchar(36);index" json:"template_id,omitempty"`
	CreatorID    string         `gorm:"type:varchar(36);index" json:"creator_id"`
	ProviderID   string         `gorm:"type:varchar(36);not null" json:"provider_id"`
	Name         string         `gorm:"type:varchar(256)" json:"name"`
	Status       string         `gorm:"type:varchar(32);default:'pending';index:idx_tenant_status" json:"status"`
	Priority     string         `gorm:"type:varchar(20);default:'normal'" json:"priority"`
	Params       datatypes.JSON `gorm:"type:json" json:"params,omitempty"`
	Description  string         `gorm:"type:text" json:"description,omitempty"`
	ErrorMessage string         `gorm:"type:text" json:"error_message,omitempty"`
	Result       datatypes.JSON `gorm:"type:json" json:"result,omitempty"`
}

// TableName returns the table name for the Task model.
func (Task) TableName() string {
	return "tasks"
}
