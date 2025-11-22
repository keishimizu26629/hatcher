# Hatcher - Git Worktree Tool
# Makefile for building, testing, and distributing

# Variables
BINARY_NAME=hatcher
MAIN_PACKAGE=./main.go
BUILD_DIR=build
DIST_DIR=dist
VERSION?=$(shell git describe --tags --always --dirty)
COMMIT?=$(shell git rev-parse --short HEAD)
DATE?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Platform targets
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

.PHONY: all build clean test coverage lint fmt vet install uninstall run help
.PHONY: build-all build-linux build-darwin build-windows
.PHONY: release package docker

# Default target
all: clean fmt vet test build

# Build for current platform
build:
	@echo "🔨 Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "✅ Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for all platforms
build-all: clean
	@echo "🔨 Building $(BINARY_NAME) for all platforms..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(BINARY_NAME)-$$os-$$arch; \
		if [ $$os = "windows" ]; then output_name=$$output_name.exe; fi; \
		echo "Building for $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$$output_name $(MAIN_PACKAGE); \
	done
	@echo "✅ Cross-compilation completed"

# Platform-specific builds
build-linux:
	@echo "🐧 Building for Linux..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)

build-darwin:
	@echo "🍎 Building for macOS..."
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)

build-windows:
	@echo "🪟 Building for Windows..."
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	GOOS=windows GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-arm64.exe $(MAIN_PACKAGE)

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	@echo "✅ Clean completed"

# Run tests
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	@echo "📊 Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

# Run tests with race detection
test-race:
	@echo "🏃 Running tests with race detection..."
	$(GOTEST) -v -race ./...

# Run benchmark tests
test-bench:
	@echo "⚡ Running benchmark tests..."
	$(GOTEST) -v -bench=. -benchmem ./...

# Run tests with verbose coverage
test-coverage-verbose:
	@echo "📊 Running tests with detailed coverage..."
	$(GOTEST) -v -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -func=coverage.out
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Detailed coverage report generated"

# Run all test suites
test-all: test test-race test-bench
	@echo "✅ All test suites completed"

# Run tests with timeout
test-timeout:
	@echo "⏰ Running tests with timeout..."
	$(GOTEST) -v -timeout=5m ./...

# Run tests for specific package
test-package:
	@echo "📦 Running tests for package: $(PKG)"
	@if [ -z "$(PKG)" ]; then \
		echo "❌ Please specify package: make test-package PKG=./internal/git"; \
		exit 1; \
	fi
	$(GOTEST) -v $(PKG)

# Generate test report
test-report:
	@echo "📋 Generating test report..."
	@mkdir -p reports
	$(GOTEST) -v -json ./... > reports/test-results.json
	$(GOTEST) -v -coverprofile=reports/coverage.out ./...
	$(GOCMD) tool cover -html=reports/coverage.out -o reports/coverage.html
	$(GOCMD) tool cover -func=reports/coverage.out > reports/coverage.txt
	@echo "✅ Test reports generated in reports/ directory"

# Lint code
lint:
	@echo "🔍 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format code
fmt:
	@echo "🎨 Formatting code..."
	$(GOFMT) -s -w .
	@echo "✅ Code formatted"

# Vet code
vet:
	@echo "🔎 Vetting code..."
	$(GOCMD) vet ./...
	@echo "✅ Code vetted"

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "✅ Dependencies installed"

# Install binary to system
install: build
	@echo "📥 Installing $(BINARY_NAME)..."
	@if [ -w /usr/local/bin ]; then \
		cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/; \
		echo "✅ Installed to /usr/local/bin/$(BINARY_NAME)"; \
	else \
		echo "⚠️  Cannot write to /usr/local/bin. Try: sudo make install"; \
		exit 1; \
	fi

# Uninstall binary from system
uninstall:
	@echo "🗑️  Uninstalling $(BINARY_NAME)..."
	@if [ -f /usr/local/bin/$(BINARY_NAME) ]; then \
		rm /usr/local/bin/$(BINARY_NAME); \
		echo "✅ Uninstalled from /usr/local/bin/$(BINARY_NAME)"; \
	else \
		echo "ℹ️  $(BINARY_NAME) not found in /usr/local/bin"; \
	fi

# Run the application
run: build
	@echo "🚀 Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Development run with arguments
dev-run:
	@echo "🚀 Running $(BINARY_NAME) in development mode..."
	$(GOCMD) run $(MAIN_PACKAGE) $(ARGS)

# Create release packages
release: build-all
	@echo "📦 Creating release packages..."
	@mkdir -p $(DIST_DIR)/packages
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		binary_name=$(BINARY_NAME)-$$os-$$arch; \
		if [ $$os = "windows" ]; then binary_name=$$binary_name.exe; fi; \
		package_name=$(BINARY_NAME)-$(VERSION)-$$os-$$arch; \
		if [ $$os = "windows" ]; then \
			zip -j $(DIST_DIR)/packages/$$package_name.zip $(DIST_DIR)/$$binary_name README.md LICENSE; \
		else \
			tar -czf $(DIST_DIR)/packages/$$package_name.tar.gz -C $(DIST_DIR) $$binary_name -C .. README.md LICENSE; \
		fi; \
		echo "Created package: $$package_name"; \
	done
	@echo "✅ Release packages created in $(DIST_DIR)/packages/"

# Generate checksums for release
checksums: release
	@echo "🔐 Generating checksums..."
	@cd $(DIST_DIR)/packages && sha256sum * > SHA256SUMS
	@echo "✅ Checksums generated: $(DIST_DIR)/packages/SHA256SUMS"

# Docker build (if Dockerfile exists)
docker:
	@if [ -f deployments/docker/Dockerfile ]; then \
		echo "🐳 Building Docker image..."; \
		docker build -f deployments/docker/Dockerfile -t $(BINARY_NAME):$(VERSION) .; \
		docker build -f deployments/docker/Dockerfile -t $(BINARY_NAME):latest .; \
		echo "✅ Docker image built: $(BINARY_NAME):$(VERSION)"; \
	else \
		echo "⚠️  Dockerfile not found"; \
	fi

# Show help
help:
	@echo "🥇 Hatcher - Git Worktree Tool"
	@echo ""
	@echo "Available targets:"
	@echo "  build        Build for current platform"
	@echo "  build-all    Build for all platforms"
	@echo "  build-linux  Build for Linux (amd64, arm64)"
	@echo "  build-darwin Build for macOS (amd64, arm64)"
	@echo "  build-windows Build for Windows (amd64, arm64)"
	@echo "  clean        Clean build artifacts"
	@echo "  test         Run tests"
	@echo "  coverage     Run tests with coverage report"
	@echo "  lint         Run linter"
	@echo "  fmt          Format code"
	@echo "  vet          Vet code"
	@echo "  deps         Install dependencies"
	@echo "  install      Install binary to system"
	@echo "  uninstall    Uninstall binary from system"
	@echo "  run          Build and run"
	@echo "  dev-run      Run in development mode (use ARGS=... for arguments)"
	@echo "  release      Create release packages"
	@echo "  checksums    Generate checksums for release"
	@echo "  docker       Build Docker image"
	@echo "  help         Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make test"
	@echo "  make install"
	@echo "  make dev-run ARGS='feature/test --dry-run'"
	@echo "  make release"

# Version info
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"
