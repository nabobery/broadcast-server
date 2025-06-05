# Broadcast Server

A real-time WebSocket broadcast server written in Go that enables multiple clients to connect and send messages to each other in a chat-like environment.

🔗 **Project Reference**: https://roadmap.sh/projects/broadcast-server

## Features

### 🚀 Core Functionality

- **Real-time messaging**: WebSocket-based communication for instant message delivery
- **Multi-client support**: Multiple clients can connect simultaneously and broadcast messages
- **Automatic user management**: Unique username assignment and collision handling
- **Connection lifecycle management**: Proper client registration/unregistration with notifications

### 🛡️ Security & Validation

- **Rate limiting**: 5 messages per second per client to prevent spam
- **Input validation**: Username and message content validation with UTF-8 support
- **Message size limits**: Maximum 512 bytes per message, 500 characters per message
- **Username constraints**: Maximum 50 characters, no control characters
- **Origin checking**: Configurable CORS policy for WebSocket connections

### 🔧 Reliability Features

- **Graceful shutdown**: Proper signal handling (SIGINT, SIGTERM) with client cleanup
- **Connection timeouts**: Automatic ping/pong heartbeat mechanism
- **Error handling**: Comprehensive error logging and connection recovery
- **Context-based cancellation**: Clean shutdown propagation across goroutines

### 📊 Monitoring & Logging

- **Connection tracking**: Real-time client count and connection status
- **Event logging**: Join/leave notifications and error reporting
- **Performance monitoring**: Built-in benchmarks and test coverage

## Installation

### Prerequisites

- Go 1.22 or higher
- Git (for cloning the repository)

### Dependencies

```bash
go mod download
```

The project uses:

- `github.com/gorilla/websocket` - WebSocket implementation
- `github.com/urfave/cli/v3` - Command-line interface

### Build

```bash
# Build the binary
make build

# Or manually
go build -o build/broadcast-server ./cmd/main.go
```

## Usage

### Starting the Server

```bash
# Using Makefile (default port 8080)
make server

# Using binary directly
./build/broadcast-server start --port 8080

# Custom port
./build/broadcast-server start --port 3000
```

### Connecting Clients

```bash
# Using Makefile with default settings
make client

# Using binary with custom parameters
./build/broadcast-server connect --host localhost --port 8080 --username john

# Connect to remote server
./build/broadcast-server connect --host example.com --port 8080 --username alice
```

### Example Session

1. **Start the server:**

   ```bash
   $ make server
   Starting server on port 8080...
   Server started on port 8080
   ```

2. **Connect first client:**

   ```bash
   $ make client USER_NAME=alice
   Connecting to localhost:8080 as alice...
   Server: alice has joined the chat
   Type your messages (press Ctrl+C to exit):
   ```

3. **Connect second client:**

   ```bash
   $ make client USER_NAME=bob
   Connecting to localhost:8080 as bob...
   Server: bob has joined the chat
   Type your messages (press Ctrl+C to exit):
   ```

4. **Chat between clients:**

   ```
   # Alice types: Hello everyone!
   # Both clients see: alice: Hello everyone!

   # Bob types: Hi Alice!
   # Both clients see: bob: Hi Alice!
   ```

## API Reference

### WebSocket Endpoint

- **URL**: `ws://localhost:8080/ws`
- **Query Parameters**:
  - `username` (optional): Client identifier. If not provided, uses IP address

### Message Format

- **Incoming**: Raw text messages from clients
- **Outgoing**: Formatted as `username: message` or `Server: notification`

### Connection Lifecycle

1. **Connection**: Client connects with optional username parameter
2. **Registration**: Server assigns unique username and registers client
3. **Welcome**: Server broadcasts join notification to all clients
4. **Messaging**: Client can send/receive messages in real-time
5. **Disconnection**: Server broadcasts leave notification and cleans up

## Configuration

### Server Limits

```go
const (
    maxMessagesPerSecond = 5        // Rate limit per client
    rateLimitWindow      = 1s       // Rate limit window
    writeWait           = 10s       // Write operation timeout
    pongWait            = 60s       // Pong response timeout
    pingPeriod          = 54s       // Ping interval
    maxMessageSize      = 512       // WebSocket message size limit
    maxUsernameLength   = 50        // Username character limit
    maxMessageLength    = 500       // Message character limit
)
```

### Environment Variables

- `PORT`: Server port (default: 8080)
- `USER_NAME`: Default username for client connection (default: anonymous)
- `HOST`: Server host for client connection (default: localhost)

## Development

### Project Structure

```
broadcast-server/
├── cmd/
│   └── main.go              # CLI entry point
├── internal/
│   ├── server/
│   │   └── server.go        # WebSocket server implementation
│   └── client/
│       └── client.go        # WebSocket client implementation
├── pkg/
│   └── websocket/
│       └── websocket.go     # WebSocket utilities and wrappers
├── build/                   # Compiled binaries
├── Makefile                 # Build and run commands
├── go.mod                   # Go module definition
└── README.md
```

### Testing

```bash
# Run all tests
go test -v ./internal/server

# Run specific test categories
go test -v ./internal/server -run "^Test[^I]"  # Unit tests only
go test -v ./internal/server -run TestIntegration  # Integration tests only

# Run with coverage
go test -v -cover ./internal/server

# Run benchmarks
go test -v -bench=. ./internal/server
```

### Code Quality

The project includes:

- **Pre-commit hooks**: Automated formatting, vetting, and testing
- **Linting**: golangci-lint integration
- **Test coverage**: Comprehensive unit and integration tests
- **Benchmarks**: Performance testing for concurrent operations

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run `go fmt`, `go vet`, and tests
5. Submit a pull request

## Architecture

### Concurrency Model

- **Goroutines per client**: Each client gets dedicated read/write goroutines
- **Central broadcaster**: Single goroutine manages client registry and message broadcasting
- **Channel-based communication**: Type-safe message passing between goroutines
- **Mutex protection**: Thread-safe access to shared client map

### Message Flow

```
Client → WebSocket → readPump → broadcast channel → run() → writePump → WebSocket → All Clients
```

### Error Handling

- **Connection errors**: Automatic cleanup and notification
- **Rate limiting**: Message dropping with logging
- **Validation errors**: Message rejection with logging
- **Graceful shutdown**: Clean client disconnection on server stop

## Performance

### Benchmarks

- **Concurrent connections**: Supports hundreds of simultaneous clients
- **Message throughput**: Optimized for real-time communication
- **Memory usage**: Efficient connection pooling and cleanup
- **CPU usage**: Lightweight goroutine-based architecture

### Scaling Considerations

- For production use, consider:
  - Load balancing across multiple server instances
  - Redis or similar for message persistence
  - Database for user authentication
  - Horizontal scaling with container orchestration

## License

MIT License - see [LICENSE](LICENSE) file for details.
