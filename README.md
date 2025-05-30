# Freyja ☕

A modern e-commerce platform specifically designed for coffee roasting businesses, enabling seamless product management, subscription services, and customer engagement.

## 🎯 Purpose

Freyja empowers coffee roasters to:
- **Manage Products**: Comprehensive coffee product catalog with origin tracking, roast profiles, and processing methods
- **Subscription Services**: Automated recurring deliveries with flexible intervals and customer preferences
- **E-commerce Operations**: Full-featured online store with inventory management and order processing
- **Customer Experience**: Rich product discovery with detailed coffee information and tasting notes

Perfect for specialty coffee roasters, coffee shops, and distributors looking to scale their online presence while maintaining the artisanal quality that defines great coffee.

## 🛠️ Technologies Used

### Backend Framework
- **Go (Golang)** - High-performance backend API
- **Echo** - Fast and minimalist web framework
- **PostgreSQL** - Robust relational database for complex queries

### Database & Migrations
- **sqlc** - Type-safe SQL code generation
- **Goose** - Database migration management
- **pgx/v5** - High-performance PostgreSQL driver

### API & Documentation
- **OpenAPI 3.1** - API specification and documentation
- **oapi-codegen** - Automatic Go server code generation from OpenAPI specs

### Payment & Authentication
- **Stripe** - Secure payment processing and subscription billing
- **JWT** - Stateless authentication and authorization

### Message Queue & Events
- **NATS** - Lightweight message bus for event-driven architecture

### Development Tools
- **Air** - Live reload for Go applications
- **golangci-lint** - Comprehensive Go linting
- **Docker** - Containerization for consistent deployments

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- PostgreSQL 14+
- Make

### Setup
```bash
# Clone the repository
git clone <repository-url>
cd freyja

# Set up development environment
make dev-setup

# Configure environment variables
# Create and edit .env file with your configuration

# Create database and run migrations
make db-create
make migrate-up

# Generate code and build
make generate
make build

# Start the application
make run
```

## 📋 Makefile Commands

### 🗄️ Database Migrations
```bash
make migrate-up                              # Run all pending migrations
make migrate-down                            # Rollback the last migration
make migrate-down-to VERSION=20240101000000 # Rollback to specific version
make migrate-reset                           # Reset database (rollback all migrations)
make migrate-status                          # Show migration status
make migrate-version                         # Show current migration version
make migrate-create NAME=add_new_table       # Create a new migration file
```

### 🔧 Development
```bash
make dev-setup        # Install development tools (goose, sqlc, oapi-codegen)
make generate         # Generate sqlc and OpenAPI code
make build            # Build the application
make run              # Run the application
make clean            # Clean build artifacts
make config           # Show current configuration
```

### 🧪 Testing & Quality
```bash
make test             # Run tests
make test-coverage    # Run tests with coverage report
make lint             # Run linter
```

### 💾 Database Management
```bash
make db-create        # Create database
make db-drop          # Drop database (with confirmation)
```

### 🐳 Docker (Optional)
```bash
make docker-build     # Build Docker image
make docker-up        # Start Docker services
make docker-down      # Stop Docker services
```

### 📚 Help
```bash
make help             # Show all available commands with descriptions
```

## 🏗️ Project Structure

```
freyja/
├── cmd/api/                 # Application entry point
├── internal/
│   ├── api/                 # Generated OpenAPI server code
│   ├── dbstore/             # Generated sqlc database code
│   ├── migrations/          # Database migration files
│   ├── handlers/            # HTTP request handlers
│   ├── services/            # Business logic
│   └── models/              # Domain models
├── config/                  # Configuration management
├── openapi.yaml             # API specification
├── sqlc.yaml               # sqlc configuration
├── oapi-codegen.yaml       # OpenAPI code generation config
├── Makefile                # Development commands
└── README.md               # This file
```

## 🌟 Key Features

### Coffee-Specific Product Management
- **Origin Tracking**: Country, region, farm, and altitude information
- **Processing Methods**: Washed, natural, honey, semi-washed processing
- **Roast Profiles**: Light to dark roast level categorization
- **Flavor Notes**: Tagged flavor descriptors for product discovery
- **Varietal Information**: Coffee variety and harvest date tracking

### Subscription System
- **Flexible Intervals**: Weekly, bi-weekly, monthly, quarterly options
- **Quantity Controls**: Min/max subscription quantities per product
- **Automatic Discounts**: Percentage-based subscriber discounts
- **Priority Products**: Featured products for subscription customers

### Developer Experience
- **Type Safety**: sqlc generates type-safe database operations
- **API Documentation**: Auto-generated from OpenAPI specification
- **Live Reload**: Fast development with automatic recompilation
- **Database Migrations**: Version-controlled schema evolution

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## ☕ About the Name

Freyja is named after the Norse goddess associated with fertility, prosperity, and abundance - qualities that perfectly embody a thriving coffee business bringing quality and joy to customers worldwide.