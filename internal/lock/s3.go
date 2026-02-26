package lock

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"VelBackuper/internal/s3"
)

const lockKeyPrefix = "locks/"

type S3Locker struct {
	client *s3.Client
	name   string
	ttl    time.Duration
	key    string
	mu     sync.Mutex
	held   bool
}

type S3Options struct {
	Client *s3.Client
	Name   string
	TTL    time.Duration
}

func NewS3(opts S3Options) (*S3Locker, error) {
	if opts.Client == nil {
		return nil, fmt.Errorf("s3 lock: client is required")
	}
	name := opts.Name
	if name == "" {
		name = "default"
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		name = "default"
	}
	key := lockKeyPrefix + name + ".lock"
	return &S3Locker{
		client: opts.Client,
		name:   name,
		ttl:    opts.TTL,
		key:    key,
	}, nil
}

func (l *S3Locker) Acquire(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.held {
		return fmt.Errorf("s3 lock already held by this process")
	}

	lastMod, err := l.client.HeadObject(ctx, l.key)
	if err != nil {
		return fmt.Errorf("s3 lock head: %w", err)
	}
	if lastMod != nil {
		if l.ttl <= 0 {
			return fmt.Errorf("s3 lock already held: %s (another process may be running)", l.key)
		}
		if time.Since(*lastMod) < l.ttl {
			return fmt.Errorf("s3 lock already held: %s (held by another process)", l.key)
		}
		if err := l.client.DeleteObject(ctx, l.key); err != nil {
			return fmt.Errorf("s3 lock stale but delete failed: %w", err)
		}
	}

	body := time.Now().UTC().Format(time.RFC3339)
	if err := l.client.PutObject(ctx, l.key, strings.NewReader(body), int64(len(body))); err != nil {
		return fmt.Errorf("s3 lock put: %w", err)
	}
	l.held = true
	return nil
}

func (l *S3Locker) Release(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.held {
		return nil
	}
	if err := l.client.DeleteObject(ctx, l.key); err != nil {
		return fmt.Errorf("s3 lock release: %w", err)
	}
	l.held = false
	return nil
}
