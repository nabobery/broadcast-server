package server

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a connected websocket client
type Client struct {
	ID   string
	Conn *websocket.Conn
}

// BroadcastServer manages websocket connections and broadcasts messages
type BroadcastServer struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mutex      sync.Mutex
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

// NewBroadcastServer creates a new broadcast server instance
func NewBroadcastServer() *BroadcastServer {
	return &BroadcastServer{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
	}
}

// Start initializes and runs the broadcast server
func Start(port string) error {
	server := NewBroadcastServer()
	go server.run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		server.handleConnection(w, r)
	})

	log.Printf("Server started on port %s", port)
	return http.ListenAndServe(":"+port, nil)
}

func (s *BroadcastServer) handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to websocket:", err)
		return
	}

	// Get username from query params or use IP address
	username := r.URL.Query().Get("username")
	if username == "" {
		username = r.RemoteAddr
	}

	client := &Client{
		ID:   username,
		Conn: conn,
	}

	// Register new client
	s.register <- client

	// Welcome message
	welcomeMsg := fmt.Sprintf("Server: %s has joined the chat", username)
	s.broadcast <- []byte(welcomeMsg)

	// Handle client messages
	go s.readPump(client)
}

func (s *BroadcastServer) readPump(client *Client) {
	defer func() {
		s.unregister <- client
		err := client.Conn.Close()
		if err != nil {
			// Log error on closing the connection if needed, but not related to ReadMessage error
			log.Printf("Error closing client connection: %v", err)
			return
		}
	}()

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			// Check specifically for normal closure
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Printf("Client %s connection closed normally", client.ID)
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log unexpected close errors as errors
				log.Printf("Unexpected error reading message from client %s: %v", client.ID, err)
			} else {
				// Log other close errors or read errors
				log.Printf("Error reading message from client %s: %v", client.ID, err)
			}
			break // Exit the read loop on any error (expected or unexpected)
		}

		// Format message with username
		formattedMsg := fmt.Sprintf("%s: %s", client.ID, string(message))
		s.broadcast <- []byte(formattedMsg)
	}
}

func (s *BroadcastServer) run() {
	for {
		select {
		case client := <-s.register:
			s.mutex.Lock()
			s.clients[client.ID] = client
			log.Printf("Client connected: %s (total: %d)", client.ID, len(s.clients))
			s.mutex.Unlock()

		case client := <-s.unregister:
			s.mutex.Lock()
			if _, ok := s.clients[client.ID]; ok {
				delete(s.clients, client.ID)
				log.Printf("Client disconnected: %s (total: %d)", client.ID, len(s.clients))

				// Notify others about the disconnection
				disconnectMsg := fmt.Sprintf("Server: %s has left the chat", client.ID)
				s.broadcast <- []byte(disconnectMsg)
			}
			s.mutex.Unlock()

		case message := <-s.broadcast:
			s.mutex.Lock()
			for _, client := range s.clients {
				err := client.Conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("Error broadcasting to client %s: %v", client.ID, err)
					err := client.Conn.Close()
					if err != nil {
						log.Printf("Error closing connection for client %s: %v", client.ID, err)
					}
					delete(s.clients, client.ID)
				}
			}
			s.mutex.Unlock()
		}
	}
}
