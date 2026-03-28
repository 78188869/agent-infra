package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/example/agent-infra/internal/monitoring"
	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/aliyun/sls"
)

// MonitoringService handles log recording and real-time event broadcasting.
type MonitoringService interface {
	// RecordTaskStatusChange broadcasts status change via WebSocket + writes to SLS.
	RecordTaskStatusChange(ctx context.Context, taskID, tenantID, oldStatus, newStatus string) error
	// RecordLogEntry writes a log entry to SLS.
	RecordLogEntry(ctx context.Context, taskID, tenantID string, eventType model.EventType, eventName string, content interface{}) error
	// RecordTaskProgress pushes progress update via WS.
	RecordTaskProgress(ctx context.Context, taskID, tenantID string, progress int64, tokensUsed int64, elapsedSecs int64) error
	// BroadcastTaskCompletion pushes task completion event via WS.
	BroadcastTaskCompletion(ctx context.Context, taskID, tenantID string) error
}

type monitoringService struct {
	hub *monitoring.Hub
	sls *monitoring.SLSClient
}

// NewMonitoringService creates a new MonitoringService.
func NewMonitoringService(hub *monitoring.Hub, slsClient *monitoring.SLSClient) MonitoringService {
	return &monitoringService{
		hub: hub,
		sls: slsClient,
	}
}

func (s *monitoringService) RecordTaskStatusChange(ctx context.Context, taskID, tenantID, oldStatus, newStatus string) error {
	payload := monitoring.StatusChangePayload{OldStatus: oldStatus, NewStatus: newStatus}
	msg := monitoring.WSMessage{
		Type:      monitoring.WSTypeTaskStatusChanged,
		TaskID:    taskID,
		TenantID:  tenantID,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}
	data, _ := json.Marshal(msg)
	s.hub.Broadcast(tenantID, data)

	slog.Info("task status changed",
		"component", "business",
		"task_id", taskID,
		"tenant_id", tenantID,
		"old_status", oldStatus,
		"new_status", newStatus,
	)

	// Write to SLS
	entry := &sls.LogEntry{
		TaskID:    taskID,
		TenantID:  tenantID,
		Timestamp: time.Now(),
		EventType: string(model.EventTypeStatusChange),
		EventName: newStatus,
		Content: map[string]interface{}{
			"old_status": oldStatus,
			"new_status": newStatus,
		},
		Source: "control-plane",
	}
	return s.sls.RecordEvent(ctx, entry)
}

func (s *monitoringService) RecordLogEntry(ctx context.Context, taskID, tenantID string, eventType model.EventType, eventName string, content interface{}) error {
	entry := &sls.LogEntry{
		TaskID:    taskID,
		TenantID:  tenantID,
		Timestamp: time.Now(),
		EventType: string(eventType),
		EventName: eventName,
		Content:   toMap(content),
		Source:    "control-plane",
	}

	slog.Info("log entry recorded",
		"component", "business",
		"task_id", taskID,
		"tenant_id", tenantID,
		"event_type", string(eventType),
		"event_name", eventName,
	)

	return s.sls.RecordEvent(ctx, entry)
}

func (s *monitoringService) RecordTaskProgress(ctx context.Context, taskID, tenantID string, progress int64, tokensUsed int64, elapsedSecs int64) error {
	payload := monitoring.ProgressPayload{
		Progress:    progress,
		TokensUsed:  tokensUsed,
		ElapsedSecs: elapsedSecs,
	}
	msg := monitoring.WSMessage{
		Type:      monitoring.WSTypeTaskProgressUpdated,
		TaskID:    taskID,
		TenantID:  tenantID,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}
	data, _ := json.Marshal(msg)
	s.hub.Broadcast(tenantID, data)
	return nil
}

func (s *monitoringService) BroadcastTaskCompletion(ctx context.Context, taskID, tenantID string) error {
	msg := monitoring.WSMessage{
		Type:      monitoring.WSTypeTaskCompleted,
		TaskID:    taskID,
		TenantID:  tenantID,
		Timestamp: time.Now().UnixMilli(),
		Payload:   nil,
	}
	data, _ := json.Marshal(msg)
	s.hub.Broadcast(tenantID, data)

	slog.Info("task completed",
		"component", "business",
		"task_id", taskID,
		"tenant_id", tenantID,
	)

	return nil
}

func toMap(v interface{}) map[string]interface{} {
	data, _ := json.Marshal(v)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	return m
}
