package doctor

import (
	"context"
	"fmt"
	"os"
	"time"

	"VelBackuper/internal/config"
	"VelBackuper/internal/lock"
	"VelBackuper/internal/s3"
)

type CheckResult struct {
	Name   string
	OK     bool
	Detail string
}

func Run(ctx context.Context, cfg *config.Config) []CheckResult {
	var results []CheckResult

	results = append(results, CheckResult{
		Name:   "config",
		OK:     cfg != nil,
		Detail: "configuration loaded",
	})

	if cfg != nil && cfg.S3 != nil {
		ok, detail := checkS3(ctx, cfg)
		results = append(results, CheckResult{Name: "s3", OK: ok, Detail: detail})
	} else {
		results = append(results, CheckResult{Name: "s3", OK: false, Detail: "s3 not configured"})
	}

	ok, detail := checkLocalLock()
	results = append(results, CheckResult{Name: "local lock", OK: ok, Detail: detail})

	ok, detail = checkDisk()
	results = append(results, CheckResult{Name: "disk", OK: ok, Detail: detail})

	return results
}

func checkS3(ctx context.Context, cfg *config.Config) (bool, string) {
	client, err := s3.New(ctx, s3.Options{
		Endpoint:           cfg.S3.Endpoint,
		Region:             cfg.S3.Region,
		AccessKey:          cfg.S3.AccessKey,
		SecretKey:          cfg.S3.SecretKey,
		Bucket:             cfg.S3.Bucket,
		Prefix:             cfg.S3.Prefix,
		InsecureSkipVerify: cfg.S3.TLS != nil && cfg.S3.TLS.InsecureSkipVerify,
	})
	if err != nil {
		return false, fmt.Sprintf("s3 client init failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = client.ListObjects(ctx, "", 1)
	if err != nil {
		return false, fmt.Sprintf("s3 list failed: %v", err)
	}
	return true, fmt.Sprintf("s3 OK (bucket=%s, prefix=%s)", cfg.S3.Bucket, cfg.S3.Prefix)
}

func checkLocalLock() (bool, string) {
	l, err := lock.NewLocal(lock.LocalOptions{Name: "doctor"})
	if err != nil {
		return false, fmt.Sprintf("local lock init failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := l.Acquire(ctx); err != nil {
		return false, fmt.Sprintf("local lock acquire failed: %v", err)
	}
	if err := l.Release(context.Background()); err != nil {
		return false, fmt.Sprintf("local lock release failed: %v", err)
	}
	return true, fmt.Sprintf("local lock dir accessible (%s)", lock.DefaultLockDir)
}

func checkDisk() (bool, string) {
	dir := os.TempDir()
	f, err := os.CreateTemp(dir, "velbackuper-doctor-*")
	if err != nil {
		return false, fmt.Sprintf("create temp file failed in %s: %v", dir, err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString("test"); err != nil {
		_ = f.Close()
		return false, fmt.Sprintf("write temp file failed: %v", err)
	}
	if err := f.Close(); err != nil {
		return false, fmt.Sprintf("close temp file failed: %v", err)
	}
	return true, fmt.Sprintf("temp dir writable (%s)", dir)
}
