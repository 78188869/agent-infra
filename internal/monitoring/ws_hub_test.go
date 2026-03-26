package monitoring

import (
	"sync"
	"testing"
)

// fakeConn is a fake WebSocket connection implementing WSConn for testing.
type fakeConn struct {
	mu      sync.Mutex
	messages [][]byte
	closed  bool
}

func (f *fakeConn) WriteMessage(_ int, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, data)
	return nil
}

func (f *fakeConn) ReadMessage() (int, []byte, error) {
	return 0, nil, nil
}

func (f *fakeConn) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *fakeConn) Messages() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.messages
}

func TestHub_NewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("NewHub returned nil")
	}
	if hub.clients == nil {
		t.Error("clients map should be initialized")
	}
}

func TestHub_RegisterAndBroadcast(t *testing.T) {
	hub := NewHub()
	conn := &fakeConn{}

	hub.Register("tenant-1", conn)

	if count := hub.ClientCount("tenant-1"); count != 1 {
		t.Errorf("expected 1 client, got %d", count)
	}

	hub.Broadcast("tenant-1", []byte(`{"type":"task.status_changed"}`))

	msgs := conn.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if string(msgs[0]) != `{"type":"task.status_changed"}` {
		t.Errorf("unexpected message: %s", string(msgs[0]))
	}
}

func TestHub_Unregister(t *testing.T) {
	hub := NewHub()
	conn := &fakeConn{}

	hub.Register("tenant-1", conn)
	if count := hub.ClientCount("tenant-1"); count != 1 {
		t.Errorf("expected 1 client, got %d", count)
	}

	hub.Unregister("tenant-1", conn)

	if count := hub.ClientCount("tenant-1"); count != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", count)
	}
}

func TestHub_BroadcastMultipleClients(t *testing.T) {
	hub := NewHub()
	conn1 := &fakeConn{}
	conn2 := &fakeConn{}

	hub.Register("tenant-1", conn1)
	hub.Register("tenant-1", conn2)

	hub.Broadcast("tenant-1", []byte(`{"type":"task.completed"}`))

	if len(conn1.Messages()) != 1 {
		t.Errorf("conn1 expected 1 message, got %d", len(conn1.Messages()))
	}
	if len(conn2.Messages()) != 1 {
		t.Errorf("conn2 expected 1 message, got %d", len(conn2.Messages()))
	}
}

func TestHub_BroadcastDifferentTenants(t *testing.T) {
	hub := NewHub()
	connTenant1 := &fakeConn{}
	connTenant2 := &fakeConn{}

	hub.Register("tenant-1", connTenant1)
	hub.Register("tenant-2", connTenant2)

	// Broadcast only to tenant-1
	hub.Broadcast("tenant-1", []byte(`{"type":"task.status_changed"}`))

	if len(connTenant1.Messages()) != 1 {
		t.Errorf("tenant-1 expected 1 message, got %d", len(connTenant1.Messages()))
	}
	if len(connTenant2.Messages()) != 0 {
		t.Errorf("tenant-2 expected 0 messages, got %d", len(connTenant2.Messages()))
	}
}

func TestHub_BroadcastNoClients(t *testing.T) {
	hub := NewHub()

	// Broadcast to a tenant with no clients — should not panic
	hub.Broadcast("nonexistent-tenant", []byte(`{"type":"test"}`))
}

func TestHub_RegisterUnregisterRace(t *testing.T) {
	hub := NewHub()
	conn := &fakeConn{}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		hub.Register("tenant-1", conn)
		hub.Unregister("tenant-1", conn)
	}()

	go func() {
		defer wg.Done()
		hub.Broadcast("tenant-1", []byte(`{"type":"test"}`))
	}()

	wg.Wait()
}

func TestHub_UnregisterClosesConnection(t *testing.T) {
	hub := NewHub()
	conn := &fakeConn{}

	hub.Register("tenant-1", conn)
	hub.Unregister("tenant-1", conn)

	if !conn.closed {
		t.Error("expected connection to be closed on unregister")
	}
}
