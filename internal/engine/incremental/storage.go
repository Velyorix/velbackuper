package incremental

import (
	"context"
	"io"
	"time"
)

type Storage interface {
	HeadObject(ctx context.Context, key string) (*time.Time, error)
	PutObject(ctx context.Context, key string, body io.Reader, contentLength int64) error
}
