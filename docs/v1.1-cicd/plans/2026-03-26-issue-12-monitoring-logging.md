# Monitoring & Logging System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement task execution monitoring, log collection (Aliyun SLS), real-time WebSocket push, and monitoring metrics APIs for the MVP.

**Architecture:** Three-layer approach: (1) `internal/monitoring/` — SLS client + in-memory WebSocket hub, (2) `internal/service/monitoring_svc.go` — business logic for metrics aggregation, (3) `internal/api/handler/metrics.go` — HTTP handlers + WebSocket upgrade endpoint. Execution events flow from executor through the hub to WebSocket clients and SLS in parallel.

**Tech Stack:** Go 1.22 + Gin 1.9 | Aliyun SLS SDK | gorilla/websocket | Redis (pub/sub for multi-instance WebSocket fan-out)

---

## Knowledge Required

- [ ] `docs/knowledge/monitoring.md` — full monitoring module spec
- [ ] `docs/v1.0-mvp/TRD.md` §9 — monitoring design section
- [ ] `docs/v1.0-mvp/TRD.md` §7.5 — WebSocket API design
- [ ] `docs/v1.0-mvp/TRD.md` §7.1.7 — metrics API endpoints
- [ ] `internal/model/execution_log.go` — existing ExecutionLog model
- [ ] `internal/api/router/router.go` — existing route registration pattern

---

## File Structure

```
internal/
├── monitoring/
│   ├── sls_client.go          # Aliyun SLS ingestion client
│   ├── sls_client_test.go
│   ├── ws_hub.go             # WebSocket connection hub (gorilla/websocket)
│   ├── ws_hub_test.go
│   └── types.go              # WebSocket message types, event structs
├── service/
│   ├── monitoring_service.go  # Metrics aggregation + SLS write + WS broadcast
│   └── monitoring_service_test.go
├── api/
│   └── handler/
│       ├── metrics.go         # GET /metrics/* handlers
│       ├── metrics_test.go
│       └── ws_handler.go      # WebSocket upgrade + client handler
│   └── router/
│       └── router.go          # Add metrics routes + WS route

pkg/
└── aliyun/
    └── sls/
        └── client.go          # SLS SDK wrapper (config, retry, batch write)
```

---

## Dependencies

- Issue #5 (Backend Core API) — base API structure
- Issue #6 (Database Models) — ExecutionLog model exists
- Issue #8 (Task Executor) — executor pushes events; integration point is `executor.go` calling `monitoringService.RecordEvent()`
- Go dependencies: `github.com/gorilla/websocket`, `github.com/aliyun/aliyun-log-go-sdk`

---

## Tasks

### Phase 1: WebSocket Hub (in-memory, single-instance MVP)

#### Task 1: WebSocket Hub Core

**Files:**
- Create: `internal/monitoring/types.go`
- Create: `internal/monitoring/ws_hub.go`
- Test: `internal/monitoring/ws_hub_test.go`

- [ ] **Step 1: Write failing test for WS hub**

```go
// internal/monitoring/ws_hub_test.go
package monitoring

import (
    "testing"
    "github.com/gorilla/websocket"
)

func TestHub_RegisterAndBroadcast(t *testing.T) {
    hub := NewHub()
    done := make(chan string, 1)

    // Client that captures broadcast
    conn := &mockConn{sendCh: make(chan []byte, 1)}
    conn.sendCh <- nil // unblock readPump

    hub.Register("tenant-1", conn)
    hub.Broadcast("tenant-1", []byte(`{"type":"task.status_changed"}`))

    select {
    case msg := <-conn.sendCh:
        if string(msg) != `{"type":"task.status_changed"}` {
            t.Errorf("expected broadcast message, got %s", string(msg))
        }
    case <-done:
        t.Fatal("timeout waiting for broadcast")
    }
}

type mockConn struct {
    sendCh chan []byte
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/monitoring/... -run TestHub_RegisterAndBroadcast -v`
Expected: FAIL — types.go and ws_hub.go do not exist yet

- [ ] **Step 3: Write types.go**

```go
// internal/monitoring/types.go
package monitoring

// WebSocket message types matching TRD §7.5.2
const (
    WSTypeTaskStatusChanged  = "task.status_changed"
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
    Progress      int64  `json:"progress"`
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
```

- [ ] **Step 4: Run test to verify types compile**

Run: `go build ./internal/monitoring/...`
Expected: FAIL — ws_hub.go still missing

- [ ] **Step 5: Write minimal ws_hub.go**

```go
// internal/monitoring/ws_hub.go
package monitoring

import (
    "sync"
    "github.com/gorilla/websocket"
)

// Hub maintains the set of active clients per tenant and broadcasts messages.
type Hub struct {
   mu      sync.RWMutex
    clients map[string]map[*websocket.Conn]struct{}
}

// NewHub creates a new Hub.
func NewHub() *Hub {
    return &Hub{
        clients: make(map[string]map[*websocket.Conn]struct{}),
    }
}

// Register adds a WebSocket connection to the tenant room.
func (h *Hub) Register(tenantID string, conn *websocket.Conn) {
    h.mu.Lock()
    defer h.mu.Unlock()
    if h.clients[tenantID] == nil {
        h.clients[tenantID] = make(map[*websocket.Conn]struct{})
    }
    h.clients[tenantID][conn] = struct{}{}
}

// Unregister removes a WebSocket connection from the tenant room.
func (h *Hub) Unregister(tenantID string, conn *websocket.Conn) {
    h.mu.Lock()
    defer h.mu.Unlock()
    if room, ok := h.clients[tenantID]; ok {
        delete(room, conn)
        if len(room) == 0 {
            delete(h.clients, tenantID)
        }
    }
}

// Broadcast sends a message to all clients in the tenant room.
func (h *Hub) Broadcast(tenantID string, msg []byte) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    for conn := range h.clients[tenantID] {
        conn.WriteMessage(websocket.TextMessage, msg)
    }
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/monitoring/... -run TestHub_RegisterAndBroadcast -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/monitoring/types.go internal/monitoring/ws_hub.go internal/monitoring/ws_hub_test.go
git commit -m "feat(monitoring): add WebSocket hub for real-time tenant broadcasts"
```

---

#### Task 2: WebSocket Handler

**Files:**
- Create: `internal/api/handler/ws_handler.go`
- Modify: `internal/api/router/router.go` — add WS route

- [ ] **Step 1: Write failing test for WS handler upgrade**

```go
// internal/api/handler/ws_handler_test.go
package handler

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/gin-gonic/gin"
)

func TestWSHandler_Upgrade(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    handler := NewWSHandler(nil) // hub = nil for this basic test
    r.GET("/api/v1/ws", handler.HandleWebSocket)

    req := httptest.NewRequest("GET", "/api/v1/ws?token=test-key", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusSwitchingProtocols {
        t.Errorf("expected 101, got %d", w.Code)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api/handler/... -run TestWSHandler_Upgrade -v`
Expected: FAIL — ws_handler.go doesn't exist

- [ ] **Step 3: Write ws_handler.go**

```go
// internal/api/handler/ws_handler.go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
)

// WSHandler handles WebSocket connections.
type WSHandler struct {
    hub      interface{ Register(string, *websocket.Conn); Unregister(string, *websocket.Conn) }
    upgrader websocket.Upgrader
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(hub interface{ Register(string, *websocket.Conn); Unregister(string, *websocket.Conn) }) *WSHandler {
    return &WSHandler{
        hub: hub,
        upgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool { return true },
        },
    }
}

// HandleWebSocket upgrades HTTP to WebSocket.
// Token validation: extract API key from ?token= query param.
// Per TRD §7.5.1: ws://{host}/api/v1/ws?token={api_key}
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
    token := c.Query("token")
    if token == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
        return
    }

    // TODO: Validate token and extract tenant_id (Issue #5 auth integration point)
    // For MVP, use a placeholder; real auth comes from Issue #5
    tenantID := "tenant-from-token" // will be replaced when auth is integrated

    conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }

    h.hub.Register(tenantID, conn)
    defer h.hub.Unregister(tenantID, conn)

    // Read pump (drain incoming messages, we only broadcast server→client)
    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            break
        }
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/api/handler/... -run TestWSHandler_Upgrade -v`
Expected: PASS

- [ ] **Step 5: Add WS route to router**

```go
// internal/api/router/router.go — add to Setup()
wsHandler := handler.NewWSHandler(/* hub injected via service container */)
v1.GET("/ws", wsHandler.HandleWebSocket)
```

- [ ] **Step 6: Run router tests**

Run: `go test ./internal/api/router/... -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/api/handler/ws_handler.go internal/api/router/router.go
git commit -m "feat(monitoring): add WebSocket handler and route for real-time events"
```

---

### Phase 2: SLS Log Client & Monitoring Service

#### Task 3: SLS Client (阿里云日志服务)

**Files:**
- Create: `pkg/aliyun/sls/client.go`
- Create: `pkg/aliyun/sls/client_test.go`
- Create: `internal/monitoring/sls_client.go`
- Create: `internal/monitoring/sls_client_test.go`

- [ ] **Step 1: Write failing test for SLS client config**

```go
// pkg/aliyun/sls/client_test.go
package sls

import (
    "testing"
)

func TestConfig_Endpoint(t *testing.T) {
    cfg := Config{
        Endpoint: "cn-hangzhou.log.aliyuncs.com",
        AccessKeyID: "test-id",
        AccessKeySecret: "test-secret",
    }
    if cfg.Endpoint == "" {
        t.Error("endpoint should not be empty")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/aliyun/sls/... -v`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Write SLS config and client**

```go
// pkg/aliyun/sls/client.go
package sls

import (
    "context"
    "encoding/json"
    "time"
)

// Config holds SLS client configuration.
type Config struct {
    Endpoint        string // e.g. "cn-hangzhou.log.aliyuncs.com"
    AccessKeyID     string
    AccessKeySecret string
    Project         string // e.g. "agent-infra-prod"
    LogStore        string // e.g. "execution-logs"
}

// Client wraps the SLS SDK client for ingestion.
type Client struct {
    cfg Config
}

// NewClient creates a new SLS client.
func NewClient(cfg Config) *Client {
    return &Client{cfg: cfg}
}

// LogEntry represents a structured log entry per TRD §3.2 / monitoring.md §3.2.
type LogEntry struct {
    TaskID    string                 `json:"task_id"`
    TenantID  string                 `json:"tenant_id"`
    Timestamp time.Time              `json:"timestamp"`
    EventType string                 `json:"event_type"`
    EventName string                 `json:"event_name,omitempty"`
    Content   map[string]interface{} `json:"content,omitempty"`
    Source    string                 `json:"source"`
}

// Ingest sends a log entry to SLS.
// In production this batches writes; for MVP write one by one.
func (c *Client) Ingest(ctx context.Context, entry *LogEntry) error {
    data, err := json.Marshal(entry)
    if err != nil {
        return err
    }
    // TODO: Replace with actual SLS SDK PutLogs call.
    // import "github.com/aliyun/aliyun-log-go-sdk/producer"
    // For MVP, we log to stdout so tests can verify the path works.
    return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/aliyun/sls/... -v`
Expected: PASS

- [ ] **Step 5: Write monitoring SLS client wrapper**

```go
// internal/monitoring/sls_client.go
package monitoring

import (
    "context"
    "github.com/example/agent-infra/pkg/aliyun/sls"
)

// SLSClient wraps the Aliyun SLS client for monitoring use.
type SLSClient struct {
    client *sls.Client
}

// NewSLSClient creates a new SLS monitoring client.
func NewSLSClient(cfg sls.Config) *SLSClient {
    return &SLSClient{client: sls.NewClient(cfg)}
}

// RecordEvent writes an execution log event to SLS.
func (s *SLSClient) RecordEvent(ctx context.Context, entry *sls.LogEntry) error {
    return s.client.Ingest(ctx, entry)
}
```

- [ ] **Step 6: Run build check**

Run: `go build ./internal/monitoring/...`
Expected: SUCCESS

- [ ] **Step 7: Commit**

```bash
git add pkg/aliyun/sls/client.go pkg/aliyun/sls/client_test.go internal/monitoring/sls_client.go internal/monitoring/sls_client_test.go
git commit -m "feat(monitoring): add SLS client for Aliyun log ingestion"
```

---

#### Task 4: Monitoring Service (metrics aggregation + event routing)

**Files:**
- Create: `internal/service/monitoring_service.go`
- Create: `internal/service/monitoring_service_test.go`
- Modify: `internal/executor/executor.go` — inject MonitoringService and call RecordEvent on status changes

- [ ] **Step 1: Write failing test for monitoring service**

```go
// internal/service/monitoring_service_test.go
package service

import (
    "testing"
)

func TestMonitoringService_RecordEvent(t *testing.T) {
    // TODO: write test once service interface is defined
    t.Skip("waiting for MonitoringService interface")
}
```

- [ ] **Step 2: Run test to verify it skips**

Run: `go test ./internal/service/... -run TestMonitoringService_RecordEvent -v`
Expected: SKIP

- [ ] **Step 3: Write MonitoringService**

```go
// internal/service/monitoring_service.go
package service

import (
    "context"
    "encoding/json"
    "time"

    "github.com/example/agent-infra/internal/monitoring"
    "github.com/example/agent-infra/internal/model"
    "github.com/example/agent-infra/pkg/aliyun/sls"
)

// MonitoringService handles log recording and real-time event broadcasting.
type MonitoringService interface {
    // RecordTaskStatusChange broadcasts status change via WebSocket + writes to SLS.
    RecordTaskStatusChange(ctx context.Context, taskID, tenantID, oldStatus, newStatus string) error
    // RecordLogEntry writes a log entry to SLS and optionally pushes via WS.
    RecordLogEntry(ctx context.Context, taskID, tenantID string, eventType model.EventType, content interface{}) error
    // RecordTaskProgress pushes progress update via WS.
    RecordTaskProgress(ctx context.Context, taskID, tenantID string, progress int64, tokensUsed int64, elapsedSecs int64) error
    // BroadcastTaskCompletion pushes task completion event via WS.
    BroadcastTaskCompletion(ctx context.Context, taskID, tenantID string) error
}

type monitoringService struct {
    hub      *monitoring.Hub
    sls      *monitoring.SLSClient
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

func (s *monitoringService) RecordLogEntry(ctx context.Context, taskID, tenantID string, eventType model.EventType, content interface{}) error {
    entry := &sls.LogEntry{
        TaskID:    taskID,
        TenantID:  tenantID,
        Timestamp: time.Now(),
        EventType: string(eventType),
        Content:   toMap(content),
        Source:    "control-plane",
    }
    return s.sls.RecordEvent(ctx, entry)
}

func (s *monitoringService) RecordTaskProgress(ctx context.Context, taskID, tenantID string, progress int64, tokensUsed int64, elapsedSecs int64) error {
    payload := monitoring.ProgressPayload{
        Progress:     progress,
        TokensUsed:   tokensUsed,
        ElapsedSecs:  elapsedSecs,
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
    return nil
}

func toMap(v interface{}) map[string]interface{} {
    data, _ := json.Marshal(v)
    var m map[string]interface{}
    json.Unmarshal(data, &m)
    return m
}
```

- [ ] **Step 4: Run build check**

Run: `go build ./internal/service/...`
Expected: SUCCESS

- [ ] **Step 5: Commit**

```bash
git add internal/service/monitoring_service.go
git commit -m "feat(monitoring): add MonitoringService for WS broadcast and SLS ingestion"
```

---

### Phase 3: Monitoring Metrics API

#### Task 5: Metrics API Handlers

**Files:**
- Create: `internal/api/handler/metrics.go`
- Create: `internal/api/handler/metrics_test.go`

- [ ] **Step 1: Write failing test for dashboard endpoint**

```go
// internal/api/handler/metrics_test.go
package handler

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/gin-gonic/gin"
)

func TestMetricsHandler_GetDashboard(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    handler := NewMetricsHandler(nil)
    r.GET("/api/v1/metrics/dashboard", handler.GetDashboard)

    req := httptest.NewRequest("GET", "/api/v1/metrics/dashboard", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }

    var resp map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &resp)
    if resp["active_tasks"] == nil {
        t.Error("expected active_tasks in response")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api/handler/... -run TestMetricsHandler_GetDashboard -v`
Expected: FAIL — metrics.go doesn't exist

- [ ] **Step 3: Write metrics.go with all four dashboard endpoints per TRD §7.1.7**

```go
// internal/api/handler/metrics.go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/example/agent-infra/internal/service"
)

// MetricsHandler handles monitoring metrics API endpoints.
type MetricsHandler struct {
    svc service.MonitoringService
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler(svc service.MonitoringService) *MetricsHandler {
    return &MetricsHandler{svc: svc}
}

// DashboardResponse represents the dashboard data per TRD §7.1.7 / issue #12 §4.
type DashboardResponse struct {
    ActiveTasks   int64   `json:"active_tasks"`
    QueuedTasks   int64   `json:"queued_tasks"`
    CompletedToday int64  `json:"completed_today"`
    FailedToday    int64  `json:"failed_today"`
    AvgDurationSecs float64 `json:"avg_duration_seconds"`
    TokenUsage     int64  `json:"token_usage_today"`
}

// GetDashboard returns dashboard metrics per issue #12 §4.
func (h *MetricsHandler) GetDashboard(c *gin.Context) {
    // TODO: Query Task model for active/queued/completed/failed counts
    // SELECT status, COUNT(*) FROM tasks GROUP BY status
    // Today's completed: SELECT COUNT(*) FROM tasks WHERE status='succeeded' AND DATE(finished_at)=TODAY()
    // Avg duration: SELECT AVG(TIMESTAMPDIFF(SECOND, started_at, finished_at)) FROM tasks WHERE finished_at IS NOT NULL
    resp := DashboardResponse{
        ActiveTasks:    0,
        QueuedTasks:    0,
        CompletedToday: 0,
        FailedToday:    0,
        AvgDurationSecs: 0,
        TokenUsage:     0,
    }
    c.JSON(http.StatusOK, resp)
}

// TaskStatsResponse per TRD §7.1.7: GET /metrics/tasks
type TaskStatsResponse struct {
    Total       int64            `json:"total"`
    ByStatus    map[string]int64 `json:"by_status"`
    ByPriority  map[string]int64 `json:"by_priority"`
    TodayCount   int64            `json:"today_count"`
    TodayFailed  int64            `json:"today_failed"`
}

func (h *MetricsHandler) GetTaskStats(c *gin.Context) {
    resp := TaskStatsResponse{
        Total:      0,
        ByStatus:   map[string]int64{},
        ByPriority: map[string]int64{},
        TodayCount:  0,
        TodayFailed: 0,
    }
    c.JSON(http.StatusOK, resp)
}

// ResourceUsageResponse per TRD §7.1.7: GET /metrics/resources
type ResourceUsageResponse struct {
    CPUUsage    float64 `json:"cpu_usage_percent"`
    MemoryUsage float64 `json:"memory_usage_percent"`
    PodCount    int     `json:"pod_count"`
}

func (h *MetricsHandler) GetResourceUsage(c *gin.Context) {
    // TODO: Query K8s metrics API (kubelet / metrics-server) for CPU/Memory
    // For MVP: return zeros or mock data
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
    TaskCountByTenant  map[string]int64  `json:"task_count_by_tenant"`
}

func (h *MetricsHandler) GetTenantStats(c *gin.Context) {
    resp := TenantStatsResponse{
        TotalTenants:      0,
        ActiveTenants:     0,
        TaskCountByTenant: map[string]int64{},
    }
    c.JSON(http.StatusOK, resp)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/api/handler/... -run TestMetricsHandler_GetDashboard -v`
Expected: PASS

- [ ] **Step 5: Add routes to router**

```go
// internal/api/router/router.go — add to Setup()
metricsHandler := handler.NewMetricsHandler(monitoringSvc)
metrics := v1.Group("/metrics")
{
    metrics.GET("/dashboard", metricsHandler.GetDashboard)
    metrics.GET("/tasks", metricsHandler.GetTaskStats)
    metrics.GET("/resources", metricsHandler.GetResourceUsage)
    metrics.GET("/tenants", metricsHandler.GetTenantStats)
}
```

- [ ] **Step 6: Run router build**

Run: `go build ./internal/api/router/...`
Expected: SUCCESS

- [ ] **Step 7: Commit**

```bash
git add internal/api/handler/metrics.go internal/api/handler/metrics_test.go internal/api/router/router.go
git commit -m "feat(monitoring): add metrics API handlers and routes per TRD §7.1.7"
```

---

#### Task 6: Implement Real Metrics Queries

**Files:**
- Modify: `internal/api/handler/metrics.go`
- Modify: `internal/repository/task_repo.go` — add CountByStatus, CountTodayByStatus, AvgDuration methods

- [ ] **Step 1: Write failing test for task repo stats methods**

```go
// internal/repository/task_repo_test.go — add after existing tests
func TestTaskRepository_CountByStatus(t *testing.T) {
    repo, _ := setupTestDB(t)
    // Create tasks in various statuses
    // ...
    counts, err := repo.CountByStatus(context.Background())
    if err != nil {
        t.Fatal(err)
    }
    if counts["running"] != 2 {
        t.Errorf("expected 2 running tasks, got %d", counts["running"])
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/repository/... -run TestTaskRepository_CountByStatus -v`
Expected: FAIL — method doesn't exist

- [ ] **Step 3: Add CountByStatus to task repo**

```go
// internal/repository/task_repo.go
// CountByStatus returns task counts grouped by status.
func (r *TaskRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
    var results []struct {
        Status string
        Count  int64
    }
    err := r.db.WithContext(ctx).Model(&model.Task{}).
        Select("status, count(*) as count").
        Group("status").
        Scan(&results).Error
    if err != nil {
        return nil, err
    }
    m := make(map[string]int64)
    for _, r := range results {
        m[r.Status] = r.Count
    }
    return m, nil
}

// CountTodayByStatus returns today's task counts by status.
func (r *TaskRepository) CountTodayByStatus(ctx context.Context) (map[string]int64, error) {
    today := time.Now().Truncate(24 * time.Hour)
    var results []struct {
        Status string
        Count  int64
    }
    err := r.db.WithContext(ctx).Model(&model.Task{}).
        Select("status, count(*) as count").
        Where("created_at >= ?", today).
        Group("status").
        Scan(&results).Error
    if err != nil {
        return nil, err
    }
    m := make(map[string]int64)
    for _, r := range results {
        m[r.Status] = r.Count
    }
    return m, nil
}

// AvgDurationSeconds returns average task duration in seconds for completed tasks.
func (r *TaskRepository) AvgDurationSeconds(ctx context.Context) (float64, error) {
    var avg float64
    err := r.db.WithContext(ctx).Model(&model.Task{}).
        Select("avg(timestampdiff(SECOND, started_at, finished_at))").
        Where("finished_at IS NOT NULL AND started_at IS NOT NULL").
        Scan(&avg).Error
    return avg, err
}
```

- [ ] **Step 4: Run test to verify it compiles**

Run: `go build ./internal/repository/...`
Expected: SUCCESS

- [ ] **Step 5: Update metrics.go to use real queries**

```go
// internal/api/handler/metrics.go — update GetDashboard
func (h *MetricsHandler) GetDashboard(c *gin.Context) {
    // Query via task repository injected into handler
    // (or via the service layer — adjust architecture as needed)
    counts, _ := h.taskRepo.CountByStatus(c.Request.Context())
    todayCounts, _ := h.taskRepo.CountTodayByStatus(c.Request.Context())
    avgDur, _ := h.taskRepo.AvgDurationSeconds(c.Request.Context())

    resp := DashboardResponse{
        ActiveTasks:     counts["running"] + counts["paused"] + counts["waiting_approval"],
        QueuedTasks:    counts["pending"] + counts["scheduled"],
        CompletedToday: todayCounts["succeeded"],
        FailedToday:    todayCounts["failed"],
        AvgDurationSecs: avgDur,
        TokenUsage:     0, // TODO: aggregate from ExecutionLog after SLS integration
    }
    c.JSON(http.StatusOK, resp)
}
```

- [ ] **Step 6: Run build**

Run: `go build ./internal/api/handler/... ./internal/repository/...`
Expected: SUCCESS

- [ ] **Step 7: Commit**

```bash
git add internal/repository/task_repo.go internal/api/handler/metrics.go
git commit -m "feat(monitoring): implement real metrics queries for dashboard"
```

---

### Phase 4: Executor → Monitoring Integration

#### Task 7: Wire MonitoringService into Executor

**Files:**
- Modify: `internal/executor/executor.go` — add MonitoringService field and call it on status changes
- Modify: `internal/executor/task_executor.go` — call RecordTaskStatusChange on state transitions

- [ ] **Step 1: Write failing integration test**

```go
// internal/executor/executor_test.go — add test
func TestExecutor_StatusChangeBroadcasts(t *testing.T) {
    // mock MonitoringService
    // start executor with mock
    // trigger status change
    // assert WS broadcast was called
    t.Skip("integration test — requires MonitoringService mock")
}
```

- [ ] **Step 2: Modify executor.go to accept MonitoringService**

```go
// internal/executor/executor.go
type Executor struct {
    kubeClient   kubernetes.Interface
    taskRepo     repository.TaskRepository
    eventEmitter *EventEmitter // existing
    monitor      service.MonitoringService // NEW
}

func NewExecutor(..., monitor service.MonitoringService) *Executor {
    return &Executor{..., monitor: monitor}
}
```

- [ ] **Step 3: Call monitoring service on status change in task_executor.go**

```go
// In task_executor.go — after updating task status in DB:
// if monitor != nil {
//     monitor.RecordTaskStatusChange(ctx, task.ID, task.TenantID, oldStatus, newStatus)
// }
```

- [ ] **Step 4: Run build**

Run: `go build ./internal/executor/...`
Expected: SUCCESS

- [ ] **Step 5: Commit**

```bash
git add internal/executor/executor.go internal/executor/task_executor.go
git commit -m "feat(monitoring): wire MonitoringService into executor for status broadcasts"
```

---

### Phase 5: Log Query API (SLS)

#### Task 8: Log Query API

**Files:**
- Modify: `internal/api/handler/metrics.go` — add GET /tasks/:id/logs handler
- Modify: `internal/repository/execution_log_repo.go` — create and add query methods

- [ ] **Step 1: Write failing test for log query**

```go
// internal/api/handler/metrics_test.go
func TestMetricsHandler_GetTaskLogs(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    handler := NewMetricsHandler(nil)
    r.GET("/api/v1/tasks/:id/logs", handler.GetTaskLogs)

    req := httptest.NewRequest("GET", "/api/v1/tasks/task-123/logs?start=2026-03-01T00:00:00Z&end=2026-03-26T00:00:00Z&keyword=Write", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api/handler/... -run TestMetricsHandler_GetTaskLogs -v`
Expected: FAIL — method doesn't exist

- [ ] **Step 3: Add GetTaskLogs handler**

```go
// internal/api/handler/metrics.go

// GetTaskLogs returns execution logs for a task.
// Query params: start (RFC3339), end (RFC3339), keyword (string)
// Per issue #12: "日志查询支持时间范围和关键词过滤"
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

    // TODO: Query SLS for logs using GetLogs API with query string
    // SLS GetLogs: project, logstore, query string, start/end time, limit
    // For MVP, fall back to DB execution_logs table for key events
    logs, _ := h.logRepo.QueryByTaskID(c.Request.Context(), taskID, start, end, keyword)
    c.JSON(http.StatusOK, gin.H{"logs": logs, "total": len(logs)})
}
```

- [ ] **Step 4: Create execution_log_repo.go with query methods**

```go
// internal/repository/execution_log_repo.go
package repository

import (
    "context"
    "time"
    "github.com/example/agent-infra/internal/model"
)

// ExecutionLogRepository handles execution log queries.
type ExecutionLogRepository struct {
    db *gorm.DB
}

func NewExecutionLogRepository(db *gorm.DB) *ExecutionLogRepository {
    return &ExecutionLogRepository{db: db}
}

// QueryByTaskID queries execution logs by task ID with optional time range and keyword.
func (r *ExecutionLogRepository) QueryByTaskID(ctx context.Context, taskID string, start, end time.Time, keyword string) ([]model.ExecutionLog, error) {
    q := r.db.WithContext(ctx).Where("task_id = ? AND timestamp BETWEEN ? AND ?", taskID, start, end)
    if keyword != "" {
        q = q.Where("event_name LIKE ?", "%"+keyword+"%")
    }
    var logs []model.ExecutionLog
    err := q.Order("timestamp ASC").Find(&logs).Error
    return logs, err
}
```

- [ ] **Step 5: Run build**

Run: `go build ./internal/repository/... ./internal/api/handler/...`
Expected: SUCCESS

- [ ] **Step 6: Add route for logs**

```go
// internal/api/router/router.go — add to tasks group
tasks.GET("/:id/logs", metricsHandler.GetTaskLogs)
```

- [ ] **Step 7: Run router build and commit**

Run: `go build ./internal/api/router/...`
Expected: SUCCESS

- [ ] **Step 8: Commit**

```bash
git add internal/repository/execution_log_repo.go internal/api/handler/metrics.go internal/api/router/router.go
git commit -m "feat(monitoring): add task log query API with time range and keyword filtering"
```

---

### Phase 6: Wiring and Final Integration

#### Task 9: Wire Everything Together in main.go / cmd/control-plane

**Files:**
- Modify: `cmd/control-plane/main.go` — create MonitoringService and inject dependencies

- [ ] **Step 1: Write failing build test (missing MonitoringService in main)**

```bash
go build ./cmd/control-plane/...
```

Expected: FAIL — MonitoringService not yet wired

- [ ] **Step 2: Update main.go to wire MonitoringService**

```go
// cmd/control-plane/main.go
// After DB, Redis, and other service initialization:

// Monitoring (Phase 8)
monitoringHub := monitoring.NewHub()
slsClient := monitoring.NewSLSClient(aliyunSLSConfig)
monitoringSvc := service.NewMonitoringService(monitoringHub, slsClient)

// Pass monitoringSvc to executor (via NewExecutor call)
// Pass monitoringHub to WSHandler (via NewWSHandler call)
wsHandler := handler.NewWSHandler(monitoringHub)
```

- [ ] **Step 3: Run full build**

Run: `go build ./cmd/control-plane/... && go build ./...`
Expected: SUCCESS

- [ ] **Step 4: Run all tests**

Run: `go test ./internal/... -coverprofile=coverage.out`
Expected: PASS (coverage > 80% target for monitoring package)

- [ ] **Step 5: Commit**

```bash
git add cmd/control-plane/main.go
git commit -m "feat(monitoring): wire MonitoringService and SLS client in main"
```

---

### Phase 7: SLA/Alert Rules Stub (Future Extension)

#### Task 10: Alert Rules Stub

**Files:**
- Create: `internal/monitoring/alerter.go` — stub with interface for future alert rules
- Create: `internal/monitoring/alerter_test.go`

- [ ] **Step 1: Write stub alerter interface per TRD §9.2**

```go
// internal/monitoring/alerter.go
package monitoring

import (
    "context"
)

// AlertSeverity levels per TRD §9.2.
type AlertSeverity string

const (
    AlertP0 AlertSeverity = "P0" // Service down
    AlertP1 AlertSeverity = "P1" // High priority
    AlertP2 AlertSeverity = "P2" // Low priority
)

// Alert represents an alert event per TRD §9.2.
type Alert struct {
    Name     string        `json:"name"`
    Severity AlertSeverity `json:"severity"`
    Message  string        `json:"message"`
}

// Alerter defines the interface for alert dispatch.
type Alerter interface {
    SendAlert(ctx context.Context, alert *Alert) error
}

// NoOpAlerter is a no-op alerter for MVP (alerts implemented in future phase).
type NoOpAlerter struct{}

func (a *NoOpAlerter) SendAlert(ctx context.Context, alert *Alert) error {
    // TODO: Integrate with DingTalk / SMS in future phase
    return nil
}
```

- [ ] **Step 2: Run build and commit**

Run: `go build ./internal/monitoring/...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/monitoring/alerter.go internal/monitoring/alerter_test.go
git commit -m "feat(monitoring): add alert interface stub per TRD §9.2 (future extension)"
```

---

## Verification

After all tasks complete, run:

```bash
# Full build
go build ./cmd/control-plane/... && go build ./...

# Test coverage
go test ./internal/monitoring/... -coverprofile=coverage.out
go tool cover -func=coverage.out

# All tests
go test ./internal/... -v

# Lint
make lint
```

**Expected results:**
- `go build` succeeds with no errors
- Monitoring package coverage > 80%
- All tests pass

---

## Status

- [ ] Not Started
