package incremental

import (
	"context"
	"encoding/json"
	"io"
	"path"
	"strings"
	"time"

	"VelBackuper/internal/config"
	"VelBackuper/internal/s3"
)

type GCResult struct {
	DeletedSnapshots int
	DeletedIndexes   int
	DeletedObjects   int
}

// gcStorage is the subset of S3 client methods used by Prune.
type gcStorage interface {
	ListObjects(ctx context.Context, prefix string, maxKeys int32) ([]string, error)
	DeleteObject(ctx context.Context, key string) error
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
}

// Prune deletes expired snapshots and their indexes for a job according to retention,
// then performs a mark-and-sweep GC over objects used by retained snapshots of this job.
// Any client that satisfies gcStorage (including *s3.Client) can be used.
func Prune(ctx context.Context, client gcStorage, job string, retention *config.RetentionConfig, now time.Time, hashPrefixLen int) (GCResult, error) {
	var result GCResult

	snapPrefix := s3.SnapshotsPrefixForJob(job)
	snapshotKeys, err := client.ListObjects(ctx, snapPrefix, 0)
	if err != nil {
		return result, err
	}

	liveHashes := make(map[string]struct{})

	for _, snapKey := range snapshotKeys {
		ts, ok := snapshotTimeFromKey(snapKey)
		if !ok {
			continue
		}
		expired := config.IsExpired(ts, now, retention)

		snap, err := readSnapshotForGC(ctx, client, snapKey)
		if err != nil {
			return result, err
		}

		if expired {
			if err := client.DeleteObject(ctx, snapKey); err != nil {
				return result, err
			}
			result.DeletedSnapshots++
			if snap.IndexKey != "" {
				if err := client.DeleteObject(ctx, snap.IndexKey); err != nil {
					return result, err
				}
				result.DeletedIndexes++
			}
			continue
		}

		if snap.IndexKey == "" {
			continue
		}
		idx, err := readIndexForGC(ctx, client, snap.IndexKey)
		if err != nil {
			return result, err
		}
		for _, ch := range idx.Chunks {
			if ch.Hash == "" {
				continue
			}
			liveHashes[ch.Hash] = struct{}{}
		}
	}

	objectsPrefix := s3.ObjectsPrefix
	objectKeys, err := client.ListObjects(ctx, objectsPrefix, 0)
	if err != nil {
		return result, err
	}

	for _, key := range objectKeys {
		hash := hashFromObjectKey(key)
		if hash == "" {
			continue
		}
		if _, ok := liveHashes[hash]; ok {
			continue
		}
		if err := client.DeleteObject(ctx, key); err != nil {
			return result, err
		}
		result.DeletedObjects++
	}

	return result, nil
}

func snapshotTimeFromKey(key string) (time.Time, bool) {
	base := path.Base(key)
	if base == "." || base == "/" {
		return time.Time{}, false
	}
	tsStr := strings.TrimSuffix(base, ".json")
	if len(tsStr) != len(timestampLayout) {
		return time.Time{}, false
	}
	t, err := time.Parse(timestampLayout, tsStr)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

func hashFromObjectKey(key string) string {
	key = strings.Trim(key, "/")
	parts := strings.Split(key, "/")
	if len(parts) != 3 || parts[0] != s3.ObjectsPrefix {
		return ""
	}
	return parts[2]
}

func readSnapshotForGC(ctx context.Context, client gcStorage, key string) (*Snapshot, error) {
	rc, err := client.GetObject(ctx, key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var s Snapshot
	if err := json.NewDecoder(rc).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func readIndexForGC(ctx context.Context, client gcStorage, key string) (*Index, error) {
	rc, err := client.GetObject(ctx, key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var idx Index
	if err := json.NewDecoder(rc).Decode(&idx); err != nil {
		return nil, err
	}
	return &idx, nil
}
