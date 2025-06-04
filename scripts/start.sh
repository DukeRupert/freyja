# =============================================================================
# scripts/start.sh - Start all services
# =============================================================================
#!/bin/bash

set -e

echo "🚀 Starting Coffee E-commerce Stack..."

# Check if .env exists
if [ ! -f ../.env ]; then
    echo "❌ .env file not found. Please run ./scripts/setup.sh first"
    exit 1
fi

# Load environment variables
source .env

# Start infrastructure services first
echo "🔧 Starting infrastructure services..."
docker-compose up -d postgres valkey nats prometheus grafana alertmanager

# Wait for databases to be ready
echo "⏳ Waiting for databases to be ready..."
sleep 30

# Start exporters
echo "📊 Starting monitoring exporters..."
docker-compose up -d postgres-exporter valkey-exporter nats-exporter

# Start the application
echo "☕ Starting coffee e-commerce application..."
docker-compose up -d ecommerce-app

# Start caddy (if configured)
if [ -f config/caddy/Caddyfile ]; then
    echo "🌐 Starting Caddy reverse proxy..."
    docker-compose up -d caddy
fi

# Show status
echo ""
echo "📋 Service Status:"
docker-compose ps

echo ""
echo "🎉 Stack started successfully!"
echo ""
echo "📱 Access URLs:"
echo "  Application:  http://localhost:8080"
echo "  Grafana:      http://localhost:3000 (admin/grafana_admin_123)"
echo "  Prometheus:   http://localhost:9090"
echo "  AlertManager: http://localhost:9093"
echo ""
echo "📊 Monitor logs with: docker-compose logs -f ecommerce-app"
