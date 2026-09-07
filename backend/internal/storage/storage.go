package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"us.ztechai.zsms/backend/internal/config"
)

// ObjectStorage defines the unified interface for object storage (MinIO / Cloudflare R2 / AWS S3).
type ObjectStorage interface {
	// Put uploads an object with metadata.
	Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error

	// Get retrieves an object reader. Caller must close the returned ReadCloser.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes an object by key.
	Delete(ctx context.Context, key string) error

	// Exists checks if an object exists by key.
	Exists(ctx context.Context, key string) (bool, error)

	// GenerateSignedURL creates a pre-signed URL for direct download or upload.
	GenerateSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// NoopStorage is a baseline in-memory/noop placeholder for Stage 2 infrastructure foundation.
type NoopStorage struct {
	provider string
	bucket   string
}

// New creates an ObjectStorage instance based on configuration.
func New(cfg *config.Config) (ObjectStorage, error) {
	switch cfg.StorageProvider {
	case "minio":
		return &NoopStorage{provider: "minio", bucket: cfg.S3Bucket}, nil
	case "r2":
		return &NoopStorage{provider: "r2", bucket: cfg.R2Bucket}, nil
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", cfg.StorageProvider)
	}
}

func (n *NoopStorage) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	return nil
}

func (n *NoopStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("object not found: %s", key)
}

func (n *NoopStorage) Delete(ctx context.Context, key string) error {
	return nil
}

func (n *NoopStorage) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (n *NoopStorage) GenerateSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("https://mock-storage.local/%s/%s?exp=%d", n.bucket, key, int(expiry.Seconds())), nil
}
