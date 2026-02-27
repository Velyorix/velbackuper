package archive

import (
	"context"
	"io"
)

// Storage is the subset of S3 operations used by retention and manifest helpers.
// *s3.Client implements this interface.
type Storage interface {
	ListObjects(ctx context.Context, prefix string, maxKeys int32) ([]string, error)
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
	DeleteObject(ctx context.Context, key string) error
	PutObject(ctx context.Context, key string, body io.Reader, contentLength int64) error
}
