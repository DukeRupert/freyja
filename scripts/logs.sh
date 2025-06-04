# =============================================================================
# scripts/logs.sh - View logs for specific service
# =============================================================================
#!/bin/bash

if [ $# -eq 0 ]; then
    echo "📋 Available services:"
    docker-compose ps --services
    echo ""
    echo "Usage: $0 <service_name> [follow]"
    echo "Example: $0 ecommerce-app follow"
    exit 1
fi

SERVICE="$1"
FOLLOW="$2"

if [ "$FOLLOW" = "follow" ] || [ "$FOLLOW" = "-f" ]; then
    docker-compose logs -f "$SERVICE"
else
    docker-compose logs "$SERVICE"
fi
