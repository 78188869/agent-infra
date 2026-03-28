package executor

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/example/agent-infra/internal/model"
)

func TestK8sRuntime_ImplementsContainerRuntime(t *testing.T) {
	// Compile-time check
	var _ ContainerRuntime = (*K8sRuntime)(nil)
}

func TestK8sRuntime_Create(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	runtime := NewK8sRuntime(k8sClient, DefaultJobConfig())

	taskID := uuid.New()
	task := &model.Task{
		BaseModel: model.BaseModel{ID: taskID},
		TenantID:  "tenant-123",
		Status:    model.TaskStatusScheduled,
	}

	info, err := runtime.Create(context.Background(), task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedName := "sandbox-" + taskID.String()
	if info.Name != expectedName {
		t.Errorf("expected name %s, got %s", expectedName, info.Name)
	}
	if info.Namespace != "sandbox" {
		t.Errorf("expected namespace sandbox, got %s", info.Namespace)
	}
	if info.Status.Phase != "Pending" {
		t.Errorf("expected Pending phase, got %s", info.Status.Phase)
	}
}

func TestK8sRuntime_Create_NilTask(t *testing.T) {
	runtime := NewK8sRuntime(nil, nil)

	_, err := runtime.Create(context.Background(), nil)
	if err != ErrInvalidJobConfig {
		t.Errorf("expected ErrInvalidJobConfig, got %v", err)
	}
}

func TestK8sRuntime_Create_DuplicateJob(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	runtime := NewK8sRuntime(k8sClient, DefaultJobConfig())

	taskID := uuid.New()
	task := &model.Task{
		BaseModel: model.BaseModel{ID: taskID},
		TenantID:  "tenant-123",
		Status:    model.TaskStatusScheduled,
	}

	_, err := runtime.Create(context.Background(), task)
	if err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}

	_, err = runtime.Create(context.Background(), task)
	if err == nil {
		t.Error("expected error for duplicate job creation")
	}
}

func TestK8sRuntime_GetStatus_NotFound(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	runtime := NewK8sRuntime(k8sClient, DefaultJobConfig())

	taskID := uuid.New().String()
	_, err := runtime.GetStatus(context.Background(), taskID)
	if err == nil {
		t.Error("expected error for non-existent job")
	}
}

func TestK8sRuntime_Delete_NotFound(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	runtime := NewK8sRuntime(k8sClient, DefaultJobConfig())

	taskID := uuid.New().String()
	err := runtime.Delete(context.Background(), taskID)
	if err == nil {
		t.Error("expected error for deleting non-existent job")
	}
}

func TestK8sRuntime_GetAddress_NotFound(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	runtime := NewK8sRuntime(k8sClient, DefaultJobConfig())

	taskID := uuid.New().String()
	_, err := runtime.GetAddress(context.Background(), taskID)
	if err == nil {
		t.Error("expected error for non-existent job address")
	}
}

func TestNewK8sRuntime_NilConfig(t *testing.T) {
	runtime := NewK8sRuntime(nil, nil)
	if runtime == nil {
		t.Error("expected non-nil K8sRuntime")
	}
	if runtime.jobManager == nil {
		t.Error("expected non-nil jobManager")
	}
}
