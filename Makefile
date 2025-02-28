.PHONY: all build-mac build-linux clean test lint run install help

# Application name
APP_NAME := dockerNav
# Go executable
GO := go

# Build directory
BUILD_DIR := build
# Main file path
MAIN_FILE := cmd/dockerNav/main.go

# Default target builds both binaries
all: clean build-mac build-linux

# Build for macOS
build-mac:
	@echo "Building $(APP_NAME) for Mac..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APP_NAME)-mac $(MAIN_FILE)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)-mac"

# Build for Linux
build-linux:
	@echo "Building $(APP_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APP_NAME)-linux $(MAIN_FILE)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)-linux"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GO) test ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Run the application (defaulting to your current platform binary)
run: build-mac
	@echo "Starting $(APP_NAME) on Mac..."
	@$(BUILD_DIR)/$(APP_NAME)-mac

# Install the application
install: build-mac
	@echo "Installing $(APP_NAME)..."
	@cp $(BUILD_DIR)/$(APP_NAME)-mac $(GOPATH)/bin/
	@echo "Installation complete: $(GOPATH)/bin/$(APP_NAME)"

# Help message
help:
	@echo "Available targets:"
	@echo "  all         - Clean and build the binaries for Mac and Linux"
	@echo "  build-mac   - Build the Mac binary"
	@echo "  build-linux - Build the Linux binary"
	@echo "  clean       - Remove build artifacts"
	@echo "  test        - Run tests"
	@echo "  lint        - Run linter"
	@echo "  run         - Build and run the Mac binary"
	@echo "  install     - Install the Mac binary to GOPATH/bin"
	@echo "  help        - Show this help message"