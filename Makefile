# Makefile for RDP HTML5 Client

# Variables
BINARY_NAME=rdp-html5
BUILD_DIR=bin
GO_VERSION=1.22
LINTER=golangci-lint

# Default target
.PHONY: all
all: deps check test build

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo ""
	@echo "  Building:"
	@echo "    build          - Build everything (WASM + JS + backend binary)"
	@echo "    build-backend  - Build Go backend binary only"
	@echo "    build-frontend - Build frontend assets (WASM + JS)"
	@echo "    build-wasm     - Build WebAssembly module"
	@echo "    build-js       - Bundle JavaScript modules"
	@echo "    build-js-min   - Bundle and minify JavaScript modules"
	@echo "    build-all      - Build binaries for all platforms"
	@echo ""
	@echo "  Quality:"
	@echo "    check       - Run all code quality checks (vet + lint)"
	@echo "    vet         - Run go vet"
	@echo "    lint        - Run golangci-lint"
	@echo "    fmt         - Format Go code"
	@echo "    security    - Run security scan"
	@echo ""
	@echo "  Testing:"
	@echo "    test        - Run unit tests with coverage"
	@echo "    test-race   - Run tests with race detection"
	@echo "    test-int    - Run integration tests"
	@echo ""
	@echo "  Development:"
	@echo "    deps        - Download and install dependencies"
	@echo "    run         - Build and run the server"
	@echo "    dev         - Run server in development mode"
	@echo "    clean       - Clean build artifacts"
	@echo "    clean-all   - Clean everything including caches"
	@echo ""
	@echo "  Deployment:"
	@echo "    docker      - Build Docker image"
	@echo "    install     - Install the binary locally"
	@echo "    ci          - Run full CI pipeline"

# Dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod verify
	@if ! command -v $(LINTER) >/dev/null 2>&1; then \
		echo "Installing $(LINTER)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin latest; \
	fi
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi

# Code quality checks
.PHONY: check
check: vet lint
	@echo "All code quality checks passed"

# Go vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "go vet passed"

# Linting (golangci-lint)
.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@if command -v $(LINTER) >/dev/null 2>&1; then \
		$(LINTER) run --timeout=5m; \
	else \
		echo "$(LINTER) not found. Run 'make deps' to install it."; \
		echo "Skipping golangci-lint (go vet already ran)"; \
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
build: build-frontend build-backend
	@echo "Full build complete:"
	@echo "  - WASM:    web/js/rle/rle.wasm"
	@echo "  - JS:      web/js/client.bundle.min.js"
	@echo "  - Binary:  $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-backend
build-backend:
	@echo "Building Go backend binary..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) cmd/server/main.go
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)

.PHONY: build-frontend
build-frontend: build-wasm build-js-min
	@echo "Frontend assets built (WASM + JS)"

.PHONY: build-all
build-all: build-frontend
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
	@echo "Building Go WebAssembly RLE module with TinyGo (optimized for size)..."
	@if command -v tinygo >/dev/null 2>&1; then \
		tinygo build -o web/js/rle/rle.wasm -target wasm -opt=z ./web/wasm/; \
		cp "$$(tinygo env TINYGOROOT)/targets/wasm_exec.js" web/js/rle/wasm_exec.js; \
	else \
		echo "TinyGo not found, using standard Go (larger output)..."; \
		GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o web/js/rle/rle.wasm ./web/wasm/; \
		GOROOT=$$(go env GOROOT); \
		if [ -f "$$GOROOT/misc/wasm/wasm_exec.js" ]; then \
			cp "$$GOROOT/misc/wasm/wasm_exec.js" web/js/rle/wasm_exec.js; \
		elif [ -f "$$GOROOT/lib/wasm/wasm_exec.js" ]; then \
			cp "$$GOROOT/lib/wasm/wasm_exec.js" web/js/rle/wasm_exec.js; \
		fi; \
	fi
	@ls -lh web/js/rle/rle.wasm
	@echo "WebAssembly module built: web/js/rle/rle.wasm"

# JavaScript bundling
.PHONY: build-js
build-js:
	@echo "Building JavaScript bundle..."
	@cd web/js/src && npm install --silent 2>/dev/null && npm run build || \
		npx --yes esbuild web/js/src/index.js --bundle --outfile=web/js/client.bundle.js --format=iife --global-name=RDP
	@ls -lh web/js/client.bundle.js
	@echo "JavaScript bundle built: web/js/client.bundle.js"

.PHONY: build-js-min
build-js-min:
	@echo "Building minified JavaScript bundle..."
	@cd web/js/src && npm install --silent 2>/dev/null && npm run build:min || \
		npx --yes esbuild web/js/src/index.js --bundle --minify --outfile=web/js/client.bundle.min.js --format=iife --global-name=RDP
	@ls -lh web/js/client.bundle.min.js
	@echo "Minified JavaScript bundle built: web/js/client.bundle.min.js"

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
	@go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi
	@echo "Formatting complete"

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
ci: deps check test build
	@echo "CI pipeline completed successfully"
