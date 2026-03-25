// Package model provides database models for the application.
package model

import (
	"gorm.io/datatypes"
)

// InterventionAction represents the type of intervention action.
type InterventionAction string

const (
	InterventionActionPause   InterventionAction = "pause"
	InterventionActionResume  InterventionAction = "resume"
	InterventionActionCancel  InterventionAction = "cancel"
	InterventionActionInject  InterventionAction = "inject"
	InterventionActionModify  InterventionAction = "modify"
)

// InterventionStatus represents the status of an intervention.
type InterventionStatus string

const (
	InterventionStatusPending InterventionStatus = "pending"
	InterventionStatusApplied InterventionStatus = "applied"
	InterventionStatusFailed  InterventionStatus = "failed"
)

// Intervention represents a human intervention record.
// Interventions are manual actions taken on running tasks.
type Intervention struct {
	BaseModel
	TaskID     string             `gorm:"type:varchar(36);not null;index" json:"task_id"`
	OperatorID string             `gorm:"type:varchar(36);not null;index" json:"operator_id"`

	// Intervention Information
	Action  InterventionAction `gorm:"type:enum('pause','resume','cancel','inject','modify');not null" json:"action"`
	Content datatypes.JSON     `gorm:"type:json" json:"content"`
	Reason  string             `gorm:"type:varchar(512)" json:"reason"`

	// Result
	Result datatypes.JSON     `gorm:"type:json" json:"result"`
	Status InterventionStatus `gorm:"type:enum('pending','applied','failed');default:'pending'" json:"status"`

	// Relations
	Task     *Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Operator *User `gorm:"foreignKey:OperatorID" json:"operator,omitempty"`
}

// TableName returns the table name for Intervention.
func (Intervention) TableName() string {
	return "interventions"
}

// IsPending checks if the intervention is pending.
func (i *Intervention) IsPending() bool {
	return i.Status == InterventionStatusPending
}

// IsApplied checks if the intervention has been applied.
func (i *Intervention) IsApplied() bool {
	return i.Status == InterventionStatusApplied
}

// IsFailed checks if the intervention has failed.
func (i *Intervention) IsFailed() bool {
	return i.Status == InterventionStatusFailed
}

// InterventionContent represents the content of an intervention.
type InterventionContent struct {
	// For inject action
	Instruction string `json:"instruction,omitempty"`

	// For modify action
	Modifications map[string]interface{} `json:"modifications,omitempty"`

	// Additional context
	Context string `json:"context,omitempty"`
}

// InterventionResult represents the result of an intervention.
type InterventionResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
}
