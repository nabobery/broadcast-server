package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestClient represents a test client for integration testing
type TestClient struct {
	conn     net.Conn
	username string
	messages []string
	mutex    sync.RWMutex
	done     chan bool
}

func NewTestClient(address, username string) (*TestClient, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	client := &TestClient{
		conn:     conn,
		username: username,
		messages: make([]string, 0),
		done:     make(chan bool),
	}

	// Start reading messages
	go client.readMessages()

	return client, nil
}

func (tc *TestClient) readMessages() {
	scanner := bufio.NewScanner(tc.conn)
	for scanner.Scan() {
		message := scanner.Text()
		tc.mutex.Lock()
		tc.messages = append(tc.messages, message)
		tc.mutex.Unlock()
	}
	close(tc.done)
}

func (tc *TestClient) SendMessage(message string) error {
	_, err := tc.conn.Write([]byte(message + "\n"))
	return err
}

func (tc *TestClient) GetMessages() []string {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	result := make([]string, len(tc.messages))
	copy(result, tc.messages)
	return result
}

func (tc *TestClient) Close() error {
	return tc.conn.Close()
}

func (tc *TestClient) WaitForDisconnect() {
	<-tc.done
}

func (tc *TestClient) WaitForMessage(contains string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		tc.mutex.RLock()
		for _, msg := range tc.messages {
			if strings.Contains(msg, contains) {
				tc.mutex.RUnlock()
				return true
			}
		}
		tc.mutex.RUnlock()
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestIntegrationSingleClient(t *testing.T) {
	// Start server
	server := NewServer(0)
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	server.listener = listener
	defer listener.Close()

	// Start broadcaster
	go server.broadcaster.Start()
	defer server.broadcaster.Stop()

	// Start accepting connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go server.handleConnection(conn)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	address := fmt.Sprintf("localhost:%d", port)

	// Create test client
	client, err := NewTestClient(address, "testuser")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}
	defer client.Close()

	// Wait for welcome message with longer timeout
	if !client.WaitForMessage("Welcome to the Broadcast Server!", 3*time.Second) {
		t.Error("Did not receive welcome message")
		t.Logf("Received messages: %v", client.GetMessages())
	}

	// Send username
	err = client.SendMessage("testuser")
	if err != nil {
		t.Fatalf("Failed to send username: %v", err)
	}

	// Wait for confirmation with longer timeout
	if !client.WaitForMessage("Hello, testuser!", 3*time.Second) {
		t.Error("Did not receive username confirmation")
		t.Logf("Received messages: %v", client.GetMessages())
	}

	// Send a test message
	testMessage := "Hello, World!"
	err = client.SendMessage(testMessage)
	if err != nil {
		t.Fatalf("Failed to send test message: %v", err)
	}

	// Wait for the message to be broadcasted back with longer timeout
	if !client.WaitForMessage("testuser: "+testMessage, 3*time.Second) {
		t.Error("Did not receive broadcasted message")
		t.Logf("Received messages: %v", client.GetMessages())
	}
}

func TestIntegrationMultipleClients(t *testing.T) {
	// Start server
	server := NewServer(0)
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	server.listener = listener
	defer listener.Close()

	// Start broadcaster
	go server.broadcaster.Start()
	defer server.broadcaster.Stop()

	// Start accepting connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go server.handleConnection(conn)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	address := fmt.Sprintf("localhost:%d", port)

	// Create multiple test clients
	numClients := 3
	clients := make([]*TestClient, numClients)

	for i := 0; i < numClients; i++ {
		username := fmt.Sprintf("user%d", i)
		client, err := NewTestClient(address, username)
		if err != nil {
			t.Fatalf("Failed to create test client %d: %v", i, err)
		}
		clients[i] = client
		defer client.Close()

		// Complete handshake
		client.WaitForMessage("Welcome to the Broadcast Server!", 1*time.Second)
		client.SendMessage(username)
		client.WaitForMessage(fmt.Sprintf("Hello, %s!", username), 1*time.Second)
	}

	// Give some time for all clients to connect
	time.Sleep(100 * time.Millisecond)

	// Test message broadcasting
	testMessage := "Hello from user0!"
	err = clients[0].SendMessage(testMessage)
	if err != nil {
		t.Fatalf("Failed to send message from client 0: %v", err)
	}

	// All clients should receive the message
	expectedMessage := "user0: " + testMessage
	for i, client := range clients {
		if !client.WaitForMessage(expectedMessage, 2*time.Second) {
			t.Errorf("Client %d did not receive broadcasted message", i)
			t.Logf("Client %d messages: %v", i, client.GetMessages())
		}
	}

	// Test multiple messages from different clients
	for i, client := range clients {
		message := fmt.Sprintf("Message from user%d", i)
		err = client.SendMessage(message)
		if err != nil {
			t.Errorf("Failed to send message from client %d: %v", i, err)
			continue
		}

		// All clients should receive this message
		expectedMsg := fmt.Sprintf("user%d: %s", i, message)
		for j, otherClient := range clients {
			if !otherClient.WaitForMessage(expectedMsg, 2*time.Second) {
				t.Errorf("Client %d did not receive message from client %d", j, i)
			}
		}
	}
}

func TestIntegrationClientDisconnection(t *testing.T) {
	// Start server
	server := NewServer(0)
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	server.listener = listener
	defer listener.Close()

	// Start broadcaster
	go server.broadcaster.Start()
	defer server.broadcaster.Stop()

	// Start accepting connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go server.handleConnection(conn)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	address := fmt.Sprintf("localhost:%d", port)

	// Create two clients
	client1, err := NewTestClient(address, "user1")
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()

	client2, err := NewTestClient(address, "user2")
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	// Complete handshakes
	client1.WaitForMessage("Welcome to the Broadcast Server!", 1*time.Second)
	client1.SendMessage("user1")
	client1.WaitForMessage("Hello, user1!", 1*time.Second)

	client2.WaitForMessage("Welcome to the Broadcast Server!", 1*time.Second)
	client2.SendMessage("user2")
	client2.WaitForMessage("Hello, user2!", 1*time.Second)

	// Both clients should see join messages
	client1.WaitForMessage("System: user2 joined the chat", 1*time.Second)
	client2.WaitForMessage("System: user1 joined the chat", 1*time.Second)

	// Disconnect client1
	client1.Close()

	// Client2 should see the disconnect message
	if !client2.WaitForMessage("System: user1 left the chat", 2*time.Second) {
		t.Error("Client2 did not receive disconnect message for user1")
		t.Logf("Client2 messages: %v", client2.GetMessages())
	}

	// Verify server cleaned up the connection
	time.Sleep(100 * time.Millisecond)
	server.connectionMutex.RLock()
	connectionCount := len(server.connections)
	server.connectionMutex.RUnlock()

	if connectionCount != 1 {
		t.Errorf("Expected 1 connection after disconnect, got %d", connectionCount)
	}
}

func TestIntegrationConcurrentMessages(t *testing.T) {
	// Start server
	server := NewServer(0)
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	server.listener = listener
	defer listener.Close()

	// Start broadcaster
	go server.broadcaster.Start()
	defer server.broadcaster.Stop()

	// Start accepting connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go server.handleConnection(conn)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	address := fmt.Sprintf("localhost:%d", port)

	// Create multiple clients
	numClients := 5
	clients := make([]*TestClient, numClients)

	for i := 0; i < numClients; i++ {
		username := fmt.Sprintf("user%d", i)
		client, err := NewTestClient(address, username)
		if err != nil {
			t.Fatalf("Failed to create client %d: %v", i, err)
		}
		clients[i] = client
		defer client.Close()

		// Complete handshake
		client.WaitForMessage("Welcome to the Broadcast Server!", 1*time.Second)
		client.SendMessage(username)
		client.WaitForMessage(fmt.Sprintf("Hello, %s!", username), 1*time.Second)
	}

	// Give time for all connections to be established
	time.Sleep(200 * time.Millisecond)

	// Send concurrent messages
	numMessages := 10
	var wg sync.WaitGroup

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientIndex int) {
			defer wg.Done()
			client := clients[clientIndex]

			for j := 0; j < numMessages; j++ {
				message := fmt.Sprintf("Message %d from user%d", j, clientIndex)
				err := client.SendMessage(message)
				if err != nil {
					t.Errorf("Failed to send message from client %d: %v", clientIndex, err)
				}
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Wait for all messages to be sent
	wg.Wait()

	// Give time for all messages to be processed
	time.Sleep(500 * time.Millisecond)

	// Verify that all clients received all messages
	totalExpectedMessages := numClients * numMessages
	for i, client := range clients {
		messages := client.GetMessages()

		// Count actual chat messages (exclude system messages)
		chatMessageCount := 0
		for _, msg := range messages {
			if strings.Contains(msg, ": Message") {
				chatMessageCount++
			}
		}

		if chatMessageCount < totalExpectedMessages {
			t.Errorf("Client %d received %d chat messages, expected at least %d",
				i, chatMessageCount, totalExpectedMessages)
		}
	}
}

func TestIntegrationEmptyUsername(t *testing.T) {
	// Start server
	server := NewServer(0)
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	server.listener = listener
	defer listener.Close()

	// Start broadcaster
	go server.broadcaster.Start()
	defer server.broadcaster.Stop()

	// Start accepting connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go server.handleConnection(conn)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	address := fmt.Sprintf("localhost:%d", port)

	// Create test client
	client, err := NewTestClient(address, "")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}
	defer client.Close()

	// Wait for welcome message
	client.WaitForMessage("Welcome to the Broadcast Server!", 1*time.Second)

	// Send empty username
	err = client.SendMessage("")
	if err != nil {
		t.Fatalf("Failed to send empty username: %v", err)
	}

	// Should still get confirmation with "anonymous" username
	if !client.WaitForMessage("Hello, anonymous!", 1*time.Second) {
		t.Error("Did not receive confirmation with anonymous username")
		t.Logf("Received messages: %v", client.GetMessages())
	}

	// Send a message
	err = client.SendMessage("Hello from anonymous!")
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Should receive message with anonymous username
	if !client.WaitForMessage("anonymous: Hello from anonymous!", 1*time.Second) {
		t.Error("Did not receive message with anonymous username")
		t.Logf("Received messages: %v", client.GetMessages())
	}
}
