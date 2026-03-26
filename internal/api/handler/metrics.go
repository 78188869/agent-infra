package handler

import (
	"net/http"
	"time"

	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
)

// MetricsHandler handles monitoring metrics API endpoints.
type MetricsHandler struct {
	svc service.MonitoringService
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler(svc service.MonitoringService) *MetricsHandler {
	return &MetricsHandler{svc: svc}
}

// DashboardResponse represents the dashboard data per issue #12 §4.
type DashboardResponse struct {
	ActiveTasks      int64   `json:"active_tasks"`
	QueuedTasks      int64   `json:"queued_tasks"`
	CompletedToday   int64   `json:"completed_today"`
	FailedToday      int64   `json:"failed_today"`
	AvgDurationSecs  float64 `json:"avg_duration_seconds"`
	TokenUsageToday   int64   `json:"token_usage_today"`
}

// GetDashboard returns dashboard metrics per issue #12 §4 and TRD §7.1.7.
func (h *MetricsHandler) GetDashboard(c *gin.Context) {
	// TODO: Query via task repository for real counts
	// Real implementation: taskRepo.CountByStatus(), taskRepo.CountTodayByStatus(), taskRepo.AvgDurationSeconds()
	resp := DashboardResponse{
		ActiveTasks:     0,
		QueuedTasks:     0,
		CompletedToday:  0,
		FailedToday:     0,
		AvgDurationSecs: 0,
		TokenUsageToday: 0,
	}
	c.JSON(http.StatusOK, resp)
}

// TaskStatsResponse per TRD §7.1.7: GET /metrics/tasks
type TaskStatsResponse struct {
	Total       int64            `json:"total"`
	ByStatus    map[string]int64 `json:"by_status"`
	ByPriority  map[string]int64 `json:"by_priority"`
	TodayCount  int64            `json:"today_count"`
	TodayFailed int64            `json:"today_failed"`
}

// GetTaskStats returns task statistics per TRD §7.1.7.
func (h *MetricsHandler) GetTaskStats(c *gin.Context) {
	// TODO: Query task repository for real stats
	resp := TaskStatsResponse{
		Total:       0,
		ByStatus:    map[string]int64{},
		ByPriority: map[string]int64{},
		TodayCount:  0,
		TodayFailed: 0,
	}
	c.JSON(http.StatusOK, resp)
}

// ResourceUsageResponse per TRD §7.1.7: GET /metrics/resources
type ResourceUsageResponse struct {
	CPUUsage     float64 `json:"cpu_usage_percent"`
	MemoryUsage  float64 `json:"memory_usage_percent"`
	PodCount     int     `json:"pod_count"`
}

// GetResourceUsage returns resource usage metrics per TRD §7.1.7.
func (h *MetricsHandler) GetResourceUsage(c *gin.Context) {
	// TODO: Query K8s metrics API for real CPU/Memory data
	resp := ResourceUsageResponse{
		CPUUsage:    0,
		MemoryUsage: 0,
		PodCount:    0,
	}
	c.JSON(http.StatusOK, resp)
}

// TenantStatsResponse per TRD §7.1.7: GET /metrics/tenants
type TenantStatsResponse struct {
	TotalTenants       int64            `json:"total_tenants"`
	ActiveTenants      int64            `json:"active_tenants"`
	TaskCountByTenant  map[string]int64 `json:"task_count_by_tenant"`
}

// GetTenantStats returns tenant-level statistics per TRD §7.1.7.
func (h *MetricsHandler) GetTenantStats(c *gin.Context) {
	// TODO: Query tenant repository and task repository for real stats
	resp := TenantStatsResponse{
		TotalTenants:      0,
		ActiveTenants:     0,
		TaskCountByTenant: map[string]int64{},
	}
	c.JSON(http.StatusOK, resp)
}

// GetTaskLogs returns execution logs for a task.
// Query params: start (RFC3339), end (RFC3339), keyword (string).
// Per issue #12: log query supports time range and keyword filtering.
func (h *MetricsHandler) GetTaskLogs(c *gin.Context) {
	taskID := c.Param("id")
	startStr := c.Query("start")
	endStr := c.Query("end")
	keyword := c.Query("keyword")

	// Parse time range
	var start, end time.Time
	if startStr != "" {
		start, _ = time.Parse(time.RFC3339, startStr)
	} else {
		start = time.Now().Add(-24 * time.Hour)
	}
	if endStr != "" {
		end, _ = time.Parse(time.RFC3339, endStr)
	} else {
		end = time.Now()
	}

	// TODO: Query SLS for logs using GetLogs API with query string.
	// For MVP, return empty logs — real SLS query in future iteration.
	c.JSON(http.StatusOK, gin.H{
		"logs":  []interface{}{},
		"total": 0,
		"task_id": taskID,
		"start":   start.Format(time.RFC3339),
		"end":     end.Format(time.RFC3339),
		"keyword": keyword,
	})
}
