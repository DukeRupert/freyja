#!/bin/bash
set -e

# Initialize Freyja project on VPS
# Creates folders and uploads config files

# Configuration
VPS_HOST="${VPS_HOST:-your-vps-hostname-or-ip}"
VPS_USER="${VPS_USER:-dukerupert}"
VPS_PATH="${VPS_PATH:-/home/dukerupert/freyja}"

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
    echo "Usage: VPS_HOST=your-server VPS_USER=deploy ./deploy/init-vps.sh"
    exit 1
fi

log_info "Initializing Freyja on ${VPS_USER}@${VPS_HOST}:${VPS_PATH}"

# Create remote directory
log_info "Creating remote directory..."
ssh "${VPS_USER}@${VPS_HOST}" "mkdir -p ${VPS_PATH}"

# Upload deployment files
log_info "Uploading deployment files..."
rsync -avz --progress \
    docker-compose.production.yml \
    deploy/.env.production.example \
    "${VPS_USER}@${VPS_HOST}:${VPS_PATH}/"

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
echo "  2. Add Caddy config for freyja.angmar.dev"
echo ""
echo "  3. Deploy the app:"
echo "     VPS_HOST=${VPS_HOST} VPS_USER=${VPS_USER} ./deploy.sh full"
