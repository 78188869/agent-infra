// Package model provides database models for the application.
package model

import (
	"time"

	"gorm.io/datatypes"
)

// EventType represents the type of an execution log event.
type EventType string

const (
	EventTypeStatusChange EventType = "status_change"
	EventTypeToolCall     EventType = "tool_call"
	EventTypeToolResult   EventType = "tool_result"
	EventTypeLLMInput     EventType = "llm_input"
	EventTypeLLMOutput    EventType = "llm_output"
	EventTypeError        EventType = "error"
	EventTypeHeartbeat    EventType = "heartbeat"
	EventTypeIntervention EventType = "intervention"
	EventTypeMetric       EventType = "metric"
	EventTypeCheckpoint   EventType = "checkpoint"
)

// ExecutionLog represents a log entry for task execution.
// Note: Full logs are stored in Alibaba Cloud SLS, this table only stores
// key event indices for quick querying.
type ExecutionLog struct {
	ID      int64       `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID  string      `gorm:"type:varchar(36);not null;index:idx_task_time;index:idx_task_event" json:"task_id"`

	// Event Information
	EventType EventType     `gorm:"type:varchar(32);not null;index:idx_task_event" json:"event_type"`
	EventName string        `gorm:"type:varchar(64)" json:"event_name"`
	Content   datatypes.JSON `gorm:"type:json" json:"content"`

	// Relation
	ParentEventID *int64 `gorm:"index" json:"parent_event_id"`

	// Timestamp (millisecond precision)
	Timestamp time.Time `gorm:"index:idx_task_time" json:"timestamp"`

	// Relations
	Task *Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
}

// TableName returns the table name for ExecutionLog.
func (ExecutionLog) TableName() string {
	return "execution_logs"
}

// StatusChangeEvent represents a status change event.
type StatusChangeEvent struct {
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
	Reason    string `json:"reason,omitempty"`
}

// ToolCallEvent represents a tool call event.
type ToolCallEvent struct {
	ToolName string                 `json:"tool_name"`
	Input    map[string]interface{} `json:"input,omitempty"`
}

// ToolResultEvent represents a tool result event.
type ToolResultEvent struct {
	ToolName string      `json:"tool_name"`
	Output   interface{} `json:"output"`
	Error    string      `json:"error,omitempty"`
}

// LLMInputEvent represents an LLM input event.
type LLMInputEvent struct {
	Model    string `json:"model,omitempty"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages,omitempty"`
}

// LLMOutputEvent represents an LLM output event.
type LLMOutputEvent struct {
	Model       string `json:"model,omitempty"`
	Content     string `json:"content,omitempty"`
	StopReason  string `json:"stop_reason,omitempty"`
	TokensUsed  int64  `json:"tokens_used,omitempty"`
}

// ErrorEvent represents an error event.
type ErrorEvent struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Stack   string `json:"stack,omitempty"`
}

// InterventionEvent represents an intervention event.
type InterventionEvent struct {
	Action    string `json:"action"`
	Content   string `json:"content,omitempty"`
	Operator  string `json:"operator,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// MetricEvent represents a metric event.
type MetricEvent struct {
	MetricName  string  `json:"metric_name"`
	MetricValue float64 `json:"metric_value"`
	Unit        string  `json:"unit,omitempty"`
}

// CheckpointEvent represents a checkpoint event.
type CheckpointEvent struct {
	CheckpointName string `json:"checkpoint_name"`
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
}
