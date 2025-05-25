package server

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockConn implements net.Conn for testing
type mockConn struct {
	readBuffer  *bytes.Buffer
	writeBuffer *bytes.Buffer
	closed      bool
	mutex       sync.RWMutex
	localAddr   net.Addr
	remoteAddr  net.Addr
}

// newMockConnWithAddr creates a new mockConn with a specific remote address string.
func newMockConnWithAddr(remoteAddrStr string) *mockConn {
	return &mockConn{
		readBuffer:  &bytes.Buffer{},
		writeBuffer: &bytes.Buffer{},
		// Assuming the functional mockAddr for this package has .network and .address fields
		localAddr:  &mockAddr{network: "tcp", address: "127.0.0.1:8080"},
		remoteAddr: &mockAddr{network: "tcp", address: remoteAddrStr},
	}
}

// newMockConn provides the original newMockConn functionality for compatibility,
// delegating to newMockConnWithAddr with a default address.
func newMockConn() *mockConn {
	return newMockConnWithAddr("127.0.0.1:12345") // Default remote address
}

func (m *mockConn) Read(b []byte) (int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if m.closed {
		return 0, net.ErrClosed
	}
	return m.readBuffer.Read(b)
}

func (m *mockConn) Write(b []byte) (int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.closed {
		return 0, net.ErrClosed
	}
	return m.writeBuffer.Write(b)
}

func (m *mockConn) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return m.localAddr
}

func (m *mockConn) RemoteAddr() net.Addr {
	return m.remoteAddr
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) WriteString(s string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.readBuffer.WriteString(s)
}

func (m *mockConn) GetWrittenData() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.writeBuffer.String()
}

func (m *mockConn) IsClosed() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.closed
}

func TestNewConnection(t *testing.T) {
	mockConn := newMockConn()
	broadcaster := NewBroadcaster()

	conn := NewConnection(mockConn, broadcaster)

	if conn == nil {
		t.Fatal("NewConnection() returned nil")
	}

	if conn.ID() == "" {
		t.Error("Connection ID should not be empty")
	}

	if conn.Username() != "anonymous" {
		t.Errorf("Expected default username 'anonymous', got '%s'", conn.Username())
	}

	if conn.conn != mockConn {
		t.Error("Connection should store the provided net.Conn")
	}

	if conn.broadcaster != broadcaster {
		t.Error("Connection should store the provided broadcaster")
	}
}

func TestConnectionSend(t *testing.T) {
	mockConn := newMockConn()
	broadcaster := NewBroadcaster()
	conn := NewConnection(mockConn, broadcaster)

	testMessage := "Hello, World!"
	err := conn.Send(testMessage)

	if err != nil {
		t.Errorf("Send() returned error: %v", err)
	}

	written := mockConn.GetWrittenData()
	expected := testMessage + "\n"

	if written != expected {
		t.Errorf("Expected written data '%s', got '%s'", expected, written)
	}
}

func TestConnectionSendToClosedConnection(t *testing.T) {
	mockConn := newMockConn()
	broadcaster := NewBroadcaster()
	conn := NewConnection(mockConn, broadcaster)

	// Close the connection
	mockConn.Close()

	err := conn.Send("test message")

	if err == nil {
		t.Error("Send() should return error when connection is closed")
	}

	if err != net.ErrClosed {
		t.Errorf("Expected net.ErrClosed, got %v", err)
	}
}

func TestConnectionHandshake(t *testing.T) {
	mockConn := newMockConn()
	broadcaster := NewBroadcaster()
	conn := NewConnection(mockConn, broadcaster)

	// Start broadcaster for this test
	go broadcaster.Start()
	defer broadcaster.Stop()

	// Simulate user input for username
	username := "testuser"
	mockConn.WriteString(username + "\n")

	// Start the connection (this will trigger handshake)
	go conn.Start()

	// Give some time for handshake to complete
	time.Sleep(50 * time.Millisecond)

	// Check if username was set
	if conn.Username() != username {
		t.Errorf("Expected username '%s', got '%s'", username, conn.Username())
	}

	// Check written messages
	written := mockConn.GetWrittenData()
	if !strings.Contains(written, "Welcome to the Broadcast Server!") {
		t.Error("Welcome message not found in output")
	}
	if !strings.Contains(written, "Please enter your username:") {
		t.Error("Username prompt not found in output")
	}
	if !strings.Contains(written, fmt.Sprintf("Hello, %s!", username)) {
		t.Error("Confirmation message not found in output")
	}

	// Clean up
	conn.Close()
}

func TestConnectionClose(t *testing.T) {
	mockConn := newMockConn()
	broadcaster := NewBroadcaster()
	conn := NewConnection(mockConn, broadcaster)

	// Start broadcaster for this test
	go broadcaster.Start()
	defer broadcaster.Stop()

	// Register the connection first
	broadcaster.Register(conn)

	// Give some time for registration
	time.Sleep(10 * time.Millisecond)

	// Close the connection
	conn.Close()

	// Check if underlying connection was closed
	if !mockConn.IsClosed() {
		t.Error("Underlying connection should be closed")
	}

	// Check if disconnected channel was closed
	select {
	case <-conn.Disconnected():
		// Channel was closed as expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Disconnected channel was not closed")
	}
}

func TestConnectionMultipleClose(t *testing.T) {
	mockConn := newMockConn()
	broadcaster := NewBroadcaster()
	conn := NewConnection(mockConn, broadcaster)

	// Start broadcaster for this test
	go broadcaster.Start()
	defer broadcaster.Stop()

	// Close multiple times - should not panic
	conn.Close()
	conn.Close()
	conn.Close()

	// Should still be closed
	if !mockConn.IsClosed() {
		t.Error("Connection should remain closed after multiple Close() calls")
	}
}

func TestConnectionEmptyMessages(t *testing.T) {
	mockConn := newMockConn()
	broadcaster := NewBroadcaster()
	conn := NewConnection(mockConn, broadcaster)

	// Start broadcaster for this test
	go broadcaster.Start()
	defer broadcaster.Stop()

	// Set username manually
	conn.username = "testuser"

	// Send empty messages and whitespace
	mockConn.WriteString("\n")
	mockConn.WriteString("   \n")
	mockConn.WriteString("\t\n")
	mockConn.WriteString("actual message\n")

	// Start the connection
	go conn.Start()

	// Give some time for processing
	time.Sleep(50 * time.Millisecond)

	// Clean up
	conn.Close()
}
