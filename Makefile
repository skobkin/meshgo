# MeshGo Build System

# Build variables
BINARY_NAME := meshgo
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS := -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.Commit=${COMMIT}"
BUILD_DIR := build
DIST_DIR := dist

# Platform-specific variables
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

.PHONY: all build test clean deps fmt vet lint run dev package install help

all: test build

## Development
dev: deps fmt vet test build run  ## Run full development cycle

run: build  ## Run the application
	./$(BUILD_DIR)/$(BINARY_NAME)

## Building
build: deps  ## Build for current platform
	@echo "Building $(BINARY_NAME) $(VERSION) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/meshgo

build-linux: deps  ## Build for Linux x64
	@echo "Building $(BINARY_NAME) for Linux x64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/meshgo

build-windows: deps  ## Build for Windows x64
	@echo "Building $(BINARY_NAME) for Windows x64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/meshgo

build-macos: deps  ## Build for macOS x64
	@echo "Building $(BINARY_NAME) for macOS x64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/meshgo

build-macos-arm64: deps  ## Build for macOS ARM64
	@echo "Building $(BINARY_NAME) for macOS ARM64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/meshgo

build-all: build-linux build-windows build-macos build-macos-arm64  ## Build for all platforms

## Testing
test: deps  ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test  ## Run tests with coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

bench: deps  ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

## Code Quality
fmt:  ## Format Go code
	@echo "Formatting code..."
	go fmt ./...

vet:  ## Run go vet
	@echo "Running go vet..."
	go vet ./...

lint:  ## Run golangci-lint (requires golangci-lint installed)
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install it with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2"; \
	fi

## Dependencies
deps:  ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

deps-update:  ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

## Packaging
package: build-all  ## Create distribution packages
	@echo "Creating packages..."
	@mkdir -p $(DIST_DIR)
	
	# Linux
	@tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-linux-amd64
	
	# Windows  
	@zip -j $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe
	
	# macOS x64
	@tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-darwin-amd64
	
	# macOS ARM64
	@tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-darwin-arm64
	
	@echo "Packages created in $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/

## Installation
install: build  ## Install binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME) to $$(go env GOPATH)/bin..."
	go install $(LDFLAGS) ./cmd/meshgo

install-tools:  ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

## Cleanup
clean:  ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -f coverage.out coverage.html

## Docker (optional)
docker-build:  ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .

docker-run: docker-build  ## Run in Docker container
	docker run --rm -it $(BINARY_NAME):$(VERSION)

## Database
db-reset:  ## Reset database (development only)
	@echo "Resetting database..."
	@if [ -d "$$HOME/.config/meshgo" ]; then \
		rm -f $$HOME/.config/meshgo/meshgo.db*; \
		echo "Database reset"; \
	else \
		echo "No database found"; \
	fi

## Git hooks
hooks:  ## Install git hooks
	@echo "Installing git hooks..."
	@mkdir -p .git/hooks
	@echo '#!/bin/sh\nmake fmt vet test' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed"

## Information
version:  ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)" 
	@echo "Commit: $(COMMIT)"
	@echo "Go Version: $$(go version)"

help:  ## Show this help message
	@echo "MeshGo - Meshtastic GUI in Go"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)