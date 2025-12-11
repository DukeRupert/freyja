# Creating a Test Tenant (Development)

This guide explains how to quickly create a tenant for local development without needing Stripe configured.

## Prerequisites

- Docker running (for PostgreSQL and Mailhog)
- Environment set to `ENV=dev` or `ENV=development`

## Quick Start

1. **Start the services:**

   ```bash
   docker compose up -d
   ```

2. **Start the server in dev mode:**

   ```bash
   ENV=dev go run cmd/server/main.go
   ```

   You should see this warning in the logs confirming the bypass is enabled:
   ```
   WARN DEV MODE: /dev/signup bypass enabled - do NOT use in production!
   ```

3. **Open the dev signup form:**

   Visit: http://localhost:3001/dev/signup

4. **Fill in the form:**

   | Field | Example |
   |-------|---------|
   | Business Name | Acme Coffee Roasters |
   | Email | owner@example.com |
   | Password | password123 |

5. **Click "Create Tenant & Login"**

   You'll be automatically logged in and redirected to `/admin`.

## What Gets Created

The dev bypass creates:

| Entity | Details |
|--------|---------|
| **Tenant** | Status: `active`, slug generated from business name |
| **Operator** | Role: `owner`, status: `active` |
| **Session** | 7-day session cookie set automatically |

## Verifying in the Database

Check the tenant was created:

```bash
docker exec freyja-postgres-1 psql -U freyja -d freyja -c \
  "SELECT name, slug, status FROM tenants ORDER BY created_at DESC LIMIT 3;"
```

Check the operator:

```bash
docker exec freyja-postgres-1 psql -U freyja -d freyja -c \
  "SELECT email, role, status FROM tenant_operators ORDER BY created_at DESC LIMIT 3;"
```

## Using cURL

You can also create a tenant via cURL:

```bash
curl -X POST http://localhost:3001/dev/signup \
  -d "business_name=Test Roasters" \
  -d "email=test@example.com" \
  -d "password=testpassword123" \
  -c cookies.txt \
  -L

# Use the session cookie for subsequent requests
curl -b cookies.txt http://localhost:3000/admin
```

## Ports

| Port | Service |
|------|---------|
| 3000 | Main app (storefront, admin, API) |
| 3001 | SaaS marketing site (includes /dev/signup) |
| 5432 | PostgreSQL |
| 8025 | Mailhog web UI |

## Troubleshooting

### "Connection refused" on startup

The database isn't running. Start it with:
```bash
docker compose up -d
```

### Dev signup route returns 404

The bypass is only enabled when `ENV=dev` or `ENV=development`. Check your environment:
```bash
echo $ENV
```

### "Tenant with this email already exists"

Each email must be unique. Use a different email or delete the existing tenant:
```bash
docker exec freyja-postgres-1 psql -U freyja -d freyja -c \
  "DELETE FROM tenants WHERE email = 'test@example.com';"
```

## Production Warning

The `/dev/signup` route is **never** available in production. It only exists when:
- `ENV=dev`, or
- `ENV=development`

In any other environment, the `DevBypassHandler` is `nil` and routes are not registered.
