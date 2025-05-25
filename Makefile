.PHONY: build run-server run-client clean

# Build variables
BINARY_NAME=broadcast-server
BUILD_DIR=build

# Define USERNAME variable with a default value
USERNAME ?= anonymous

build:
	@echo "Building broadcast-server..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/main.go

run-server: build
	@echo "Starting broadcast server..."
	@$(BUILD_DIR)/$(BINARY_NAME) start

run-client: build
	@echo "Starting broadcast client..."
	# Pass the USERNAME variable as a command-line argument
	@$(BUILD_DIR)/$(BINARY_NAME) connect --username=$(USERNAME)

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)

test:
	@echo "Running all tests..."
	@go test ./...

test-unit:
	@echo "Running unit tests..."
	@go test -v ./internal/server -run "^Test[^I]"

test-integration:
	@echo "Running integration tests..."
	@go test -v ./internal/server -run "TestIntegration"

test-benchmark:
	@echo "Running benchmark tests..."
	@go test -v ./internal/server -bench=. -benchmem

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v ./internal/server -cover -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race:
	@echo "Running tests with race detection..."
	@go test -v ./internal/server -race

test-verbose:
	@echo "Running all tests with verbose output..."
	@go test -v ./...
