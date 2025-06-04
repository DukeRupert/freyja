.PHONY: help build up down logs restart clean db-migrate dev-setup

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

db-migrate: ## Run database migrations
	docker-compose exec app sh -c "make db-migrate" || echo "Run migrations manually with: goose -dir migrations postgres \$$DATABASE_URL up"

dev-setup: build up ## Full development setup
	@echo "✅ Development environment ready!"
	@echo "🔗 Services:"
	@echo "  • API: http://localhost:8080"
	@echo "  • Health: http://localhost:8080/health"
	@echo "  • Grafana: http://localhost:3000 (admin/admin123)"
	@echo "  • MinIO Console: http://localhost:9001 (minioadmin/minioadmin123)"

test-api: ## Test the API endpoints
	@echo "Testing API endpoints..."
	curl -s http://localhost:8080/health | jq . || echo "Health check failed"
	curl -s http://localhost:8080/api/v1/products | jq . || echo "Products endpoint failed"

# Development helpers
dev-logs: ## Follow development logs
	docker-compose logs -f

dev-shell: ## Open shell in app container
	docker-compose exec app sh

dev-db-shell: ## Open PostgreSQL shell
	docker-compose exec postgres psql -U postgres -d coffee_ecommerce
