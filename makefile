# Makefile for Coffee Roasting E-commerce Application

# Include environment variables from .env file
include .env
export

# Set default values if not provided in .env
GOOSE_DRIVER ?= postgres
GOOSE_MIGRATION_DIR ?= internal/migrations
GOOSE_TABLE ?= goose_db_version

# Construct database connection string from individual components
GOOSE_DBSTRING ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)

# Default target
.DEFAULT_GOAL := help

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: help
help: ## Show this help message
	@echo "$(BLUE)Coffee Roasting E-commerce - Available Commands$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Database Migration Commands
.PHONY: migrate-up
migrate-up: ## Run all pending migrations
	@echo "$(YELLOW)Running migrations up...$(NC)"
	@goose up
	@echo "$(GREEN)Migrations completed successfully!$(NC)"

.PHONY: migrate-down
migrate-down: ## Rollback the last migration
	@echo "$(YELLOW)Rolling back last migration...$(NC)"
	@goose down
	@echo "$(GREEN)Migration rollback completed!$(NC)"

.PHONY: migrate-down-to
migrate-down-to: ## Rollback to specific version (usage: make migrate-down-to VERSION=20240101000000)
	@if [ -z "$(VERSION)" ]; then \
		echo "$(RED)Error: VERSION is required. Usage: make migrate-down-to VERSION=20240101000000$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Rolling back to migration version $(VERSION)...$(NC)"
	@goose -dir $(GOOSE_MIGRATION_DIR) $(GOOSE_DRIVER) "$(GOOSE_DBSTRING)" down-to $(VERSION)
	@echo "$(GREEN)Migration rollback to $(VERSION) completed!$(NC)"

.PHONY: migrate-reset
migrate-reset: ## Reset database by rolling back all migrations
	@echo "$(RED)WARNING: This will rollback ALL migrations!$(NC)"
	@echo "$(YELLOW)Press Ctrl+C to cancel, or any key to continue...$(NC)"
	@read -n 1
	@goose -dir $(GOOSE_MIGRATION_DIR) $(GOOSE_DRIVER) "$(GOOSE_DBSTRING)" reset
	@echo "$(GREEN)Database reset completed!$(NC)"

.PHONY: migrate-status
migrate-status: ## Show migration status
	@echo "$(BLUE)Current migration status:$(NC)"
	@goose -dir $(GOOSE_MIGRATION_DIR) $(GOOSE_DRIVER) "$(GOOSE_DBSTRING)" status

.PHONY: migrate-version
migrate-version: ## Show current migration version
	@goose -dir $(GOOSE_MIGRATION_DIR) $(GOOSE_DRIVER) "$(GOOSE_DBSTRING)" version

.PHONY: migrate-create
migrate-create: ## Create a new migration file (usage: make migrate-create NAME=add_new_table)
	@if [ -z "$(NAME)" ]; then \
		echo "$(RED)Error: NAME is required. Usage: make migrate-create NAME=add_new_table$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Creating new migration: $(NAME)$(NC)"
	@goose -dir $(GOOSE_MIGRATION_DIR) create $(NAME) sql
	@echo "$(GREEN)Migration file created successfully!$(NC)"

# Development Commands
.PHONY: dev-setup
dev-setup: ## Set up development environment
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@go mod download
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@echo "$(GREEN)Development setup completed!$(NC)"

.PHONY: generate
generate: ## Generate sqlc and OpenAPI code
	@echo "$(YELLOW)Generating sqlc code...$(NC)"
	@cd internal && sqlc generate
	@echo "$(YELLOW)Generating OpenAPI server code...$(NC)"
	@oapi-codegen --config=internal/oapi-codegen.yaml internal/openapi.yaml
	@echo "$(GREEN)Code generation completed!$(NC)"

.PHONY: build
build: ## Build the application
	@echo "$(YELLOW)Building application...$(NC)"
	@go build -o bin/freyja ./cmd/api
	@echo "$(GREEN)Build completed! Binary: bin/freyja$(NC)"

.PHONY: run
run: ## Run the application
	@echo "$(YELLOW)Starting application...$(NC)"
	@go run ./cmd/api

.PHONY: test
test: ## Run tests
	@echo "$(YELLOW)Running tests...$(NC)"
	@go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "$(YELLOW)Running tests with coverage...$(NC)"
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

.PHONY: lint
lint: ## Run linter
	@echo "$(YELLOW)Running linter...$(NC)"
	@golangci-lint run

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)Clean completed!$(NC)"

# Docker Commands (if using Docker)
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "$(YELLOW)Building Docker image...$(NC)"
	@docker build -t freyja:latest .
	@echo "$(GREEN)Docker image built successfully!$(NC)"

.PHONY: docker-up
docker-up: ## Start Docker services
	@echo "$(YELLOW)Starting Docker services...$(NC)"
	@docker-compose up -d
	@echo "$(GREEN)Docker services started!$(NC)"

.PHONY: docker-down
docker-down: ## Stop Docker services
	@echo "$(YELLOW)Stopping Docker services...$(NC)"
	@docker-compose down
	@echo "$(GREEN)Docker services stopped!$(NC)"

# Database Commands
.PHONY: db-create
db-create: ## Create database
	@echo "$(YELLOW)Creating database $(DB_NAME)...$(NC)"
	@createdb -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) $(DB_NAME)
	@echo "$(GREEN)Database $(DB_NAME) created successfully!$(NC)"

.PHONY: db-drop
db-drop: ## Drop database
	@echo "$(RED)WARNING: This will permanently delete the database $(DB_NAME)!$(NC)"
	@echo "$(YELLOW)Press Ctrl+C to cancel, or any key to continue...$(NC)"
	@read -n 1
	@dropdb -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) $(DB_NAME)
	@echo "$(GREEN)Database $(DB_NAME) dropped successfully!$(NC)"

# Show current configuration
.PHONY: config
config: ## Show current configuration
	@echo "$(BLUE)Current Configuration:$(NC)"
	@echo "  GOOSE_DRIVER: $(GOOSE_DRIVER)"
	@echo "  GOOSE_MIGRATION_DIR: $(GOOSE_MIGRATION_DIR)"
	@echo "  GOOSE_TABLE: $(GOOSE_TABLE)"
	@echo "  Database: $(DB_NAME)"
	@echo "  Host: $(DB_HOST):$(DB_PORT)"
	@echo "  User: $(DB_USER)"