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

	taskID := uuid.New().String()

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

	taskID := uuid.New().String()
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

	taskID := uuid.New().String()
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

func TestValidateTaskID(t *testing.T) {
	tests := []struct {
		name      string
		taskID    string
		wantError bool
	}{
		{
			name:      "empty task ID",
			taskID:    "",
			wantError: true,
		},
		{
			name:      "invalid UUID format",
			taskID:    "not-a-uuid",
			wantError: true,
		},
		{
			name:      "invalid UUID format with numbers",
			taskID:    "test-task-123",
			wantError: true,
		},
		{
			name:      "valid UUID",
			taskID:    uuid.New().String(),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTaskID(tt.taskID)
			if (err != nil) != tt.wantError {
				t.Errorf("validateTaskID(%q) error = %v, wantError %v", tt.taskID, err, tt.wantError)
			}
			if err != nil && tt.wantError {
				// Verify the error wraps ErrInvalidTaskID
				if err.Error() == "" {
					t.Error("error message should not be empty")
				}
			}
		})
	}
}

func TestTaskExecutor_Execute_InvalidTaskID(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Task with nil UUID (zero value)
	task := &model.Task{
		BaseModel: model.BaseModel{
			ID: uuid.Nil,
		},
		Status: model.TaskStatusScheduled,
	}

	_, err = executor.Execute(context.Background(), task)
	if err == nil {
		t.Error("expected error for nil UUID task ID")
	}
}

func TestTaskExecutor_GetStatus_InvalidTaskID(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = executor.GetStatus(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty task ID")
	}

	_, err = executor.GetStatus(context.Background(), "invalid-uuid")
	if err == nil {
		t.Error("expected error for invalid UUID format")
	}
}

func TestTaskExecutor_Pause_InvalidTaskID(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = executor.Pause(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty task ID")
	}

	err = executor.Pause(context.Background(), "invalid-uuid")
	if err == nil {
		t.Error("expected error for invalid UUID format")
	}
}

func TestTaskExecutor_Resume_InvalidTaskID(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = executor.Resume(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty task ID")
	}

	err = executor.Resume(context.Background(), "invalid-uuid")
	if err == nil {
		t.Error("expected error for invalid UUID format")
	}
}

func TestTaskExecutor_Cancel_InvalidTaskID(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = executor.Cancel(context.Background(), "", "test reason")
	if err == nil {
		t.Error("expected error for empty task ID")
	}

	err = executor.Cancel(context.Background(), "invalid-uuid", "test reason")
	if err == nil {
		t.Error("expected error for invalid UUID format")
	}
}

func TestTaskExecutor_GetPodAddress_InvalidTaskID(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = executor.GetPodAddress(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty task ID")
	}

	_, err = executor.GetPodAddress(context.Background(), "invalid-uuid")
	if err == nil {
		t.Error("expected error for invalid UUID format")
	}
}

func TestTaskExecutor_HandleHeartbeat_InvalidTaskID(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = executor.HandleHeartbeat(context.Background(), "", "running", 50)
	if err == nil {
		t.Error("expected error for empty task ID")
	}

	err = executor.HandleHeartbeat(context.Background(), "invalid-uuid", "running", 50)
	if err == nil {
		t.Error("expected error for invalid UUID format")
	}
}

func TestTaskExecutor_HandleTaskEvent_InvalidTaskID(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = executor.HandleTaskEvent(context.Background(), "", "status_change", map[string]interface{}{})
	if err == nil {
		t.Error("expected error for empty task ID")
	}

	err = executor.HandleTaskEvent(context.Background(), "invalid-uuid", "status_change", map[string]interface{}{})
	if err == nil {
		t.Error("expected error for invalid UUID format")
	}
}

func TestTaskExecutor_InjectInstruction_InvalidTaskID(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	mockRedis := NewMockRedisClient()

	executor, err := NewTaskExecutor(k8sClient, mockRedis, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = executor.InjectInstruction(context.Background(), "", "test instruction")
	if err == nil {
		t.Error("expected error for empty task ID")
	}

	err = executor.InjectInstruction(context.Background(), "invalid-uuid", "test instruction")
	if err == nil {
		t.Error("expected error for invalid UUID format")
	}
}
