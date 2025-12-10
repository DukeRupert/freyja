# Hiri Deployment Guide

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│  GitHub                                                     │
│  ┌───────────┐    ┌───────────────┐    ┌─────────────────┐ │
│  │ Push to   │───▶│ GitHub Actions│───▶│ GitHub Container│ │
│  │ main/tag  │    │ (build/test)  │    │ Registry (ghcr) │ │
│  └───────────┘    └───────────────┘    └────────┬────────┘ │
└─────────────────────────────────────────────────┼──────────┘
                                                  │
                                                  ▼ SSH deploy
┌─────────────────────────────────────────────────────────────┐
│  Hetzner VPS (Ubuntu 24.04)                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Docker Compose                                       │   │
│  │  ┌─────────┐  ┌─────────────┐  ┌──────────────────┐ │   │
│  │  │ Caddy   │  │ Go App      │  │ PostgreSQL 16    │ │   │
│  │  │ (TLS)   │  │ (hiri-app)  │  │ (hiri-postgres)  │ │   │
│  │  └─────────┘  └─────────────┘  └──────────────────┘ │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Domain & DNS

- **Domain:** hiri.coffee
- **DNS Provider:** Hetzner DNS
- **Wildcard Support:** `*.hiri.coffee` for multi-tenant subdomains

### Domain Structure

| Domain | Purpose |
|--------|---------|
| `hiri.coffee` | Marketing site (landing, pricing, about) |
| `app.hiri.coffee` | Application (admin, storefront) |
| `*.hiri.coffee` | Future: per-tenant custom subdomains |

### Required DNS Records

| Type | Name | Value |
|------|------|-------|
| A | `@` | VPS IPv4 |
| A | `*` | VPS IPv4 |

## CI/CD Pipeline

### Workflows

Two GitHub Actions workflows in `.github/workflows/`:

#### 1. CI (`ci.yml`) - Runs on push/PR to main

- Runs `go vet`
- Runs `go test -race -short`
- Verifies build compiles
- Uses PostgreSQL service container for tests

#### 2. Build & Deploy (`build-push.yml`) - Runs on version tags

- Builds Docker image
- Pushes to GitHub Container Registry (ghcr.io)
- SSHs to VPS and deploys

### Triggering a Deploy

```bash
git tag v1.0.0
git push origin v1.0.0
```

## VPS Configuration

### Server Details

- **Provider:** Hetzner
- **OS:** Ubuntu 24.04 LTS
- **Deploy User:** `deploy`
- **App Directory:** `/opt/freyja`

### File Structure on VPS

```
/opt/freyja/
├── docker-compose.yml
├── .env
└── deploy/
    └── caddy/
        ├── Dockerfile
        └── Caddyfile
```

### SSH Access

- **Deploy user:** `deploy`
- **Auth:** SSH key (ed25519)
- **Key location:** `~/.ssh/freyja_deploy` (local machine)

GitHub Actions uses these secrets:
- `VPS_HOST` - VPS IP address
- `VPS_USER` - `deploy`
- `VPS_SSH_KEY` - Private key contents

## Docker Compose Services

### docker-compose.yml

```yaml
services:
  caddy:
    build: ./deploy/caddy
    container_name: hiri-caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
    environment:
      - HETZNER_DNS_API_TOKEN=${HETZNER_DNS_API_TOKEN}
    volumes:
      - ./deploy/caddy/Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    networks:
      - hiri-network

  app:
    image: ${HIRI_IMAGE:-ghcr.io/dukerupert/hiri:latest}
    container_name: hiri-app
    restart: unless-stopped
    expose:
      - "3000"
    environment:
      - ENV=prod
      - PORT=3000
      - BASE_URL=https://app.hiri.coffee
      - DATABASE_URL=postgres://hiri:${POSTGRES_PASSWORD}@postgres:5432/hiri?sslmode=disable
      - SESSION_SECRET=${SESSION_SECRET}
      # Host-based routing (marketing at root, app at subdomain)
      - HOST_ROUTING_ENABLED=${HOST_ROUTING_ENABLED:-true}
      - MARKETING_DOMAIN=${MARKETING_DOMAIN:-hiri.coffee}
      - APP_DOMAIN=${APP_DOMAIN:-app.hiri.coffee}
      # Stripe
      - STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
      - STRIPE_PUBLISHABLE_KEY=${STRIPE_PUBLISHABLE_KEY}
      - STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET}
      # Email
      - SMTP_HOST=${SMTP_HOST}
      - SMTP_PORT=${SMTP_PORT}
      - SMTP_USERNAME=${SMTP_USERNAME}
      - SMTP_PASSWORD=${SMTP_PASSWORD}
      - SMTP_FROM=${SMTP_FROM}
    depends_on:
      postgres:
        condition: service_healthy
    volumes:
      - uploads:/app/uploads
    networks:
      - hiri-network

  postgres:
    image: postgres:16-alpine
    container_name: hiri-postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: hiri
      POSTGRES_USER: hiri
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U hiri"]
      interval: 5s
      timeout: 5s
      retries: 5
    networks:
      - hiri-network

volumes:
  postgres_data:
  uploads:
  caddy_data:
  caddy_config:

networks:
  hiri-network:
    driver: bridge
```

## Caddy Configuration

### deploy/caddy/Dockerfile

```dockerfile
FROM caddy:2-builder AS builder

RUN xcaddy build \
    --with github.com/caddy-dns/hetzner

FROM caddy:2-alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
```

### deploy/caddy/Caddyfile

```caddyfile
{
    email your@email.com
}

hiri.coffee, *.hiri.coffee {
    tls {
        dns hetzner {env.HETZNER_DNS_API_TOKEN}
    }

    reverse_proxy app:3000
}
```

## Environment Variables

### .env file on VPS (`/opt/freyja/.env`)

```
POSTGRES_PASSWORD=<strong-random-password>
SESSION_SECRET=<strong-random-string>
HETZNER_DNS_API_TOKEN=<read-write-token>

# Host-based routing
HOST_ROUTING_ENABLED=true
MARKETING_DOMAIN=hiri.coffee
APP_DOMAIN=app.hiri.coffee

# Stripe
STRIPE_SECRET_KEY=sk_live_xxx
STRIPE_PUBLISHABLE_KEY=pk_live_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx

# Email
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=<user>
SMTP_PASSWORD=<password>
SMTP_FROM=noreply@hiri.coffee
```

Generate strong passwords with:
```bash
openssl rand -base64 32
```

## Deployment Flow

```
1. Push tag: git tag v1.0.0 && git push origin v1.0.0
2. GitHub Actions builds image
3. Image pushed to ghcr.io/dukerupert/hiri:v1.0.0
4. Actions SSHs to VPS as deploy user
5. Runs: docker compose pull app
6. Runs: docker compose up -d --no-deps app
7. Old containers pruned
```

## Database Migrations

Migrations are embedded in the Go binary and run automatically on application startup using Goose.

## Rollback Procedure

### Quick Rollback (to previous image)

```bash
ssh deploy@hiri.coffee
cd /opt/freyja
HIRI_IMAGE=ghcr.io/dukerupert/hiri:v0.9.0 docker compose up -d --no-deps app
```

### View Available Tags

```bash
# List recent images
docker images ghcr.io/dukerupert/hiri
```

### Database Rollback (if needed)

The app embeds Goose migrations. To rollback a migration manually:

```bash
# SSH to VPS, exec into app container or use a migration tool
docker exec -it hiri-app /app/freyja migrate down
```

## Useful Commands

### On VPS

```bash
# View logs
docker compose logs -f app
docker compose logs -f caddy
docker compose logs -f postgres

# Restart services
docker compose restart app

# Check status
docker compose ps

# Pull latest and restart
docker compose pull app
docker compose up -d --no-deps app

# Full restart (all services)
docker compose down
docker compose up -d

# Database shell
docker exec -it hiri-postgres psql -U hiri -d hiri
```

### Local Development

```bash
# Run tests
go test -race -short ./...

# Vet code
go vet ./...

# Build
go build -o freyja ./cmd/server

# Create release tag
git tag v1.0.0
git push origin v1.0.0
```

## Host-Based Routing

The Go application handles host-based routing internally. Caddy forwards all requests to the app, which routes based on the `Host` header:

```
Request: https://hiri.coffee/pricing
         ↓
Caddy: Terminates TLS, forwards to app:3000
         ↓
App: Host = "hiri.coffee" → serves marketing site

Request: https://app.hiri.coffee/admin
         ↓
Caddy: Terminates TLS, forwards to app:3000
         ↓
App: Host = "app.hiri.coffee" → serves application
```

### Domain Routing Summary

| Host | Routes To |
|------|-----------|
| `hiri.coffee` | Marketing site (/, /pricing, /about, etc.) |
| `www.hiri.coffee` | Redirects to `hiri.coffee` |
| `app.hiri.coffee` | Application (admin, storefront, APIs) |
| Other subdomains | Application (for future custom domains) |

## Multi-Tenant Routing (Future)

Tenant routing for custom subdomains will be handled by the Go application:

```
Request: https://bluemountain.hiri.coffee
         ↓
Caddy: Terminates TLS, forwards to app:3000
         ↓
App: Reads Host header → "bluemountain.hiri.coffee"
     Looks up tenant → serves response
```

## Security Notes

- SSH key auth only (password auth disabled)
- Non-root deploy user for CI/CD
- App runs as non-root user in container
- PostgreSQL not exposed to internet (internal network only)
- TLS via Let's Encrypt with auto-renewal
- `.env` file has `600` permissions
