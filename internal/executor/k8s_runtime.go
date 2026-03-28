package executor

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"

	"github.com/example/agent-infra/internal/model"
)

// K8sRuntime implements ContainerRuntime using Kubernetes Jobs.
type K8sRuntime struct {
	jobManager *JobManager
}

// NewK8sRuntime creates a new K8sRuntime instance.
func NewK8sRuntime(client kubernetes.Interface, config *JobConfig) *K8sRuntime {
	return &K8sRuntime{
		jobManager: NewJobManager(client, config),
	}
}

// Create creates a K8s Job for the given task.
func (r *K8sRuntime) Create(ctx context.Context, task *model.Task) (*RuntimeInfo, error) {
	if task == nil {
		return nil, ErrInvalidJobConfig
	}

	jobInfo, err := r.jobManager.CreateJob(ctx, task)
	if err != nil {
		return nil, err
	}

	return &RuntimeInfo{
		Name:      jobInfo.Name,
		Namespace: jobInfo.Namespace,
		Status: RuntimeStatus{
			Phase: jobInfo.Status.Phase,
		},
		CreatedAt: jobInfo.CreatedAt,
	}, nil
}

// GetStatus returns the status of the K8s Job for the given task.
func (r *K8sRuntime) GetStatus(ctx context.Context, taskID string) (*RuntimeStatus, error) {
	jobStatus, err := r.jobManager.GetJobStatus(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return &RuntimeStatus{
		Phase:          jobStatus.Phase,
		Message:        jobStatus.Message,
		StartTime:      jobStatus.StartTime,
		CompletionTime: jobStatus.CompletionTime,
		ExitCode:       jobStatus.ExitCode,
	}, nil
}

// Delete removes the K8s Job for the given task.
func (r *K8sRuntime) Delete(ctx context.Context, taskID string) error {
	return r.jobManager.DeleteJob(ctx, taskID)
}

// GetAddress returns the Pod IP address for the task's K8s Job.
func (r *K8sRuntime) GetAddress(ctx context.Context, taskID string) (string, error) {
	addr, err := r.jobManager.GetPodAddress(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get runtime address: %w", err)
	}
	return addr, nil
}
