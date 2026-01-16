# Makefile for RDP HTML5 Client

# Variables
BINARY_NAME=rdp-html5
BUILD_DIR=bin
GO_VERSION=1.22
LINTER=golangci-lint

# Default target
.PHONY: all
all: deps lint test build

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  deps        - Download and install dependencies"
	@echo "  lint        - Run linter and code quality checks"
	@echo "  test        - Run unit tests with coverage"
	@echo "  test-race  - Run tests with race detection"
	@echo "  test-int    - Run integration tests"
	@echo "  build       - Build the binary for current platform"
	@echo "  build-all   - Build binaries for all platforms"
	@echo "  build-wasm  - Build WebAssembly module"
	@echo "  docker      - Build Docker image"
	@echo "  run         - Build and run the server"
	@echo "  clean       - Clean build artifacts and dependencies"
	@echo "  clean-all   - Clean everything including caches"
	@echo "  fmt         - Format Go code"
	@echo "  vet         - Run go vet"
	@echo "  security    - Run security scan"
	@echo "  install     - Install the binary locally"

# Dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod verify
	@if ! command -v $(LINTER) >/dev/null 2>&1; then \
		echo "Installing $(LINTER)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
	fi
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	fi

# Linting
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v $(LINTER) >/dev/null 2>&1; then \
		$(LINTER) run --timeout=5m; \
	else \
		echo "Linter not found. Run 'make deps' to install it."; \
		exit 1; \
	fi

# Testing
.PHONY: test
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	go test -v -race -count=1 ./...

.PHONY: test-int
test-int:
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

# Building
.PHONY: build
build:
	@echo "Building binary..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) cmd/server/main.go
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-all
build-all:
	@echo "Building binaries for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 cmd/server/main.go
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 cmd/server/main.go
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/server/main.go
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/server/main.go
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/server/main.go
	
	@echo "All binaries built in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

.PHONY: build-wasm
build-wasm:
	@echo "Building WebAssembly module..."
	@mkdir -p $(BUILD_DIR)
	@cd web/js/rle && GOOS=js GOARCH=wasm go build -o $(BUILD_DIR)/ms-rle-wasm.wasm .
	@echo "WebAssembly module built: $(BUILD_DIR)/ms-rle-wasm.wasm"

# Docker
.PHONY: docker
docker:
	@echo "Building Docker image..."
	docker build -t rdp-html5:latest .
	docker build -t rdp-html5:$$(git rev-parse --short HEAD) .

# Development
.PHONY: run
run: build
	@echo "Starting server..."
	./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: dev
dev:
	@echo "Starting development server..."
	go run cmd/server/main.go

# Code quality
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	go fmt ./...
	goimports -w .

.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

.PHONY: security
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found. Run 'make deps' to install it."; \
		exit 1; \
	fi

# Installation
.PHONY: install
install: build
	@echo "Installing binary..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $$(go env GOPATH)/bin/
	@echo "Binary installed to $$(go env GOPATH)/bin/$(BINARY_NAME)"

# Cleanup
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean -cache
	go clean -testcache

.PHONY: clean-all
clean-all: clean
	@echo "Cleaning everything..."
	rm -rf vendor/
	go mod tidy -cache
	docker system prune -f

# Development helpers
.PHONY: setup-dev
setup-dev: deps
	@echo "Setting up development environment..."
	pre-commit install
	go install github.com/air-verse/air@latest

.PHONY: watch
watch:
	@echo "Watching for changes..."
	@which air > /dev/null || (echo "air not found. Run 'make setup-dev' to install it." && exit 1)
	air -c .air.toml

# Production deployment helpers
.PHONY: docker-run
docker-run: docker
	@echo "Running Docker container..."
	docker run -p 8080:8080 -v $$(pwd)/web:/app/web rdp-html5:latest

.PHONY: docker-compose
docker-compose:
	@echo "Starting with Docker Compose..."
	docker-compose up -d

# Check Go version
.PHONY: check-go
check-go:
	@go_version=$$(go version | awk '{print $$3}' | cut -c3-); \
	if [ "$$(printf '%s\n' "$(GO_VERSION)" "$$go_version" | sort -V | head -n1)" != "$(GO_VERSION)" ]; then \
		echo "Error: Go version $$go_version is not supported. Please install Go $(GO_VERSION) or later."; \
		exit 1; \
	else \
		echo "Go version $$go_version is compatible."; \
	fi

# Show configuration
.PHONY: config
config:
	@echo "Build configuration:"
	@echo "  Go Version: $(GO_VERSION)"
	@echo "  Binary Name: $(BINARY_NAME)"
	@echo "  Build Directory: $(BUILD_DIR)"
	@echo "  Linter: $(LINTER)"

# Quick test and build
.PHONY: ci
ci: deps lint test build
	@echo "CI pipeline completed successfully"
