package archive

import (
	"context"
	"path"
	"strings"
	"time"

	"VelBackuper/internal/config"
	"VelBackuper/internal/s3"
)

const timestampLayout = "20060102150405"

func ApplyRetention(ctx context.Context, client Storage, job string, retention *config.RetentionConfig, now time.Time) (deleted int, err error) {
	if retention == nil || config.RetainUntil(now, retention).IsZero() {
		return 0, nil
	}

	manifestPrefix := path.Join(s3.ManifestsPrefix, job) + "/"
	manifestKeys, err := client.ListObjects(ctx, manifestPrefix, 0)
	if err != nil {
		return 0, err
	}

	deletedKeys := make(map[string]struct{})
	for _, manifestKey := range manifestKeys {
		ts, ok := parseTimestampFromManifestKey(manifestKey, job)
		if !ok {
			continue
		}
		if !config.IsExpired(ts, now, retention) {
			continue
		}
		m, err := ReadManifestByKey(ctx, client, manifestKey)
		if err != nil {
			return deleted, err
		}
		if m.Key != "" {
			if err := client.DeleteObject(ctx, m.Key); err != nil {
				return deleted, err
			}
			deletedKeys[m.Key] = struct{}{}
		}
		if err := client.DeleteObject(ctx, manifestKey); err != nil {
			return deleted, err
		}
		deleted++
	}

	_, latestKey, err := ReadLatest(ctx, client, job)
	if err != nil || latestKey == "" {
		return deleted, nil
	}
	if _, removed := deletedKeys[latestKey]; !removed {
		return deleted, nil
	}

	manifestKeys, err = client.ListObjects(ctx, manifestPrefix, 0)
	if err != nil {
		return deleted, err
	}
	var newestTs string
	var newestKey string
	for _, manifestKey := range manifestKeys {
		tsStr, ok := timestampStringFromManifestKey(manifestKey, job)
		if !ok {
			continue
		}
		if newestTs == "" || tsStr > newestTs {
			newestTs = tsStr
			m, err := ReadManifestByKey(ctx, client, manifestKey)
			if err != nil {
				return deleted, err
			}
			newestKey = m.Key
		}
	}
	if newestTs != "" && newestKey != "" {
		return deleted, WriteLatest(ctx, client, job, newestTs, newestKey)
	}
	return deleted, client.DeleteObject(ctx, s3.LatestKey(job))
}

func parseTimestampFromManifestKey(manifestKey, job string) (time.Time, bool) {
	tsStr, ok := timestampStringFromManifestKey(manifestKey, job)
	if !ok {
		return time.Time{}, false
	}
	t, err := time.Parse(timestampLayout, tsStr)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

func timestampStringFromManifestKey(manifestKey, job string) (string, bool) {
	base := path.Base(manifestKey)
	if base == "." || base == "/" {
		return "", false
	}
	tsStr := strings.TrimSuffix(base, ".json")
	if len(tsStr) != 14 || !strings.HasSuffix(manifestKey, "/"+base) {
		return "", false
	}
	return tsStr, true
}
