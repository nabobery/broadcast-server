package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"broadcast-server/pkg/logger"
)

// Connection represents a client connection to the server
type Connection struct {
	id           string
	username     string
	conn         net.Conn
	broadcaster  *Broadcaster
	disconnected chan struct{}
	closeOnce    sync.Once
}

// NewConnection creates a new client connection
func NewConnection(conn net.Conn, broadcaster *Broadcaster) *Connection {
	// Create a unique connection ID based on the remote address and timestamp
	connID := fmt.Sprintf("%s-%d", conn.RemoteAddr().String(), time.Now().UnixNano())

	return &Connection{
		id:           connID,
		username:     "anonymous", // Default username will be updated during handshake
		conn:         conn,
		broadcaster:  broadcaster,
		disconnected: make(chan struct{}),
	}
}

// ID returns the connection ID
func (c *Connection) ID() string {
	return c.id
}

// Username returns the client's username
func (c *Connection) Username() string {
	return c.username
}

// Send sends a message to the client
func (c *Connection) Send(message string) error {
	_, err := c.conn.Write([]byte(message + "\n"))
	return err
}

// Start begins processing messages from the client
func (c *Connection) Start() {
	// Register this connection with the broadcaster
	c.broadcaster.Register(c)

	// Handle handshake first
	c.handleHandshake()

	// Start reading messages
	go c.readLoop()
}

// handleHandshake performs the initial handshake with the client
func (c *Connection) handleHandshake() {
	// Send welcome message
	err := c.Send("Welcome to the Broadcast Server!")
	if err != nil {
		logger.Error("Error sending welcome message to %s: %v", c.id, err)
		c.Close()
		return
	}

	err = c.Send("Please enter your username:")
	if err != nil {
		logger.Error("Error sending username prompt to %s: %v", c.id, err)
		c.Close()
		return
	}

	// Read username
	scanner := bufio.NewScanner(c.conn)
	if scanner.Scan() {
		username := strings.TrimSpace(scanner.Text())
		if username != "" {
			c.username = username
		}
	} else {
		// If scanner.Scan() returns false, it means the connection was closed or an error occurred
		if scanner.Err() != nil {
			logger.Error("Error reading username from %s: %v", c.id, scanner.Err())
		} else {
			logger.Info("Connection closed by client during username prompt: %s", c.id)
		}
		c.Close()
		return
	}

	// Send confirmation
	err = c.Send(fmt.Sprintf("Hello, %s! You are now connected. Type a message to broadcast it.", c.username))
	if err != nil {
		logger.Error("Error sending confirmation message to %s: %v", c.id, err)
		c.Close()
		return
	}
}

// readLoop continuously reads messages from the client
func (c *Connection) readLoop() {
	scanner := bufio.NewScanner(c.conn)

	for scanner.Scan() {
		message := scanner.Text()
		if message == "" {
			continue
		}

		// Format the message with the sender's username
		formattedMsg := fmt.Sprintf("%s: %s", c.username, message)

		// Broadcast the message
		c.broadcaster.Broadcast(formattedMsg)
	}

	// Client disconnected or error occurred
	if scanner.Err() != nil {
		logger.Error("Error reading from connection %s: %v", c.id, scanner.Err())
	} else {
		logger.Info("Connection closed by client: %s", c.id)
	}
	c.Close()
}

// Disconnected returns a channel that is closed when the connection is closed
func (c *Connection) Disconnected() <-chan struct{} {
	return c.disconnected
}

// Close closes the connection
func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		// Unregister from broadcaster
		c.broadcaster.Unregister(c)

		// Close the connection
		err := c.conn.Close()
		if err != nil {
			return
		}

		// Signal disconnection
		close(c.disconnected)
	})
}
