# Track Studio Orchestrator - Makefile
# Professional build and deployment automation

# Configuration
APP_NAME := track-studio-orchestrator
VERSION := 0.1.0
BUILD_DIR := bin
MAIN_PATH := cmd/server/main.go
BINARY_NAME := trackstudio-server

# Go build flags
GOOS ?= linux
GOARCH ?= amd64
GO_BUILD_FLAGS := -ldflags="-s -w -X main.Version=$(VERSION)"

# Deployment targets
MULE_HOST := mule.nlaakstudios
MULE_USER := andrew
MULE_PATH := /home/$(MULE_USER)/trackstudio/orchestrator
MULE_SERVICE := track-studio-orchestrator

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

.PHONY: all help build clean test run dev install deps docker deploy-mule status-mule logs-mule restart-mule

## help: Display this help message
help:
	@echo "$(COLOR_BOLD)Track Studio Orchestrator - Available Commands$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)Development:$(COLOR_RESET)"
	@echo "  make dev          - Run in development mode (hot reload)"
	@echo "  make run          - Build and run locally"
	@echo "  make test         - Run all tests"
	@echo "  make test-verbose - Run tests with verbose output"
	@echo "  make fmt          - Format Go code"
	@echo "  make lint         - Run linters"
	@echo ""
	@echo "$(COLOR_BLUE)Build:$(COLOR_RESET)"
	@echo "  make build        - Build binary for current platform"
	@echo "  make build-linux  - Build binary for Linux (production)"
	@echo "  make build-all    - Build for all platforms"
	@echo "  make clean        - Remove build artifacts"
	@echo ""
	@echo "$(COLOR_BLUE)Dependencies:$(COLOR_RESET)"
	@echo "  make deps         - Download Go dependencies"
	@echo "  make deps-update  - Update all dependencies"
	@echo "  make deps-verify  - Verify dependencies"
	@echo ""
	@echo "$(COLOR_BLUE)Deployment:$(COLOR_RESET)"
	@echo "  make deploy-mule  - Deploy to mule.nlaakstudios"
	@echo "  make status-mule  - Check service status on mule"
	@echo "  make logs-mule    - View service logs on mule"
	@echo "  make restart-mule - Restart service on mule"
	@echo "  make ssh-mule     - SSH into mule server"
	@echo ""
	@echo "$(COLOR_BLUE)Database:$(COLOR_RESET)"
	@echo "  make db-init      - Initialize local database"
	@echo "  make db-migrate   - Run database migrations"
	@echo "  make db-seed      - Seed database with test data"
	@echo "  make db-reset     - Reset database (drop and recreate)"
	@echo ""

## build: Build the application binary
build:
	@echo "$(COLOR_GREEN)Building $(APP_NAME) v$(VERSION)...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(COLOR_GREEN)✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(COLOR_RESET)"

## build-linux: Build for Linux (production)
build-linux:
	@echo "$(COLOR_GREEN)Building for Linux ($(GOARCH))...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=$(GOARCH) go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux $(MAIN_PATH)
	@echo "$(COLOR_GREEN)✓ Linux build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux$(COLOR_RESET)"

## build-all: Build for all platforms
build-all: build-linux
	@echo "$(COLOR_GREEN)Building for macOS (amd64)...$(COLOR_RESET)"
	GOOS=darwin GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	@echo "$(COLOR_GREEN)Building for macOS (arm64)...$(COLOR_RESET)"
	GOOS=darwin GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "$(COLOR_GREEN)Building for Windows (amd64)...$(COLOR_RESET)"
	GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows.exe $(MAIN_PATH)
	@echo "$(COLOR_GREEN)✓ All builds complete$(COLOR_RESET)"

## clean: Remove build artifacts and temporary files
clean:
	@echo "$(COLOR_YELLOW)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(BUILD_DIR)
	@rm -rf storage/temp/*
	@echo "$(COLOR_GREEN)✓ Clean complete$(COLOR_RESET)"

## deps: Download Go dependencies
deps:
	@echo "$(COLOR_BLUE)Downloading dependencies...$(COLOR_RESET)"
	go mod download
	go mod verify
	@echo "$(COLOR_GREEN)✓ Dependencies ready$(COLOR_RESET)"

## deps-update: Update all dependencies
deps-update:
	@echo "$(COLOR_BLUE)Updating dependencies...$(COLOR_RESET)"
	go get -u ./...
	go mod tidy
	@echo "$(COLOR_GREEN)✓ Dependencies updated$(COLOR_RESET)"

## deps-verify: Verify dependencies
deps-verify:
	@echo "$(COLOR_BLUE)Verifying dependencies...$(COLOR_RESET)"
	go mod verify
	@echo "$(COLOR_GREEN)✓ Dependencies verified$(COLOR_RESET)"

## test: Run all tests
test:
	@echo "$(COLOR_BLUE)Running tests...$(COLOR_RESET)"
	go test -v -race -coverprofile=coverage.out ./...
	@echo "$(COLOR_GREEN)✓ Tests complete$(COLOR_RESET)"

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "$(COLOR_BLUE)Running tests (verbose)...$(COLOR_RESET)"
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

## fmt: Format Go code
fmt:
	@echo "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	go fmt ./...
	@echo "$(COLOR_GREEN)✓ Code formatted$(COLOR_RESET)"

## lint: Run linters
lint:
	@echo "$(COLOR_BLUE)Running linters...$(COLOR_RESET)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "$(COLOR_YELLOW)⚠ golangci-lint not installed, skipping$(COLOR_RESET)"; \
	fi
	@echo "$(COLOR_GREEN)✓ Lint complete$(COLOR_RESET)"

## run: Build and run locally
run: build
	@echo "$(COLOR_GREEN)Starting $(APP_NAME)...$(COLOR_RESET)"
	./$(BUILD_DIR)/$(BINARY_NAME)

## dev: Run in development mode (requires air for hot reload)
dev:
	@echo "$(COLOR_GREEN)Starting development server...$(COLOR_RESET)"
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "$(COLOR_YELLOW)⚠ 'air' not installed, using 'go run' instead$(COLOR_RESET)"; \
		echo "$(COLOR_YELLOW)  Install air: go install github.com/cosmtrek/air@latest$(COLOR_RESET)"; \
		go run $(MAIN_PATH); \
	fi

## db-init: Initialize local database
db-init:
	@echo "$(COLOR_BLUE)Initializing database...$(COLOR_RESET)"
	@mkdir -p data
	@sqlite3 data/songs.db < scripts/schema.sql 2>/dev/null || echo "$(COLOR_YELLOW)⚠ Schema file not found, skipping$(COLOR_RESET)"
	@echo "$(COLOR_GREEN)✓ Database initialized$(COLOR_RESET)"

## db-migrate: Run database migrations
db-migrate:
	@echo "$(COLOR_BLUE)Running migrations...$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)⚠ Migration system not implemented yet$(COLOR_RESET)"

## db-seed: Seed database with test data
db-seed:
	@echo "$(COLOR_BLUE)Seeding database...$(COLOR_RESET)"
	@sqlite3 data/songs.db < scripts/seed.sql 2>/dev/null || echo "$(COLOR_YELLOW)⚠ Seed file not found, skipping$(COLOR_RESET)"
	@echo "$(COLOR_GREEN)✓ Database seeded$(COLOR_RESET)"

## db-reset: Reset database (drop and recreate)
db-reset:
	@echo "$(COLOR_YELLOW)Resetting database...$(COLOR_RESET)"
	@rm -f data/songs.db
	@$(MAKE) db-init
	@$(MAKE) db-seed
	@echo "$(COLOR_GREEN)✓ Database reset complete$(COLOR_RESET)"

## deploy-mule: Deploy to mule.nlaakstudios
deploy-mule: build-linux
	@echo "$(COLOR_GREEN)Deploying to $(MULE_HOST)...$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)→ Creating directories...$(COLOR_RESET)"
	@ssh $(MULE_USER)@$(MULE_HOST) "mkdir -p $(MULE_PATH)/{bin,data,storage/{songs,videos,temp},config,scripts}"
	@echo "$(COLOR_BLUE)→ Uploading binary...$(COLOR_RESET)"
	@scp $(BUILD_DIR)/$(BINARY_NAME)-linux $(MULE_USER)@$(MULE_HOST):$(MULE_PATH)/bin/$(BINARY_NAME)
	@ssh $(MULE_USER)@$(MULE_HOST) "chmod +x $(MULE_PATH)/bin/$(BINARY_NAME)"
	@echo "$(COLOR_BLUE)→ Uploading configuration...$(COLOR_RESET)"
	@rsync -avz --exclude='data/' --exclude='storage/' --exclude='bin/' --exclude='.git/' \
		./config/ $(MULE_USER)@$(MULE_HOST):$(MULE_PATH)/config/ || true
	@rsync -avz --exclude='.git/' \
		./scripts/ $(MULE_USER)@$(MULE_HOST):$(MULE_PATH)/scripts/ || true
	@echo "$(COLOR_BLUE)→ Restarting service...$(COLOR_RESET)"
	@ssh $(MULE_USER)@$(MULE_HOST) "sudo systemctl restart $(MULE_SERVICE) 2>/dev/null || echo 'Service not configured yet'"
	@echo "$(COLOR_GREEN)✓ Deployment complete!$(COLOR_RESET)"
	@echo ""
	@echo "Check status with: $(COLOR_BOLD)make status-mule$(COLOR_RESET)"
	@echo "View logs with:    $(COLOR_BOLD)make logs-mule$(COLOR_RESET)"

## status-mule: Check service status on mule
status-mule:
	@echo "$(COLOR_BLUE)Checking service status on $(MULE_HOST)...$(COLOR_RESET)"
	@ssh $(MULE_USER)@$(MULE_HOST) "sudo systemctl status $(MULE_SERVICE) --no-pager"

## logs-mule: View service logs on mule
logs-mule:
	@echo "$(COLOR_BLUE)Viewing logs on $(MULE_HOST) (Ctrl+C to exit)...$(COLOR_RESET)"
	@ssh $(MULE_USER)@$(MULE_HOST) "sudo journalctl -u $(MULE_SERVICE) -f"

## restart-mule: Restart service on mule
restart-mule:
	@echo "$(COLOR_BLUE)Restarting service on $(MULE_HOST)...$(COLOR_RESET)"
	@ssh $(MULE_USER)@$(MULE_HOST) "sudo systemctl restart $(MULE_SERVICE)"
	@echo "$(COLOR_GREEN)✓ Service restarted$(COLOR_RESET)"
	@$(MAKE) status-mule

## ssh-mule: SSH into mule server
ssh-mule:
	@echo "$(COLOR_BLUE)Connecting to $(MULE_HOST)...$(COLOR_RESET)"
	@ssh $(MULE_USER)@$(MULE_HOST)

## install: Install development tools
install:
	@echo "$(COLOR_BLUE)Installing development tools...$(COLOR_RESET)"
	@echo "→ Installing air (hot reload)..."
	@go install github.com/cosmtrek/air@latest
	@echo "→ Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(COLOR_GREEN)✓ Development tools installed$(COLOR_RESET)"

## version: Display version information
version:
	@echo "$(COLOR_BOLD)$(APP_NAME) v$(VERSION)$(COLOR_RESET)"
	@echo "Go version: $(shell go version)"
	@echo "Build dir:  $(BUILD_DIR)"

# Default target
all: deps build

# Development quick commands
.PHONY: quick quick-deploy
quick: clean build run
quick-deploy: clean build-linux deploy-mule
