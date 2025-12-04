# Freyja Deployment Guide

This guide covers two deployment methods for Freyja:
- **Path A: Local Build** - Build locally and upload tarball (no registry needed)
- **Path B: Registry** - Build via GitHub Actions and pull from ghcr.io

Both paths end with your app running on your VPS behind Caddy.

## Prerequisites

- VPS with Docker and Docker Compose installed
- SSH access to VPS (key-based auth recommended)
- Caddy running on VPS for reverse proxy
- GitHub repository for Path B

## Initial VPS Setup (One Time)

### 1. Initialize the VPS directory

```bash
# Set your VPS details
export VPS_HOST=your-vps-ip-or-hostname
export VPS_USER=dukerupert

# Run init script
./deploy/init-vps.sh
```

This creates the directory structure and uploads config templates.

### 2. Configure environment on VPS

```bash
# SSH to VPS and edit .env
ssh $VPS_USER@$VPS_HOST
cd ~/freyja
nano .env
```

Fill in all values in `.env`:
- `POSTGRES_PASSWORD` - Strong database password
- `TENANT_ID` - Your tenant UUID
- `SESSION_SECRET` - 32+ character random string
- `STRIPE_*` - Your Stripe keys (live for prod)
- `SMTP_*` - Your email provider credentials

### 3. Configure Caddy

Add to your Caddyfile on VPS:

```
freyja.yourdomain.com {
    reverse_proxy localhost:3000
}
```

Reload Caddy:
```bash
sudo systemctl reload caddy
```

---

## Path A: Local Build Deployment

Use this when you want to build locally and push via rsync. Good for quick iterations or when GitHub Actions isn't available.

### Workflow

```
Local machine                          VPS
─────────────                          ───
1. docker build
2. docker save | gzip     ─────────►   3. docker load
                           (rsync)     4. docker compose up
```

### Commands

```bash
# Set VPS details (or add to your shell profile)
export VPS_HOST=your-vps-ip-or-hostname
export VPS_USER=dukerupert

# Full deployment (build + save + upload + deploy)
./deploy/deploy.sh full

# Or step by step:
./deploy/deploy.sh build    # Build image
./deploy/deploy.sh save     # Create tarball
./deploy/deploy.sh upload   # Upload to VPS
./deploy/deploy.sh deploy   # Load and restart on VPS
```

### With Versioning

```bash
# Bump version
./scripts/bump-version.sh patch   # 0.1.0 → 0.1.1

# Deploy with version tag
./deploy/deploy.sh full

# Create git tag (optional)
./scripts/bump-version.sh tag
git push origin main --tags
```

---

## Path B: Registry Deployment (GitHub Actions)

Use this for production releases. GitHub Actions builds and pushes to ghcr.io when you create a version tag.

### Setup (One Time)

#### 1. Enable GitHub Container Registry

The workflow uses `GITHUB_TOKEN` which is automatic. No additional secrets needed for building.

#### 2. Create a Personal Access Token for VPS

Your VPS needs to pull images from ghcr.io. Create a PAT:

1. Go to https://github.com/settings/tokens
2. Generate new token (classic)
3. Select scope: `read:packages`
4. Copy the token

#### 3. Login to Registry on VPS

```bash
export VPS_HOST=your-vps-ip-or-hostname
export GITHUB_TOKEN=your_pat_here
export GITHUB_USER=dukerupert

./deploy/pull-deploy.sh login
```

This stores credentials on VPS. You only need to do this once (or when token expires).

### Release Workflow

```
Local machine                 GitHub Actions              VPS
─────────────                 ──────────────              ───
1. bump version
2. git tag v1.2.3
3. git push --tags   ────►    4. Build image
                              5. Push to ghcr.io  ────►   6. docker pull
                                                          7. docker compose up
```

### Commands

```bash
# 1. Bump version
./scripts/bump-version.sh patch   # or minor/major

# 2. Create and push tag
./scripts/bump-version.sh tag
git push origin main --tags

# 3. Wait for GitHub Actions to complete
#    Check: https://github.com/dukerupert/freyja/actions

# 4. Deploy on VPS
export VPS_HOST=your-vps-ip-or-hostname
./deploy/pull-deploy.sh deploy
```

### Deploy Specific Version

```bash
# Deploy a specific version (not necessarily latest)
IMAGE_TAG=1.2.3 ./deploy/pull-deploy.sh deploy
```

### Rollback

```bash
# Rollback to previous version
./deploy/pull-deploy.sh rollback 1.2.2
```

---

## Operations

These commands work for both deployment paths:

```bash
export VPS_HOST=your-vps-ip-or-hostname

# View logs
./deploy/deploy.sh logs          # or ./deploy/pull-deploy.sh logs

# Check status
./deploy/deploy.sh status        # or ./deploy/pull-deploy.sh status

# Restart services
./deploy/deploy.sh restart       # or ./deploy/pull-deploy.sh restart

# Stop services
./deploy/deploy.sh stop          # or ./deploy/pull-deploy.sh stop
```

## Check Deployed Version (Registry Path)

```bash
./deploy/pull-deploy.sh version
```

---

## Quick Reference

### Path A: Local Build
```bash
export VPS_HOST=myserver.com
./scripts/bump-version.sh patch
./deploy/deploy.sh full
```

### Path B: Registry
```bash
export VPS_HOST=myserver.com
./scripts/bump-version.sh patch
./scripts/bump-version.sh tag
git push origin main --tags
# wait for Actions...
./deploy/pull-deploy.sh deploy
```

---

## Troubleshooting

### Build fails on GitHub Actions
- Check Actions tab for error logs
- Verify Dockerfile builds locally: `docker build -t freyja:test .`

### Can't pull image on VPS
- Re-run login: `GITHUB_TOKEN=xxx ./deploy/pull-deploy.sh login`
- Check token has `read:packages` scope
- Verify image exists: https://github.com/dukerupert/freyja/pkgs/container/freyja

### App won't start
- Check logs: `./deploy/deploy.sh logs`
- Verify .env on VPS has all required values
- Check database is healthy: `docker compose -f docker-compose.production.yml ps`

### Database connection issues
- Ensure postgres service is healthy before app starts
- Check DATABASE_URL in .env matches postgres service name

### Caddy not proxying
- Verify Caddy config points to `localhost:3000`
- Check Caddy logs: `sudo journalctl -u caddy -f`
- Ensure app container is binding to `127.0.0.1:3000`
