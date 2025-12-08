package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage implements Storage using the local filesystem.
// This is the MVP implementation for development and small deployments.
type LocalStorage struct {
	basePath string // Root directory for file storage (e.g., "./web/static/uploads")
	baseURL  string // Base URL for serving files (e.g., "/uploads")
}

// NewLocalStorage creates a new local filesystem storage implementation.
//
// basePath is the directory where files will be stored (created if it doesn't exist).
// baseURL is the URL prefix for accessing files (e.g., "/uploads").
func NewLocalStorage(basePath, baseURL string) (*LocalStorage, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}, nil
}

// Put stores a file in the local filesystem.
func (s *LocalStorage) Put(ctx context.Context, key string, content io.Reader, contentType string) (string, error) {
	fullPath := filepath.Join(s.basePath, key)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy content to file
	if _, err := io.Copy(file, content); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return s.URL(key), nil
}

// Get retrieves a file from the local filesystem.
func (s *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, key)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound(key)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes a file from the local filesystem.
func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(s.basePath, key)

	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// URL returns the public URL for accessing a file.
func (s *LocalStorage) URL(key string) string {
	return filepath.Join(s.baseURL, key)
}

// Exists checks if a file exists in the local filesystem.
func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	fullPath := filepath.Join(s.basePath, key)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}
