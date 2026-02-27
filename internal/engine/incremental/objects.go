package incremental

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"VelBackuper/internal/s3"
)

type ChunkObject struct {
	Hash string
	Data []byte
}

type UploadOptions struct {
	Concurrency   int
	HashPrefixLen int
}

type UploadResult struct {
	Uploaded int
	Skipped  int
}

func objectKeyForHash(hash string, prefixLen int) string {
	if prefixLen <= 0 {
		prefixLen = DefaultHashPrefixLen
	}
	return s3.ObjectKey(ObjectKeyPrefix(hash, prefixLen), hash)
}

func UploadChunk(ctx context.Context, store Storage, hash string, data []byte, prefixLen int) (uploaded bool, err error) {
	key := objectKeyForHash(hash, prefixLen)
	existsAt, err := store.HeadObject(ctx, key)
	if err != nil {
		return false, fmt.Errorf("head object %s: %w", key, err)
	}
	if existsAt != nil {
		return false, nil
	}
	if err := store.PutObject(ctx, key, bytes.NewReader(data), int64(len(data))); err != nil {
		return false, fmt.Errorf("put object %s: %w", key, err)
	}
	return true, nil
}

func UploadChunks(ctx context.Context, store Storage, chunks []ChunkObject, opts UploadOptions) (UploadResult, error) {
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}
	prefixLen := opts.HashPrefixLen

	seen := make(map[string]struct{}, len(chunks))
	var unique []ChunkObject
	for _, c := range chunks {
		if c.Hash == "" {
			continue
		}
		if _, ok := seen[c.Hash]; ok {
			continue
		}
		seen[c.Hash] = struct{}{}
		unique = append(unique, c)
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	var mu sync.Mutex
	var res UploadResult
	var firstErr error

	for _, c := range unique {
		c := c
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				mu.Lock()
				if firstErr == nil {
					firstErr = ctx.Err()
				}
				mu.Unlock()
				return
			}
			defer func() { <-sem }()

			uploaded, err := UploadChunk(ctx, store, c.Hash, c.Data, prefixLen)
			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = err
				return
			}
			if err == nil {
				if uploaded {
					res.Uploaded++
				} else {
					res.Skipped++
				}
			}
		}()
	}
	wg.Wait()

	if firstErr != nil {
		return UploadResult{}, firstErr
	}
	return res, nil
}
