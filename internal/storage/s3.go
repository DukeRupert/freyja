package storage

import (
	"context"
	"io"
)

// S3Storage implements Storage using AWS S3 or S3-compatible storage.
// This is a placeholder for post-MVP implementation.
type S3Storage struct {
	bucket string
	region string
	// s3Client *s3.Client // AWS SDK client (not implemented yet)
}

// NewS3Storage creates a new S3 storage implementation.
// This is a stub - full implementation will use AWS SDK for Go v2.
func NewS3Storage(bucket, region string) (*S3Storage, error) {
	// TODO: Initialize S3 client with credentials
	// client := s3.NewFromConfig(cfg)

	return &S3Storage{
		bucket: bucket,
		region: region,
	}, nil
}

// Put stores a file in S3.
func (s *S3Storage) Put(ctx context.Context, key string, content io.Reader, contentType string) (string, error) {
	// TODO: Implement S3 PutObject
	// Use s3.PutObjectInput with ContentType, ServerSideEncryption, etc.
	panic("S3Storage not implemented yet")
}

// Get retrieves a file from S3.
func (s *S3Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	// TODO: Implement S3 GetObject
	panic("S3Storage not implemented yet")
}

// Delete removes a file from S3.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	// TODO: Implement S3 DeleteObject
	panic("S3Storage not implemented yet")
}

// URL returns the public URL for accessing a file in S3.
func (s *S3Storage) URL(key string) string {
	// TODO: Generate S3 URL (https://bucket.s3.region.amazonaws.com/key)
	// Or use CloudFront URL if CDN is configured
	panic("S3Storage not implemented yet")
}

// Exists checks if a file exists in S3.
func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	// TODO: Implement S3 HeadObject
	panic("S3Storage not implemented yet")
}
