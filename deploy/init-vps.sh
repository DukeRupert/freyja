#!/bin/bash
set -e

# Initialize Hiri/Freyja project on VPS
# Creates folders and uploads config files

# Configuration
VPS_HOST="${VPS_HOST:-your-vps-hostname-or-ip}"
VPS_USER="${VPS_USER:-deploy}"
VPS_PATH="${VPS_PATH:-/opt/freyja}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check config
if [ "$VPS_HOST" = "your-vps-hostname-or-ip" ]; then
    log_error "VPS_HOST not set"
    echo "Usage: VPS_HOST=your-server ./deploy/init-vps.sh"
    exit 1
fi

log_info "Initializing Hiri on ${VPS_USER}@${VPS_HOST}:${VPS_PATH}"

# Create remote directories
log_info "Creating remote directories..."
ssh "${VPS_USER}@${VPS_HOST}" "mkdir -p ${VPS_PATH}/deploy/caddy"

# Upload deployment files
log_info "Uploading deployment files..."
rsync -avz --progress \
    deploy/.env.production.example \
    deploy/caddy/Dockerfile \
    deploy/caddy/Caddyfile \
    "${VPS_USER}@${VPS_HOST}:${VPS_PATH}/deploy/caddy/" 2>/dev/null || true

# Upload and rename docker-compose file
rsync -avz --progress \
    docker-compose.production.yml \
    "${VPS_USER}@${VPS_HOST}:${VPS_PATH}/docker-compose.yml"

# Move .env.production.example to correct location
ssh "${VPS_USER}@${VPS_HOST}" "mv ${VPS_PATH}/deploy/caddy/.env.production.example ${VPS_PATH}/ 2>/dev/null || true"

# Create .env from example if it doesn't exist
log_info "Setting up .env file..."
ssh "${VPS_USER}@${VPS_HOST}" bash << EOF
    cd ${VPS_PATH}
    if [ ! -f .env ]; then
        cp .env.production.example .env
        chmod 600 .env
        echo "Created .env from template - EDIT THIS FILE"
    else
        echo ".env already exists - skipping"
    fi
EOF

log_info "Done!"
echo ""
echo "Next steps:"
echo "  1. Edit the .env file on VPS:"
echo "     ssh ${VPS_USER}@${VPS_HOST} 'nano ${VPS_PATH}/.env'"
echo ""
echo "  2. Start the services:"
echo "     ssh ${VPS_USER}@${VPS_HOST} 'cd ${VPS_PATH} && docker compose up -d'"
echo ""
echo "  3. Or trigger a deploy via git tag:"
echo "     git tag v1.0.0 && git push origin v1.0.0"
