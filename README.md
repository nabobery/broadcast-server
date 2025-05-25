# Broadcast Server

https://roadmap.sh/projects/broadcast-server

A simple TCP-based broadcast server implemented in Go. This application allows multiple clients to connect to a central server and broadcast messages to all connected clients.

## Features

- Start a server that listens for client connections
- Connect multiple clients to the server
- Send messages from clients to the server
- Broadcast messages from server to all connected clients
- Handle client disconnections gracefully
- Clean CLI interface with configurable options

## Project Structure

This project follows the [golang-standards/project-layout](https://github.com/golang-standards/project-layout) guidelines:

```
broadcast-server/
├── cmd/              # Application entry points
├── internal/         # Private application code
├── pkg/              # Public libraries that can be used by external applications
├── build/            # Build artifacts
├── Makefile          # Build automation
└── README.md         # Project documentation
```

## Prerequisites

- Go 1.18 or later

## Installation

### From Source

1. Clone the repository:
   ```
   git clone https://github.com/nabobery/broadcast-server.git
   cd broadcast-server
   ```

2. Build the project:
   ```
   make build
   ```

## Usage

### Starting the Server

```
./build/broadcast-server start
```

Optional flags:
- `--port, -p`: Specify the port to listen on (default: 8080)

Example:
```
./build/broadcast-server start --port 9090
```

### Connecting as a Client

```
./build/broadcast-server connect
```

Optional flags:
- `--host, -H`: Specify the host to connect to (default: localhost)
- `--port, -p`: Specify the port to connect to (default: 8080)
- `--username, -u`: Specify the username to use (default: anonymous)

Example:
```
./build/broadcast-server connect --host 192.168.1.100 --port 9090 --username Alice
```

## How It Works

1. The server starts and listens for incoming TCP connections on the specified port.
2. When a client connects, it's added to the list of connected clients.
3. The client provides a username which is used to identify messages.
4. When a client sends a message, the server broadcasts it to all connected clients.
5. When a client disconnects, it's removed from the list of connected clients.

## Architecture

The application follows a clean architecture approach:

- **CLI Layer**: Handles command-line arguments and forwards them to the appropriate service.
- **Service Layer**: Contains the business logic for the server and client.
- **Infrastructure Layer**: Handles low-level details like TCP connections and message broadcasting.

## Development

### Running Tests

```
make test
```

### Cleaning Build Artifacts

```
make clean
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
