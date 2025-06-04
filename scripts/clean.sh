# =============================================================================
# scripts/clean.sh - Clean up old data and images
# =============================================================================
#!/bin/bash

echo "🧹 Cleaning up Coffee E-commerce Stack..."

# Stop services
./scripts/stop.sh

# Remove containers and volumes
echo "🗑️ Removing containers and volumes..."
docker-compose down -v --remove-orphans

# Clean up Docker images
echo "🐳 Cleaning up Docker images..."
docker image prune -f

# Clean up old backups (keep last 7 days)
if [ -d "backups" ]; then
    echo "📦 Cleaning old backups..."
    find backups -name "*.tar.gz" -mtime +7 -delete
fi

# Clean up logs
if [ -d "logs" ]; then
    echo "📝 Cleaning old logs..."
    find logs -name "*.log" -mtime +30 -delete
fi

echo "✅ Cleanup completed!"
