package incremental

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"VelBackuper/internal/s3"
)

type IndexChunk struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

type Index struct {
	Job       string       `json:"job"`
	Timestamp string       `json:"timestamp"`
	Chunks    []IndexChunk `json:"chunks"`
}

func WriteIndex(ctx context.Context, client *s3.Client, idx Index) error {
	key := s3.IndexKey(idx.Job, idx.Timestamp)
	body, err := json.Marshal(idx)
	if err != nil {
		return fmt.Errorf("index marshal: %w", err)
	}
	return client.PutObject(ctx, key, bytes.NewReader(body), int64(len(body)))
}

func ReadIndex(ctx context.Context, client *s3.Client, job, timestamp string) (*Index, error) {
	key := s3.IndexKey(job, timestamp)
	return ReadIndexByKey(ctx, client, key)
}

func ReadIndexByKey(ctx context.Context, client *s3.Client, key string) (*Index, error) {
	rc, err := client.GetObject(ctx, key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var idx Index
	if err := json.NewDecoder(rc).Decode(&idx); err != nil {
		return nil, fmt.Errorf("index decode: %w", err)
	}
	return &idx, nil
}
