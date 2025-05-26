PORT = 8080
USER_NAME ?= anonymous
HOST ?= localhost

BINARY_NAME=broadcast-server
BUILD_DIR=build

.PHONY: build server client clean

build:
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/main.go

server:
	@$(BUILD_DIR)/$(BINARY_NAME) start --port $(PORT)

client:
	@$(BUILD_DIR)/$(BINARY_NAME) connect --host $(HOST) --port $(PORT) --username $(USER_NAME)

clean:
	rm -rf $(BUILD_DIR)
