# Dockerfile - Fixed for cmd/server/main.go structure
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Create and set working directory
RUN mkdir -p /build
WORKDIR /build

# Copy go.mod and go.sum first for dependency caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy entire project (needed for internal packages)
COPY . .

# Debug: Verify project structure
RUN echo "=== Project structure ===" && \
    ls -la && \
    echo "=== cmd directory ===" && \
    ls -la cmd/ && \
    echo "=== cmd/server directory ===" && \
    ls -la cmd/server/ && \
    echo "=== Go files ===" && \
    find . -name "*.go" | head -10

# Build the application from cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Verify binary was created
RUN ls -la main && echo "✅ Binary created: $(du -h main)"

# Production stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

# Create app directory and user
RUN mkdir -p /app && adduser -D -s /bin/sh appuser

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/main .

# Set permissions
RUN chmod +x main && chown appuser:appuser main

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./main"]
