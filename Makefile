# Makefile for RDP HTML5 Client

# Variables
.DEFAULT_GOAL := help
BINARY_NAME=go-rdp
BUILD_DIR=bin
GO_VERSION=1.22
LINTER=golangci-lint
VERSION=$(shell cat VERSION)
LDFLAGS=-s -w -X main.appVersion=$(VERSION)

.PHONY: help
help: ## Show this help
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; OFS=""; print ""} /^[a-zA-Z0-9_.-]+:.*##/ {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: all
all: deps check test build ## Run deps, checks, tests, and build (default)

# Dependencies
.PHONY: deps
deps: ## Install Go and tooling dependencies
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
check: vet lint ## Run vet and lint
	@echo "All code quality checks passed"

# Go vet
.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...
	@echo "go vet passed"

# Linting (golangci-lint)
.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@if command -v $(LINTER) >/dev/null 2>&1; then \
		$(LINTER) run --timeout=5m; \
	else \
		echo "$(LINTER) not found. Run 'make deps' to install it."; \
		echo "Skipping golangci-lint (go vet already ran)"; \
	fi

# Testing
.PHONY: test
test: ## Run unit tests with race + coverage
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage_all.out ./...
	cp coverage_all.out coverage.out
	go tool cover -html=coverage_all.out -o coverage.html

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	go test -v -race -count=1 ./...

.PHONY: test-int
test-int: ## Run integration tests
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

.PHONY: test-rfx
test-rfx: ## Run RemoteFX codec tests
	@echo "Running RemoteFX codec tests..."
	go test -v -race ./internal/codec/rfx/...

.PHONY: test-js
test-js: ## Run JavaScript fallback codec tests
	@echo "Running JavaScript tests..."
	cd web/src/js && node --test codec-fallback.test.js

.PHONY: test-e2e
test-e2e: ## Run Playwright browser tests (requires server running on :8080)
	@echo "Running end-to-end browser tests..."
	@echo "NOTE: Server must be running on localhost:8080"
	@echo "      Start with: make run (in another terminal)"
	cd web/src/js && npx playwright test

.PHONY: test-e2e-firefox
test-e2e-firefox: ## Run Playwright tests in Firefox only
	cd web/src/js && npx playwright test --project=firefox

.PHONY: test-e2e-chromium
test-e2e-chromium: ## Run Playwright tests in Chromium only
	cd web/src/js && npx playwright test --project=chromium

.PHONY: test-e2e-headed
test-e2e-headed: ## Run Playwright tests with visible browser
	cd web/src/js && npx playwright test --headed

.PHONY: test-e2e-debug
test-e2e-debug: ## Run Playwright tests in debug mode
	cd web/src/js && npx playwright test --debug

# Building
.PHONY: build
build: build-frontend build-backend ## Build frontend (WASM+JS) and backend
	@echo "Full build complete:"
	@echo "  - WASM:    web/dist/js/rle/rle.wasm"
	@echo "  - JS:      web/dist/js/client.bundle.min.js"
	@echo "  - Binary:  $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-backend
build-backend: ## Build Go backend only
	@echo "Building Go backend binary (version $(VERSION))..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) cmd/server/main.go
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)

.PHONY: build-frontend
build-frontend: build-html build-wasm build-js-min ## Build WASM and JS assets
	@echo "Frontend assets built (HTML + WASM + JS)"

.PHONY: build-html
build-html: ## Copy HTML/manifest/PWA assets from src to dist
	@echo "Copying HTML/manifest/PWA assets to dist..."
	@mkdir -p web/dist web/dist/pwa
	@cp web/src/*.html web/dist/
	@cp web/src/*.webmanifest web/dist/
	@cp web/src/*.js web/dist/
	@cp web/src/pwa/*.png web/dist/pwa/
	@echo "HTML/manifest/PWA assets copied to web/dist/"

.PHONY: build-all
build-all: build-frontend ## Build binaries for common platforms
	@echo "Building binaries for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 cmd/server/main.go
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 cmd/server/main.go
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/server/main.go
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/server/main.go
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/server/main.go
	
	@echo "All binaries built in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

.PHONY: build-wasm
build-wasm: ## Build WebAssembly module (requires TinyGo)
	@echo "Building Go WebAssembly RLE module with TinyGo (optimized for size)..."
	@mkdir -p web/dist/js/rle
	@if ! command -v tinygo >/dev/null 2>&1; then \
		echo ""; \
		echo "ERROR: TinyGo is required to build WASM artifacts but was not found."; \
		echo ""; \
		echo "  Install it with:  brew install tinygo-org/tools/tinygo"; \
		echo "  Or see:           https://tinygo.org/getting-started/install/"; \
		echo ""; \
		echo "  Standard Go WASM builds produce ~2 MB+ output vs ~400 KB with TinyGo."; \
		echo "  TinyGo is mandatory for distribution builds."; \
		echo ""; \
		exit 1; \
	fi
	tinygo build -o web/dist/js/rle/rle.wasm -target wasm -opt=z ./web/src/wasm/
	cp "$$(tinygo env TINYGOROOT)/targets/wasm_exec.js" web/dist/js/rle/wasm_exec.js
	@ls -lh web/dist/js/rle/rle.wasm
	@echo "WebAssembly module built: web/dist/js/rle/rle.wasm"

# JavaScript bundling
.PHONY: build-js
build-js: ## Build JavaScript bundle (non-minified)
	@echo "Building JavaScript bundle..."
	@mkdir -p web/dist/js
	@cd web/src/js && npm install --silent 2>/dev/null && npm run build || \
		npx --yes esbuild index.js --bundle --outfile=../../dist/js/client.bundle.js --format=iife --global-name=RDP
	@ls -lh web/dist/js/client.bundle.js
	@echo "JavaScript bundle built: web/dist/js/client.bundle.js"

.PHONY: build-js-min
build-js-min: ## Build minified JavaScript bundle
	@echo "Building minified JavaScript bundle..."
	@mkdir -p web/dist/js
	@cd web/src/js && npm install --silent 2>/dev/null && npm run build:min || \
		npx --yes esbuild index.js --bundle --minify --outfile=../../dist/js/client.bundle.min.js --format=iife --global-name=RDP
	@ls -lh web/dist/js/client.bundle.min.js
	@echo "Minified JavaScript bundle built: web/dist/js/client.bundle.min.js"

# Docker
.PHONY: docker
docker: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t rdp-html5:latest .
	docker build -t rdp-html5:$$(git rev-parse --short HEAD) .

# Development
.PHONY: run
run: build ## Build and run server locally
	@echo "Starting server..."
	./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: dev
dev: ## Run server in development mode
	@echo "Starting development server..."
	go run cmd/server/main.go

# Code quality
.PHONY: fmt
fmt: ## Format Go code (fmt + goimports if available)
	@echo "Formatting Go code..."
	@go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi
	@echo "Formatting complete"

.PHONY: security
security: ## Run gosec security scan
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found. Run 'make deps' to install it."; \
		exit 1; \
	fi

# Installation
.PHONY: install
install: build ## Install binary to GOPATH/bin
	@echo "Installing binary..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $$(go env GOPATH)/bin/
	@echo "Binary installed to $$(go env GOPATH)/bin/$(BINARY_NAME)"

# Cleanup
.PHONY: clean
clean: ## Clean build artifacts and caches
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage_all.out coverage.html
	go clean -cache
	go clean -testcache

.PHONY: clean-frontend
clean-frontend: ## Clean frontend build artifacts
	@echo "Cleaning frontend build artifacts..."
	rm -f web/dist/*.html web/dist/*.webmanifest web/dist/*.js
	rm -rf web/dist/pwa
	rm -f web/dist/js/*.js web/dist/js/*.js.map
	rm -f web/dist/js/rle/*.wasm web/dist/js/rle/*.js

.PHONY: clean-all
clean-all: clean clean-frontend ## Deep clean (vendor, go mod cache, project docker images)
	@echo "Cleaning everything..."
	rm -rf vendor/
	go mod tidy -cache
	@echo "Removing project Docker images..."
	-docker rmi $$(docker images --filter "reference=rdp-html5*" -q) 2>/dev/null || true
	-docker rmi $$(docker images --filter "reference=*/go-rdp*" -q) 2>/dev/null || true

# Development helpers
.PHONY: setup-dev
setup-dev: deps ## Install pre-commit hooks and air
	@echo "Setting up development environment..."
	pre-commit install
	go install github.com/air-verse/air@latest

.PHONY: watch
watch: ## Live-reload with air (requires setup-dev)
	@echo "Watching for changes..."
	@which air > /dev/null || (echo "air not found. Run 'make setup-dev' to install it." && exit 1)
	air -c .air.toml

# Production deployment helpers
.PHONY: docker-run
docker-run: docker ## Build and run Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 -v $$(pwd)/web:/app/web rdp-html5:latest

.PHONY: docker-compose
docker-compose: ## Start services with docker-compose
	@echo "Starting with Docker Compose..."
	docker-compose up -d

# Check Go version
.PHONY: check-go
check-go: ## Validate minimum Go version
	@go_version=$$(go version | awk '{print $$3}' | cut -c3-); \
	if [ "$$(printf '%s\n' "$(GO_VERSION)" "$$go_version" | sort -V | head -n1)" != "$(GO_VERSION)" ]; then \
		echo "Error: Go version $$go_version is not supported. Please install Go $(GO_VERSION) or later."; \
		exit 1; \
	else \
		echo "Go version $$go_version is compatible."; \
	fi

# Show configuration
.PHONY: config
config: ## Show build configuration
	@echo "Build configuration:"
	@echo "  Version: $(VERSION)"
	@echo "  Go Version: $(GO_VERSION)"
	@echo "  Binary Name: $(BINARY_NAME)"
	@echo "  Build Directory: $(BUILD_DIR)"
	@echo "  Linter: $(LINTER)"

# Quick test and build
.PHONY: ci
ci: deps check security test build ## Run full CI pipeline
	@echo "CI pipeline completed successfully"

# Release helpers
.PHONY: bump-patch
bump-patch: ## Bump patch version and create git tag
	@OLD=$$(cat VERSION); \
	MAJOR=$$(echo $$OLD | cut -d. -f1); \
	MINOR=$$(echo $$OLD | cut -d. -f2); \
	PATCH=$$(echo $$OLD | cut -d. -f3); \
	NEW="$$MAJOR.$$MINOR.$$((PATCH + 1))"; \
	echo "$$NEW" > VERSION; \
	git add VERSION; \
	git commit -m "Bump version to $$NEW"; \
	git tag "v$$NEW"; \
	echo "Bumped version: $$OLD -> $$NEW (tagged v$$NEW)"

.PHONY: push
push: ## Push commits and current tag to origin
	@TAG=$$(git describe --tags --exact-match 2>/dev/null); \
	git push origin $$(git rev-parse --abbrev-ref HEAD); \
	if [ -n "$$TAG" ]; then \
		echo "Pushing tag $$TAG..."; \
		git push origin "$$TAG"; \
	else \
		echo "No tag on current commit"; \
	fi
