package incremental

import (
	"context"
	"io"
	"time"

	"VelBackuper/internal/s3"
)

const timestampLayout = "20060102150405"

type RunOptions struct {
	ChunkSize     int64
	Concurrency   int
	HashPrefixLen int
}

func Run(ctx context.Context, store Storage, job string, r io.Reader, opts RunOptions) (backupID string, idx *Index, snap *Snapshot, err error) {
	now := time.Now().UTC()
	timestamp := now.Format(timestampLayout)

	chunkSize := opts.ChunkSize
	if chunkSize <= 0 {
		chunkSize = ChunkSizeMin
	}

	var chunks []ChunkObject
	var indexChunks []IndexChunk

	err = ReadChunks(r, chunkSize, func(chunk []byte) error {
		if len(chunk) == 0 {
			return nil
		}
		hash := HashChunkHex(chunk)
		chunks = append(chunks, ChunkObject{
			Hash: hash,
			Data: append([]byte(nil), chunk...),
		})
		indexChunks = append(indexChunks, IndexChunk{
			Hash: hash,
			Size: int64(len(chunk)),
		})
		return nil
	})
	if err != nil {
		return "", nil, nil, err
	}

	_, err = UploadChunks(ctx, store, chunks, UploadOptions{
		Concurrency:   opts.Concurrency,
		HashPrefixLen: opts.HashPrefixLen,
	})
	if err != nil {
		return "", nil, nil, err
	}

	index := &Index{
		Job:       job,
		Timestamp: timestamp,
		Chunks:    indexChunks,
	}
	if err := WriteIndex(ctx, store.(*s3.Client), *index); err != nil {
		return "", nil, nil, err
	}

	snapshot := &Snapshot{
		Job:       job,
		Timestamp: timestamp,
		IndexKey:  s3.IndexKey(job, timestamp),
		Files:     nil,
	}
	if err := WriteSnapshot(ctx, store.(*s3.Client), *snapshot); err != nil {
		return "", nil, nil, err
	}

	return timestamp, index, snapshot, nil
}
