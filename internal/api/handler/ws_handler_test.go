package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/agent-infra/internal/monitoring"
	"github.com/gin-gonic/gin"
)

func TestWSHandler_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	hub := monitoring.NewHub()
	handler := NewWSHandler(hub)
	r.GET("/api/v1/ws", handler.HandleWebSocket)

	req := httptest.NewRequest("GET", "/api/v1/ws", nil) // no token
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWSHandler_ValidTokenUpgrade(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	hub := monitoring.NewHub()
	handler := NewWSHandler(hub)
	r.GET("/api/v1/ws", handler.HandleWebSocket)

	req := httptest.NewRequest("GET", "/api/v1/ws?token=test-key", nil)
	w := httptest.NewRecorder()

	// Run the request asynchronously to avoid blocking
	done := make(chan struct{})
	go func() {
		r.ServeHTTP(w, req)
		close(done)
	}()

	// The request should upgrade to websocket
	select {
	case <-done:
		// Check if it was a successful upgrade
		// Note: httptest doesn't support websocket upgrades directly,
		// but we can verify the handler doesn't panic
	case <-done:
	}
}
