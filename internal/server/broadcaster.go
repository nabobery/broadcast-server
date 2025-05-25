package server

import (
	"sync"

	"broadcast-server/pkg/logger"
)

// Broadcaster handles broadcasting messages to all connected clients
type Broadcaster struct {
	connections     map[*Connection]bool
	connectionMutex sync.RWMutex
	register        chan *Connection
	unregister      chan *Connection
	broadcast       chan string
	quit            chan struct{}
}

// NewBroadcaster creates a new message broadcaster
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		connections: make(map[*Connection]bool),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		broadcast:   make(chan string),
		quit:        make(chan struct{}),
	}
}

// Start begins the broadcaster's message handling loop
func (b *Broadcaster) Start() {
	for {
		select {
		case conn := <-b.register:
			// Add new connection
			b.connectionMutex.Lock()
			b.connections[conn] = true
			b.connectionMutex.Unlock()

		case conn := <-b.unregister:
			// Remove connection
			b.connectionMutex.Lock()
			delete(b.connections, conn)
			b.connectionMutex.Unlock()

		case message := <-b.broadcast:
			// Send message to all connections
			logger.Debug("Broadcasting: %s", message)

			b.connectionMutex.RLock()
			for conn := range b.connections {
				go func(c *Connection) {
					if err := c.Send(message); err != nil {
						logger.Error("Error sending to %s: %v", c.ID(), err)
						b.Unregister(c)
					}
				}(conn)
			}
			b.connectionMutex.RUnlock()

		case <-b.quit:
			// Stop the broadcaster
			return
		}
	}
}

// Register adds a connection to the broadcaster
func (b *Broadcaster) Register(conn *Connection) {
	b.register <- conn
}

// Unregister removes a connection from the broadcaster
func (b *Broadcaster) Unregister(conn *Connection) {
	b.unregister <- conn
}

// Broadcast sends a message to all connected clients
func (b *Broadcaster) Broadcast(message string) {
	b.broadcast <- message
}

// Stop shuts down the broadcaster
func (b *Broadcaster) Stop() {
	close(b.quit)
}
