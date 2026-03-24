package executor

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/example/agent-infra/internal/model"
)

func TestJobManager_JobName(t *testing.T) {
	cfg := &JobConfig{
		NamePrefix: "sandbox-",
		Namespace:  "default",
	}
	mgr := NewJobManager(nil, cfg)

	taskID := "test-123"
	expected := "sandbox-test-123"
	result := mgr.jobName(taskID)

	if result != expected {
		t.Errorf("expected job name %s, got %s", expected, result)
	}
}

func TestJobManager_CreateJob_NilTask(t *testing.T) {
	mgr := NewJobManager(nil, nil)

	_, err := mgr.CreateJob(context.Background(), nil)
	if err != ErrInvalidJobConfig {
		t.Errorf("expected ErrInvalidJobConfig, got %v", err)
	}
}

func TestJobManager_GetJobStatus_NoJob(t *testing.T) {
	// This test would require a mock K8s client
	// For now, we test the job name generation
	cfg := DefaultJobConfig()
	mgr := NewJobManager(nil, cfg)

	taskID := "test-task"
	expected := "sandbox-test-task"
	if mgr.jobName(taskID) != expected {
		t.Errorf("expected %s, got %s", expected, mgr.jobName(taskID))
	}
}

func TestJobManager_BuildJobSpec(t *testing.T) {
	cfg := DefaultJobConfig()
	mgr := NewJobManager(nil, cfg)

	taskID := uuid.New()
	task := &model.Task{
		BaseModel: model.BaseModel{
			ID: taskID,
		},
		TenantID:   "tenant-123",
		CreatorID:  "user-123",
		ProviderID: "provider-123",
		Name:       "Test Task",
		Status:     model.TaskStatusPending,
	}

	job := mgr.buildJobSpec(task)

	expectedName := "sandbox-" + taskID.String()
	if job.Name != expectedName {
		t.Errorf("expected job name %s, got %s", expectedName, job.Name)
	}

	if job.Namespace != cfg.Namespace {
		t.Errorf("expected namespace %s, got %s", cfg.Namespace, job.Namespace)
	}

	// Check labels
	if job.Labels["app"] != "agent-sandbox" {
		t.Errorf("expected app label 'agent-sandbox', got %s", job.Labels["app"])
	}
	if job.Labels["task-id"] != taskID.String() {
		t.Errorf("expected task-id label %s, got %s", taskID.String(), job.Labels["task-id"])
	}

	// Check containers
	if len(job.Spec.Template.Spec.Containers) != 2 {
		t.Errorf("expected 2 containers, got %d", len(job.Spec.Template.Spec.Containers))
	}

	// Check cli-runner container
	cliRunner := job.Spec.Template.Spec.Containers[0]
	if cliRunner.Name != "cli-runner" {
		t.Errorf("expected container name 'cli-runner', got %s", cliRunner.Name)
	}

	// Check wrapper container
	wrapper := job.Spec.Template.Spec.Containers[1]
	if wrapper.Name != "wrapper" {
		t.Errorf("expected container name 'wrapper', got %s", wrapper.Name)
	}
}

func TestJobManager_BuildVolumes(t *testing.T) {
	mgr := NewJobManager(nil, nil)
	volumes := mgr.buildVolumes()

	if len(volumes) != 2 {
		t.Errorf("expected 2 volumes, got %d", len(volumes))
	}

	// Check workspace volume
	foundWorkspace := false
	foundAgentState := false
	for _, vol := range volumes {
		if vol.Name == "workspace" {
			foundWorkspace = true
		}
		if vol.Name == "agent-state" {
			foundAgentState = true
		}
	}

	if !foundWorkspace {
		t.Error("workspace volume not found")
	}
	if !foundAgentState {
		t.Error("agent-state volume not found")
	}
}

func TestJobManager_GetJobPhase(t *testing.T) {
	// This test would require batchv1.Job objects
	// For now, we just verify the helper functions work
	mgr := NewJobManager(nil, nil)
	if mgr == nil {
		t.Error("JobManager should not be nil")
	}
}

func TestNewJobManager_NilConfig(t *testing.T) {
	mgr := NewJobManager(nil, nil)
	if mgr == nil {
		t.Error("NewJobManager should not return nil")
	}
	if mgr.config == nil {
		t.Error("config should be initialized with defaults")
	}
}

func TestJobManager_HelperFunctions(t *testing.T) {
	// Test int32Ptr
	val := int32Ptr(42)
	if *val != 42 {
		t.Errorf("expected 42, got %d", *val)
	}

	// Test boolPtr
	b := boolPtr(true)
	if !*b {
		t.Error("expected true")
	}
}
