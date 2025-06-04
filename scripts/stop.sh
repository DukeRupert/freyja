# =============================================================================
# scripts/stop.sh - Stop all services
# =============================================================================
#!/bin/bash

echo "🛑 Stopping Coffee E-commerce Stack..."

docker-compose down

echo "✅ All services stopped"
