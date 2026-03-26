// Package monitoring provides monitoring, logging, and real-time event services.
package monitoring

// WebSocket message types matching TRD §7.5.2
const (
	WSTypeTaskStatusChanged   = "task.status_changed"
	WSTypeTaskProgressUpdated = "task.progress_updated"
	WSTypeTaskLogEntry       = "task.log_entry"
	WSTypeTaskCompleted      = "task.completed"
)

// WSMessage represents a WebSocket message per TRD §7.5.2
type WSMessage struct {
	Type      string      `json:"type"`
	TaskID    string      `json:"task_id,omitempty"`
	TenantID  string      `json:"tenant_id,omitempty"`
	Timestamp int64       `json:"timestamp"`
	Payload   interface{} `json:"payload,omitempty"`
}

// StatusChangePayload per TRD §7.5.3
type StatusChangePayload struct {
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
	Reason    string `json:"reason,omitempty"`
}

// ProgressPayload per TRD §7.5.4
type ProgressPayload struct {
	Progress     int64  `json:"progress"`
	Stage        string `json:"stage,omitempty"`
	TokensUsed   int64  `json:"tokens_used,omitempty"`
	ElapsedSecs  int64  `json:"elapsed_seconds,omitempty"`
}

// LogEntryPayload per TRD §7.5.5
type LogEntryPayload struct {
	EventType string      `json:"event_type"`
	EventName string      `json:"event_name,omitempty"`
	Content   interface{} `json:"content,omitempty"`
}
