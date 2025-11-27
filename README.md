# Freyja

E-commerce platform built exclusively for coffee roasters.

## Quick Start

### Prerequisites

- Go 1.25.4 or later
- Docker and Docker Compose
- Make

### Initial Setup

1. **Install development tools:**
   ```bash
   make install-tools
   ```

   This installs:
   - `goose` - Database migrations
   - `sqlc` - Type-safe SQL queries
   - `air` - Live reload for development

2. **Start Docker services:**
   ```bash
   make docker-up
   ```

   This starts:
   - PostgreSQL on port 5432
   - Mailhog on ports 1025 (SMTP) and 8025 (Web UI)

3. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your settings (defaults work for local development)
   ```

4. **Install Go dependencies:**
   ```bash
   make deps
   ```

5. **Run migrations** (once database schema is created):
   ```bash
   make migrate
   ```

6. **Generate sqlc code** (once queries are written):
   ```bash
   make sqlc-gen
   ```

7. **Start development server:**
   ```bash
   make dev
   ```

   The application will start on http://localhost:3000 with live reload enabled.

## Development

### Available Commands

Run `make help` to see all available commands:

```bash
make help              # Show all available commands
make dev               # Start development server with live reload
make build             # Build the application
make test              # Run tests
make test-coverage     # Run tests with coverage report
make migrate           # Run database migrations
make migrate-down      # Rollback last migration
make migrate-create    # Create new migration (NAME=migration_name)
make sqlc-gen          # Generate sqlc code
make docker-up         # Start Docker services
make docker-down       # Stop Docker services
make clean             # Clean build artifacts
```

### Project Structure

```
freyja/
├── cmd/server/              # Application entry point
├── internal/
│   ├── config/              # Configuration
│   ├── domain/              # Business domain types
│   ├── billing/             # Payment processing (Stripe)
│   ├── shipping/            # Shipping providers
│   ├── email/               # Email sending
│   ├── repository/          # Database queries (sqlc generated)
│   ├── handler/             # HTTP handlers
│   ├── middleware/          # HTTP middleware
│   ├── jobs/                # Background jobs
│   └── worker/              # Job processing
├── migrations/              # SQL migrations
├── sqlc/                    # sqlc configuration and queries
├── web/                     # Templates and static assets
└── planning/                # Project documentation
```

### Development Workflow

1. Write database migrations in `migrations/`
2. Run `make migrate` to apply migrations
3. Write SQL queries in `sqlc/queries/`
4. Run `make sqlc-gen` to generate type-safe Go code
5. Write handlers, middleware, and business logic
6. Run `make dev` for live reload during development
7. Run `make test` to verify changes

### Services

- **Application**: http://localhost:3000
- **PostgreSQL**: localhost:5432
  - Database: `freyja`
  - User: `freyja`
  - Password: `password`
- **Mailhog Web UI**: http://localhost:8025
- **Mailhog SMTP**: localhost:1025

## Documentation

- [Project Purpose](planning/PURPOSE.md) - Vision and target customer
- [Technical Decisions](planning/TECHNICAL.md) - Architecture and technology choices
- [Roadmap](planning/ROADMAP.md) - Feature roadmap and milestones
- [Business Model](planning/BUSINESS.md) - Market positioning and economics
- [UI Direction](planning/UI_DIRECTION.md) - Design philosophy and guidelines
- [Claude Code Guide](CLAUDE.md) - Guide for AI-assisted development

## License

Proprietary - All rights reserved
