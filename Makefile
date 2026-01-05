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
MULE_HOST := andrew@192.168.1.200
MULE_PATH := /home/andrew/trackstudio/orchestrator
MULE_DATA_PATH := /home/andrew/track-studio-data
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
	@echo "  make deploy-mule       - Deploy binary to mule.nlaakstudios"
	@echo "  make deploy-mule-data  - Sync database and storage to mule"
	@echo "  make status-mule       - Check service status on mule"
	@echo "  make logs-mule         - View service logs on mule"
	@echo "  make restart-mule      - Restart service on mule"
	@echo "  make test-cqai         - Test connection to cqai from mule"
	@echo "  make ssh-mule          - SSH into mule server"
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
	@echo "$(COLOR_GREEN)Copying Python scripts to data directory...$(COLOR_RESET)"
	@mkdir -p ~/track-studio-data/python-scripts
	@cp -r python-scripts/* ~/track-studio-data/python-scripts/
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
	@echo ""
	@echo "$(COLOR_BLUE)→ Stopping existing server...$(COLOR_RESET)"
	@ssh $(MULE_HOST) "pkill -f trackstudio-server" 2>/dev/null || true
	@sleep 1
	@echo "$(COLOR_BLUE)→ Creating directories...$(COLOR_RESET)"
	@ssh $(MULE_HOST) "mkdir -p $(MULE_PATH)/{bin,config,scripts}"
	@ssh $(MULE_HOST) "mkdir -p $(MULE_DATA_PATH)/{audio,images,videos,temp,logs,python-scripts,branding}"
	@echo "$(COLOR_BLUE)→ Uploading binary...$(COLOR_RESET)"
	@scp $(BUILD_DIR)/$(BINARY_NAME)-linux $(MULE_HOST):$(MULE_PATH)/bin/$(BINARY_NAME)
	@ssh $(MULE_HOST) "chmod +x $(MULE_PATH)/bin/$(BINARY_NAME)"
	@echo "$(COLOR_BLUE)→ Uploading Python scripts...$(COLOR_RESET)"
	@rsync -avz --progress python-scripts/ $(MULE_HOST):$(MULE_DATA_PATH)/python-scripts/
	@echo "$(COLOR_BLUE)→ Starting server...$(COLOR_RESET)"
	@ssh $(MULE_HOST) "cd $(MULE_PATH) && nohup ./bin/$(BINARY_NAME) > server.log 2>&1 &"
	@sleep 2
	@echo "$(COLOR_GREEN)✓ Deployment complete!$(COLOR_RESET)"
	@echo ""
	@echo "Server running on: $(COLOR_BOLD)http://192.168.1.200:8080$(COLOR_RESET)"
	@echo ""
	@echo "To copy database and storage: $(COLOR_BOLD)make deploy-mule-data$(COLOR_RESET)"
	@echo "Check status with:            $(COLOR_BOLD)make status-mule$(COLOR_RESET)"
	@echo "View logs with:               $(COLOR_BOLD)make logs-mule$(COLOR_RESET)"

## deploy-mule-data: Sync database and storage to mule
deploy-mule-data:
	@echo "$(COLOR_YELLOW)Syncing database and storage to $(MULE_HOST)...$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)⚠ This will overwrite data on mule!$(COLOR_RESET)"
	@echo ""
	@read -p "Continue? (y/N): " confirm; \
	if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then \
		echo "$(COLOR_YELLOW)Cancelled.$(COLOR_RESET)"; \
		exit 1; \
	fi; \
	echo "$(COLOR_BLUE)→ Stopping server...$(COLOR_RESET)"; \
	ssh $(MULE_HOST) "pkill -f trackstudio-server" 2>/dev/null || true; \
	sleep 1; \
	echo "$(COLOR_BLUE)→ Syncing track-studio-data (this may take a while)...$(COLOR_RESET)"; \
	rsync -avz --progress ~/track-studio-data/ $(MULE_HOST):$(MULE_DATA_PATH)/ \
		--exclude='*.log' --exclude='temp/*' --exclude='.venv'; \
	echo "$(COLOR_BLUE)→ Starting server...$(COLOR_RESET)"; \
	ssh $(MULE_HOST) "cd $(MULE_PATH) && nohup ./bin/$(BINARY_NAME) > server.log 2>&1 &"; \
	sleep 2; \
	echo "$(COLOR_GREEN)✓ Data sync complete!$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)→ Starting server...$(COLOR_RESET)"
	@ssh $(MULE_HOST) "cd $(MULE_PATH) && nohup ./bin/$(BINARY_NAME) > server.log 2>&1 &"
	@sleep 2
	@echo "$(COLOR_GREEN)✓ Deployment complete!$(COLOR_RESET)"
	@echo ""
	@echo "Server running on: $(COLOR_BOLD)http://192.168.1.200:8080$(COLOR_RESET)"
	@echo "Check status with: $(COLOR_BOLD)make status-mule$(COLOR_RESET)"
	@echo "View logs with:    $(COLOR_BOLD)make logs-mule$(COLOR_RESET)"

## status-mule: Check service status on mule
status-mule:
	@echo "$(COLOR_BLUE)Checking service status on $(MULE_HOST)...$(COLOR_RESET)"
	@ssh $(MULE_HOST) "ps aux | grep trackstudio-server | grep -v grep || echo 'Server not running'"

## logs-mule: View service logs on mule
logs-mule:
	@echo "$(COLOR_BLUE)Viewing logs on $(MULE_HOST) (Ctrl+C to exit)...$(COLOR_RESET)"
	@ssh $(MULE_HOST) "tail -f $(MULE_PATH)/server.log"

## restart-mule: Restart service on mule
restart-mule:
	@echo "$(COLOR_BLUE)Restarting service on $(MULE_HOST)...$(COLOR_RESET)"
	@ssh $(MULE_HOST) "pkill -f trackstudio-server || true"
	@sleep 1
	@ssh $(MULE_HOST) "cd $(MULE_PATH) && nohup ./bin/$(BINARY_NAME) > server.log 2>&1 &"
	@sleep 2
	@echo "$(COLOR_GREEN)✓ Service restarted$(COLOR_RESET)"
	@$(MAKE) status-mule

## test-cqai: Test connection to cqai from mule
test-cqai:
	@echo "$(COLOR_BLUE)Testing connections from mule to cqai.nlaakstudios...$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_YELLOW)Testing Ollama API (LLM):$(COLOR_RESET)"
	@ssh $(MULE_HOST) "curl -s http://cqai.nlaakstudios:11434/api/tags | head -20 && echo '$(COLOR_GREEN)✓ Ollama API accessible$(COLOR_RESET)' || echo '$(COLOR_YELLOW)✗ Ollama API not accessible$(COLOR_RESET)'"
	@echo ""
	@echo "$(COLOR_YELLOW)Testing Image Generation API:$(COLOR_RESET)"
	@ssh $(MULE_HOST) "curl -s http://cqai.nlaakstudios/health && echo '$(COLOR_GREEN)✓ Image API accessible$(COLOR_RESET)' || echo '$(COLOR_YELLOW)✗ Image API not accessible$(COLOR_RESET)'"

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
