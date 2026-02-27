package incremental

import (
	"context"
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

func Prune(ctx context.Context, client *s3.Client, job string, retention *config.RetentionConfig, now time.Time, hashPrefixLen int) (GCResult, error) {
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

		snap, err := ReadSnapshotByKey(ctx, client, snapKey)
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
		idx, err := ReadIndexByKey(ctx, client, snap.IndexKey)
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
