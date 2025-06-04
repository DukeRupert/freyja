# =============================================================================
# scripts/restore.sh - Restore from backup
# =============================================================================
#!/bin/bash

set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <backup_file.tar.gz>"
    exit 1
fi

BACKUP_FILE="$1"
RESTORE_DIR="restore_$(date +%Y%m%d_%H%M%S)"

if [ ! -f "$BACKUP_FILE" ]; then
    echo "❌ Backup file not found: $BACKUP_FILE"
    exit 1
fi

echo "🔄 Restoring from $BACKUP_FILE..."

# Extract backup
mkdir -p "$RESTORE_DIR"
tar -xzf "$BACKUP_FILE" -C "$RESTORE_DIR"

BACKUP_CONTENT=$(find "$RESTORE_DIR" -mindepth 1 -maxdepth 1 -type d | head -1)

# Stop services
./scripts/stop.sh

# Restore PostgreSQL
if [ -f "$BACKUP_CONTENT/postgres_backup.sql" ]; then
    echo "📊 Restoring PostgreSQL..."
    docker-compose up -d postgres
    sleep 10
    docker-compose exec -T postgres psql -U ecommerce_user -d ecommerce < "$BACKUP_CONTENT/postgres_backup.sql"
fi

# Restore configuration
if [ -d "$BACKUP_CONTENT/config" ]; then
    echo "⚙️ Restoring configuration..."
    cp -r "$BACKUP_CONTENT/config" .
fi

if [ -f "$BACKUP_CONTENT/env_backup" ]; then
    echo "🔧 Restoring environment..."
    cp "$BACKUP_CONTENT/env_backup" .env
fi

# Cleanup
rm -rf "$RESTORE_DIR"

echo "✅ Restore completed. Run ./scripts/start.sh to start services."
