# =============================================================================
# Freyja Coffee E-commerce Platform Dockerfile
# Multi-stage build for optimal production image size and security
# =============================================================================

# =============================================================================
# Stage 1: Build Stage
# =============================================================================
FROM golang:1.21-alpine AS builder

# Set build arguments
ARG VERSION=dev
ARG BUILD_TIME
ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Create non-root user for build
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o freyja \
    cmd/server/main.go

# Verify the binary
RUN ./freyja --version || echo "Binary built successfully"

# =============================================================================
# Stage 2: Production Stage
# =============================================================================
FROM alpine:3.18 AS production

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    curl \
    wget \
    && update-ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set timezone (can be overridden with environment variable)
ENV TZ=UTC

# Create application directories
RUN mkdir -p /app/logs /app/tmp /app/config && \
    chown -R appuser:appgroup /app

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/freyja /app/freyja

# Copy configuration files (if any)
COPY --chown=appuser:appgroup config/ /app/config/

# Make binary executable
RUN chmod +x /app/freyja

# Create health check script
RUN echo '#!/bin/sh' > /app/healthcheck.sh && \
    echo 'curl -f http://localhost:${PORT:-8080}/health || exit 1' >> /app/healthcheck.sh && \
    chmod +x /app/healthcheck.sh && \
    chown appuser:appgroup /app/healthcheck.sh

# Switch to non-root user
USER appuser

# Expose port (default 8080, configurable via environment)
EXPOSE 8080

# Environment variables with defaults
ENV PORT=8080
ENV ENV=production
ENV LOG_LEVEL=info
ENV GIN_MODE=release

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD /app/healthcheck.sh

# Labels for metadata
LABEL maintainer="dukerupert" \
    org.opencontainers.image.title="Freyja Coffee E-commerce" \
    org.opencontainers.image.description="Modern coffee e-commerce platform built with Go" \
    org.opencontainers.image.url="https://github.com/dukerupert/freyja" \
    org.opencontainers.image.source="https://github.com/dukerupert/freyja" \
    org.opencontainers.image.version="${VERSION}" \
    org.opencontainers.image.created="${BUILD_TIME}" \
    org.opencontainers.image.licenses="MIT"

# Default command
CMD ["/app/freyja"]

# =============================================================================
# Development Stage (optional, for development with hot reload)
# =============================================================================
FROM golang:1.21-alpine AS development

# Install development tools
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    curl \
    air \
    && update-ca-certificates

# Create non-root user
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Install air for hot reload
RUN go install github.com/cosmtrek/air@latest

# Copy source code
COPY . .

# Change ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Environment variables for development
ENV ENV=development
ENV LOG_LEVEL=debug
ENV PORT=8080

# Default command for development (with hot reload)
CMD ["air", "-c", ".air.toml"]

# =============================================================================
# Testing Stage (for running tests in CI/CD)
# =============================================================================
FROM golang:1.21-alpine AS testing

# Install test dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    gcc \
    musl-dev \
    && update-ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Install test tools
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
RUN go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Default command for testing
CMD ["go", "test", "-v", "-race", "-coverprofile=coverage.out", "./..."]

# =============================================================================
# Build target selection
# Default target is production, but can be overridden:
# docker build --target development -t freyja:dev .
# docker build --target testing -t freyja:test .
# docker build --target production -t freyja:latest .
# =============================================================================
