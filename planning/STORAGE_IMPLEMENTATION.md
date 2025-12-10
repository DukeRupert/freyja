# Storage Implementation Plan

## Overview

Platform-controlled file storage using Cloudflare R2 in production and local filesystem in development. Configuration via environment variables (not per-tenant database config).

**Key decisions:**
- Single storage backend for all tenants (Freyja controls infrastructure)
- Tenant isolation via key prefixes: `{tenant_id}/products/{id}/image.jpg`
- R2 chosen for free egress and S3-compatible API
- Storage included in $149/month plan (no separate billing)

## Architecture

```
Production:  App → R2Storage → Cloudflare R2 (S3-compatible)
Development: App → LocalStorage → ./web/static/uploads/
```

Both implement the existing `storage.Storage` interface.

## Configuration

### Environment Variables

```bash
# .env.example additions

# Storage Configuration
# Provider: "local" (development) or "r2" (production)
STORAGE_PROVIDER=local

# Cloudflare R2 Configuration (required when STORAGE_PROVIDER=r2)
R2_ACCOUNT_ID=your_account_id
R2_ACCESS_KEY_ID=your_access_key
R2_SECRET_ACCESS_KEY=your_secret_key
R2_BUCKET_NAME=freyja-files
R2_PUBLIC_URL=https://files.your-domain.com  # Optional: custom domain or R2.dev URL

# Local Storage Configuration (used when STORAGE_PROVIDER=local)
LOCAL_STORAGE_PATH=./web/static/uploads
LOCAL_STORAGE_URL=/uploads
```

### Config Struct Addition

```go
// internal/config.go

type StorageConfig struct {
    Provider        string // "local" or "r2"
    // Local storage
    LocalPath       string
    LocalURL        string
    // R2 storage
    R2AccountID     string
    R2AccessKeyID   string
    R2SecretKey     string
    R2BucketName    string
    R2PublicURL     string // Optional custom domain
}
```

## Implementation

### File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/config.go` | Modify | Add `StorageConfig` struct and parsing |
| `internal/storage/s3.go` | Replace | Rename to `r2.go`, implement R2Storage |
| `internal/storage/storage.go` | Modify | Add `NewStorage()` factory function |
| `main.go` | Modify | Initialize storage from config |
| `.env.example` | Modify | Add storage environment variables |

### 1. Config Changes (`internal/config.go`)

Add to Config struct:
```go
type Config struct {
    // ... existing fields ...
    Storage StorageConfig
}

type StorageConfig struct {
    Provider      string
    LocalPath     string
    LocalURL      string
    R2AccountID   string
    R2AccessKeyID string
    R2SecretKey   string
    R2BucketName  string
    R2PublicURL   string
}
```

Add to NewConfig():
```go
Storage: StorageConfig{
    Provider:      getEnv("STORAGE_PROVIDER", "local"),
    LocalPath:     getEnv("LOCAL_STORAGE_PATH", "./web/static/uploads"),
    LocalURL:      getEnv("LOCAL_STORAGE_URL", "/uploads"),
    R2AccountID:   getEnv("R2_ACCOUNT_ID", ""),
    R2AccessKeyID: getEnv("R2_ACCESS_KEY_ID", ""),
    R2SecretKey:   getEnv("R2_SECRET_ACCESS_KEY", ""),
    R2BucketName:  getEnv("R2_BUCKET_NAME", ""),
    R2PublicURL:   getEnv("R2_PUBLIC_URL", ""),
},
```

Add validation in production:
```go
if cfg.Env == "prod" && cfg.Storage.Provider == "r2" {
    if cfg.Storage.R2AccessKeyID == "" || cfg.Storage.R2SecretKey == "" {
        return nil, fmt.Errorf("R2 credentials required in production")
    }
    if cfg.Storage.R2BucketName == "" {
        return nil, fmt.Errorf("R2_BUCKET_NAME required in production")
    }
}
```

### 2. R2 Storage Implementation (`internal/storage/r2.go`)

Replace the existing `s3.go` stub:

```go
package storage

import (
    "context"
    "fmt"
    "io"
    "strings"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Config contains configuration for Cloudflare R2 storage.
type R2Config struct {
    AccountID   string // Cloudflare account ID
    AccessKeyID string // R2 access key ID
    SecretKey   string // R2 secret access key
    BucketName  string // R2 bucket name
    PublicURL   string // Public URL for serving files (custom domain or R2.dev)
}

// R2Storage implements Storage using Cloudflare R2.
type R2Storage struct {
    client    *s3.Client
    bucket    string
    publicURL string
}

// NewR2Storage creates a new Cloudflare R2 storage implementation.
func NewR2Storage(cfg R2Config) (*R2Storage, error) {
    if cfg.AccountID == "" {
        return nil, fmt.Errorf("R2 account ID is required")
    }
    if cfg.AccessKeyID == "" || cfg.SecretKey == "" {
        return nil, fmt.Errorf("R2 credentials are required")
    }
    if cfg.BucketName == "" {
        return nil, fmt.Errorf("R2 bucket name is required")
    }

    // R2 endpoint format: https://<account_id>.r2.cloudflarestorage.com
    endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

    // Create credentials provider
    credsProvider := credentials.NewStaticCredentialsProvider(
        cfg.AccessKeyID,
        cfg.SecretKey,
        "",
    )

    // Load AWS config with R2-specific settings
    awsCfg, err := config.LoadDefaultConfig(context.Background(),
        config.WithRegion("auto"), // R2 uses "auto" for region
        config.WithCredentialsProvider(credsProvider),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }

    // Create S3 client with R2 endpoint
    client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
        o.BaseEndpoint = aws.String(endpoint)
        o.UsePathStyle = true // R2 requires path-style URLs
    })

    return &R2Storage{
        client:    client,
        bucket:    cfg.BucketName,
        publicURL: strings.TrimSuffix(cfg.PublicURL, "/"),
    }, nil
}

// Put stores a file in R2.
func (s *R2Storage) Put(ctx context.Context, key string, content io.Reader, contentType string) (string, error) {
    input := &s3.PutObjectInput{
        Bucket:      aws.String(s.bucket),
        Key:         aws.String(key),
        Body:        content,
        ContentType: aws.String(contentType),
    }

    _, err := s.client.PutObject(ctx, input)
    if err != nil {
        return "", fmt.Errorf("failed to upload to R2: %w", err)
    }

    return s.URL(key), nil
}

// Get retrieves a file from R2.
func (s *R2Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
    result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get from R2: %w", err)
    }

    return result.Body, nil
}

// Delete removes a file from R2.
func (s *R2Storage) Delete(ctx context.Context, key string) error {
    _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return fmt.Errorf("failed to delete from R2: %w", err)
    }

    return nil
}

// URL returns the public URL for accessing a file.
func (s *R2Storage) URL(key string) string {
    if s.publicURL != "" {
        return fmt.Sprintf("%s/%s", s.publicURL, key)
    }
    // Fallback: R2 doesn't have a default public URL
    // Files are private by default; publicURL should be configured
    return key
}

// Exists checks if a file exists in R2.
func (s *R2Storage) Exists(ctx context.Context, key string) (bool, error) {
    _, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        // Check if it's a "not found" error
        if isNotFoundError(err) {
            return false, nil
        }
        return false, fmt.Errorf("failed to check existence in R2: %w", err)
    }

    return true, nil
}

// isNotFoundError checks if an error indicates the object doesn't exist.
func isNotFoundError(err error) bool {
    if err == nil {
        return false
    }
    errStr := err.Error()
    return strings.Contains(errStr, "NoSuchKey") ||
        strings.Contains(errStr, "NotFound") ||
        strings.Contains(errStr, "404")
}
```

### 3. Factory Function (`internal/storage/storage.go`)

Add factory function to create storage from config:

```go
// Add to storage.go

import "github.com/dukerupert/hiri/internal"

// NewStorage creates a Storage implementation based on configuration.
// Returns LocalStorage for "local" provider, R2Storage for "r2" provider.
func NewStorage(cfg internal.StorageConfig) (Storage, error) {
    switch cfg.Provider {
    case "local", "":
        return NewLocalStorage(cfg.LocalPath, cfg.LocalURL)
    case "r2":
        return NewR2Storage(R2Config{
            AccountID:   cfg.R2AccountID,
            AccessKeyID: cfg.R2AccessKeyID,
            SecretKey:   cfg.R2SecretKey,
            BucketName:  cfg.R2BucketName,
            PublicURL:   cfg.R2PublicURL,
        })
    default:
        return nil, fmt.Errorf("unknown storage provider: %s", cfg.Provider)
    }
}
```

### 4. Application Initialization (`main.go`)

Add storage initialization:

```go
// In main() or application setup

// Initialize storage
store, err := storage.NewStorage(cfg.Storage)
if err != nil {
    log.Fatalf("Failed to initialize storage: %v", err)
}

// Pass store to handlers that need it
```

### 5. Delete Old S3 Stub

Remove `internal/storage/s3.go` after creating `r2.go`.

## Dependencies

Add AWS SDK v2 for S3:

```bash
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/credentials
go get github.com/aws/aws-sdk-go-v2/service/s3
```

## Tenant Isolation

Storage keys MUST include tenant ID prefix for multi-tenant isolation:

```go
// Helper function for generating storage keys
func StorageKey(tenantID uuid.UUID, category string, filename string) string {
    return fmt.Sprintf("%s/%s/%s", tenantID.String(), category, filename)
}

// Usage examples:
// Product image:  "550e8400-e29b-41d4-a716-446655440000/products/image.jpg"
// Invoice PDF:    "550e8400-e29b-41d4-a716-446655440000/invoices/INV-001.pdf"
// Customer doc:   "550e8400-e29b-41d4-a716-446655440000/customers/avatar.png"
```

## Cloudflare R2 Setup

### 1. Create R2 Bucket

```bash
# Via Cloudflare Dashboard or Wrangler CLI
wrangler r2 bucket create freyja-files
```

### 2. Create API Token

1. Go to Cloudflare Dashboard → R2 → Manage R2 API Tokens
2. Create token with "Object Read & Write" permissions
3. Copy Access Key ID and Secret Access Key

### 3. Configure Public Access (for serving images)

Option A: R2.dev subdomain (quick, free)
- Enable in bucket settings
- URL format: `https://pub-{hash}.r2.dev/{key}`

Option B: Custom domain (recommended for production)
- Add custom domain in bucket settings
- Configure DNS CNAME to R2
- URL format: `https://files.your-domain.com/{key}`

### 4. CORS Configuration (if needed for direct uploads)

```json
[
  {
    "AllowedOrigins": ["https://your-app.com"],
    "AllowedMethods": ["GET", "PUT"],
    "AllowedHeaders": ["*"],
    "MaxAgeSeconds": 3600
  }
]
```

## Testing

### Unit Tests

```go
// internal/storage/r2_test.go

func TestR2Storage_URL(t *testing.T) {
    s := &R2Storage{
        bucket:    "test-bucket",
        publicURL: "https://files.example.com",
    }

    got := s.URL("tenant-id/products/image.jpg")
    want := "https://files.example.com/tenant-id/products/image.jpg"

    if got != want {
        t.Errorf("URL() = %q, want %q", got, want)
    }
}
```

### Integration Tests

Use LocalStorage for tests by default. For R2 integration tests:

```go
func TestR2Integration(t *testing.T) {
    if os.Getenv("R2_ACCESS_KEY_ID") == "" {
        t.Skip("R2 credentials not configured")
    }

    store, err := storage.NewR2Storage(storage.R2Config{
        AccountID:   os.Getenv("R2_ACCOUNT_ID"),
        AccessKeyID: os.Getenv("R2_ACCESS_KEY_ID"),
        SecretKey:   os.Getenv("R2_SECRET_ACCESS_KEY"),
        BucketName:  os.Getenv("R2_BUCKET_NAME") + "-test",
        PublicURL:   os.Getenv("R2_PUBLIC_URL"),
    })
    require.NoError(t, err)

    // Test Put
    key := "test/" + uuid.New().String() + ".txt"
    content := strings.NewReader("test content")
    url, err := store.Put(context.Background(), key, content, "text/plain")
    require.NoError(t, err)
    assert.Contains(t, url, key)

    // Test Exists
    exists, err := store.Exists(context.Background(), key)
    require.NoError(t, err)
    assert.True(t, exists)

    // Test Get
    reader, err := store.Get(context.Background(), key)
    require.NoError(t, err)
    defer reader.Close()
    data, _ := io.ReadAll(reader)
    assert.Equal(t, "test content", string(data))

    // Cleanup
    err = store.Delete(context.Background(), key)
    require.NoError(t, err)
}
```

## Cost Estimate

**Cloudflare R2 Pricing:**
- Storage: $0.015/GB/month
- Class A ops (write): $4.50/million
- Class B ops (read): $0.36/million
- Egress: **Free**

**Estimated costs for 100 tenants:**
- Storage: 100 tenants × 500MB = 50GB → $0.75/month
- Operations: ~100k/month → ~$0.05/month
- **Total: ~$1/month**

**Free tier (generous):**
- 10GB storage/month
- 1M Class A ops/month
- 10M Class B ops/month

## Implementation Checklist

- [ ] Add AWS SDK v2 dependencies
- [ ] Add `StorageConfig` to `internal/config.go`
- [ ] Create `internal/storage/r2.go` with R2Storage implementation
- [ ] Add `NewStorage()` factory to `internal/storage/storage.go`
- [ ] Delete `internal/storage/s3.go` (old stub)
- [ ] Initialize storage in `main.go`
- [ ] Update `.env.example` with storage variables
- [ ] Write unit tests for R2Storage
- [ ] Create R2 bucket in Cloudflare
- [ ] Configure public access (R2.dev or custom domain)
- [ ] Test in development with local storage
- [ ] Test in staging with R2
