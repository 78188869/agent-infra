package handler

import (
	"net/http"

	"github.com/example/agent-infra/internal/monitoring"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WSHandler handles WebSocket connections.
type WSHandler struct {
	hub      *monitoring.Hub
	upgrader websocket.Upgrader
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(hub *monitoring.Hub) *WSHandler {
	return &WSHandler{
		hub: hub,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // TODO: restrict in production with proper origin check
			},
		},
	}
}

// HandleWebSocket upgrades HTTP to WebSocket.
// Token validation: extract API key from ?token= query param.
// Per TRD §7.5.1: ws://{host}/api/v1/ws?token={api_key}
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	// TODO: Validate token and extract tenant_id (auth integration point, Issue #5)
	// For MVP, use placeholder; real auth comes from Issue #5
	tenantID := "tenant-from-token"

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// Register with our custom Hub
	h.hub.Register(tenantID, conn)
	defer h.hub.Unregister(tenantID, conn)

	// Read pump: drain incoming client messages (MVP doesn't process client→server messages)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
