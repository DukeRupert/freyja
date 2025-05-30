# Updated Makefile with optimized test commands

.PHONY: help generate sqlc oapi-gen migrate-up migrate-down migrate-create dev build test test-unit test-integration test-handler clean

# Default target
help:
	@echo "Available commands:"
	@echo "  generate        - Generate all code (sqlc + oapi-codegen)"
	@echo "  sqlc            - Generate database code with sqlc"
	@echo "  oapi-gen        - Generate API code with oapi-codegen"
	@echo "  migrate-up      - Run database migrations up"
	@echo "  migrate-down    - Run database migrations down"
	@echo "  migrate-create NAME=migration_name - Create new migration"
	@echo "  test            - Run all tests"
	@echo "  test-unit       - Run unit tests only"
	@echo "  test-integration- Run integration tests only"
	@echo "  test-handler    - Run handler tests only"
	@echo "  test-short      - Run tests in short mode (skip integration)"
	@echo "  test-verbose    - Run tests with verbose output"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  dev             - Run development server with hot reload"
	@echo "  build           - Build the application"
	@echo "  clean           - Clean generated files"

# Install tools if they don't exist
tools:
	@which sqlc > /dev/null || go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@which oapi-codegen > /dev/null || go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@which goose > /dev/null || go install github.com/pressly/goose/v3/cmd/goose@latest
	@which air > /dev/null || go install github.com/air-verse/air

# Generate all code
generate: tools sqlc oapi-gen

# Generate sqlc code
sqlc:
	@echo "Generating sqlc code..."
	@cd internal && sqlc generate

# Generate oapi-codegen code
oapi-gen:
	@echo "Generating API code..."
	@oapi-codegen -config internal/oapi-codegen.yaml internal/openapi.yaml

# Database migrations
migrate-up:
	@echo "Running migrations up..."
	goose -dir internal/migrations postgres "$(shell go run cmd/config/main.go db-url)" up

migrate-down:
	@echo "Running migrations down..."
	goose -dir internal/migrations postgres "$(shell go run cmd/config/main.go db-url)" down

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	@echo "Creating migration: $(NAME)"
	goose -dir internal/migrations create $(NAME) sql

# Test commands
test:
	@echo "Running all tests..."
	go test ./...

test-unit:
	@echo "Running unit tests..."
	go test -short ./...

test-integration:
	@echo "Running integration tests..."
	go test -run Integration ./...

test-handler:
	@echo "Running handler tests..."
	go test ./internal/handler -v

test-short:
	@echo "Running tests in short mode (skipping integration tests)..."
	go test -short ./...

test-verbose:
	@echo "Running tests with verbose output..."
	go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Test database setup
setup-test-db:
	@echo "Setting up test database..."
	@if [ -z "$(TEST_DB_NAME)" ]; then \
		export TEST_DB_NAME=coffee_subscriptions_test; \
	fi
	@echo "Creating test database: $$TEST_DB_NAME"
	@psql -h localhost -U postgres -c "DROP DATABASE IF EXISTS $$TEST_DB_NAME;" || true
	@psql -h localhost -U postgres -c "CREATE DATABASE $$TEST_DB_NAME;"
	@echo "Running migrations on test database..."
	@APP_ENV=test DB_NAME=$$TEST_DB_NAME goose -dir internal/migrations postgres "postgres://postgres:postgres@localhost:5432/$$TEST_DB_NAME?sslmode=disable" up

clean-test-db:
	@echo "Cleaning up test database..."
	@psql -h localhost -U postgres -c "DROP DATABASE IF EXISTS coffee_subscriptions_test;" || true

# Development
dev: tools
	@echo "Starting development server..."
	air

# Build
build:
	@echo "Building application..."
	go build -o bin/freyja cmd/main.go

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	rm -f internal/api/api.gen.go
	rm -f internal/repo/*.sql.go
	rm -f internal/repo/models.go
	rm -f internal/repo/querier.go
	rm -f internal/repo/db.go
	rm -f coverage.out coverage.html
	rm -rf bin/

# Docker development environment
docker-up:
	@echo "Starting Docker development environment..."
	docker-compose up -d

docker-down:
	@echo "Stopping Docker development environment..."
	docker-compose down

# Complete test workflow
test-all: setup-test-db test clean-test-db
	@echo "Complete test workflow finished"

# Example curl commands for testing
test-api:
	@echo "Testing API endpoints..."
	@echo "Health check:"
	curl -s http://localhost:8080/health | jq .
	@echo "\nList products:"
	curl -s http://localhost:8080/api/v1/products | jq .
	@echo "\nCreate product:"
	curl -s -X POST http://localhost:8080/api/v1/products \
		-H "Content-Type: application/json" \
		-d '{"title":"Test Coffee","handle":"test-coffee"}' | jq .

# Load test data
load-test-data:
	@echo "Loading test data..."
	@curl -s -X POST http://localhost:8080/api/v1/products \
		-H "Content-Type: application/json" \
		-d '{"title":"Ethiopian Yirgacheffe","handle":"ethiopian-yirgacheffe","origin_country":"Ethiopia","region":"Yirgacheffe","roast_level":"light","processing_method":"washed","flavor_notes":["floral","citrus","tea-like"]}' > /dev/null
	@curl -s -X POST http://localhost:8080/api/v1/products \
		-H "Content-Type: application/json" \
		-d '{"title":"Colombian Supremo","handle":"colombian-supremo","origin_country":"Colombia","region":"Huila","roast_level":"medium","processing_method":"washed","flavor_notes":["chocolate","caramel","nutty"]}' > /dev/null
	@curl -s -X POST http://localhost:8080/api/v1/products \
		-H "Content-Type: application/json" \
		-d '{"title":"Brazilian Pulped Natural","handle":"brazilian-pulped-natural","origin_country":"Brazil","region":"Cerrado","roast_level":"medium_dark","processing_method":"natural","flavor_notes":["chocolate","nuts","brown_sugar"]}' > /dev/null
	@echo "Test data loaded successfully!"