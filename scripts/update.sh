# =============================================================================
# scripts/update.sh - Update and restart services
# =============================================================================
#!/bin/bash

set -e

echo "🔄 Updating Coffee E-commerce Stack..."

# Pull latest images
echo "📥 Pulling latest Docker images..."
docker-compose pull

# Rebuild application if Dockerfile changed
echo "🔨 Rebuilding application..."
docker-compose build ecommerce-app

# Rolling update - restart services one by one
echo "🔄 Performing rolling update..."

# Update infrastructure services first
for service in prometheus grafana alertmanager postgres-exporter redis-exporter; do
    echo "Updating $service..."
    docker-compose up -d --no-deps "$service"
    sleep 5
done

# Update application
echo "Updating application..."
docker-compose up -d --no-deps ecommerce-app

echo "✅ Update completed!"

# Run health check
./scripts/health-check.sh
