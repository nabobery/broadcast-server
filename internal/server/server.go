package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Rate limiting: max 5 messages per second per client
	maxMessagesPerSecond = 5
	rateLimitWindow      = time.Second

	// Connection timeouts
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// Client represents a connected websocket client
type Client struct {
	ID            string
	Conn          *websocket.Conn
	lastMessage   time.Time
	messageCount  int
	rateLimitTime time.Time
}

// BroadcastServer manages websocket connections and broadcasts messages
type BroadcastServer struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mutex      sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Only allow localhost for development
		// In production, this should be configured to allow specific origins
		origin := r.Header.Get("Origin")
		return origin == "" ||
			origin == "http://localhost:8080" ||
			origin == "http://127.0.0.1:8080" ||
			origin == "ws://localhost:8080" ||
			origin == "ws://127.0.0.1:8080"
	},
}

// NewBroadcastServer creates a new broadcast server instance
func NewBroadcastServer() *BroadcastServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &BroadcastServer{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start initializes and runs the broadcast server
func Start(port string) error {
	server := NewBroadcastServer()
	go server.run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		server.handleConnection(w, r)
	})

	// Create HTTP server
	httpServer := &http.Server{
		Addr: ":" + port,
	}

	// Handle shutdown signals
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		<-c
		log.Println("Shutting down server gracefully...")

		// Cancel the context to stop the run loop
		server.cancel()

		// Shutdown HTTP server with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Server started on port %s", port)
	return httpServer.ListenAndServe()
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

	// Ensure unique username
	s.mutex.Lock()
	originalUsername := username
	counter := 1
	for _, exists := s.clients[username]; exists; _, exists = s.clients[username] {
		username = fmt.Sprintf("%s_%d", originalUsername, counter)
		counter++
	}
	s.mutex.Unlock()

	client := &Client{
		ID:            username,
		Conn:          conn,
		lastMessage:   time.Now(),
		messageCount:  0,
		rateLimitTime: time.Now(),
	}

	// Register new client
	s.register <- client

	// Welcome message
	welcomeMsg := fmt.Sprintf("Server: %s has joined the chat", username)
	s.broadcast <- []byte(welcomeMsg)

	// Handle client messages
	go s.readPump(client)
	go s.writePump(client)
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

	// Set read deadline and message size limit
	client.Conn.SetReadLimit(maxMessageSize)
	if err := client.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("Error setting read deadline: %v", err)
		return
	}
	client.Conn.SetPongHandler(func(string) error {
		if err := client.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Printf("Error setting read deadline in pong handler: %v", err)
		}
		return nil
	})

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

		// Rate limiting check
		now := time.Now()
		if now.Sub(client.rateLimitTime) >= rateLimitWindow {
			// Reset rate limit window
			client.rateLimitTime = now
			client.messageCount = 0
		}

		client.messageCount++
		if client.messageCount > maxMessagesPerSecond {
			log.Printf("Rate limit exceeded for client %s, dropping message", client.ID)
			continue
		}

		// Format message with username
		formattedMsg := fmt.Sprintf("%s: %s", client.ID, string(message))
		s.broadcast <- []byte(formattedMsg)
	}
}

func (s *BroadcastServer) writePump(client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err := client.Conn.Close(); err != nil {
			log.Printf("Error closing connection in writePump: %v", err)
		}
	}()

	for {
		select {
		case <-ticker.C:
			if err := client.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *BroadcastServer) run() {
	for {
		select {
		case <-s.ctx.Done():
			log.Println("Broadcast server shutting down...")
			// Close all client connections
			s.mutex.Lock()
			for _, client := range s.clients {
				if err := client.Conn.Close(); err != nil {
					log.Printf("Error closing client connection during shutdown: %v", err)
				}
			}
			s.mutex.Unlock()
			return

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
				if err := client.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
					log.Printf("Error setting write deadline for client %s: %v", client.ID, err)
					continue
				}
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
