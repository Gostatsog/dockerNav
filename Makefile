.PHONY: build clean test lint run install

# Application name
APP_NAME := dockerNav
# Go executable
GO := go

# Build directory
BUILD_DIR := build
# Main file path
MAIN_FILE := cmd/dockerNav/main.go

# Default target
all: clean build

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GO) test ./...

# Run linting
lint:
	@echo "Running linter..."
	golangci-lint run

# Run the application
run: build
	@echo "Starting $(APP_NAME)..."
	@$(BUILD_DIR)/$(APP_NAME)

# Install the application
install: build
	@echo "Installing $(APP_NAME)..."
	@cp $(BUILD_DIR)/$(APP_NAME) $(GOPATH)/bin/
	@echo "Installation complete: $(GOPATH)/bin/$(APP_NAME)"

# List targets
help:
	@echo "Available targets:"
	@echo "  all      - Clean and build the application"
	@echo "  build    - Build the application"
	@echo "  clean    - Remove build artifacts"
	@echo "  test     - Run tests"
	@echo "  lint     - Run linter"
	@echo "  run      - Build and run the application"
	@echo "  install  - Install the application to GOPATH/bin"
	@echo "  help     - Show this help message"