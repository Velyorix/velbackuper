package archive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"VelBackuper/internal/s3"
)

type Manifest struct {
	Job       string `json:"job"`
	Timestamp string `json:"timestamp"`
	Key       string `json:"key"`
	Size      int64  `json:"size"`
	Host      string `json:"host"`
	Format    string `json:"format"`
}

type LatestPointer struct {
	Timestamp string `json:"timestamp"`
	Key       string `json:"key"`
}

func WriteManifest(ctx context.Context, client *s3.Client, m Manifest) error {
	key := s3.ManifestKey(m.Job, m.Timestamp)
	body, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("manifest marshal: %w", err)
	}
	return client.PutObject(ctx, key, bytes.NewReader(body), int64(len(body)))
}

func WriteLatest(ctx context.Context, client Storage, job, timestamp, archiveKey string) error {
	p := LatestPointer{Timestamp: timestamp, Key: archiveKey}
	key := s3.LatestKey(job)
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("latest pointer marshal: %w", err)
	}
	return client.PutObject(ctx, key, bytes.NewReader(body), int64(len(body)))
}

func ReadLatest(ctx context.Context, client Storage, job string) (timestamp, archiveKey string, err error) {
	key := s3.LatestKey(job)
	rc, err := client.GetObject(ctx, key)
	if err != nil {
		return "", "", err
	}
	defer rc.Close()
	var p LatestPointer
	if err := json.NewDecoder(rc).Decode(&p); err != nil {
		return "", "", err
	}
	return p.Timestamp, p.Key, nil
}

func ReadManifestByKey(ctx context.Context, client Storage, manifestKey string) (*Manifest, error) {
	rc, err := client.GetObject(ctx, manifestKey)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var m Manifest
	if err := json.NewDecoder(rc).Decode(&m); err != nil {
		return nil, fmt.Errorf("manifest decode: %w", err)
	}
	return &m, nil
}
