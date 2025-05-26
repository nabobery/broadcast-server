package websocket

import (
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Connection wraps the websocket connection with additional functionality
type Connection struct {
	conn      *websocket.Conn
	writeMu   sync.Mutex
	closeChan chan struct{}
	isClosed  bool
}

// Upgrader configuration for websocket
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin
	},
}

// NewConnection creates a new websocket connection wrapper
func NewConnection(conn *websocket.Conn) *Connection {
	return &Connection{
		conn:      conn,
		closeChan: make(chan struct{}),
		isClosed:  false,
	}
}

// Upgrade upgrades an HTTP connection to a websocket connection
func Upgrade(w http.ResponseWriter, r *http.Request) (*Connection, error) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return NewConnection(conn), nil
}

// WriteMessage sends a message through the websocket connection
func (c *Connection) WriteMessage(messageType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.isClosed {
		return errors.New("connection is closed")
	}

	return c.conn.WriteMessage(messageType, data)
}

// ReadMessage reads a message from the websocket connection
func (c *Connection) ReadMessage() (messageType int, p []byte, err error) {
	return c.conn.ReadMessage()
}

// Close closes the websocket connection
func (c *Connection) Close() error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if !c.isClosed {
		c.isClosed = true
		close(c.closeChan)
		return c.conn.Close()
	}
	return nil
}

// IsClosed returns whether the connection is closed
func (c *Connection) IsClosed() bool {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.isClosed
}

// ClosedChan returns a channel that's closed when the connection is closed
func (c *Connection) ClosedChan() <-chan struct{} {
	return c.closeChan
}
