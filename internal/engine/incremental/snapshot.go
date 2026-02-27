package incremental

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"VelBackuper/internal/s3"
)

type FileChunk struct {
	Hash   string `json:"hash"`
	Offset int64  `json:"offset"`
	Length int64  `json:"length"`
}

type FileEntry struct {
	Path    string      `json:"path"`
	Mode    uint32      `json:"mode"`
	Size    int64       `json:"size"`
	ModTime time.Time   `json:"mod_time"`
	Chunks  []FileChunk `json:"chunks"`
}

type Snapshot struct {
	Job       string      `json:"job"`
	Timestamp string      `json:"timestamp"`
	IndexKey  string      `json:"index_key"`
	Files     []FileEntry `json:"files"`
}

func WriteSnapshot(ctx context.Context, client *s3.Client, s Snapshot) error {
	key := s3.SnapshotKey(s.Job, s.Timestamp)
	body, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("snapshot marshal: %w", err)
	}
	return client.PutObject(ctx, key, bytes.NewReader(body), int64(len(body)))
}

func ReadSnapshot(ctx context.Context, client *s3.Client, job, timestamp string) (*Snapshot, error) {
	key := s3.SnapshotKey(job, timestamp)
	return ReadSnapshotByKey(ctx, client, key)
}

func ReadSnapshotByKey(ctx context.Context, client *s3.Client, key string) (*Snapshot, error) {
	rc, err := client.GetObject(ctx, key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var s Snapshot
	if err := json.NewDecoder(rc).Decode(&s); err != nil {
		return nil, fmt.Errorf("snapshot decode: %w", err)
	}
	return &s, nil
}
