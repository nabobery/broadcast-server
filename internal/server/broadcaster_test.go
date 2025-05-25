package server

import (
	"net"
	"sync"
	"testing"
	"time"
)

// mockConnection implements a mock connection for testing
type mockConnection struct {
	id       string
	username string
	messages []string
	closed   bool
	mutex    sync.RWMutex
}

func newMockConnection(id, username string) *mockConnection {
	return &mockConnection{
		id:       id,
		username: username,
		messages: make([]string, 0),
	}
}

func (m *mockConnection) ID() string {
	return m.id
}

func (m *mockConnection) Username() string {
	return m.username
}

func (m *mockConnection) Send(message string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.closed {
		return net.ErrClosed
	}
	m.messages = append(m.messages, message)
	return nil
}

func (m *mockConnection) GetMessages() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	result := make([]string, len(m.messages))
	copy(result, m.messages)
	return result
}

func (m *mockConnection) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.closed = true
	return nil
}

func (m *mockConnection) IsClosed() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.closed
}

// Implement net.Conn interface methods
func (m *mockConnection) Read(b []byte) (int, error) {
	return 0, nil
}

func (m *mockConnection) Write(b []byte) (int, error) {
	return len(b), nil
}

func (m *mockConnection) LocalAddr() net.Addr {
	return &mockAddr{network: "tcp", address: "127.0.0.1:8080"}
}

func (m *mockConnection) RemoteAddr() net.Addr {
	return &mockAddr{network: "tcp", address: "127.0.0.1:12345"}
}

func (m *mockConnection) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConnection) SetWriteDeadline(t time.Time) error {
	return nil
}

// mockAddr implements net.Addr for testing
type mockAddr struct {
	network string
	address string
}

func (m *mockAddr) Network() string {
	return m.network
}

func (m *mockAddr) String() string {
	return m.address
}

func TestNewBroadcaster(t *testing.T) {
	broadcaster := NewBroadcaster()

	if broadcaster == nil {
		t.Fatal("NewBroadcaster() returned nil")
	}

	if broadcaster.connections == nil {
		t.Error("connections map not initialized")
	}

	if broadcaster.register == nil {
		t.Error("register channel not initialized")
	}

	if broadcaster.unregister == nil {
		t.Error("unregister channel not initialized")
	}

	if broadcaster.broadcast == nil {
		t.Error("broadcast channel not initialized")
	}

	if broadcaster.quit == nil {
		t.Error("quit channel not initialized")
	}
}

func TestBroadcasterRegisterUnregister(t *testing.T) {
	broadcaster := NewBroadcaster()

	// Start broadcaster in a goroutine
	go broadcaster.Start()
	defer broadcaster.Stop()

	// Create mock connections
	conn1 := newMockConnection("conn1", "user1")
	_ = newMockConnection("conn2", "user2") // unused in this test

	// Convert mock connections to Connection interface
	// Note: We need to create actual Connection objects for this test
	// For now, we'll test the broadcaster channels directly

	// Test registration
	go func() {
		broadcaster.register <- &Connection{id: conn1.ID(), username: conn1.Username()}
	}()

	// Give some time for registration
	time.Sleep(10 * time.Millisecond)

	// Check if connection was registered
	broadcaster.connectionMutex.RLock()
	connectionCount := len(broadcaster.connections)
	broadcaster.connectionMutex.RUnlock()

	if connectionCount != 1 {
		t.Errorf("Expected 1 connection, got %d", connectionCount)
	}
}

func TestBroadcasterBroadcast(t *testing.T) {
	broadcaster := NewBroadcaster()

	// Start broadcaster in a goroutine
	go broadcaster.Start()
	defer broadcaster.Stop()

	// Create a test message
	testMessage := "Hello, World!"

	// Test broadcasting (this will test the channel mechanism)
	done := make(chan bool)
	go func() {
		broadcaster.Broadcast(testMessage)
		done <- true
	}()

	// Wait for broadcast to complete or timeout
	select {
	case <-done:
		// Broadcast completed successfully
	case <-time.After(100 * time.Millisecond):
		t.Error("Broadcast operation timed out")
	}
}

func TestBroadcasterStop(t *testing.T) {
	broadcaster := NewBroadcaster()

	// Start broadcaster in a goroutine
	started := make(chan bool)
	stopped := make(chan bool)

	go func() {
		started <- true
		broadcaster.Start()
		stopped <- true
	}()

	// Wait for broadcaster to start
	<-started

	// Stop the broadcaster
	broadcaster.Stop()

	// Wait for broadcaster to stop or timeout
	select {
	case <-stopped:
		// Broadcaster stopped successfully
	case <-time.After(100 * time.Millisecond):
		t.Error("Broadcaster did not stop within timeout")
	}
}

func TestBroadcasterConcurrentOperations(t *testing.T) {
	broadcaster := NewBroadcaster()

	// Start broadcaster
	go broadcaster.Start()
	defer broadcaster.Stop()

	// Number of concurrent operations
	numOperations := 10
	var wg sync.WaitGroup

	// Test concurrent registrations
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()
			mockConn := newMockConnection(string(rune('A'+id)), "user"+string(rune('0'+id)))
			conn := &Connection{
				id:           mockConn.ID(),
				username:     mockConn.Username(),
				conn:         mockConn, // Use the mock connection
				broadcaster:  broadcaster,
				disconnected: make(chan struct{}),
			}
			broadcaster.register <- conn
		}(i)
	}

	// Test concurrent broadcasts
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()
			broadcaster.Broadcast("Message " + string(rune('0'+id)))
		}(i)
	}

	// Wait for all operations to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// All operations completed
	case <-time.After(1 * time.Second):
		t.Error("Concurrent operations timed out")
	}

	// Give some time for all operations to be processed
	time.Sleep(50 * time.Millisecond)

	// Check final state
	broadcaster.connectionMutex.RLock()
	connectionCount := len(broadcaster.connections)
	broadcaster.connectionMutex.RUnlock()

	if connectionCount != numOperations {
		t.Errorf("Expected %d connections, got %d", numOperations, connectionCount)
	}
}
