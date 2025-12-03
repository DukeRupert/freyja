#!/bin/bash
set -e

# Version Bump Script for Freyja
# Manages semantic versioning in VERSION file

VERSION_FILE="VERSION"
DEFAULT_VERSION="0.1.0"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Get current version
get_version() {
    if [ -f "$VERSION_FILE" ]; then
        cat "$VERSION_FILE"
    else
        echo "$DEFAULT_VERSION"
    fi
}

# Parse version into components
parse_version() {
    local version="$1"
    # Remove 'v' prefix if present
    version="${version#v}"

    IFS='.' read -r MAJOR MINOR PATCH <<< "$version"

    # Handle pre-release suffix (e.g., 1.0.0-beta.1)
    if [[ "$PATCH" == *-* ]]; then
        PRERELEASE="${PATCH#*-}"
        PATCH="${PATCH%%-*}"
    else
        PRERELEASE=""
    fi
}

# Bump version
bump() {
    local type="$1"
    local current=$(get_version)
    parse_version "$current"

    case "$type" in
        major)
            MAJOR=$((MAJOR + 1))
            MINOR=0
            PATCH=0
            PRERELEASE=""
            ;;
        minor)
            MINOR=$((MINOR + 1))
            PATCH=0
            PRERELEASE=""
            ;;
        patch)
            PATCH=$((PATCH + 1))
            PRERELEASE=""
            ;;
        *)
            log_error "Unknown bump type: $type"
            echo "Use: major, minor, or patch"
            exit 1
            ;;
    esac

    local new_version="${MAJOR}.${MINOR}.${PATCH}"
    echo "$new_version" > "$VERSION_FILE"

    echo -e "${CYAN}$current${NC} → ${GREEN}$new_version${NC}"
    log_info "Version bumped to $new_version"

    echo "$new_version"
}

# Set specific version
set_version() {
    local new_version="$1"

    # Validate format
    if ! [[ "$new_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
        log_error "Invalid version format: $new_version"
        echo "Expected format: X.Y.Z or X.Y.Z-prerelease"
        exit 1
    fi

    local current=$(get_version)
    echo "$new_version" > "$VERSION_FILE"

    echo -e "${CYAN}$current${NC} → ${GREEN}$new_version${NC}"
    log_info "Version set to $new_version"
}

# Create git tag
tag() {
    local version=$(get_version)
    local tag_name="v$version"

    if git rev-parse "$tag_name" >/dev/null 2>&1; then
        log_error "Tag $tag_name already exists"
        exit 1
    fi

    git add "$VERSION_FILE"
    git commit -m "chore: bump version to $version" || true
    git tag -a "$tag_name" -m "Release $version"

    log_info "Created tag: $tag_name"
    echo ""
    echo "Push with: git push origin main --tags"
}

# Show current version
show() {
    local version=$(get_version)
    echo "$version"
}

# Show help
help() {
    echo "Freyja Version Bump Script"
    echo ""
    echo "Usage: ./scripts/bump-version.sh <command> [args]"
    echo ""
    echo "Commands:"
    echo "  show           Show current version"
    echo "  major          Bump major version (X.0.0)"
    echo "  minor          Bump minor version (x.X.0)"
    echo "  patch          Bump patch version (x.x.X)"
    echo "  set <version>  Set specific version"
    echo "  tag            Create git tag for current version"
    echo "  help           Show this help"
    echo ""
    echo "Examples:"
    echo "  ./scripts/bump-version.sh show        # Show: 0.1.0"
    echo "  ./scripts/bump-version.sh patch       # 0.1.0 → 0.1.1"
    echo "  ./scripts/bump-version.sh minor       # 0.1.1 → 0.2.0"
    echo "  ./scripts/bump-version.sh major       # 0.2.0 → 1.0.0"
    echo "  ./scripts/bump-version.sh set 2.0.0   # Set to 2.0.0"
    echo "  ./scripts/bump-version.sh tag         # Create v2.0.0 tag"
}

# Initialize VERSION file if it doesn't exist
init() {
    if [ ! -f "$VERSION_FILE" ]; then
        echo "$DEFAULT_VERSION" > "$VERSION_FILE"
        log_info "Created $VERSION_FILE with version $DEFAULT_VERSION"
    fi
}

# Main
init

case "${1:-help}" in
    show)   show ;;
    major)  bump major ;;
    minor)  bump minor ;;
    patch)  bump patch ;;
    set)    set_version "$2" ;;
    tag)    tag ;;
    help)   help ;;
    *)
        log_error "Unknown command: $1"
        help
        exit 1
        ;;
esac
