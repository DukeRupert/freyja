.PHONY: help dev build run test clean migrate migrate-down migrate-status sqlc-gen docker-up docker-down

# Load environment variables from .env file
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev: ## Start development server with Air live reload
	@air

build: ## Build the application binary
	@echo "Building application..."
	@go build -o bin/server cmd/server/main.go
	@echo "Build complete: bin/server"

run: build ## Build and run the application
	@./bin/server

test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts and temporary files
	@echo "Cleaning..."
	@rm -rf bin/ tmp/ coverage.out coverage.html
	@echo "Clean complete"

migrate: ## Run database migrations up
	@echo "Running migrations..."
	@goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down: ## Rollback last database migration
	@echo "Rolling back last migration..."
	@goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-status: ## Show migration status
	@goose -dir migrations postgres "$(DATABASE_URL)" status

migrate-create: ## Create a new migration file (usage: make migrate-create NAME=create_users)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=create_users"; \
		exit 1; \
	fi
	@goose -dir migrations create $(NAME) sql

sqlc-gen: ## Generate sqlc code from queries
	@echo "Generating sqlc code..."
	@sqlc generate
	@echo "sqlc generation complete"

docker-up: ## Start Docker Compose services
	@echo "Starting Docker services..."
	@docker-compose up -d
	@echo "Docker services started"
	@echo "PostgreSQL: localhost:5432"
	@echo "Mailhog Web UI: http://localhost:8025"

docker-down: ## Stop Docker Compose services
	@echo "Stopping Docker services..."
	@docker-compose down
	@echo "Docker services stopped"

docker-logs: ## Show Docker Compose logs
	@docker-compose logs -f

install-tools: ## Install development tools (goose, sqlc, air)
	@echo "Installing development tools..."
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/cosmtrek/air@latest
	@echo "Tools installed successfully"
	@echo "Make sure $(shell go env GOPATH)/bin is in your PATH"

deps: ## Download Go dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies downloaded"

fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Code formatted"

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@golangci-lint run
	@echo "Linting complete"

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...
	@echo "Vet complete"

check: fmt vet test ## Run formatting, vetting, and tests
	@echo "All checks passed"
