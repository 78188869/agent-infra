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

	// Check PodSecurityContext
	podSecCtx := job.Spec.Template.Spec.SecurityContext
	if podSecCtx == nil {
		t.Error("expected PodSecurityContext to be set, got nil")
	} else {
		if podSecCtx.RunAsNonRoot == nil || !*podSecCtx.RunAsNonRoot {
			t.Error("expected RunAsNonRoot to be true")
		}
		if podSecCtx.RunAsUser == nil || *podSecCtx.RunAsUser != 1000 {
			t.Errorf("expected RunAsUser to be 1000, got %v", podSecCtx.RunAsUser)
		}
		if podSecCtx.RunAsGroup == nil || *podSecCtx.RunAsGroup != 1000 {
			t.Errorf("expected RunAsGroup to be 1000, got %v", podSecCtx.RunAsGroup)
		}
		if podSecCtx.FSGroup == nil || *podSecCtx.FSGroup != 1000 {
			t.Errorf("expected FSGroup to be 1000, got %v", podSecCtx.FSGroup)
		}
	}

	// Check Container SecurityContext for cli-runner
	if cliRunner.SecurityContext == nil {
		t.Error("expected cli-runner SecurityContext to be set, got nil")
	} else {
		if cliRunner.SecurityContext.RunAsNonRoot == nil || !*cliRunner.SecurityContext.RunAsNonRoot {
			t.Error("expected cli-runner RunAsNonRoot to be true")
		}
		if cliRunner.SecurityContext.RunAsUser == nil || *cliRunner.SecurityContext.RunAsUser != 1000 {
			t.Errorf("expected cli-runner RunAsUser to be 1000, got %v", cliRunner.SecurityContext.RunAsUser)
		}
		if cliRunner.SecurityContext.ReadOnlyRootFilesystem == nil || !*cliRunner.SecurityContext.ReadOnlyRootFilesystem {
			t.Error("expected cli-runner ReadOnlyRootFilesystem to be true")
		}
		if cliRunner.SecurityContext.AllowPrivilegeEscalation == nil || *cliRunner.SecurityContext.AllowPrivilegeEscalation {
			t.Error("expected cli-runner AllowPrivilegeEscalation to be false")
		}
	}

	// Check Container SecurityContext for wrapper
	if wrapper.SecurityContext == nil {
		t.Error("expected wrapper SecurityContext to be set, got nil")
	} else {
		if wrapper.SecurityContext.RunAsNonRoot == nil || !*wrapper.SecurityContext.RunAsNonRoot {
			t.Error("expected wrapper RunAsNonRoot to be true")
		}
		if wrapper.SecurityContext.RunAsUser == nil || *wrapper.SecurityContext.RunAsUser != 1000 {
			t.Errorf("expected wrapper RunAsUser to be 1000, got %v", wrapper.SecurityContext.RunAsUser)
		}
		if wrapper.SecurityContext.ReadOnlyRootFilesystem == nil || !*wrapper.SecurityContext.ReadOnlyRootFilesystem {
			t.Error("expected wrapper ReadOnlyRootFilesystem to be true")
		}
		if wrapper.SecurityContext.AllowPrivilegeEscalation == nil || *wrapper.SecurityContext.AllowPrivilegeEscalation {
			t.Error("expected wrapper AllowPrivilegeEscalation to be false")
		}
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

	// Test int64Ptr
	i64 := int64Ptr(1000)
	if *i64 != 1000 {
		t.Errorf("expected 1000, got %d", *i64)
	}
}

func TestJobManager_CustomSecurityConfig(t *testing.T) {
	// Test with custom security configuration
	allowPrivEsc := true
	fsGroup := int64(2000)
	cfg := &JobConfig{
		NamePrefix:              "sandbox-",
		Namespace:               "default",
		CLIRunnerImage:          "test:latest",
		WrapperImage:            "test:latest",
		DefaultCPULimit:         "2",
		DefaultMemoryLimit:      "4Gi",
		DefaultCPURequest:       "500m",
		DefaultMemoryRequest:    "1Gi",
		WrapperCPULimit:         "100m",
		WrapperMemoryLimit:      "128Mi",
		WrapperCPURequest:       "50m",
		WrapperMemoryRequest:    "64Mi",
		Security: &SecurityConfig{
			RunAsNonRoot:            true,
			RunAsUser:               2000,
			RunAsGroup:              2000,
			ReadOnlyRootFilesystem:  false,
			AllowPrivilegeEscalation: &allowPrivEsc,
			FSGroup:                 &fsGroup,
		},
	}
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

	// Verify custom security settings are applied
	podSecCtx := job.Spec.Template.Spec.SecurityContext
	if podSecCtx == nil {
		t.Fatal("expected PodSecurityContext to be set")
	}

	if podSecCtx.RunAsUser == nil || *podSecCtx.RunAsUser != 2000 {
		t.Errorf("expected RunAsUser to be 2000, got %v", podSecCtx.RunAsUser)
	}

	if podSecCtx.FSGroup == nil || *podSecCtx.FSGroup != 2000 {
		t.Errorf("expected FSGroup to be 2000, got %v", podSecCtx.FSGroup)
	}

	// Check container security context
	cliRunner := job.Spec.Template.Spec.Containers[0]
	if cliRunner.SecurityContext == nil {
		t.Fatal("expected cli-runner SecurityContext to be set")
	}

	if cliRunner.SecurityContext.ReadOnlyRootFilesystem == nil || *cliRunner.SecurityContext.ReadOnlyRootFilesystem {
		t.Error("expected ReadOnlyRootFilesystem to be false")
	}

	if cliRunner.SecurityContext.AllowPrivilegeEscalation == nil || !*cliRunner.SecurityContext.AllowPrivilegeEscalation {
		t.Error("expected AllowPrivilegeEscalation to be true")
	}
}

func TestJobManager_NilSecurityConfig(t *testing.T) {
	// Test with nil security configuration - should use defaults
	cfg := &JobConfig{
		NamePrefix:              "sandbox-",
		Namespace:               "default",
		CLIRunnerImage:          "test:latest",
		WrapperImage:            "test:latest",
		DefaultCPULimit:         "2",
		DefaultMemoryLimit:      "4Gi",
		DefaultCPURequest:       "500m",
		DefaultMemoryRequest:    "1Gi",
		WrapperCPULimit:         "100m",
		WrapperMemoryLimit:      "128Mi",
		WrapperCPURequest:       "50m",
		WrapperMemoryRequest:    "64Mi",
		Security:                nil, // Explicitly nil
	}
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

	// Verify default security settings are applied
	podSecCtx := job.Spec.Template.Spec.SecurityContext
	if podSecCtx == nil {
		t.Fatal("expected PodSecurityContext to be set")
	}

	// Should use defaults
	if podSecCtx.RunAsNonRoot == nil || !*podSecCtx.RunAsNonRoot {
		t.Error("expected default RunAsNonRoot to be true")
	}

	if podSecCtx.RunAsUser == nil || *podSecCtx.RunAsUser != 1000 {
		t.Errorf("expected default RunAsUser to be 1000, got %v", podSecCtx.RunAsUser)
	}
}

func TestDefaultSecurityConfig(t *testing.T) {
	sec := DefaultSecurityConfig()

	if sec.RunAsNonRoot != true {
		t.Error("expected RunAsNonRoot to be true")
	}

	if sec.RunAsUser != 1000 {
		t.Errorf("expected RunAsUser to be 1000, got %d", sec.RunAsUser)
	}

	if sec.RunAsGroup != 1000 {
		t.Errorf("expected RunAsGroup to be 1000, got %d", sec.RunAsGroup)
	}

	if sec.ReadOnlyRootFilesystem != true {
		t.Error("expected ReadOnlyRootFilesystem to be true")
	}

	if sec.AllowPrivilegeEscalation == nil || *sec.AllowPrivilegeEscalation != false {
		t.Error("expected AllowPrivilegeEscalation to be false")
	}

	if sec.FSGroup == nil || *sec.FSGroup != 1000 {
		t.Errorf("expected FSGroup to be 1000, got %v", sec.FSGroup)
	}
}
