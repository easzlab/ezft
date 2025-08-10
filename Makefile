# EZFT - Easy File Transfer Tool Makefile

# Project information
PROJECT_NAME := ezft
MODULE_NAME := github.com/easzlab/ezft
BUILD_TIME := $(shell date +%Y-%m-%d\ %H:%M:%S)
BUILD_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Build configuration
BUILD_DIR := build
MAIN_FILE := cmd/main.go
BINARY_NAME := $(PROJECT_NAME)
BINARY_PATH := $(BUILD_DIR)/$(BINARY_NAME)

# Go configuration
GO := go
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Linker flags for version information
LDFLAGS := -ldflags "-X '$(MODULE_NAME)/internal/config.BuildTime=$(BUILD_TIME)' \
                     -X '$(MODULE_NAME)/internal/config.BuildCommit=$(BUILD_COMMIT)' \
					 -X '$(MODULE_NAME)/internal/config.BuildBranch=$(BUILD_BRANCH)' \
                     -s -w -extldflags -static"

# Default target
.PHONY: all
all: clean build

.PHONY: help
help: ## show help
	@echo "Available targets: "
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## build the binary
	@echo "Building $(PROJECT_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BINARY_PATH) $(MAIN_FILE)
	@echo "✓ Build completed: $(BINARY_PATH)"

.PHONY: build-dev
build-dev: ## build the binary with debug info
	@echo "Building $(PROJECT_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -race -o $(BINARY_PATH) $(MAIN_FILE)
	@echo "✓ Development build completed: $(BINARY_PATH)"

.PHONY: build-linux
build-linux: ## build the binary for Linux
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_FILE)
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_FILE)
	@echo "✓ Linux build completed"

.PHONY: build-windows
build-windows: ## build the binary for Windows
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_FILE)
	@echo "✓ Windows build completed"

.PHONY: build-darwin
build-darwin: ## build the binary for macOS
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_FILE)
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_FILE)
	@echo "✓ macOS builds completed"

.PHONY: build-all
build-all: build-linux build-windows build-darwin ## build the binary for all platforms
	@echo "✓ All platform builds completed"

.PHONY: test
test: ## run tests
	@echo "Running tests..."
	$(GO) test -v ./...
	@echo "✓ Tests completed"

.PHONY: test-cov
test-cov: ## run tests with coverage
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

.PHONY: bench
bench: ## run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

.PHONY: deps
deps: ## download dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download
	@echo "✓ Dependencies downloaded"

.PHONY: clean
clean: ## clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "✓ Clean completed"

.PHONY: update
update: ## update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "✓ Dependencies updated"