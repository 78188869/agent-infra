package executor

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestNewWrapperClient(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *WrapperClientConfig
		expected int
	}{
		{
			name:     "nil config uses defaults",
			cfg:      nil,
			expected: 9090,
		},
		{
			name: "custom port",
			cfg: &WrapperClientConfig{
				Port: 8080,
			},
			expected: 8080,
		},
		{
			name: "zero port uses default",
			cfg: &WrapperClientConfig{
				Port: 0,
			},
			expected: 9090,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewWrapperClient(tt.cfg)
			if client.port != tt.expected {
				t.Errorf("expected port %d, got %d", tt.expected, client.port)
			}
		})
	}
}

// parseTestServer extracts host and port from test server URL
func parseTestServer(ts *httptest.Server) (host string, port int) {
	// ts.URL is like "http://127.0.0.1:52618"
	h, p, err := net.SplitHostPort(ts.URL[len("http://"):])
	if err != nil {
		return "127.0.0.1", 9090
	}
	portInt, _ := strconv.Atoi(p)
	return h, portInt
}

func TestWrapperClient_Health(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","task_id":"test-123","uptime":100,"version":"1.0.0"}`))
	}))
	defer testServer.Close()

	host, port := parseTestServer(testServer)
	client := NewWrapperClient(&WrapperClientConfig{
		Port:    port,
		Timeout: 5 * time.Second,
	})
	client.httpClient = testServer.Client()

	health, err := client.Health(context.Background(), host)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %s", health.Status)
	}
	if health.TaskID != "test-123" {
		t.Errorf("expected task_id 'test-123', got %s", health.TaskID)
	}
}

func TestWrapperClient_GetStatus(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			t.Errorf("expected /status, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"running","progress":50,"stage":"executing","timestamp":1234567890,"message":"Task is running"}`))
	}))
	defer testServer.Close()

	host, port := parseTestServer(testServer)
	client := NewWrapperClient(&WrapperClientConfig{Port: port})
	client.httpClient = testServer.Client()

	status, err := client.GetStatus(context.Background(), host)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Status != "running" {
		t.Errorf("expected status 'running', got %s", status.Status)
	}
	if status.Progress != 50 {
		t.Errorf("expected progress 50, got %d", status.Progress)
	}
}

func TestWrapperClient_Pause(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pause" {
			t.Errorf("expected /pause, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"message":"Task paused successfully"}`))
	}))
	defer testServer.Close()

	host, port := parseTestServer(testServer)
	client := NewWrapperClient(&WrapperClientConfig{Port: port})
	client.httpClient = testServer.Client()

	err := client.Pause(context.Background(), host)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWrapperClient_Pause_Failed(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":false,"message":"Cannot pause: task not running"}`))
	}))
	defer testServer.Close()

	host, port := parseTestServer(testServer)
	client := NewWrapperClient(&WrapperClientConfig{Port: port})
	client.httpClient = testServer.Client()

	err := client.Pause(context.Background(), host)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWrapperClient_Resume(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/resume" {
			t.Errorf("expected /resume, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"message":"Task resumed successfully"}`))
	}))
	defer testServer.Close()

	host, port := parseTestServer(testServer)
	client := NewWrapperClient(&WrapperClientConfig{Port: port})
	client.httpClient = testServer.Client()

	err := client.Resume(context.Background(), host)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWrapperClient_Inject(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/inject" {
			t.Errorf("expected /inject, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"injected"}`))
	}))
	defer testServer.Close()

	host, port := parseTestServer(testServer)
	client := NewWrapperClient(&WrapperClientConfig{Port: port})
	client.httpClient = testServer.Client()

	err := client.Inject(context.Background(), host, "test instruction content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWrapperClient_ErrorHandling(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer testServer.Close()

	host, port := parseTestServer(testServer)
	client := NewWrapperClient(&WrapperClientConfig{Port: port})
	client.httpClient = testServer.Client()

	_, err := client.Health(context.Background(), host)
	if err == nil {
		t.Error("expected error for 500 status")
	}
}

func TestWrapperClient_Health_WithHostname(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","task_id":"test-123","uptime":100,"version":"1.0.0"}`))
	}))
	defer testServer.Close()

	host, port := parseTestServer(testServer)
	client := NewWrapperClient(&WrapperClientConfig{
		Port:    port,
		Timeout: 5 * time.Second,
	})
	client.httpClient = testServer.Client()

	health, err := client.Health(context.Background(), host)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if health.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %s", health.Status)
	}
}

func TestWrapperClient_InvalidAddress(t *testing.T) {
	client := NewWrapperClient(nil)

	_, err := client.Health(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty address")
	}
}

func TestWrapperClient_Interrupt(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/interrupt" {
			t.Errorf("expected /interrupt, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"interrupted","message":"Task interrupted successfully"}`))
	}))
	defer testServer.Close()

	host, port := parseTestServer(testServer)
	client := NewWrapperClient(&WrapperClientConfig{Port: port})
	client.httpClient = testServer.Client()

	err := client.Interrupt(context.Background(), host)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWrapperClient_Interrupt_Failed(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"error","message":"Cannot interrupt: task not running"}`))
	}))
	defer testServer.Close()

	host, port := parseTestServer(testServer)
	client := NewWrapperClient(&WrapperClientConfig{Port: port})
	client.httpClient = testServer.Client()

	err := client.Interrupt(context.Background(), host)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWrapperClient_Interrupt_ServerError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer testServer.Close()

	host, port := parseTestServer(testServer)
	client := NewWrapperClient(&WrapperClientConfig{Port: port})
	client.httpClient = testServer.Client()

	err := client.Interrupt(context.Background(), host)
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

func TestWrapperClient_Interrupt_InvalidAddress(t *testing.T) {
	client := NewWrapperClient(nil)
	err := client.Interrupt(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty address")
	}
}
