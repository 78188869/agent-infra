package executor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// MockRedisClient implements RedisClient interface for testing.
type MockRedisClient struct {
	data     map[string]map[string]string
	err      error
	closed   bool
	commands []*mockCommand
}

type mockCommand struct {
	cmd  string
	key  string
	args []interface{}
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]map[string]string),
	}
}

func (m *MockRedisClient) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	m.commands = append(m.commands, &mockCommand{cmd: "HSet", key: key, args: values})
	if m.data[key] == nil {
		m.data[key] = make(map[string]string)
	}
	// Simple implementation for testing
	if len(values) >= 2 {
		if field, ok := values[0].(string); ok {
			if val, ok := values[1].(string); ok {
				m.data[key][field] = val
			}
		}
	}
	return redis.NewIntCmd(ctx)
}

func (m *MockRedisClient) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	m.commands = append(m.commands, &mockCommand{cmd: "HGet", key: key, args: []interface{}{field}})
	cmd := redis.NewStringCmd(ctx)
	if data, ok := m.data[key]; ok {
		if val, ok := data[field]; ok {
			cmd.SetVal(val)
		}
	}
	return cmd
}

func (m *MockRedisClient) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	m.commands = append(m.commands, &mockCommand{cmd: "HGetAll", key: key})
	cmd := redis.NewMapStringStringCmd(ctx)
	if data, ok := m.data[key]; ok {
		cmd.SetVal(data)
	}
	return cmd
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	m.commands = append(m.commands, &mockCommand{cmd: "Del", args: []interface{}{keys}})
	for _, key := range keys {
		delete(m.data, key)
	}
	return redis.NewIntCmd(ctx)
}

func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	m.commands = append(m.commands, &mockCommand{cmd: "Expire", key: key, args: []interface{}{expiration}})
	return redis.NewBoolCmd(ctx)
}

func (m *MockRedisClient) Pipeline() redis.Pipeliner {
	// Return nil for now - tests should avoid using pipeline operations
	return nil
}

func TestNewHeartbeatManager(t *testing.T) {
	mockRedis := NewMockRedisClient()

	t.Run("with config", func(t *testing.T) {
		cfg := &HeartbeatManagerConfig{
			Interval:  10 * time.Second,
			Timeout:   30 * time.Second,
			OnTimeout: func(ctx context.Context, taskID string) error { return nil },
		}
		mgr, err := NewHeartbeatManager(mockRedis, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if mgr.interval != 10*time.Second {
			t.Errorf("expected interval 10s, got %v", mgr.interval)
		}
		if mgr.timeout != 30*time.Second {
			t.Errorf("expected timeout 30s, got %v", mgr.timeout)
		}
	})

	t.Run("nil config uses defaults", func(t *testing.T) {
		mgr, err := NewHeartbeatManager(mockRedis, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if mgr.interval != DefaultHeartbeatInterval {
			t.Errorf("expected default interval, got %v", mgr.interval)
		}
		if mgr.timeout != DefaultHeartbeatTimeout {
			t.Errorf("expected default timeout, got %v", mgr.timeout)
		}
	})

	t.Run("nil redis client returns error", func(t *testing.T) {
		mgr, err := NewHeartbeatManager(nil, nil)
		if err != ErrNilRedisClient {
			t.Errorf("expected ErrNilRedisClient, got %v", err)
		}
		if mgr != nil {
			t.Error("expected nil manager when redis client is nil")
		}
	})
}

func TestHeartbeatManager_Register(t *testing.T) {
	mockRedis := NewMockRedisClient()
	mgr, err := NewHeartbeatManager(mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	taskID := "test-task-123"
	podIP := "10.0.0.1"

	mgr.Register(taskID, podIP)

	if mgr.GetTaskCount() != 1 {
		t.Errorf("expected 1 task, got %d", mgr.GetTaskCount())
	}

	mgr.mu.RLock()
	info, exists := mgr.tasks[taskID]
	mgr.mu.RUnlock()

	if !exists {
		t.Fatal("task should be registered")
	}
	if info.PodIP != podIP {
		t.Errorf("expected pod IP %s, got %s", podIP, info.PodIP)
	}
}

func TestHeartbeatManager_Unregister(t *testing.T) {
	mockRedis := NewMockRedisClient()
	mgr, err := NewHeartbeatManager(mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	taskID := "test-task-123"
	mgr.Register(taskID, "10.0.0.1")

	if mgr.GetTaskCount() != 1 {
		t.Errorf("expected 1 task, got %d", mgr.GetTaskCount())
	}

	mgr.Unregister(taskID)

	if mgr.GetTaskCount() != 0 {
		t.Errorf("expected 0 tasks, got %d", mgr.GetTaskCount())
	}
}

func TestHeartbeatManager_UpdateHeartbeat(t *testing.T) {
	// Skip this test as it requires a full Redis mock with pipeline support
	// In production, use miniredis or a real Redis for integration testing
	t.Skip("Requires full Redis mock with pipeline support")
}

func TestHeartbeatManager_GetHeartbeat(t *testing.T) {
	mockRedis := NewMockRedisClient()
	mgr, err := NewHeartbeatManager(mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	taskID := "test-task-123"

	// Set up mock data
	mockRedis.data["executor:heartbeat:"+taskID] = map[string]string{
		"last_seen": "1234567890",
		"status":    "running",
		"progress":  "75",
	}

	info, err := mgr.GetHeartbeat(context.Background(), taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info == nil {
		t.Fatal("expected heartbeat info, got nil")
	}
	if info.Status != "running" {
		t.Errorf("expected status 'running', got %s", info.Status)
	}
	if info.Progress != 75 {
		t.Errorf("expected progress 75, got %d", info.Progress)
	}
}

func TestHeartbeatManager_GetHeartbeat_NotFound(t *testing.T) {
	mockRedis := NewMockRedisClient()
	mgr, err := NewHeartbeatManager(mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := mgr.GetHeartbeat(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil for nonexistent heartbeat, got %+v", info)
	}
}

func TestHeartbeatManager_StartStop(t *testing.T) {
	mockRedis := NewMockRedisClient()
	mgr, err := NewHeartbeatManager(mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Start
	err = mgr.Start(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on start: %v", err)
	}

	if !mgr.IsRunning() {
		t.Error("manager should be running")
	}

	// Double start should fail
	err = mgr.Start(context.Background())
	if err != ErrExecutorAlreadyRunning {
		t.Errorf("expected ErrExecutorAlreadyRunning, got %v", err)
	}

	// Stop
	err = mgr.Stop(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on stop: %v", err)
	}

	if mgr.IsRunning() {
		t.Error("manager should not be running")
	}
}

func TestHeartbeatManager_GetTaskCount(t *testing.T) {
	mockRedis := NewMockRedisClient()
	mgr, err := NewHeartbeatManager(mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mgr.GetTaskCount() != 0 {
		t.Errorf("expected 0 tasks, got %d", mgr.GetTaskCount())
	}

	mgr.Register("task-1", "10.0.0.1")
	mgr.Register("task-2", "10.0.0.2")
	mgr.Register("task-3", "10.0.0.3")

	if mgr.GetTaskCount() != 3 {
		t.Errorf("expected 3 tasks, got %d", mgr.GetTaskCount())
	}

	mgr.Unregister("task-2")

	if mgr.GetTaskCount() != 2 {
		t.Errorf("expected 2 tasks, got %d", mgr.GetTaskCount())
	}
}

func TestHeartbeatManager_ErrorsChannel(t *testing.T) {
	mockRedis := NewMockRedisClient()

	t.Run("no error channel by default", func(t *testing.T) {
		mgr, err := NewHeartbeatManager(mockRedis, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mgr.Errors() != nil {
			t.Error("expected nil error channel by default")
		}
	})

	t.Run("error channel created when configured", func(t *testing.T) {
		cfg := &HeartbeatManagerConfig{
			ErrorChanSize: 10,
		}
		mgr, err := NewHeartbeatManager(mockRedis, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mgr.Errors() == nil {
			t.Error("expected error channel to be created")
		}
	})
}

func TestHeartbeatManager_HandleTimeout_CallbackError(t *testing.T) {
	mockRedis := NewMockRedisClient()

	testError := errors.New("callback failed")

	cfg := &HeartbeatManagerConfig{
		ErrorChanSize: 10,
		OnTimeout: func(ctx context.Context, taskID string) error {
			return testError
		},
	}
	mgr, err := NewHeartbeatManager(mockRedis, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	taskID := "test-task-timeout"
	mgr.Register(taskID, "10.0.0.1")

	// Call handleTimeout directly
	mgr.handleTimeout(context.Background(), taskID)

	// Task should be unregistered
	if mgr.GetTaskCount() != 0 {
		t.Errorf("expected 0 tasks after timeout, got %d", mgr.GetTaskCount())
	}

	// Error should be sent to channel
	select {
	case err := <-mgr.Errors():
		if err == nil {
			t.Error("expected error from channel")
		}
		if !strings.Contains(err.Error(), "callback failed") {
			t.Errorf("expected error to contain 'callback failed', got: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("expected error in channel, but got none")
	}
}

func TestHeartbeatManager_HandleTimeout_CallbackSuccess(t *testing.T) {
	mockRedis := NewMockRedisClient()

	callbackCalled := false
	cfg := &HeartbeatManagerConfig{
		ErrorChanSize: 10,
		OnTimeout: func(ctx context.Context, taskID string) error {
			callbackCalled = true
			return nil // Success
		},
	}
	mgr, err := NewHeartbeatManager(mockRedis, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	taskID := "test-task-success"
	mgr.Register(taskID, "10.0.0.1")

	// Call handleTimeout directly
	mgr.handleTimeout(context.Background(), taskID)

	// Task should be unregistered
	if mgr.GetTaskCount() != 0 {
		t.Errorf("expected 0 tasks after timeout, got %d", mgr.GetTaskCount())
	}

	// Callback should have been called
	if !callbackCalled {
		t.Error("expected callback to be called")
	}

	// No error should be sent to channel
	select {
	case err := <-mgr.Errors():
		t.Errorf("unexpected error in channel: %v", err)
	case <-time.After(100 * time.Millisecond):
		// Expected - no error
	}
}

func TestHeartbeatManager_HandleTimeout_NoCallback(t *testing.T) {
	mockRedis := NewMockRedisClient()

	cfg := &HeartbeatManagerConfig{
		ErrorChanSize: 10,
		// No OnTimeout callback
	}
	mgr, err := NewHeartbeatManager(mockRedis, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	taskID := "test-task-nocallback"
	mgr.Register(taskID, "10.0.0.1")

	// Call handleTimeout directly - should not panic
	mgr.handleTimeout(context.Background(), taskID)

	// Task should be unregistered
	if mgr.GetTaskCount() != 0 {
		t.Errorf("expected 0 tasks after timeout, got %d", mgr.GetTaskCount())
	}
}
