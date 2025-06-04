 =============================================================================
# scripts/backup.sh - Backup data
# =============================================================================
#!/bin/bash

set -e

BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "💾 Creating backup in $BACKUP_DIR..."

# Backup PostgreSQL
echo "📊 Backing up PostgreSQL..."
docker-compose exec -T postgres pg_dump -U ecommerce_user ecommerce > "$BACKUP_DIR/postgres_backup.sql"

# Backup Valkey (if needed)
echo "🔴 Backing up Valkey..."
docker-compose exec -T valkey valkey-cli --rdb - > "$BACKUP_DIR/valkey_backup.rdb"

# Backup Grafana dashboards
echo "📈 Backing up Grafana..."
if [ -d "config/grafana/dashboard-configs" ]; then
    cp -r config/grafana/dashboard-configs "$BACKUP_DIR/grafana_dashboards"
fi

# Backup configuration
echo "⚙️ Backing up configuration..."
cp -r config "$BACKUP_DIR/"
cp .env "$BACKUP_DIR/env_backup"

# Create archive
echo "📦 Creating archive..."
tar -czf "${BACKUP_DIR}.tar.gz" -C backups "$(basename $BACKUP_DIR)"
rm -rf "$BACKUP_DIR"

echo "✅ Backup created: ${BACKUP_DIR}.tar.gz"
