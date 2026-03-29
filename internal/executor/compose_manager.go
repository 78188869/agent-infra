package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// GenerateSingleContainerConfig creates a docker-compose.yml for the
// single-container Agent SDK wrapper mode. Unlike GenerateConfig, this
// produces a compose file with only the wrapper service that combines
// CLI runner and wrapper functionality.
func (cm *ComposeManager) GenerateSingleContainerConfig(ctx context.Context, data *SingleContainerTemplateData) error {
	if data == nil {
		return fmt.Errorf("template data must not be nil")
	}

	taskDir := filepath.Join(cm.config.ComposeDir, "task-"+data.TaskID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("failed to create task compose directory: %w", err)
	}

	tmpl, err := template.New("single-container-compose").Parse(singleContainerComposeTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse single-container compose template: %w", err)
	}

	// Apply defaults from ComposeManager config.
	templateData := map[string]string{
		"TaskID":          data.TaskID,
		"WrapperImage":    data.WrapperImage,
		"WorkspaceDir":    data.WorkspaceDir,
		"ControlPlaneURL": data.ControlPlaneURL,
		"AnthropicAPIKey": data.AnthropicAPIKey,
		"TaskPrompt":      data.TaskPrompt,
		"MaxTimeout":      data.MaxTimeout,
		"GitRepo":         data.GitRepo,
		"GitBranch":       data.GitBranch,
		"ClaudeMdContent": data.ClaudeMdContent,
		"AllowedTools":    data.AllowedTools,
	}

	composeFile := filepath.Join(taskDir, "docker-compose.yml")
	f, err := os.Create(composeFile)
	if err != nil {
		return fmt.Errorf("failed to create compose file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, templateData); err != nil {
		return fmt.Errorf("failed to execute single-container compose template: %w", err)
	}

	// Restrict compose file permissions since it contains sensitive data (API key)
	if err := os.Chmod(composeFile, 0600); err != nil {
		return fmt.Errorf("failed to set compose file permissions: %w", err)
	}

	return nil
}

// TaskDir returns the compose directory path for a task.
func (cm *ComposeManager) TaskDir(taskID string) string {
	return filepath.Join(cm.config.ComposeDir, "task-"+taskID)
}

// Up starts the containers for the given task.
func (cm *ComposeManager) Up(ctx context.Context, taskID string) error {
	return cm.composeCommand(ctx, taskID, "up", "-d")
}

// Down stops and removes containers for the given task.
func (cm *ComposeManager) Down(ctx context.Context, taskID string) error {
	err := cm.composeCommand(ctx, taskID, "down")
	// Cleanup task directory after successful down
	if err == nil {
		os.RemoveAll(cm.TaskDir(taskID))
	}
	return err
}

// DockerServiceStatus represents a container status from docker compose ps.
type DockerServiceStatus struct {
	ID      string `json:"ID"`
	Name    string `json:"Name"`
	State   string `json:"State"`   // running, exited, paused, dead
	Health  string `json:"Health"`
	Service string `json:"Service"`
}

// GetStatus returns the status of containers for the given task.
func (cm *ComposeManager) GetStatus(ctx context.Context, taskID string) ([]DockerServiceStatus, error) {
	out, err := cm.composeOutput(ctx, taskID, "ps", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to get compose status: %w", err)
	}

	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return nil, nil
	}

	// Docker Compose v2 returns a JSON array; try array parse first.
	var statuses []DockerServiceStatus
	if err := json.Unmarshal([]byte(trimmed), &statuses); err == nil {
		return statuses, nil
	}

	// Fallback: JSON Lines (one object per line) for older versions.
	for _, line := range splitLines(out) {
		if line == "" {
			continue
		}
		var s DockerServiceStatus
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			continue // skip unparseable lines
		}
		statuses = append(statuses, s)
	}
	return statuses, nil
}

// GetServicePort returns the mapped host port for a service.
func (cm *ComposeManager) GetServicePort(ctx context.Context, taskID string, service string, port int) (int, error) {
	out, err := cm.composeOutput(ctx, taskID, "port", service, fmt.Sprintf("%d", port))
	if err != nil {
		return 0, fmt.Errorf("failed to get service port: %w", err)
	}
	// Output format: "0.0.0.0:32768" or "[::]:32768"
	var hostPort int
	if _, err := fmt.Sscanf(out, "%*s:%d", &hostPort); err != nil {
		return 0, fmt.Errorf("failed to parse port from %q: %w", out, err)
	}
	if hostPort == 0 {
		return 0, fmt.Errorf("no port mapping found for %s:%d", service, port)
	}
	return hostPort, nil
}

// composeCommand runs a docker compose command in the task's directory.
func (cm *ComposeManager) composeCommand(ctx context.Context, taskID string, args ...string) error {
	cmd := cm.buildComposeCmd(ctx, taskID, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose %v failed: %w\noutput: %s", args, err, string(out))
	}
	return nil
}

// composeOutput runs a docker compose command and returns stdout.
func (cm *ComposeManager) composeOutput(ctx context.Context, taskID string, args ...string) (string, error) {
	cmd := cm.buildComposeCmd(ctx, taskID, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker compose %v failed: %w\nstderr: %s", args, err, stderr.String())
	}
	return stdout.String(), nil
}

// buildComposeCmd constructs an exec.Cmd for docker compose.
func (cm *ComposeManager) buildComposeCmd(ctx context.Context, taskID string, args ...string) *exec.Cmd {
	allArgs := append([]string{"compose", "-f", filepath.Join(cm.TaskDir(taskID), "docker-compose.yml")}, args...)
	return exec.CommandContext(ctx, "docker", allArgs...)
}

// splitLines splits output into non-empty trimmed lines.
func splitLines(s string) []string {
	var lines []string
	for _, l := range bytes.Split([]byte(s), []byte("\n")) {
		trimmed := string(bytes.TrimSpace(l))
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
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

// singleContainerComposeTemplate defines a single-container compose config
// for the Agent SDK wrapper mode where CLI runner and wrapper are combined.
const singleContainerComposeTemplate = `services:
  wrapper:
    image: {{.WrapperImage}}
    ports:
      - "9090"
    volumes:
      - {{.WorkspaceDir}}/{{.TaskID}}:/workspace
    environment:
      - TASK_ID={{.TaskID}}
      - CONTROL_PLANE_URL={{.ControlPlaneURL}}
      - ANTHROPIC_API_KEY={{.AnthropicAPIKey}}
      - TASK_PROMPT={{.TaskPrompt}}
      - MAX_TIMEOUT={{.MaxTimeout}}
      - WORKSPACE_DIR=/workspace
      - GIT_REPO={{.GitRepo}}
      - GIT_BRANCH={{.GitBranch}}
      - CLAUDE_MD_CONTENT={{.ClaudeMdContent}}
      - ALLOWED_TOOLS={{.AllowedTools}}
`

// SingleContainerTemplateData holds the data for rendering a single-container
// docker-compose.yml template used by the Agent SDK wrapper mode.
type SingleContainerTemplateData struct {
	TaskID          string
	WrapperImage    string
	WorkspaceDir    string
	ControlPlaneURL string
	AnthropicAPIKey string
	TaskPrompt      string
	MaxTimeout      string
	GitRepo         string
	GitBranch       string
	ClaudeMdContent string
	AllowedTools    string
}
