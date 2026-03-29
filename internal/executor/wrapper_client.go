package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// WrapperClient provides HTTP client for communicating with the Wrapper sidecar.
type WrapperClient struct {
	httpClient *http.Client
	port       int
	timeout    time.Duration
}

// WrapperClientConfig holds configuration for WrapperClient.
type WrapperClientConfig struct {
	Port    int
	Timeout time.Duration
}

// isValidAddress validates that the given string is a non-empty address.
func isValidAddress(addr string) bool {
	return addr != ""
}

// NewWrapperClient creates a new WrapperClient instance.
func NewWrapperClient(cfg *WrapperClientConfig) *WrapperClient {
	if cfg == nil {
		cfg = &WrapperClientConfig{
			Port:    9090,
			Timeout: 10 * time.Second,
		}
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.Port == 0 {
		cfg.Port = 9090
	}

	return &WrapperClient{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		port:    cfg.Port,
		timeout: cfg.Timeout,
	}
}

// WrapperStatus represents the status response from Wrapper.
type WrapperStatus struct {
	Status    string `json:"status"`     // running, paused, completed, failed
	Progress  int    `json:"progress"`   // 0-100
	Stage     string `json:"stage"`      // Current execution stage
	Timestamp int64  `json:"timestamp"`  // Unix timestamp
	Message   string `json:"message"`    // Human-readable message
}

// WrapperHealth represents the health check response.
type WrapperHealth struct {
	Status  string `json:"status"`  // healthy, unhealthy
	TaskID  string `json:"task_id"` // Current task ID
	Uptime  int64  `json:"uptime"`  // Uptime in seconds
	Version string `json:"version"` // Wrapper version
}

// PauseResponse represents the response from pause operation.
type PauseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ResumeResponse represents the response from resume operation.
type ResumeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// InjectRequest represents a request to inject instructions.
type InjectRequest struct {
	Prompt string `json:"prompt"`
}

// InjectResponse represents the response from inject operation.
type InjectResponse struct {
	Status  string `json:"status"`
	Success bool   `json:"success,omitempty"`
}

// Health checks the health of the Wrapper.
func (c *WrapperClient) Health(ctx context.Context, address string) (*WrapperHealth, error) {
	if !isValidAddress(address) {
		return nil, fmt.Errorf("invalid runtime address: %s", address)
	}
	url := fmt.Sprintf("http://%s:%d/health", address, c.port)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	defer drainAndClose(resp.Body, "Health")

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	var health WrapperHealth
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}

	return &health, nil
}

// GetStatus gets the current status from Wrapper.
func (c *WrapperClient) GetStatus(ctx context.Context, address string) (*WrapperStatus, error) {
	if !isValidAddress(address) {
		return nil, fmt.Errorf("invalid runtime address: %s", address)
	}
	url := fmt.Sprintf("http://%s:%d/status", address, c.port)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("get status failed: %w", err)
	}
	defer drainAndClose(resp.Body, "GetStatus")

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get status returned status %d", resp.StatusCode)
	}

	var status WrapperStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return &status, nil
}

// Pause sends a pause request to the Wrapper.
func (c *WrapperClient) Pause(ctx context.Context, address string) error {
	if !isValidAddress(address) {
		return fmt.Errorf("invalid runtime address: %s", address)
	}
	url := fmt.Sprintf("http://%s:%d/pause", address, c.port)
	resp, err := c.doRequest(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("pause request failed: %w", err)
	}
	defer drainAndClose(resp.Body, "Pause")

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pause returned status %d", resp.StatusCode)
	}

	var pauseResp PauseResponse
	if err := json.NewDecoder(resp.Body).Decode(&pauseResp); err != nil {
		return fmt.Errorf("failed to decode pause response: %w", err)
	}

	if !pauseResp.Success {
		return fmt.Errorf("pause failed: %s", pauseResp.Message)
	}

	return nil
}

// Resume sends a resume request to the Wrapper.
//
// Deprecated: For Agent SDK wrapper, use InjectInstruction instead of
// Pause/Resume, as the SDK does not support pause semantics. Resume is
// retained for backward compatibility with the CLI runner wrapper.
func (c *WrapperClient) Resume(ctx context.Context, address string) error {
	if !isValidAddress(address) {
		return fmt.Errorf("invalid runtime address: %s", address)
	}
	url := fmt.Sprintf("http://%s:%d/resume", address, c.port)
	resp, err := c.doRequest(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("resume request failed: %w", err)
	}
	defer drainAndClose(resp.Body, "Resume")

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("resume returned status %d", resp.StatusCode)
	}

	var resumeResp ResumeResponse
	if err := json.NewDecoder(resp.Body).Decode(&resumeResp); err != nil {
		return fmt.Errorf("failed to decode resume response: %w", err)
	}

	if !resumeResp.Success {
		return fmt.Errorf("resume failed: %s", resumeResp.Message)
	}

	return nil
}

// Inject sends an instruction injection request to the Wrapper.
func (c *WrapperClient) Inject(ctx context.Context, address string, content string) error {
	if !isValidAddress(address) {
		return fmt.Errorf("invalid runtime address: %s", address)
	}
	req := &InjectRequest{
		Prompt: content,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal inject request: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/inject", address, c.port)
	resp, err := c.doRequest(ctx, "POST", url, body)
	if err != nil {
		return fmt.Errorf("inject request failed: %w", err)
	}
	defer drainAndClose(resp.Body, "Inject")

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("inject returned status %d", resp.StatusCode)
	}

	var injectResp InjectResponse
	if err := json.NewDecoder(resp.Body).Decode(&injectResp); err != nil {
		return fmt.Errorf("failed to decode inject response: %w", err)
	}

	if injectResp.Status != "injected" {
		return fmt.Errorf("inject failed: unexpected status %s", injectResp.Status)
	}

	return nil
}

// InterruptResponse represents the response from interrupt operation.
type InterruptResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Interrupt sends an interrupt request to the Wrapper, signaling it to stop
// the current execution gracefully. This is used with the single-container
// Agent SDK wrapper where Pause/Resume is not applicable.
func (c *WrapperClient) Interrupt(ctx context.Context, address string) error {
	if !isValidAddress(address) {
		return fmt.Errorf("invalid runtime address: %s", address)
	}
	url := fmt.Sprintf("http://%s:%d/interrupt", address, c.port)
	resp, err := c.doRequest(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("interrupt request failed: %w", err)
	}
	defer drainAndClose(resp.Body, "Interrupt")

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("interrupt returned status %d", resp.StatusCode)
	}

	var interruptResp InterruptResponse
	if err := json.NewDecoder(resp.Body).Decode(&interruptResp); err != nil {
		return fmt.Errorf("failed to decode interrupt response: %w", err)
	}

	if interruptResp.Status != "interrupted" {
		return fmt.Errorf("interrupt failed: %s", interruptResp.Message)
	}

	return nil
}

// doRequest performs an HTTP request.
func (c *WrapperClient) doRequest(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// drainAndClose drains the response body and closes it, logging any errors.
// This is important for connection reuse - if the body is not fully drained,
// the connection cannot be reused for subsequent requests.
func drainAndClose(body io.ReadCloser, methodName string) {
	if body == nil {
		return
	}
	_, drainErr := io.Copy(io.Discard, body)
	if drainErr != nil {
		slog.Warn("failed to drain response body, connection reuse may be affected",
			"method", methodName,
			"error", drainErr,
		)
	}
	if closeErr := body.Close(); closeErr != nil {
		slog.Warn("failed to close response body",
			"method", methodName,
			"error", closeErr,
		)
	}
}
