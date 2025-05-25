package server

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

// BenchmarkClient represents a simple client for benchmarking
type BenchmarkClient struct {
	conn net.Conn
	id   int
}

func NewBenchmarkClient(address string, id int) (*BenchmarkClient, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	client := &BenchmarkClient{
		conn: conn,
		id:   id,
	}

	// Complete handshake
	username := fmt.Sprintf("benchuser%d", id)
	_, err = conn.Write([]byte(username + "\n"))
	if err != nil {
		conn.Close()
		return nil, err
	}

	return client, nil
}

func (bc *BenchmarkClient) SendMessage(message string) error {
	_, err := bc.conn.Write([]byte(message + "\n"))
	return err
}

func (bc *BenchmarkClient) Close() error {
	return bc.conn.Close()
}

func setupBenchmarkServer() (net.Listener, int, func()) {
	server := NewServer(0)
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(fmt.Sprintf("Failed to start benchmark server: %v", err))
	}

	server.listener = listener

	// Start broadcaster
	go server.broadcaster.Start()

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

	cleanup := func() {
		server.broadcaster.Stop()
		listener.Close()
	}

	return listener, port, cleanup
}

func BenchmarkSingleClientMessages(b *testing.B) {
	_, port, cleanup := setupBenchmarkServer()
	defer cleanup()

	address := fmt.Sprintf("localhost:%d", port)

	// Create a single client
	client, err := NewBenchmarkClient(address, 0)
	if err != nil {
		b.Fatalf("Failed to create benchmark client: %v", err)
	}
	defer client.Close()

	// Give time for connection to be established
	time.Sleep(50 * time.Millisecond)

	b.ResetTimer()

	// Benchmark sending messages
	for i := 0; i < b.N; i++ {
		message := fmt.Sprintf("Benchmark message %d", i)
		err := client.SendMessage(message)
		if err != nil {
			b.Fatalf("Failed to send message: %v", err)
		}
	}
}

func BenchmarkMultipleClientsMessages(b *testing.B) {
	_, port, cleanup := setupBenchmarkServer()
	defer cleanup()

	address := fmt.Sprintf("localhost:%d", port)

	// Create multiple clients
	numClients := 10
	clients := make([]*BenchmarkClient, numClients)

	for i := 0; i < numClients; i++ {
		client, err := NewBenchmarkClient(address, i)
		if err != nil {
			b.Fatalf("Failed to create benchmark client %d: %v", i, err)
		}
		clients[i] = client
		defer client.Close()
	}

	// Give time for connections to be established
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()

	// Benchmark concurrent message sending
	var wg sync.WaitGroup
	messagesPerClient := b.N / numClients

	for i, client := range clients {
		wg.Add(1)
		go func(clientID int, c *BenchmarkClient) {
			defer wg.Done()
			for j := 0; j < messagesPerClient; j++ {
				message := fmt.Sprintf("Message %d from client %d", j, clientID)
				err := c.SendMessage(message)
				if err != nil {
					b.Errorf("Failed to send message from client %d: %v", clientID, err)
					return
				}
			}
		}(i, client)
	}

	wg.Wait()
}

func BenchmarkConnectionEstablishment(b *testing.B) {
	_, port, cleanup := setupBenchmarkServer()
	defer cleanup()

	address := fmt.Sprintf("localhost:%d", port)

	b.ResetTimer()

	// Benchmark connection establishment and teardown
	for i := 0; i < b.N; i++ {
		client, err := NewBenchmarkClient(address, i)
		if err != nil {
			b.Fatalf("Failed to create benchmark client: %v", err)
		}
		client.Close()
	}
}

func BenchmarkBroadcasterRegisterUnregister(b *testing.B) {
	broadcaster := NewBroadcaster()
	go broadcaster.Start()
	defer broadcaster.Stop()

	b.ResetTimer()

	// Benchmark register/unregister operations
	for i := 0; i < b.N; i++ {
		conn := &Connection{
			id:           fmt.Sprintf("bench-conn-%d", i),
			username:     fmt.Sprintf("benchuser%d", i),
			conn:         nil, // Not needed for this benchmark
			broadcaster:  broadcaster,
			disconnected: make(chan struct{}),
		}

		broadcaster.Register(conn)
		broadcaster.Unregister(conn)
	}
}

func BenchmarkBroadcasterBroadcast(b *testing.B) {
	broadcaster := NewBroadcaster()
	go broadcaster.Start()
	defer broadcaster.Stop()

	// Register some connections
	numConnections := 100
	for i := 0; i < numConnections; i++ {
		conn := &Connection{
			id:           fmt.Sprintf("bench-conn-%d", i),
			username:     fmt.Sprintf("benchuser%d", i),
			conn:         nil, // Not needed for this benchmark
			broadcaster:  broadcaster,
			disconnected: make(chan struct{}),
		}
		broadcaster.Register(conn)
	}

	// Give time for registrations to complete
	time.Sleep(50 * time.Millisecond)

	b.ResetTimer()

	// Benchmark broadcasting messages
	for i := 0; i < b.N; i++ {
		message := fmt.Sprintf("Benchmark broadcast message %d", i)
		broadcaster.Broadcast(message)
	}
}

func BenchmarkConcurrentBroadcasts(b *testing.B) {
	broadcaster := NewBroadcaster()
	go broadcaster.Start()
	defer broadcaster.Stop()

	// Register some connections
	numConnections := 50
	for i := 0; i < numConnections; i++ {
		conn := &Connection{
			id:           fmt.Sprintf("bench-conn-%d", i),
			username:     fmt.Sprintf("benchuser%d", i),
			conn:         nil, // Not needed for this benchmark
			broadcaster:  broadcaster,
			disconnected: make(chan struct{}),
		}
		broadcaster.Register(conn)
	}

	// Give time for registrations to complete
	time.Sleep(50 * time.Millisecond)

	b.ResetTimer()

	// Benchmark concurrent broadcasting
	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := b.N / numGoroutines

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				message := fmt.Sprintf("Concurrent message %d from goroutine %d", j, goroutineID)
				broadcaster.Broadcast(message)
			}
		}(i)
	}

	wg.Wait()
}

func BenchmarkHighLoadScenario(b *testing.B) {
	_, port, cleanup := setupBenchmarkServer()
	defer cleanup()

	address := fmt.Sprintf("localhost:%d", port)

	// Create many clients
	numClients := 50
	clients := make([]*BenchmarkClient, numClients)

	for i := 0; i < numClients; i++ {
		client, err := NewBenchmarkClient(address, i)
		if err != nil {
			b.Fatalf("Failed to create benchmark client %d: %v", i, err)
		}
		clients[i] = client
		defer client.Close()
	}

	// Give time for connections to be established
	time.Sleep(200 * time.Millisecond)

	b.ResetTimer()

	// Simulate high load with concurrent message sending
	var wg sync.WaitGroup
	messagesPerClient := b.N / numClients

	for i, client := range clients {
		wg.Add(1)
		go func(clientID int, c *BenchmarkClient) {
			defer wg.Done()
			for j := 0; j < messagesPerClient; j++ {
				message := fmt.Sprintf("High load message %d from client %d", j, clientID)
				err := c.SendMessage(message)
				if err != nil {
					// Don't fail the benchmark for individual message failures
					// in high load scenarios
					continue
				}

				// Small delay to simulate realistic usage
				if j%10 == 0 {
					time.Sleep(time.Microsecond)
				}
			}
		}(i, client)
	}

	wg.Wait()
}

// Memory allocation benchmarks
func BenchmarkConnectionCreation(b *testing.B) {
	broadcaster := NewBroadcaster()

	// Create a real TCP connection for benchmarking
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	address := listener.Addr().String()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create real connection for more accurate benchmarking
		tcpConn, err := net.Dial("tcp", address)
		if err != nil {
			b.Fatalf("Failed to create connection: %v", err)
		}

		conn := NewConnection(tcpConn, broadcaster)
		_ = conn // Prevent optimization

		tcpConn.Close()
	}
}

func BenchmarkMessageFormatting(b *testing.B) {
	username := "testuser"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		message := fmt.Sprintf("Test message %d", i)
		formattedMsg := fmt.Sprintf("%s: %s", username, message)
		_ = formattedMsg // Prevent optimization
	}
}
