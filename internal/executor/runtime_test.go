package executor

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/example/agent-infra/internal/model"
)

// MockContainerRuntime implements ContainerRuntime for testing.
type MockContainerRuntime struct {
	createFunc     func(ctx context.Context, task *model.Task) (*RuntimeInfo, error)
	getStatusFunc  func(ctx context.Context, taskID string) (*RuntimeStatus, error)
	deleteFunc     func(ctx context.Context, taskID string) error
	getAddressFunc func(ctx context.Context, taskID string) (string, error)
}

// NewMockContainerRuntime creates a MockContainerRuntime with default success behaviors.
func NewMockContainerRuntime() *MockContainerRuntime {
	return &MockContainerRuntime{
		createFunc: func(ctx context.Context, task *model.Task) (*RuntimeInfo, error) {
			return &RuntimeInfo{
				Name:      "sandbox-" + task.ID.String(),
				Namespace: "sandbox",
				Status: RuntimeStatus{
					Phase: "Pending",
				},
				CreatedAt: 1234567890,
			}, nil
		},
		getStatusFunc: func(ctx context.Context, taskID string) (*RuntimeStatus, error) {
			return &RuntimeStatus{
				Phase: "Running",
			}, nil
		},
		deleteFunc: func(ctx context.Context, taskID string) error {
			return nil
		},
		getAddressFunc: func(ctx context.Context, taskID string) (string, error) {
			return "10.0.0.1", nil
		},
	}
}

func (m *MockContainerRuntime) Create(ctx context.Context, task *model.Task) (*RuntimeInfo, error) {
	return m.createFunc(ctx, task)
}

func (m *MockContainerRuntime) GetStatus(ctx context.Context, taskID string) (*RuntimeStatus, error) {
	return m.getStatusFunc(ctx, taskID)
}

func (m *MockContainerRuntime) Delete(ctx context.Context, taskID string) error {
	return m.deleteFunc(ctx, taskID)
}

func (m *MockContainerRuntime) GetAddress(ctx context.Context, taskID string) (string, error) {
	return m.getAddressFunc(ctx, taskID)
}

// Verify MockContainerRuntime satisfies ContainerRuntime at compile time.
var _ ContainerRuntime = (*MockContainerRuntime)(nil)

func TestMockContainerRuntime_Interface(t *testing.T) {
	mock := NewMockContainerRuntime()

	t.Run("create returns expected info", func(t *testing.T) {
		task := &model.Task{
			BaseModel: model.BaseModel{ID: uuid.New()},
			TenantID:  "test-tenant",
		}
		info, err := mock.Create(context.Background(), task)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Name == "" {
			t.Error("expected non-empty name")
		}
	})

	t.Run("get status returns running", func(t *testing.T) {
		status, err := mock.GetStatus(context.Background(), "test-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status.Phase != "Running" {
			t.Errorf("expected Running, got %s", status.Phase)
		}
	})

	t.Run("delete succeeds", func(t *testing.T) {
		err := mock.Delete(context.Background(), "test-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("get address returns IP", func(t *testing.T) {
		addr, err := mock.GetAddress(context.Background(), "test-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if addr == "" {
			t.Error("expected non-empty address")
		}
	})

	t.Run("custom create function", func(t *testing.T) {
		mock := NewMockContainerRuntime()
		mock.createFunc = func(ctx context.Context, task *model.Task) (*RuntimeInfo, error) {
			return nil, fmt.Errorf("runtime unavailable")
		}
		_, err := mock.Create(context.Background(), &model.Task{
			BaseModel: model.BaseModel{ID: uuid.New()},
		})
		if err == nil {
			t.Error("expected error from custom create func")
		}
	})
}
