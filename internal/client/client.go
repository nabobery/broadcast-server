package client

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gorilla/websocket"
)

// Connect establishes a connection to the broadcast server and handles I/O
func Connect(host, port, username string) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	u := url.URL{
		Scheme:   "ws",
		Host:     host + ":" + port,
		Path:     "/ws",
		RawQuery: "username=" + username,
	}

	log.Printf("Connecting to %s", u.String())
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("error connecting to server: %v", err)
	}
	defer func(conn *websocket.Conn) {
		err := conn.Close()
		if err != nil {
			log.Println("error closing websocket connection")
		}
	}(conn)

	done := make(chan struct{})

	// Read incoming messages from the server
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Connection closed by server")
				return
			}
			fmt.Println(string(message))
		}
	}()

	// Handle user input
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("Type your messages (press Ctrl+C to exit):")
		for scanner.Scan() {
			message := scanner.Text()
			if strings.TrimSpace(message) == "" {
				continue
			}

			err := conn.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				log.Println("Error sending message:", err)
				return
			}
		}
	}()

	// Wait for interrupt signal or server connection close
	select {
	case <-interrupt:
		fmt.Println("\nInterrupt received, closing connection...")
		err := conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("Error during closing handshake:", err)
		}
		<-done
		return nil
	case <-done:
		return fmt.Errorf("connection closed by server")
	}
}
