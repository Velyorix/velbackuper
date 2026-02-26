package lock

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const DefaultLockDir = "/var/run/velbackuper"

type LocalLocker struct {
	path string
	ttl  time.Duration
	file *os.File
	mu   sync.Mutex
	held bool
}

type LocalOptions struct {
	Dir  string
	Name string
	TTL  time.Duration
}

func NewLocal(opts LocalOptions) (*LocalLocker, error) {
	dir := opts.Dir
	if dir == "" {
		dir = DefaultLockDir
	}
	name := opts.Name
	if name == "" {
		name = "default"
	}
	if filepath.Base(name) != name {
		name = "default"
	}
	path := filepath.Join(dir, name+".lock")
	return &LocalLocker{path: path, ttl: opts.TTL}, nil
}

func (l *LocalLocker) Acquire(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.held {
		return fmt.Errorf("lock already held by this process")
	}

	if err := os.MkdirAll(filepath.Dir(l.path), 0755); err != nil {
		return fmt.Errorf("create lock dir: %w", err)
	}

	tryAcquire := func() (*os.File, error) {
		return os.OpenFile(l.path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0640)
	}

	file, err := tryAcquire()
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("create lock file: %w", err)
		}
		if l.ttl <= 0 {
			return fmt.Errorf("lock file exists: %s (another process may be running)", l.path)
		}
		info, statErr := os.Stat(l.path)
		if statErr != nil {
			return fmt.Errorf("lock file exists and stat failed: %w", statErr)
		}
		if time.Since(info.ModTime()) < l.ttl {
			return fmt.Errorf("lock file exists: %s (held by another process)", l.path)
		}
		if removeErr := os.Remove(l.path); removeErr != nil {
			return fmt.Errorf("stale lock file exists, remove failed: %w", removeErr)
		}
		file, err = tryAcquire()
		if err != nil {
			return fmt.Errorf("retry acquire after stale remove: %w", err)
		}
	}

	if _, err := file.WriteString(fmt.Sprintf("%d\n", os.Getpid())); err != nil {
		_ = file.Close()
		_ = os.Remove(l.path)
		return fmt.Errorf("write lock file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(l.path)
		return fmt.Errorf("sync lock file: %w", err)
	}

	l.file = file
	l.held = true
	return nil
}

func (l *LocalLocker) Release(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.held {
		return nil
	}
	var errs []error
	if l.file != nil {
		if err := l.file.Close(); err != nil {
			errs = append(errs, err)
		}
		l.file = nil
	}
	if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		errs = append(errs, err)
	}
	l.held = false
	if len(errs) > 0 {
		return fmt.Errorf("release lock: %v", errs)
	}
	return nil
}
