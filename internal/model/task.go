// Package model provides database models for the application.
package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TaskStatus represents the status of a task.
type TaskStatus string

const (
	TaskStatusPending          TaskStatus = "pending"
	TaskStatusScheduled        TaskStatus = "scheduled"
	TaskStatusRunning          TaskStatus = "running"
	TaskStatusPaused           TaskStatus = "paused"
	TaskStatusWaitingApproval  TaskStatus = "waiting_approval"
	TaskStatusRetrying         TaskStatus = "retrying"
	TaskStatusSucceeded        TaskStatus = "succeeded"
	TaskStatusFailed           TaskStatus = "failed"
	TaskStatusCancelled        TaskStatus = "cancelled"
)

// Priority represents the priority level of a task.
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityNormal Priority = "normal"
	PriorityLow    Priority = "low"
)

// Task represents a task in the system.
// A task is an execution instance created from a template.
type Task struct {
	ID           string  `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID     string  `gorm:"type:varchar(36);not null;index:idx_tenant_status" json:"tenant_id"`
	TemplateID   *string `gorm:"type:varchar(36);index" json:"template_id"`
	CreatorID    string  `gorm:"type:varchar(36);not null;index" json:"creator_id"`
	ParentTaskID *string `gorm:"type:varchar(36)" json:"parent_task_id"`

	// Task Information
	Name        string `gorm:"type:varchar(256)" json:"name"`
	Description string `gorm:"type:text" json:"description"`

	// Provider Configuration
	ProviderID string `gorm:"type:varchar(36);not null" json:"provider_id"`

	// Status and Progress
	Status        TaskStatus `gorm:"type:enum('pending','scheduled','running','paused','waiting_approval','retrying','succeeded','failed','cancelled');default:'pending';index:idx_tenant_status;index:idx_status_created" json:"status"`
	Progress      int        `gorm:"default:0" json:"progress"` // 0-100
	CurrentStage  string     `gorm:"type:varchar(64)" json:"current_stage"`

	// Configuration and Parameters
	Params       datatypes.JSON `gorm:"type:json" json:"params"`
	ResolvedSpec string         `gorm:"type:mediumtext" json:"resolved_spec"`
	Priority     Priority       `gorm:"type:enum('high','normal','low');default:'normal'" json:"priority"`

	// Execution Information
	PodName    string `gorm:"type:varchar(128)" json:"pod_name"`
	SandboxID  string `gorm:"type:varchar(64)" json:"sandbox_id"`
	RetryCount int    `gorm:"default:0" json:"retry_count"`
	MaxRetries int    `gorm:"default:3" json:"max_retries"`

	// Results and Metrics
	Result        datatypes.JSON `gorm:"type:json" json:"result"`
	ErrorMessage  string         `gorm:"type:text" json:"error_message"`
	ErrorCode     string         `gorm:"type:varchar(32)" json:"error_code"`
	Metrics       datatypes.JSON `gorm:"type:json" json:"metrics"`

	// Timestamps
	ScheduledAt *time.Time     `gorm:"type:timestamp;index" json:"scheduled_at"`
	StartedAt   *time.Time     `gorm:"type:timestamp" json:"started_at"`
	FinishedAt  *time.Time     `gorm:"type:timestamp" json:"finished_at"`
	CreatedAt   time.Time      `gorm:"autoCreateTime;index:idx_status_created" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Tenant      *Tenant       `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Template    *Template     `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	Creator     *User         `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
	Provider    *Provider     `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
	Logs        []ExecutionLog `gorm:"foreignKey:TaskID" json:"logs,omitempty"`
	Interventions []Intervention `gorm:"foreignKey:TaskID" json:"interventions,omitempty"`
}

// TableName returns the table name for Task.
func (Task) TableName() string {
	return "tasks"
}

// BeforeCreate is a GORM hook that generates a UUID before creating a record.
func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = generateUUID()
	}
	return nil
}

// IsTerminal checks if the task is in a terminal state.
func (t *Task) IsTerminal() bool {
	return t.Status == TaskStatusSucceeded ||
		t.Status == TaskStatusFailed ||
		t.Status == TaskStatusCancelled
}

// IsRunning checks if the task is currently running.
func (t *Task) IsRunning() bool {
	return t.Status == TaskStatusRunning
}

// CanPause checks if the task can be paused.
func (t *Task) CanPause() bool {
	return t.Status == TaskStatusRunning
}

// CanResume checks if the task can be resumed.
func (t *Task) CanResume() bool {
	return t.Status == TaskStatusPaused
}

// CanCancel checks if the task can be cancelled.
func (t *Task) CanCancel() bool {
	return t.Status == TaskStatusPending ||
		t.Status == TaskStatusScheduled ||
		t.Status == TaskStatusRunning ||
		t.Status == TaskStatusPaused
}

// CanInject checks if instructions can be injected into the task.
func (t *Task) CanInject() bool {
	return t.Status == TaskStatusRunning
}

// CanRetry checks if the task can be retried.
func (t *Task) CanRetry() bool {
	return t.Status == TaskStatusFailed && t.RetryCount < t.MaxRetries
}

// TaskResult represents the result of a task execution.
type TaskResult struct {
	Status       string                 `json:"status"`
	Output       string                 `json:"output,omitempty"`
	Artifacts    []TaskArtifact         `json:"artifacts,omitempty"`
	Summary      string                 `json:"summary,omitempty"`
	FilesChanged []string               `json:"files_changed,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

// TaskArtifact represents an artifact produced by a task.
type TaskArtifact struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size,omitempty"`
	Type string `json:"type,omitempty"`
}

// TaskMetrics represents execution metrics for a task.
type TaskMetrics struct {
	TokensUsed      int64 `json:"tokens_used,omitempty"`
	InputTokens     int64 `json:"input_tokens,omitempty"`
	OutputTokens    int64 `json:"output_tokens,omitempty"`
	ElapsedSeconds  int64 `json:"elapsed_seconds,omitempty"`
	PeakMemoryMB    int64 `json:"peak_memory_mb,omitempty"`
	ToolCalls       int   `json:"tool_calls,omitempty"`
	FileOperations  int   `json:"file_operations,omitempty"`
}
