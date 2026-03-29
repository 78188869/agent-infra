package executor

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/example/agent-infra/internal/model"
)

// JobManager manages K8s Job lifecycle for tasks.
type JobManager struct {
	client    kubernetes.Interface
	config    *JobConfig
	namespace string
}

// NewJobManager creates a new JobManager instance.
func NewJobManager(client kubernetes.Interface, config *JobConfig) *JobManager {
	if config == nil {
		config = DefaultJobConfig()
	}
	return &JobManager{
		client:    client,
		config:    config,
		namespace: config.Namespace,
	}
}

// jobName generates the Job name from task ID.
func (m *JobManager) jobName(taskID string) string {
	return fmt.Sprintf("%s%s", m.config.NamePrefix, taskID)
}

// CreateJob creates a K8s Job for the given task.
func (m *JobManager) CreateJob(ctx context.Context, task *model.Task) (*JobInfo, error) {
	if task == nil {
		return nil, ErrInvalidJobConfig
	}

	taskID := task.ID.String()
	jobName := m.jobName(taskID)

	// Check if job already exists
	_, err := m.client.BatchV1().Jobs(m.namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err == nil {
		return nil, ErrJobAlreadyExists
	}

	// Build Job spec
	job := m.buildJobSpec(task)

	// Create Job
	createdJob, err := m.client.BatchV1().Jobs(m.namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJobCreationFailed, err)
	}

	return &JobInfo{
		Name:      createdJob.Name,
		Namespace: createdJob.Namespace,
		Status: JobStatus{
			Phase: "Pending",
		},
		CreatedAt: createdJob.CreationTimestamp.Unix(),
	}, nil
}

// GetJob retrieves a Job by task ID.
func (m *JobManager) GetJob(ctx context.Context, taskID string) (*batchv1.Job, error) {
	jobName := m.jobName(taskID)
	job, err := m.client.BatchV1().Jobs(m.namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJobNotFound, err)
	}
	return job, nil
}

// GetJobStatus returns the status of a Job.
func (m *JobManager) GetJobStatus(ctx context.Context, taskID string) (*JobStatus, error) {
	job, err := m.GetJob(ctx, taskID)
	if err != nil {
		return nil, err
	}

	status := &JobStatus{
		Phase:   m.getJobPhase(job),
		Message: m.getJobMessage(job),
	}

	if job.Status.StartTime != nil {
		t := job.Status.StartTime.Unix()
		status.StartTime = &t
	}

	if job.Status.CompletionTime != nil {
		t := job.Status.CompletionTime.Unix()
		status.CompletionTime = &t
	}

	// Get exit code from container if available
	if len(job.Status.Conditions) > 0 {
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobFailed && cond.Status == corev1.ConditionTrue {
				exitCode := int32(1)
				status.ExitCode = &exitCode
				status.Message = cond.Message
			}
		}
	}

	return status, nil
}

// DeleteJob deletes a Job by task ID.
func (m *JobManager) DeleteJob(ctx context.Context, taskID string) error {
	jobName := m.jobName(taskID)
	propagationPolicy := metav1.DeletePropagationBackground
	err := m.client.BatchV1().Jobs(m.namespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrJobDeletionFailed, err)
	}
	return nil
}

// GetPodForJob returns the Pod associated with a Job.
func (m *JobManager) GetPodForJob(ctx context.Context, taskID string) (*corev1.Pod, error) {
	jobName := m.jobName(taskID)

	// List pods with job name label
	pods, err := m.client.CoreV1().Pods(m.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, ErrPodNotFound
	}

	// Return the first (and should be only) pod
	return &pods.Items[0], nil
}

// GetPodAddress returns the Pod IP address.
func (m *JobManager) GetPodAddress(ctx context.Context, taskID string) (string, error) {
	pod, err := m.GetPodForJob(ctx, taskID)
	if err != nil {
		return "", err
	}

	if pod.Status.PodIP == "" {
		return "", ErrPodNotFound
	}

	return pod.Status.PodIP, nil
}

// buildJobSpec builds the K8s Job specification for a task.
func (m *JobManager) buildJobSpec(task *model.Task) *batchv1.Job {
	taskID := task.ID.String()
	jobName := m.jobName(taskID)

	// Build labels
	labels := map[string]string{
		"app":       "agent-sandbox",
		"task-id":   taskID,
		"tenant-id": task.TenantID,
	}
	for k, v := range m.config.Labels {
		labels[k] = v
	}

	// Build annotations
	annotations := map[string]string{
		"task-id": taskID,
	}
	for k, v := range m.config.Annotations {
		annotations[k] = v
	}

	// Calculate timeout
	timeoutSeconds := m.config.DefaultTimeoutSeconds
	// TODO: Get timeout from task template if specified

	// Build PodSpec
	podSpec := corev1.PodSpec{
		ShareProcessNamespace: boolPtr(true),
		RestartPolicy:         corev1.RestartPolicyNever,
		SecurityContext:       m.buildPodSecurityContext(),
		Containers: []corev1.Container{
			m.buildCLIRunnerContainer(task),
			m.buildWrapperContainer(task),
		},
		Volumes: m.buildVolumes(),
	}

	// Set ServiceAccountName if configured
	if m.config.ServiceAccountName != "" {
		podSpec.ServiceAccountName = m.config.ServiceAccountName
	}

	// Build Job spec
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   m.namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            int32Ptr(0), // No auto-retry
			TTLSecondsAfterFinished: &m.config.TTLSecondsAfterFinished,
			ActiveDeadlineSeconds:   &timeoutSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: podSpec,
			},
		},
	}

	return job
}

// buildCLIRunnerContainer builds the main CLI runner container.
func (m *JobManager) buildCLIRunnerContainer(task *model.Task) corev1.Container {
	taskID := task.ID.String()
	return corev1.Container{
		Name:            "cli-runner",
		Image:           m.config.WrapperImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: m.buildContainerSecurityContext(),
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(m.config.DefaultCPULimit),
				corev1.ResourceMemory: resource.MustParse(m.config.DefaultMemoryLimit),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(m.config.DefaultCPURequest),
				corev1.ResourceMemory: resource.MustParse(m.config.DefaultMemoryRequest),
			},
		},
		Env: []corev1.EnvVar{
			{Name: "TASK_ID", Value: taskID},
			{Name: "TENANT_ID", Value: task.TenantID},
			{Name: "CONTROL_PLANE_URL", Value: m.config.ControlPlaneURL},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "workspace", MountPath: "/workspace"},
			{Name: "agent-state", MountPath: "/workspace/.agent-state"},
		},
	}
}

// buildWrapperContainer builds the wrapper sidecar container.
func (m *JobManager) buildWrapperContainer(task *model.Task) corev1.Container {
	taskID := task.ID.String()

	// Use config values with fallback to defaults for backward compatibility
	cpuLimit := m.config.WrapperCPULimit
	if cpuLimit == "" {
		cpuLimit = "100m"
	}
	memoryLimit := m.config.WrapperMemoryLimit
	if memoryLimit == "" {
		memoryLimit = "128Mi"
	}
	cpuRequest := m.config.WrapperCPURequest
	if cpuRequest == "" {
		cpuRequest = "50m"
	}
	memoryRequest := m.config.WrapperMemoryRequest
	if memoryRequest == "" {
		memoryRequest = "64Mi"
	}

	return corev1.Container{
		Name:            "wrapper",
		Image:           m.config.WrapperImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: m.buildContainerSecurityContext(),
		Ports: []corev1.ContainerPort{
			{ContainerPort: int32(m.config.WrapperPort), Name: "http"},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpuLimit),
				corev1.ResourceMemory: resource.MustParse(memoryLimit),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpuRequest),
				corev1.ResourceMemory: resource.MustParse(memoryRequest),
			},
		},
		Env: []corev1.EnvVar{
			{Name: "TASK_ID", Value: taskID},
			{Name: "TENANT_ID", Value: task.TenantID},
			{Name: "CONTROL_PLANE_URL", Value: m.config.ControlPlaneURL},
			{Name: "WRAPPER_PORT", Value: fmt.Sprintf("%d", m.config.WrapperPort)},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "workspace", MountPath: "/workspace"},
			{Name: "agent-state", MountPath: "/workspace/.agent-state"},
		},
	}
}

// buildVolumes builds the shared volumes for the Pod.
func (m *JobManager) buildVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "agent-state",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

// getJobPhase extracts the phase from Job status.
func (m *JobManager) getJobPhase(job *batchv1.Job) string {
	if job.Status.Succeeded > 0 {
		return string(batchv1.JobComplete)
	}
	if job.Status.Failed > 0 {
		return string(batchv1.JobFailed)
	}
	if job.Status.Active > 0 {
		return "Running"
	}
	return "Pending"
}

// getJobMessage extracts the message from Job conditions.
func (m *JobManager) getJobMessage(job *batchv1.Job) string {
	for _, cond := range job.Status.Conditions {
		if cond.Status == corev1.ConditionTrue {
			return cond.Message
		}
	}
	return ""
}

// Helper functions
func int32Ptr(i int32) *int32 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int64) *int64 {
	return &i
}

// getSecurityConfig returns the security configuration, using defaults if not set.
func (m *JobManager) getSecurityConfig() *SecurityConfig {
	if m.config.Security != nil {
		return m.config.Security
	}
	return DefaultSecurityConfig()
}

// buildPodSecurityContext creates a PodSecurityContext from the security config.
func (m *JobManager) buildPodSecurityContext() *corev1.PodSecurityContext {
	sec := m.getSecurityConfig()

	podSecCtx := &corev1.PodSecurityContext{
		RunAsNonRoot: &sec.RunAsNonRoot,
		RunAsUser:    &sec.RunAsUser,
		RunAsGroup:   &sec.RunAsGroup,
		FSGroup:      sec.FSGroup,
	}

	return podSecCtx
}

// buildContainerSecurityContext creates a SecurityContext for a container.
func (m *JobManager) buildContainerSecurityContext() *corev1.SecurityContext {
	sec := m.getSecurityConfig()

	return &corev1.SecurityContext{
		RunAsNonRoot:            &sec.RunAsNonRoot,
		RunAsUser:               &sec.RunAsUser,
		ReadOnlyRootFilesystem:  &sec.ReadOnlyRootFilesystem,
		AllowPrivilegeEscalation: sec.AllowPrivilegeEscalation,
	}
}
