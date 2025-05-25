# Testing Guide for Broadcast Server

This document provides comprehensive information about testing the broadcast server implementation. The testing philosophy emphasizes a multi-layered approach, combining unit, integration, and benchmark tests to ensure correctness, robustness, and performance.

## Test Structure

The test suite is organized into several categories:

### 1. Unit Tests

Located in `internal/server/*_test.go` files, these test individual components in isolation. The primary goal is to verify the logic of each small piece of the server independently.

- **Approach**: Unit tests heavily utilize mock objects to simulate dependencies. For instance, when testing connection logic (`connection_test.go`), the actual network I/O (`net.Conn`) is replaced with a `mockConn` (defined in `connection_test.go`). This `mockConn` allows tests to simulate client behavior (e.g., sending data, disconnecting) and inspect what the server writes back, without needing real network connections. This makes tests fast, reliable, and focused on the specific unit's logic.
- **`broadcaster_test.go`** - Tests the message broadcasting system (e.g., registration, unregistration, message fan-out).
- **`connection_test.go`** - Tests client connection handling (e.g., handshake, message reading/writing, graceful close) using `mockConn`.
- **`server_test.go`** - Tests server startup, overall connection management (registering/unregistering connections with the server instance), and graceful shutdown, sometimes using `mockConn` for direct connection registration tests and sometimes using real local TCP connections for broader server behavior tests.

### 2. Integration Tests

Located in `internal/server/integration_test.go`, these test the entire system end-to-end. They verify that different components of the server (listener, connection handler, broadcaster) work together correctly.

- **Approach**: Integration tests use real TCP/IP connections on the local machine. Test functions typically start a server instance on an available port (`:0`), then create one or more test clients that connect to this server. These clients send messages, perform actions like disconnecting, and the test verifies that the server behaves as expected from an external perspective (e.g., messages are broadcast correctly to other clients, disconnections are handled gracefully, server logs reflect correct states).
- **Single client scenarios** - Basic connection and message flow
- **Multiple client scenarios** - Message broadcasting between clients
- **Client disconnection** - Proper cleanup and notification
- **Concurrent messaging** - High-load scenarios with multiple clients
- **Edge cases** - Empty usernames, malformed input

### 3. Benchmark Tests

Located in `internal/server/benchmark_test.go`, these measure performance characteristics of key server operations.

- **Approach**: Benchmarks use Go's `testing.B` type and typically run a piece of code (e.g., sending N messages, establishing N connections) multiple times (controlled by `b.N`) to get stable performance metrics. They often involve setting up a server and then simulating client load, focusing on throughput and resource usage (like memory allocations when `-benchmem` is used).
- **Message throughput** - Single and multiple client message sending
- **Connection establishment** - Speed of new connections
- **Broadcasting performance** - Efficiency of message distribution
- **Memory allocation** - Resource usage patterns

## Running Tests

### Using Make Commands

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests only
make test-integration

# Run benchmark tests
make test-benchmark

# Run tests with coverage report
make test-coverage

# Run tests with race detection
make test-race

# Run all tests with verbose output
make test-verbose
```

### Using Go Commands Directly

```bash
# Run all tests
go test ./...

# Run tests in server package only
go test ./internal/server

# Run specific test
go test ./internal/server -run TestNewServer

# Run tests with verbose output
go test -v ./internal/server

# Run benchmarks
go test -bench=. ./internal/server

# Run benchmarks with memory allocation stats
go test -bench=. -benchmem ./internal/server

# Run tests with coverage
go test -cover ./internal/server

# Generate coverage report
go test -coverprofile=coverage.out ./internal/server
go tool cover -html=coverage.out -o coverage.html

# Run tests with race detection
go test -race ./internal/server
```

### Using the Test Runner

```bash
# Run specific test categories
go run test_runner.go unit
go run test_runner.go integration
go run test_runner.go benchmark
go run test_runner.go all
go run test_runner.go coverage
go run test_runner.go race
```

## Test Categories Explained

### Unit Tests

#### Broadcaster Tests

- **TestNewBroadcaster** - Verifies proper initialization
- **TestBroadcasterRegisterUnregister** - Tests connection management
- **TestBroadcasterBroadcast** - Tests message broadcasting mechanism
- **TestBroadcasterStop** - Tests graceful shutdown
- **TestBroadcasterConcurrentOperations** - Tests thread safety

#### Connection Tests

- **TestNewConnection** - Verifies connection creation
- **TestConnectionSend** - Tests message sending
- **TestConnectionHandshake** - Tests initial client handshake
- **TestConnectionMessageBroadcasting** - Tests message processing
- **TestConnectionClose** - Tests connection cleanup
- **TestConnectionEmptyMessages** - Tests edge cases

#### Server Tests

- **TestNewServer** - Verifies server initialization
- **TestServerStartAndStop** - Tests server lifecycle
- **TestServerConnectionHandling** - Tests connection management
- **TestServerConcurrentConnections** - Tests concurrent client handling

### Integration Tests

#### End-to-End Scenarios

- **TestIntegrationSingleClient** - Complete single client flow
- **TestIntegrationMultipleClients** - Multi-client message broadcasting
- **TestIntegrationClientDisconnection** - Disconnect handling
- **TestIntegrationConcurrentMessages** - High-load concurrent messaging
- **TestIntegrationEmptyUsername** - Edge case handling

### Benchmark Tests

#### Performance Measurements

- **BenchmarkSingleClientMessages** - Single client message throughput
- **BenchmarkMultipleClientsMessages** - Multi-client performance
- **BenchmarkConnectionEstablishment** - Connection setup speed
- **BenchmarkBroadcasterOperations** - Broadcasting efficiency
- **BenchmarkHighLoadScenario** - Stress testing

## Test Coverage

To generate and view test coverage:

```bash
# Generate coverage report
make test-coverage

# View coverage in terminal
go test -cover ./internal/server

# Generate detailed coverage profile
go test -coverprofile=coverage.out ./internal/server
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

## Race Detection

The broadcast server uses goroutines extensively, so race detection is crucial:

```bash
# Run tests with race detection
make test-race

# Or directly with go
go test -race ./internal/server
```

## Continuous Integration

For CI/CD pipelines, use:

```bash
# Complete test suite with race detection and coverage
go test -v -race -cover ./...

# Benchmark tests (optional in CI)
go test -bench=. -benchmem ./internal/server
```

## Test Data and Mocks

The test suite includes several mock implementations:

- **mockConn** - Implements `net.Conn` for testing connections
- **mockBroadcaster** - Simple broadcaster for isolated testing
- **TestClient** - Full-featured test client for integration tests
- **BenchmarkClient** - Lightweight client for performance testing

## Writing New Tests

### Unit Test Guidelines

1. **Isolation** - Test components in isolation using mocks
2. **Coverage** - Test both success and error paths
3. **Concurrency** - Test thread safety where applicable
4. **Edge Cases** - Test boundary conditions and invalid inputs

### Integration Test Guidelines

1. **Real Connections** - Use actual TCP connections
2. **Multiple Scenarios** - Test various client configurations
3. **Cleanup** - Ensure proper resource cleanup
4. **Timeouts** - Use appropriate timeouts for async operations

### Benchmark Guidelines

1. **Realistic Load** - Use realistic message sizes and frequencies
2. **Multiple Scales** - Test with different numbers of clients
3. **Memory Profiling** - Include memory allocation measurements
4. **Baseline** - Establish performance baselines

## Troubleshooting Tests

### Common Issues

1. **Port Conflicts** - Tests use random ports (`:0`) to avoid conflicts
2. **Timing Issues** - Integration tests include appropriate delays
3. **Resource Cleanup** - All tests properly clean up connections
4. **Race Conditions** - Use `-race` flag to detect concurrency issues

### Debug Tips

```bash
# Run specific failing test with verbose output
go test -v ./internal/server -run TestSpecificTest

# Run with race detection for concurrency issues
go test -race ./internal/server -run TestSpecificTest

# Add debug logging (modify logger level in tests)
# Use t.Logf() for test-specific logging
```

## Performance Expectations

### Benchmark Targets

- **Single Client**: >1000 messages/second
- **Multiple Clients**: >500 messages/second per client
- **Connection Setup**: <10ms per connection
- **Memory Usage**: <1MB per 100 connections

### Scaling Characteristics

The server should handle:

- 100+ concurrent connections
- 1000+ messages/second total throughput
- <100ms message latency under normal load

## Test Environment

### Requirements

- Go 1.21.3 or later
- Available TCP ports for testing
- Sufficient memory for concurrent connections

### Configuration

Tests automatically:

- Use random available ports
- Clean up resources
- Handle timeouts appropriately
- Provide detailed error messages

## Contributing

When adding new features:

1. **Add Unit Tests** - Test new components in isolation
2. **Add Integration Tests** - Test end-to-end functionality
3. **Add Benchmarks** - Measure performance impact
4. **Update Documentation** - Update this guide if needed
5. **Run Full Suite** - Ensure all tests pass

```bash
# Before submitting changes
make test-race
make test-coverage
make test-benchmark
```
