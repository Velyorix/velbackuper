package restore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"VelBackuper/internal/engine/incremental"
	"VelBackuper/internal/s3"
)

type IncrementalRestoreOptions struct {
	DryRun       bool
	VerifyChunks bool
}

func RestoreIncremental(ctx context.Context, client *s3.Client, job, timestamp, targetDir string, opts IncrementalRestoreOptions) error {
	if targetDir == "" {
		return fmt.Errorf("targetDir is required")
	}

	snap, err := incremental.ReadSnapshot(ctx, client, job, timestamp)
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}
	if snap.IndexKey == "" {
		return fmt.Errorf("snapshot has no index_key")
	}

	idx, err := incremental.ReadIndexByKey(ctx, client, snap.IndexKey)
	if err != nil {
		return fmt.Errorf("read index: %w", err)
	}

	chunkData := make(map[string][]byte, len(idx.Chunks))
	for _, ch := range idx.Chunks {
		if ch.Hash == "" {
			continue
		}
		if _, ok := chunkData[ch.Hash]; ok {
			continue
		}
		prefix := incremental.ObjectKeyPrefix(ch.Hash, incremental.DefaultHashPrefixLen)
		key := s3.ObjectKey(prefix, ch.Hash)
		rc, err := client.GetObject(ctx, key)
		if err != nil {
			return fmt.Errorf("get chunk %s: %w", key, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("read chunk %s: %w", key, err)
		}
		if opts.VerifyChunks {
			if got := incremental.HashChunkHex(data); got != ch.Hash {
				return fmt.Errorf("chunk hash mismatch for %s: got %s, want %s", key, got, ch.Hash)
			}
		}
		chunkData[ch.Hash] = data
	}

	for _, fe := range snap.Files {
		rel := cleanRelativePath(fe.Path)
		if rel == "" {
			continue
		}
		fullPath := filepath.Join(targetDir, rel)

		if opts.DryRun {
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(fe.Mode))
		if err != nil {
			return err
		}

		for _, fc := range fe.Chunks {
			data, ok := chunkData[fc.Hash]
			if !ok {
				_ = f.Close()
				return fmt.Errorf("missing chunk data for hash %s", fc.Hash)
			}
			if fc.Offset < 0 || fc.Length < 0 || int(fc.Offset+fc.Length) > len(data) {
				_ = f.Close()
				return fmt.Errorf("invalid chunk range for hash %s", fc.Hash)
			}
			if fc.Length == 0 {
				continue
			}
			if _, err := f.Write(data[fc.Offset : fc.Offset+fc.Length]); err != nil {
				_ = f.Close()
				return err
			}
		}

		if err := f.Close(); err != nil {
			return err
		}
	}

	return nil
}

func cleanRelativePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = filepath.ToSlash(p)
	p = strings.TrimLeft(p, "/")
	if p == "" || strings.HasPrefix(p, "..") {
		return ""
	}
	p = filepath.Clean(p)
	if p == "." || strings.HasPrefix(p, "..") {
		return ""
	}
	return p
}
