package incremental

import (
	"context"
	"io"
	"time"

	"VelBackuper/internal/lock"
	"VelBackuper/internal/notifier"
	"VelBackuper/internal/s3"
)

const timestampLayout = "20060102150405"

type RunOptions struct {
	ChunkSize     int64
	Concurrency   int
	HashPrefixLen int
	Notifier      notifier.Notifier
	StrictNotify  bool
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

func RunWithS3Lock(ctx context.Context, client *s3.Client, job string, r io.Reader, opts RunOptions, lockTTL time.Duration) (backupID string, idx *Index, snap *Snapshot, err error) {
	locker, err := lock.NewS3(lock.S3Options{
		Client: client,
		Name:   job,
		TTL:    lockTTL,
	})
	if err != nil {
		return "", nil, nil, err
	}
	if err := locker.Acquire(ctx); err != nil {
		return "", nil, nil, err
	}
	defer func() {
		_ = locker.Release(context.Background())
	}()

	if opts.Notifier != nil {
		if nErr := opts.Notifier.NotifyStart(ctx, job, ""); nErr != nil && opts.StrictNotify {
			return "", nil, nil, nErr
		}
	}

	start := time.Now()
	backupID, idx, snap, err = Run(ctx, client, job, r, opts)
	duration := time.Since(start)

	if opts.Notifier != nil {
		if err != nil {
			nErr := opts.Notifier.NotifyError(ctx, job, backupID, err)
			if nErr != nil && opts.StrictNotify && backupID == "" {
				return backupID, idx, snap, nErr
			}
			return backupID, idx, snap, err
		}

		nErr := opts.Notifier.NotifySuccess(ctx, job, backupID, duration, 0)
		if nErr != nil && opts.StrictNotify {
			return backupID, idx, snap, nErr
		}
	}

	return backupID, idx, snap, err
}
