package executor

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/example/agent-infra/internal/model"
)

func TestNewTaskExecutor(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	t.Run("with nil config", func(t *testing.T) {
		executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if executor == nil {
			t.Fatal("executor should not be nil")
		}
	})

	t.Run("with config", func(t *testing.T) {
		cfg := &ExecutorConfig{
			JobConfig: DefaultJobConfig(),
		}
		executor, err := NewTaskExecutor(k8sClient, mockRedis, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if executor.config != cfg {
			t.Error("config should be set")
		}
	})
}

func TestTaskExecutor_Execute_NilTask(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = executor.Execute(context.Background(), nil)
	if err != ErrInvalidJobConfig {
		t.Errorf("expected ErrInvalidJobConfig, got %v", err)
	}
}

func TestTaskExecutor_Execute_ValidTask(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	taskID := uuid.New()
	task := &model.Task{
		BaseModel: model.BaseModel{
			ID: taskID,
		},
		TenantID:   "tenant-123",
		CreatorID:  "user-123",
		ProviderID: "provider-123",
		Name:       "Test Task",
		Status:     model.TaskStatusScheduled,
	}

	statusUpdated := false
	cfg := &ExecutorConfig{
		JobConfig: DefaultJobConfig(),
		UpdateTaskStatus: func(ctx context.Context, taskID string, status string, message string) error {
			statusUpdated = true
			return nil
		},
	}

	executor, err := NewTaskExecutor(k8sClient, mockRedis, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Start the executor first
	err = executor.Start(context.Background())
	if err != nil {
		t.Fatalf("unexpected error starting executor: %v", err)
	}
	defer executor.Stop(context.Background())

	_, err = executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !statusUpdated {
		t.Error("status should have been updated")
	}
}

func TestTaskExecutor_canExecute(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		status   string
		expected bool
	}{
		{model.TaskStatusPending, true},
		{model.TaskStatusScheduled, true},
		{model.TaskStatusRetrying, true},
		{model.TaskStatusRunning, false},
		{model.TaskStatusPaused, false},
		{model.TaskStatusSucceeded, false},
		{model.TaskStatusFailed, false},
		{model.TaskStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			task := &model.Task{Status: tt.status}
			result := executor.canExecute(task)
			if result != tt.expected {
				t.Errorf("status %s: expected %v, got %v", tt.status, tt.expected, result)
			}
		})
	}
}

func TestTaskExecutor_StartStop(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Not running initially
	if executor.IsRunning() {
		t.Error("executor should not be running initially")
	}

	// Start
	err = executor.Start(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on start: %v", err)
	}

	if !executor.IsRunning() {
		t.Error("executor should be running")
	}

	// Double start should fail
	err = executor.Start(context.Background())
	if err != ErrExecutorAlreadyRunning {
		t.Errorf("expected ErrExecutorAlreadyRunning, got %v", err)
	}

	// Stop
	err = executor.Stop(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on stop: %v", err)
	}

	if executor.IsRunning() {
		t.Error("executor should not be running after stop")
	}
}

func TestTaskExecutor_HandleTaskEvent(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	statusUpdated := false
	cfg := &ExecutorConfig{
		JobConfig: DefaultJobConfig(),
		UpdateTaskStatus: func(ctx context.Context, taskID string, status string, message string) error {
			statusUpdated = true
			return nil
		},
	}

	executor, err := NewTaskExecutor(k8sClient, mockRedis, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = executor.Start(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer executor.Stop(context.Background())

	taskID := "test-task-123"

	// Test status_change event
	err = executor.HandleTaskEvent(context.Background(), taskID, "status_change", map[string]interface{}{
		"status":  "running",
		"message": "task started",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !statusUpdated {
		t.Error("status should have been updated for status_change event")
	}
}

func TestTaskExecutor_HandleTaskEvent_Complete(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	completed := false
	cfg := &ExecutorConfig{
		JobConfig: DefaultJobConfig(),
		OnTaskComplete: func(ctx context.Context, taskID string, result map[string]interface{}) error {
			completed = true
			return nil
		},
	}

	executor, err := NewTaskExecutor(k8sClient, mockRedis, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = executor.Start(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer executor.Stop(context.Background())

	taskID := "test-task-123"
	executor.heartbeat.Register(taskID, "10.0.0.1")

	err = executor.HandleTaskEvent(context.Background(), taskID, "complete", map[string]interface{}{
		"result": map[string]interface{}{
			"status": "succeeded",
			"output": "task completed",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !completed {
		t.Error("OnTaskComplete should have been called")
	}

	if executor.heartbeat.GetTaskCount() != 0 {
		t.Error("task should be unregistered from heartbeat")
	}
}

func TestTaskExecutor_HandleTaskEvent_Failed(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	failed := false
	cfg := &ExecutorConfig{
		JobConfig: DefaultJobConfig(),
		OnTaskFailed: func(ctx context.Context, taskID string, err error) error {
			failed = true
			return nil
		},
	}

	executor, err := NewTaskExecutor(k8sClient, mockRedis, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = executor.Start(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer executor.Stop(context.Background())

	taskID := "test-task-123"
	executor.heartbeat.Register(taskID, "10.0.0.1")

	err = executor.HandleTaskEvent(context.Background(), taskID, "failed", map[string]interface{}{
		"error": "task failed with error",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !failed {
		t.Error("OnTaskFailed should have been called")
	}

	if executor.heartbeat.GetTaskCount() != 0 {
		t.Error("task should be unregistered from heartbeat")
	}
}

func TestTaskExecutor_GetJobManager(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mgr := executor.GetJobManager()
	if mgr == nil {
		t.Error("JobManager should not be nil")
	}
}

func TestTaskExecutor_GetHeartbeatManager(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mgr := executor.GetHeartbeatManager()
	if mgr == nil {
		t.Error("HeartbeatManager should not be nil")
	}
}

func TestTaskExecutor_HandleHeartbeat(t *testing.T) {
	// Skip this test as it requires a full Redis mock with pipeline support
	t.Skip("Requires full Redis mock with pipeline support")
}
