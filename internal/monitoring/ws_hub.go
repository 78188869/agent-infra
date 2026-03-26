package monitoring

import (
	"sync"

	"github.com/gorilla/websocket"
)

// WSConn defines the interface for a WebSocket connection used by the Hub.
// This allows for easy testing via mock implementations.
type WSConn interface {
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
}

// Hub maintains the set of active clients per tenant and broadcasts messages.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[WSConn]struct{}
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[WSConn]struct{}),
	}
}

// Register adds a WebSocket connection to the tenant room.
func (h *Hub) Register(tenantID string, conn WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[tenantID] == nil {
		h.clients[tenantID] = make(map[WSConn]struct{})
	}
	h.clients[tenantID][conn] = struct{}{}
}

// Unregister removes a WebSocket connection from the tenant room.
func (h *Hub) Unregister(tenantID string, conn WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.clients[tenantID]; ok {
		delete(room, conn)
		if len(room) == 0 {
			delete(h.clients, tenantID)
		}
	}
	if conn != nil {
		conn.Close()
	}
}

// Broadcast sends a message to all clients in the tenant room.
func (h *Hub) Broadcast(tenantID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.clients[tenantID] {
		conn.WriteMessage(websocket.TextMessage, msg)
	}
}

// ClientCount returns the number of connected clients for a tenant.
func (h *Hub) ClientCount(tenantID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[tenantID])
}
