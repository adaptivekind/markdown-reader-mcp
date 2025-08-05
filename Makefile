.PHONY: build test test-verbose test-specific vet run clean install dev help

# Default target
all: build test

# Build the server binary
build:
	go build

# Run all tests
test:
	go test

# Run tests with verbose output
test-verbose:
	go test -v

# Run a specific test (usage: make test-specific TEST=TestServerInitialization)
test-specific:
	go test -v -run $(TEST)

# Run go vet
vet:
	go vet ./...

# Run the server with current directory
run: build
	./markdown-reader-mcp .

# Run the server with example directories
run-multi: build
	./markdown-reader-mcp docs guides .

# Clean build artifacts
clean:
	rm -f markdown-reader-mcp

# Install dependencies
install:
	go mod tidy

# Development mode - build and run with current directory
dev: build
	./markdown-reader-mcp .

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the server binary"
	@echo "  test         - Run all tests"
	@echo "  test-verbose - Run tests with verbose output"
	@echo "  test-specific - Run specific test (use TEST=TestName)"
	@echo "  vet          - Run go vet"
	@echo "  run          - Build and run server with current directory"
	@echo "  run-multi    - Build and run server with multiple directories"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install/update dependencies"
	@echo "  dev          - Development mode (build and run)"
	@echo "  help         - Show this help message"
