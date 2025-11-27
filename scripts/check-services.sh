#!/bin/bash
# Check if development services are running

echo "Checking development services..."
echo ""

# Check PostgreSQL
if docker-compose ps postgres | grep -q "Up"; then
    if docker exec freyja-postgres-1 pg_isready -U freyja > /dev/null 2>&1; then
        echo "✓ PostgreSQL is running and accepting connections on localhost:5432"
    else
        echo "✗ PostgreSQL container is up but not ready"
    fi
else
    echo "✗ PostgreSQL is not running"
fi

# Check Mailhog
if docker-compose ps mailhog | grep -q "Up"; then
    echo "✓ Mailhog is running"
    echo "  - SMTP: localhost:1025"
    echo "  - Web UI: http://localhost:8025"
else
    echo "✗ Mailhog is not running"
fi

echo ""
echo "To start services: make docker-up"
echo "To stop services: make docker-down"
