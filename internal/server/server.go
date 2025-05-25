package server

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"broadcast-server/pkg/logger"
)

// Server represents the broadcast server
type Server struct {
	port            int
	listener        net.Listener
	connectionMutex sync.RWMutex
	connections     map[string]*Connection
	broadcaster     *Broadcaster
}

// NewServer creates a new Server instance
func NewServer(port int) *Server {
	return &Server{
		port:        port,
		connections: make(map[string]*Connection),
		broadcaster: NewBroadcaster(),
	}
}

// Start starts the server and listens for connections
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	s.listener = listener
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			logger.Error("Error closing listener: %v", err)
		}
	}(s.listener)

	logger.PrintServerBanner(s.port)

	// Handle graceful shutdown
	go s.handleSignals()

	// Start the broadcaster
	go s.broadcaster.Start()

	// Accept connections
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// Check if the listener was closed intentionally
			if isClosedErr(err) {
				return nil
			}
			logger.Error("Error accepting connection: %v", err)
			continue
		}

		// Handle connection in a goroutine
		go s.handleConnection(conn)
	}
}

// handleConnection processes a new client connection
func (s *Server) handleConnection(conn net.Conn) {
	// Create a new connection
	client := NewConnection(conn, s.broadcaster)
	clientID := client.ID()

	// Register the connection
	s.connectionMutex.Lock()
	s.connections[clientID] = client
	connectionCount := len(s.connections)
	s.connectionMutex.Unlock()

	// Announce new connection
	logger.Info("New client connected: %s (Total: %d)", clientID, connectionCount)
	s.broadcaster.Broadcast(fmt.Sprintf("System: %s joined the chat", client.Username()))

	// Start handling messages from this client
	client.Start()

	// Wait for the client to disconnect
	<-client.Disconnected()

	// Remove the connection
	s.connectionMutex.Lock()
	delete(s.connections, clientID)
	connectionCount = len(s.connections)
	s.connectionMutex.Unlock()

	// Announce disconnection
	logger.Info("Client disconnected: %s (Total: %d)", clientID, connectionCount)
	s.broadcaster.Broadcast(fmt.Sprintf("System: %s left the chat", client.Username()))
}

// handleSignals handles system signals for graceful shutdown
func (s *Server) handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down server...")

	// Close the listener to stop accepting new connections
	if s.listener != nil {
		err := s.listener.Close()
		if err != nil {
			return
		}
	}

	// Close all existing connections
	s.connectionMutex.Lock()
	for _, conn := range s.connections {
		conn.Close()
	}
	s.connectionMutex.Unlock()

	// Stop the broadcaster
	s.broadcaster.Stop()

	logger.Info("Server shutdown complete")
	os.Exit(0)
}

// isClosedErr checks if the error is because the listener was closed
func isClosedErr(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return opErr.Err.Error() == "use of closed network connection"
	}
	return false
}
