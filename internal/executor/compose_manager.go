package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// ComposeManager manages per-task docker-compose.yml files and lifecycle.
type ComposeManager struct {
	config *DockerConfig
}

// NewComposeManager creates a new ComposeManager instance.
func NewComposeManager(cfg *DockerConfig) (*ComposeManager, error) {
	if cfg == nil {
		cfg = DefaultDockerConfig()
	}
	return &ComposeManager{config: cfg}, nil
}

// GenerateConfig creates a docker-compose.yml for the given task.
func (cm *ComposeManager) GenerateConfig(ctx context.Context, taskID string, envVars map[string]string) error {
	taskDir := filepath.Join(cm.config.ComposeDir, "task-"+taskID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("failed to create task compose directory: %w", err)
	}

	tmpl, err := template.New("compose").Parse(composeTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse compose template: %w", err)
	}

	data := map[string]interface{}{
		"TaskID":         taskID,
		"WorkspaceDir":   cm.config.WorkspaceDir,
		"CLIRunnerImage": cm.config.CLIRunnerImage,
		"WrapperImage":   cm.config.WrapperImage,
		"WrapperPort":    cm.config.WrapperPort,
		"GitRepoURL":     envVars["GIT_REPO_URL"],
		"TaskPrompt":     envVars["TASK_PROMPT"],
	}

	f, err := os.Create(filepath.Join(taskDir, "docker-compose.yml"))
	if err != nil {
		return fmt.Errorf("failed to create compose file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute compose template: %w", err)
	}

	return nil
}

// TaskDir returns the compose directory path for a task.
func (cm *ComposeManager) TaskDir(taskID string) string {
	return filepath.Join(cm.config.ComposeDir, "task-"+taskID)
}

const composeTemplate = `services:
  cli-runner:
    image: {{.CLIRunnerImage}}
    volumes:
      - {{.WorkspaceDir}}/{{.TaskID}}:/workspace
    environment:
      - TASK_ID={{.TaskID}}
      - GIT_REPO_URL={{.GitRepoURL}}
      - TASK_PROMPT={{.TaskPrompt}}
      - AGENT_STATE_DIR=/workspace/.agent-state

  wrapper:
    image: {{.WrapperImage}}
    ports:
      - "{{.WrapperPort}}"
    volumes:
      - {{.WorkspaceDir}}/{{.TaskID}}:/workspace
    environment:
      - TASK_ID={{.TaskID}}
      - SHARED_STATE_DIR=/workspace/.agent-state
    depends_on:
      - cli-runner
`
