package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Client represents a client that connects to the broadcast server
type Client struct {
	host     string
	port     int
	username string
	conn     net.Conn
	wg       sync.WaitGroup
}

// NewClient creates a new client instance
func NewClient(host string, port int, username string) *Client {
	return &Client{
		host:     host,
		port:     port,
		username: username,
	}
}

// Connect connects to the server and starts handling messages
func (c *Client) Connect() error {
	// Connect to the server
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.host, c.port))
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	c.conn = conn
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("Error closing connection: %v\n", err)
		}
	}(c.conn)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nDisconnecting from server...")
		err := c.conn.Close()
		if err != nil {
			return
		}
		os.Exit(0)
	}()

	// Start receiving messages
	c.wg.Add(1)
	go c.receiveMessages()

	// Send the username when prompted
	c.sendUsername()

	// Start sending messages
	c.sendMessages()

	// Wait for the receive goroutine to finish
	c.wg.Wait()

	return nil
}

// sendUsername sends the username to the server during initial handshake
func (c *Client) sendUsername() {
	// The server will ask for a username, so we send it
	_, err := fmt.Fprintln(c.conn, c.username)
	if err != nil {
		return
	}
}

// receiveMessages continuously receives and displays messages from the server
func (c *Client) receiveMessages() {
	defer c.wg.Done()

	scanner := bufio.NewScanner(c.conn)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading from server: %v\n", err)
	}

	fmt.Println("Disconnected from server.")
}

// sendMessages reads input from the user and sends it to the server
func (c *Client) sendMessages() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		message := scanner.Text()
		_, err := fmt.Fprintln(c.conn, message)
		if err != nil {
			fmt.Printf("Error sending message: %v\n", err)
			return
		}
	}
}
