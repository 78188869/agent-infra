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
func (cm *ComposeManager) GenerateConfig(ctx context.Context, data *ComposeTemplateData) error {
	if data == nil {
		return fmt.Errorf("template data must not be nil")
	}

	if data.WrapperImage == "" {
		data.WrapperImage = cm.config.WrapperImage
	}
	if data.WorkspaceDir == "" {
		data.WorkspaceDir = cm.config.WorkspaceDir
	}

	taskDir := filepath.Join(cm.config.ComposeDir, "task-"+data.TaskID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("failed to create task compose directory: %w", err)
	}

	tmpl, err := template.New("compose").Parse(composeTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse compose template: %w", err)
	}

	templateData := map[string]string{
		"TaskID":          data.TaskID,
		"WrapperImage":    data.WrapperImage,
		"WorkspaceDir":    data.WorkspaceDir,
		"ControlPlaneURL": data.ControlPlaneURL,
		"AnthropicAPIKey": data.AnthropicAPIKey,
		"TaskPrompt":      yamlQuote(data.TaskPrompt),
		"MaxTimeout":      data.MaxTimeout,
		"GitRepo":         data.GitRepo,
		"GitBranch":       data.GitBranch,
		"ClaudeMdContent": yamlQuote(data.ClaudeMdContent),
		"AllowedTools":    data.AllowedTools,
	}

	composeFile := filepath.Join(taskDir, "docker-compose.yml")
	f, err := os.Create(composeFile)
	if err != nil {
		return fmt.Errorf("failed to create compose file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, templateData); err != nil {
		return fmt.Errorf("failed to execute compose template: %w", err)
	}

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

// GetExitCode returns the exit code of the first exited container for the task.
// Returns 0 if containers are still running or exit code is unavailable.
func (cm *ComposeManager) GetExitCode(ctx context.Context, taskID string) (int, error) {
	statuses, err := cm.GetStatus(ctx, taskID)
	if err != nil {
		return 0, fmt.Errorf("failed to get status for exit code: %w", err)
	}
	for _, s := range statuses {
		if s.State == "exited" {
			cmd := exec.CommandContext(ctx, "docker", "inspect",
				"--format", "{{.State.ExitCode}}", s.ID)
			out, err := cmd.Output()
			if err != nil {
				return 1, nil
			}
			code := strings.TrimSpace(string(out))
			if code == "0" {
				return 0, nil
			}
			var exitCode int
			if _, err := fmt.Sscanf(code, "%d", &exitCode); err == nil && exitCode != 0 {
				return exitCode, nil
			}
			return 1, nil
		}
	}
	return 0, nil
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

// yamlQuote wraps a string in single quotes for safe YAML embedding.
func yamlQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

const composeTemplate = `services:
  sandbox:
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

// ComposeTemplateData holds the data for rendering a docker-compose.yml template.
type ComposeTemplateData struct {
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
