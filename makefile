# Coffee E-commerce Makefile
# NOTE: All command lines under targets MUST be indented with TABS, not spaces

.PHONY: help build up down logs restart clean dev test-api dev-logs dev-shell dev-db-shell dev-local dev-services dev-stop

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@egrep '^(.+)\s*:.*?##\s*(.+)' $(MAKEFILE_LIST) | column -t -c 2 -s ':#'

build: ## Build the Docker images
	docker-compose build

up: ## Start all services
	docker-compose up -d

down: ## Stop all services
	docker-compose down

logs: ## View application logs
	docker-compose logs -f app

restart: ## Restart the application
	docker-compose restart app

clean: ## Clean up Docker resources
	docker-compose down -v
	docker system prune -f

dev: ## Start development environment
	docker-compose -f docker-compose.working.yml up -d
	@echo "✅ Development environment ready!"
	@echo "🔗 Services:"
	@echo "  • API: http://localhost:8080"
	@echo "  • Health: http://localhost:8080/health"
	@echo "  • Grafana: http://localhost:3000 (admin/admin123)"
	@echo "  • MinIO Console: http://localhost:9001 (minioadmin/minioadmin123)"

dev-local: ## Run app locally with Docker services
	@echo "🏠 Starting local development..."
	docker-compose -f docker-compose.working.yml up -d postgres valkey
	@sleep 5
	@echo "📡 Services started. Run with local environment:"
	@echo "export DATABASE_URL='postgres://postgres:password@localhost:5432/coffee_ecommerce?sslmode=disable'"
	@echo "export VALKEY_ADDR='localhost:6379'"
	@echo "go run cmd/server/main.go"

dev-services: ## Start just the Docker services (DB, cache, etc.)
	docker-compose -f docker-compose.working.yml up -d postgres valkey
	@echo "✅ Services started: PostgreSQL and Valkey"

dev-stop: ## Stop development services
	docker-compose -f docker-compose.working.yml down

test-api: ## Test the API endpoints
	@echo "Testing API endpoints..."
	curl -s http://localhost:8080/health | jq . || echo "Health check failed"
	curl -s http://localhost:8080/api/v1/products | jq . || echo "Products endpoint failed"

dev-logs: ## Follow development logs
	docker-compose -f docker-compose.working.yml logs -f

dev-shell: ## Open shell in app container
	docker-compose -f docker-compose.working.yml exec app sh

dev-db-shell: ## Open PostgreSQL shell
	docker exec -it freyja-postgres-1 psql -U postgres -d coffee_ecommerce

# Database operations
db-migrate: ## Run database migrations
	@echo "🗄️  Running database migrations..."
	goose -dir migrations postgres "postgres://postgres:password@localhost:5432/coffee_ecommerce?sslmode=disable" up

db-status: ## Check migration status
	goose -dir migrations postgres "postgres://postgres:password@localhost:5432/coffee_ecommerce?sslmode=disable" status

db-create-migration: ## Create new migration (usage: make db-create-migration NAME=add_something)
	@if [ -z "$(NAME)" ]; then \
		echo "❌ NAME is required. Usage: make db-create-migration NAME=add_something"; \
		exit 1; \
	fi
	goose -dir migrations create $(NAME) sql

# Quick development workflow
quick-start: ## Quick start for development (services + local app setup)
	@echo "🚀 Quick development setup..."
	make dev-services
	@echo ""
	@echo "✅ Services started! Now run your app locally:"
	@echo "   export DATABASE_URL='postgres://postgres:password@localhost:5432/coffee_ecommerce?sslmode=disable'"
	@echo "   export VALKEY_ADDR='localhost:6379'"
	@echo "   go run cmd/server/main.go"
	@echo ""
	@echo "Or run full Docker setup:"
	@echo "   make dev"

# Helper to check if services are running
status: ## Check status of development services
	@echo "📊 Development environment status:"
	@echo ""
	@echo "Docker containers:"
	@docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "(freyja|coffee)" || echo "No development containers running"
	@echo ""
	@echo "Quick health checks:"
	@curl -s http://localhost:8080/health >/dev/null 2>&1 && echo "✅ API responding" || echo "❌ API not responding"
	@curl -s http://localhost:5432 >/dev/null 2>&1 && echo "✅ PostgreSQL port open" || echo "❌ PostgreSQL not accessible"
