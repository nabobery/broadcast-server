.PHONY: build run-server run-client clean

# Build variables
BINARY_NAME=broadcast-server
BUILD_DIR=build

build:
	@echo "Building broadcast-server..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/broadcast-server

run-server: build
	@echo "Starting broadcast server..."
	@$(BUILD_DIR)/$(BINARY_NAME) start

run-client: build
	@echo "Starting broadcast client..."
	@$(BUILD_DIR)/$(BINARY_NAME) connect

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)

test:
	@echo "Running tests..."
	@go test ./...
