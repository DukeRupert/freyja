#!/bin/bash
set -e

# Registry-based deployment script for Freyja
# Pulls image from GitHub Container Registry and deploys

# Configuration - update these for your setup
VPS_HOST="${VPS_HOST:-your-vps-hostname-or-ip}"
VPS_USER="${VPS_USER:-deploy}"
VPS_PATH="${VPS_PATH:-/opt/freyja}"

# Registry configuration
REGISTRY="${REGISTRY:-ghcr.io}"
IMAGE_NAME="${IMAGE_NAME:-dukerupert/freyja}"

# Use VERSION file if exists, otherwise default to latest
if [ -f "VERSION" ]; then
    IMAGE_TAG="${IMAGE_TAG:-$(cat VERSION | tr -d '[:space:]')}"
else
    IMAGE_TAG="${IMAGE_TAG:-latest}"
fi

FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

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

# Login to GitHub Container Registry on VPS
login() {
    check_config
    log_info "Logging into ${REGISTRY} on VPS..."

    if [ -z "$GITHUB_TOKEN" ]; then
        log_error "GITHUB_TOKEN not set. Generate a PAT with 'read:packages' scope."
        echo ""
        echo "Create token at: https://github.com/settings/tokens"
        echo "Then run: GITHUB_TOKEN=your_token ./deploy/pull-deploy.sh login"
        exit 1
    fi

    ssh "${VPS_USER}@${VPS_HOST}" "echo '${GITHUB_TOKEN}' | docker login ${REGISTRY} -u ${GITHUB_USER:-dukerupert} --password-stdin"
    log_info "Login successful"
}

# Pull latest image on VPS
pull() {
    check_config
    log_info "Pulling ${FULL_IMAGE} on VPS..."

    ssh "${VPS_USER}@${VPS_HOST}" "docker pull ${FULL_IMAGE}"
    log_info "Pull complete"
}

# Deploy: pull and restart services
deploy() {
    check_config
    log_info "Deploying ${FULL_IMAGE}..."

    ssh "${VPS_USER}@${VPS_HOST}" bash << EOF
        set -e
        cd ${VPS_PATH}

        echo "Pulling image..."
        docker pull ${FULL_IMAGE}

        echo "Updating docker-compose with new image..."
        # Export for docker compose to use
        export HIRI_IMAGE=${FULL_IMAGE}

        echo "Stopping existing containers..."
        docker compose down || true

        echo "Starting services with new image..."
        docker compose up -d

        echo "Cleaning up old images..."
        docker image prune -f

        echo "Checking service status..."
        docker compose ps
EOF

    log_info "Deployment complete!"
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

# Rollback to previous image
rollback() {
    check_config
    local previous_tag="${1:-}"

    if [ -z "$previous_tag" ]; then
        log_error "Please specify the version to rollback to"
        echo "Usage: ./deploy/pull-deploy.sh rollback 1.2.3"
        exit 1
    fi

    log_warn "Rolling back to version ${previous_tag}..."
    IMAGE_TAG="${previous_tag}" deploy
}

# Show current deployed version
version() {
    check_config
    log_info "Checking deployed version..."
    ssh "${VPS_USER}@${VPS_HOST}" bash << 'EOF'
        docker inspect hiri-app --format '{{.Config.Image}}' 2>/dev/null || echo "Container not running"
EOF
}

# List available tags in registry
list-tags() {
    log_info "Available tags for ${IMAGE_NAME}:"
    echo ""
    echo "View at: https://github.com/${IMAGE_NAME}/pkgs/container/freyja"
}

# Show help
help() {
    echo "Freyja Registry Deployment Script"
    echo ""
    echo "Usage: ./deploy/pull-deploy.sh <command>"
    echo ""
    echo "Commands:"
    echo "  login         Login to GitHub Container Registry on VPS"
    echo "  pull          Pull latest image on VPS"
    echo "  deploy        Pull image and restart services"
    echo "  logs          Show live logs from VPS"
    echo "  status        Show service status on VPS"
    echo "  restart       Restart services on VPS"
    echo "  stop          Stop services on VPS"
    echo "  rollback <v>  Rollback to specific version"
    echo "  version       Show currently deployed version"
    echo "  list-tags     Show link to available tags"
    echo "  help          Show this help"
    echo ""
    echo "Environment variables:"
    echo "  VPS_HOST      VPS hostname or IP (required)"
    echo "  VPS_USER      SSH user (default: deploy)"
    echo "  VPS_PATH      Remote path (default: /opt/freyja)"
    echo "  IMAGE_TAG     Image tag to deploy (default: from VERSION file or 'latest')"
    echo "  GITHUB_TOKEN  GitHub PAT for registry login (required for login command)"
    echo "  GITHUB_USER   GitHub username (default: dukerupert)"
    echo ""
    echo "Examples:"
    echo "  VPS_HOST=myserver.com ./deploy/pull-deploy.sh deploy"
    echo "  VPS_HOST=myserver.com IMAGE_TAG=1.2.3 ./deploy/pull-deploy.sh deploy"
    echo "  VPS_HOST=myserver.com ./deploy/pull-deploy.sh rollback 1.2.2"
}

# Main
case "${1:-help}" in
    login)      login ;;
    pull)       pull ;;
    deploy)     deploy ;;
    logs)       logs ;;
    status)     status ;;
    restart)    restart ;;
    stop)       stop ;;
    rollback)   rollback "$2" ;;
    version)    version ;;
    list-tags)  list-tags ;;
    help)       help ;;
    *)
        log_error "Unknown command: $1"
        help
        exit 1
        ;;
esac
