#!/bin/bash
set -e

# Configuration - update these for your setup
VPS_HOST="${VPS_HOST:-your-vps-hostname-or-ip}"
VPS_USER="${VPS_USER:-deploy}"
VPS_PATH="${VPS_PATH:-/opt/freyja}"
IMAGE_NAME="freyja"

# Use VERSION file if exists, otherwise default to latest
if [ -f "VERSION" ]; then
    IMAGE_TAG="${IMAGE_TAG:-$(cat VERSION | tr -d '[:space:]')}"
else
    IMAGE_TAG="${IMAGE_TAG:-latest}"
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check required environment variables
check_config() {
    if [ "$VPS_HOST" = "your-vps-hostname-or-ip" ]; then
        log_error "VPS_HOST not set. Set it via environment variable or edit this script."
        exit 1
    fi
}

# Build the Docker image locally
build() {
    log_info "Building Docker image: ${IMAGE_NAME}:${IMAGE_TAG}"
    docker build -t "${IMAGE_NAME}:${IMAGE_TAG}" .
    log_info "Build complete"
}

# Save image to tarball
save() {
    log_info "Saving image to tarball..."
    docker save "${IMAGE_NAME}:${IMAGE_TAG}" | gzip > "${IMAGE_NAME}.tar.gz"
    log_info "Saved to ${IMAGE_NAME}.tar.gz ($(du -h ${IMAGE_NAME}.tar.gz | cut -f1))"
}

# Upload to VPS via rsync
upload() {
    check_config
    log_info "Uploading to ${VPS_USER}@${VPS_HOST}:${VPS_PATH}..."

    # Ensure remote directory exists
    ssh "${VPS_USER}@${VPS_HOST}" "mkdir -p ${VPS_PATH}"

    # Upload image and compose file
    rsync -avz --progress \
        "${IMAGE_NAME}.tar.gz" \
        docker-compose.production.yml \
        "${VPS_USER}@${VPS_HOST}:${VPS_PATH}/"

    log_info "Upload complete"
}

# Deploy on VPS (load image and restart services)
deploy() {
    check_config
    log_info "Deploying on VPS..."

    ssh "${VPS_USER}@${VPS_HOST}" bash << EOF
        set -e
        cd ${VPS_PATH}

        echo "Loading Docker image..."
        docker load < ${IMAGE_NAME}.tar.gz

        echo "Stopping existing containers..."
        docker compose down || true

        echo "Starting services..."
        docker compose up -d

        echo "Cleaning up..."
        rm -f ${IMAGE_NAME}.tar.gz
        docker image prune -f

        echo "Checking service status..."
        docker compose ps
EOF

    log_info "Deployment complete!"
}

# Full deploy: build -> save -> upload -> deploy
full() {
    build
    save
    upload
    deploy

    # Cleanup local tarball
    rm -f "${IMAGE_NAME}.tar.gz"
    log_info "Full deployment complete!"
}

# Show logs from VPS
logs() {
    check_config
    ssh "${VPS_USER}@${VPS_HOST}" "cd ${VPS_PATH} && docker compose logs -f"
}

# Show status on VPS
status() {
    check_config
    ssh "${VPS_USER}@${VPS_HOST}" "cd ${VPS_PATH} && docker compose ps"
}

# Restart services on VPS
restart() {
    check_config
    ssh "${VPS_USER}@${VPS_HOST}" "cd ${VPS_PATH} && docker compose restart"
}

# Stop services on VPS
stop() {
    check_config
    ssh "${VPS_USER}@${VPS_HOST}" "cd ${VPS_PATH} && docker compose down"
}

# Show current version
version() {
    echo "Image: ${IMAGE_NAME}:${IMAGE_TAG}"
}

# Show help
help() {
    echo "Freyja Deployment Script"
    echo ""
    echo "Usage: ./deploy.sh <command>"
    echo ""
    echo "Commands:"
    echo "  build    Build Docker image locally"
    echo "  save     Save image to tarball"
    echo "  upload   Upload tarball and compose file to VPS"
    echo "  deploy   Load image and restart services on VPS"
    echo "  full     Do all of the above (build -> save -> upload -> deploy)"
    echo "  logs     Show live logs from VPS"
    echo "  status   Show service status on VPS"
    echo "  restart  Restart services on VPS"
    echo "  stop     Stop services on VPS"
    echo "  version  Show current version"
    echo "  help     Show this help"
    echo ""
    echo "Environment variables:"
    echo "  VPS_HOST   VPS hostname or IP (required)"
    echo "  VPS_USER   SSH user (default: deploy)"
    echo "  VPS_PATH   Remote path (default: /opt/freyja)"
    echo "  IMAGE_TAG  Docker image tag (default: latest)"
    echo ""
    echo "Example:"
    echo "  VPS_HOST=192.168.1.100 ./deploy.sh full"
}

# Main
case "${1:-help}" in
    build)   build ;;
    save)    save ;;
    upload)  upload ;;
    deploy)  deploy ;;
    full)    full ;;
    logs)    logs ;;
    status)  status ;;
    restart) restart ;;
    stop)    stop ;;
    version) version ;;
    help)    help ;;
    *)
        log_error "Unknown command: $1"
        help
        exit 1
        ;;
esac
