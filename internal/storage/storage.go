package storage

import (
	"context"
	"io"
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
