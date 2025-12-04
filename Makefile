.PHONY: help dev build build\:server build\:saas build\:all run run\:saas test test\:coverage clean migrate migrate\:down migrate\:status migrate\:create sqlc\:gen docker\:up docker\:down docker\:logs css\:build css\:watch css\:clean install\:tools deps fmt lint vet check

# Load environment variables from .env file
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@sed -n 's/^\([a-zA-Z_:\\-]*\):.*## \(.*\)/  \1\t\2/p' $(MAKEFILE_LIST) | sed 's/\\:/:/g' | awk -F'\t' '{printf "  %-18s %s\n", $$1, $$2}'

dev: ## Start development server with Air live reload
	@air

css\:build: ## Build CSS with Tailwind
	@echo "Building CSS..."
	@./tailwind -i ./web/static/css/input.css -o ./web/static/css/output.css --minify
	@echo "CSS build complete"

css\:watch: ## Watch and rebuild CSS on changes
	@echo "Watching CSS files..."
	@./tailwind -i ./web/static/css/input.css -o ./web/static/css/output.css --watch

css\:clean: ## Remove generated CSS file
	@echo "Cleaning CSS..."
	@rm -f web/static/css/output.css
	@echo "CSS cleaned"

build: build\:server ## Alias for build:server (default)

build\:server: css\:build ## Build the tenant server binary
	@echo "Building tenant server..."
	@go build -o bin/server cmd/server/main.go
	@echo "Build complete: bin/server"

build\:saas: css\:build ## Build the SaaS marketing server binary
	@echo "Building SaaS server..."
	@go build -o bin/saas cmd/saas/main.go
	@echo "Build complete: bin/saas"

build\:all: css\:build ## Build all server binaries
	@echo "Building all servers..."
	@go build -o bin/server cmd/server/main.go
	@go build -o bin/saas cmd/saas/main.go
	@echo "Build complete: bin/server, bin/saas"

run: build\:server ## Build and run the tenant server
	@./bin/server

run\:saas: build\:saas ## Build and run the SaaS marketing server
	@./bin/saas

test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test\:coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts and temporary files
	@echo "Cleaning..."
	@rm -rf bin/ tmp/ coverage.out coverage.html web/static/css/output.css
	@echo "Clean complete"

migrate: ## Run database migrations up
	@echo "Running migrations..."
	@goose -dir migrations postgres "$(DATABASE_URL)" up

migrate\:down: ## Rollback last database migration
	@echo "Rolling back last migration..."
	@goose -dir migrations postgres "$(DATABASE_URL)" down

migrate\:status: ## Show migration status
	@goose -dir migrations postgres "$(DATABASE_URL)" status

migrate\:create: ## Create a new migration file (usage: make migrate:create NAME=create_users)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate:create NAME=create_users"; \
		exit 1; \
	fi
	@goose -dir migrations create $(NAME) sql

sqlc\:gen: ## Generate sqlc code from queries
	@echo "Generating sqlc code..."
	@sqlc generate
	@echo "sqlc generation complete"

docker\:up: ## Start Docker Compose services
	@echo "Starting Docker services..."
	@docker-compose up -d
	@echo "Docker services started"
	@echo "PostgreSQL: localhost:5432"
	@echo "Mailhog Web UI: http://localhost:8025"

docker\:down: ## Stop Docker Compose services
	@echo "Stopping Docker services..."
	@docker-compose down
	@echo "Docker services stopped"

docker\:logs: ## Show Docker Compose logs
	@docker-compose logs -f

install\:tools: ## Install development tools (goose, sqlc, air)
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
