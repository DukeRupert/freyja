# =============================================================================
# scripts/health-check.sh - Check health of all services
# =============================================================================
#!/bin/bash

echo "🏥 Coffee E-commerce Health Check"
echo "=================================="

# Function to check HTTP endpoint
check_http() {
    local name="$1"
    local url="$2"
    local expected_code="${3:-200}"

    if curl -s -o /dev/null -w "%{http_code}" "$url" | grep -q "$expected_code"; then
        echo "✅ $name: Healthy"
        return 0
    else
        echo "❌ $name: Unhealthy"
        return 1
    fi
}

# Function to check container health
check_container() {
    local name="$1"
    local container="$2"

    if docker-compose ps "$container" | grep -q "Up"; then
        echo "✅ $name: Running"
        return 0
    else
        echo "❌ $name: Not running"
        return 1
    fi
}

FAILED=0

# Check containers
echo ""
echo "📦 Container Status:"
check_container "PostgreSQL" "postgres" || ((FAILED++))
check_container "Valkey" "valkey" || ((FAILED++))
check_container "NATS" "nats" || ((FAILED++))
check_container "Prometheus" "prometheus" || ((FAILED++))
check_container "Grafana" "grafana" || ((FAILED++))
check_container "Application" "ecommerce-app" || ((FAILED++))

# Check HTTP endpoints
echo ""
echo "🌐 HTTP Endpoints:"
check_http "Application Health" "http://localhost:8080/health" || ((FAILED++))
check_http "Application Metrics" "http://localhost:8080/metrics" || ((FAILED++))
check_http "Prometheus" "http://localhost:9090/-/healthy" || ((FAILED++))
check_http "Grafana" "http://localhost:3000/api/health" || ((FAILED++))
check_http "AlertManager" "http://localhost:9093/-/healthy" || ((FAILED++))

# Check database connectivity
echo ""
echo "🗄️ Database Connectivity:"
if docker-compose exec -T postgres pg_isready -U ecommerce_user -d ecommerce > /dev/null 2>&1; then
    echo "✅ PostgreSQL: Connected"
else
    echo "❌ PostgreSQL: Connection failed"
    ((FAILED++))
fi

if docker-compose exec -T valkey valkey-cli ping | grep -q "PONG"; then
    echo "✅ Valkey: Connected"
else
    echo "❌ Valkey: Connection failed"
    ((FAILED++))
fi

echo ""
echo "=================================="
if [ $FAILED -eq 0 ]; then
    echo "🎉 All services are healthy!"
    exit 0
else
    echo "⚠️  $FAILED service(s) are unhealthy"
    exit 1
fi
