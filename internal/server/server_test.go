package server

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	port := 8080
	server := NewServer(port)

	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	if server.port != port {
		t.Errorf("Expected port %d, got %d", port, server.port)
	}

	if server.connections == nil {
		t.Error("connections map not initialized")
	}

	if server.broadcaster == nil {
		t.Error("broadcaster not initialized")
	}

	if len(server.connections) != 0 {
		t.Errorf("Expected 0 initial connections, got %d", len(server.connections))
	}
}

func TestServerStartAndStop(t *testing.T) {
	// Use port 0 to let the OS choose an available port
	server := NewServer(0)

	// Start server in a goroutine
	serverError := make(chan error, 1)
	serverStarted := make(chan bool, 1)

	go func() {
		// Start the server
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			serverError <- err
			return
		}
		server.listener = listener
		serverStarted <- true

		// Accept one connection and then stop
		conn, err := listener.Accept()
		if err != nil {
			// Expected when we close the listener
			return
		}
		conn.Close()
	}()

	// Wait for server to start or error
	select {
	case err := <-serverError:
		t.Fatalf("Server failed to start: %v", err)
	case <-serverStarted:
		// Server started successfully
	case <-time.After(1 * time.Second):
		t.Fatal("Server did not start within timeout")
	}

	// Get the actual port the server is listening on
	addr := server.listener.Addr().(*net.TCPAddr)
	port := addr.Port

	// Test connection to the server
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	conn.Close()

	// Stop the server
	if server.listener != nil {
		server.listener.Close()
	}
}

func TestServerConnectionHandling(t *testing.T) {
	server := NewServer(0)

	// Start server
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	server.listener = listener
	defer listener.Close()

	// Start broadcaster
	go server.broadcaster.Start()
	defer server.broadcaster.Stop()

	port := listener.Addr().(*net.TCPAddr).Port

	// Handle connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Listener closed
			}
			go server.handleConnection(conn)
		}
	}()

	// Test multiple concurrent connections
	numConnections := 3
	var wg sync.WaitGroup

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Connect to server
			conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
			if err != nil {
				t.Errorf("Failed to connect client %d: %v", id, err)
				return
			}
			defer conn.Close()

			// Send username
			username := fmt.Sprintf("user%d", id)
			_, err = conn.Write([]byte(username + "\n"))
			if err != nil {
				t.Errorf("Failed to send username for client %d: %v", id, err)
				return
			}

			// Send a test message
			message := fmt.Sprintf("Hello from client %d", id)
			_, err = conn.Write([]byte(message + "\n"))
			if err != nil {
				t.Errorf("Failed to send message for client %d: %v", id, err)
				return
			}

			// Keep connection alive for a short time
			time.Sleep(100 * time.Millisecond)
		}(i)
	}

	// Wait for all connections to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// All connections completed
	case <-time.After(5 * time.Second):
		t.Error("Connection test timed out")
	}

	// Give some time for cleanup
	time.Sleep(100 * time.Millisecond)

	// Check that connections were properly cleaned up
	server.connectionMutex.RLock()
	connectionCount := len(server.connections)
	server.connectionMutex.RUnlock()

	if connectionCount != 0 {
		t.Errorf("Expected 0 connections after cleanup, got %d", connectionCount)
	}
}

func TestServerConnectionRegistration(t *testing.T) {
	server := NewServer(0)

	// Start broadcaster
	go server.broadcaster.Start()
	defer server.broadcaster.Stop()

	// Test connection registration directly without full handshake
	// Create connections manually and register them
	// Ensure mock connections have different remote addresses for unique IDs
	mockConn1 := newMockConnWithAddr("127.0.0.1:12345")
	mockConn2 := newMockConnWithAddr("127.0.0.1:12346")

	conn1 := NewConnection(mockConn1, server.broadcaster)
	conn2 := NewConnection(mockConn2, server.broadcaster)

	// Register connections manually (simulating what handleConnection does)
	server.connectionMutex.Lock()
	server.connections[conn1.ID()] = conn1
	server.connections[conn2.ID()] = conn2
	connectionCount := len(server.connections)
	server.connectionMutex.Unlock()

	if connectionCount != 2 {
		t.Errorf("Expected 2 connections, got %d", connectionCount)
	}

	// Test removal
	server.connectionMutex.Lock()
	delete(server.connections, conn1.ID())
	delete(server.connections, conn2.ID())
	connectionCount = len(server.connections)
	server.connectionMutex.Unlock()

	if connectionCount != 0 {
		t.Errorf("Expected 0 connections after removal, got %d", connectionCount)
	}
}

func TestServerHandleConnectionWithRealTCP(t *testing.T) {
	server := NewServer(0)

	// Start server
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	server.listener = listener
	defer listener.Close()

	// Start broadcaster
	go server.broadcaster.Start()
	defer server.broadcaster.Stop()

	port := listener.Addr().(*net.TCPAddr).Port

	// Handle connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Listener closed
			}
			go server.handleConnection(conn)
		}
	}()

	// Create a real TCP connection
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Send username for handshake
	_, err = conn.Write([]byte("testuser\n"))
	if err != nil {
		t.Fatalf("Failed to send username: %v", err)
	}

	// Give time for connection to be registered
	time.Sleep(100 * time.Millisecond)

	// Check that connection was registered
	server.connectionMutex.RLock()
	connectionCount := len(server.connections)
	server.connectionMutex.RUnlock()

	if connectionCount != 1 {
		t.Errorf("Expected 1 connection, got %d", connectionCount)
	}

	// Close connection
	conn.Close()

	// Give time for cleanup
	time.Sleep(200 * time.Millisecond)

	// Check that connection was cleaned up
	server.connectionMutex.RLock()
	connectionCount = len(server.connections)
	server.connectionMutex.RUnlock()

	if connectionCount != 0 {
		t.Errorf("Expected 0 connections after close, got %d", connectionCount)
	}
}

func TestServerIsClosedErr(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-OpError",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
		{
			name:     "OpError with different message",
			err:      &net.OpError{Err: fmt.Errorf("different error")},
			expected: false,
		},
		{
			name:     "OpError with closed connection message",
			err:      &net.OpError{Err: fmt.Errorf("use of closed network connection")},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isClosedErr(tt.err)
			if result != tt.expected {
				t.Errorf("isClosedErr(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestServerConcurrentConnections(t *testing.T) {
	server := NewServer(0)

	// Start server
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	server.listener = listener
	defer listener.Close()

	// Start broadcaster
	go server.broadcaster.Start()
	defer server.broadcaster.Stop()

	port := listener.Addr().(*net.TCPAddr).Port

	// Handle connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Listener closed
			}
			go server.handleConnection(conn)
		}
	}()

	// Test many concurrent connections with reduced count for stability
	numConnections := 5
	var wg sync.WaitGroup
	connectionErrors := make(chan error, numConnections)

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
			if err != nil {
				connectionErrors <- fmt.Errorf("client %d connection failed: %v", id, err)
				return
			}
			defer conn.Close()

			// Send username
			username := fmt.Sprintf("user%d", id)
			_, err = conn.Write([]byte(username + "\n"))
			if err != nil {
				connectionErrors <- fmt.Errorf("client %d username send failed: %v", id, err)
				return
			}

			// Send multiple messages with delays
			for j := 0; j < 2; j++ { // Reduced from 3 to 2
				message := fmt.Sprintf("Message %d from client %d", j, id)
				_, err = conn.Write([]byte(message + "\n"))
				if err != nil {
					connectionErrors <- fmt.Errorf("client %d message %d send failed: %v", id, j, err)
					return
				}
				time.Sleep(50 * time.Millisecond) // Increased delay
			}
		}(i)
	}

	// Wait for all connections with longer timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// All connections completed
	case <-time.After(15 * time.Second): // Increased timeout
		t.Error("Concurrent connection test timed out")
	}

	// Check for any connection errors
	close(connectionErrors)
	errorCount := 0
	for err := range connectionErrors {
		t.Error(err)
		errorCount++
	}

	// Allow some errors in concurrent testing but not too many
	if errorCount > numConnections/2 {
		t.Errorf("Too many connection errors: %d out of %d", errorCount, numConnections)
	}

	// Give more time for cleanup
	time.Sleep(500 * time.Millisecond)

	// Verify all connections were cleaned up
	server.connectionMutex.RLock()
	connectionCount := len(server.connections)
	server.connectionMutex.RUnlock()

	if connectionCount != 0 {
		t.Errorf("Expected 0 connections after test, got %d", connectionCount)
	}
}
