package storage

import (
	"context"
	"io"

	"github.com/dukerupert/freyja/internal"
)

// Storage defines the interface for file storage operations.
// Implementations can use local filesystem, S3, or any other storage backend.
type Storage interface {
	// Put stores a file and returns its URL/path for retrieval.
	// The key should be a unique identifier (e.g., "products/uuid/image.jpg").
	Put(ctx context.Context, key string, content io.Reader, contentType string) (string, error)

	// Get retrieves a file by its key.
	// Returns an io.ReadCloser that must be closed by the caller.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes a file by its key.
	// Returns nil if the file doesn't exist (idempotent).
	Delete(ctx context.Context, key string) error

	// URL returns the public URL for accessing a stored file.
	// For local storage, this might be a relative path.
	// For S3, this would be the full HTTPS URL.
	URL(key string) string

	// Exists checks if a file exists at the given key.
	Exists(ctx context.Context, key string) (bool, error)
}

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
		return nil, ErrUnknownProvider(cfg.Provider)
	}
}
