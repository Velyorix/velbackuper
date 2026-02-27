package archive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"VelBackuper/internal/config"
	"VelBackuper/internal/s3"
)

func TestTimestampStringFromManifestKey(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		ts, ok := timestampStringFromManifestKey("manifests/myjob/20250226120000.json", "myjob")
		if !ok || ts != "20250226120000" {
			t.Errorf("timestampStringFromManifestKey = %q, %v; want 20250226120000, true", ts, ok)
		}
	})
	t.Run("wrong job in path still extracts timestamp", func(t *testing.T) {
		ts, ok := timestampStringFromManifestKey("manifests/myjob/20250101120000.json", "other")
		// Function only checks basename and suffix; job is not validated
		if !ok || ts != "20250101120000" {
			t.Errorf("timestampStringFromManifestKey = %q, %v", ts, ok)
		}
	})
	t.Run("invalid no json suffix", func(t *testing.T) {
		_, ok := timestampStringFromManifestKey("manifests/job/20250226120000.txt", "job")
		if ok {
			t.Error("expected false for .txt")
		}
	})
	t.Run("invalid short timestamp", func(t *testing.T) {
		_, ok := timestampStringFromManifestKey("manifests/job/2025022612000.json", "job")
		if ok {
			t.Error("expected false for 13-char timestamp")
		}
	})
	t.Run("invalid long timestamp", func(t *testing.T) {
		_, ok := timestampStringFromManifestKey("manifests/job/202502261200001.json", "job")
		if ok {
			t.Error("expected false for 15-char timestamp")
		}
	})
}

func TestParseTimestampFromManifestKey(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		ts, ok := parseTimestampFromManifestKey("manifests/j1/20250226120000.json", "j1")
		if !ok {
			t.Fatal("expected true")
		}
		want := time.Date(2025, 2, 26, 12, 0, 0, 0, time.UTC)
		if !ts.Equal(want) {
			t.Errorf("parsed time = %v, want %v", ts, want)
		}
	})
	t.Run("invalid returns zero", func(t *testing.T) {
		ts, ok := parseTimestampFromManifestKey("manifests/job/invalid.json", "job")
		if ok {
			t.Error("expected false for invalid timestamp")
		}
		if !ts.IsZero() {
			t.Errorf("expected zero time, got %v", ts)
		}
	})
}

func TestApplyRetention_NilOrZero_NoCalls(t *testing.T) {
	ctx := context.Background()
	fake := &fakeStorage{listErr: errors.New("should not be called")}

	t.Run("nil retention", func(t *testing.T) {
		n, err := ApplyRetention(ctx, fake, "job", nil, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("deleted = %d, want 0", n)
		}
	})

	t.Run("zero retention", func(t *testing.T) {
		n, err := ApplyRetention(ctx, fake, "job", &config.RetentionConfig{Days: 0, Weeks: 0, Months: 0}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("deleted = %d, want 0", n)
		}
	})
}

func TestApplyRetention_DeletesExpired_UpdatesLatest(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)
	retention := &config.RetentionConfig{Days: 30, Weeks: 0, Months: 0}
	// Cutoff = 2025-01-30 12:00. So 20250201000000 is kept, 20250101000000 is expired.

	oldManifest := Manifest{Job: "job1", Timestamp: "20250101000000", Key: "archives/job1/2025/01/01/backup-h-20250101000000.tar.gz"}
	newManifest := Manifest{Job: "job1", Timestamp: "20250201000000", Key: "archives/job1/2025/02/01/backup-h-20250201000000.tar.gz"}
	oldBody, _ := json.Marshal(oldManifest)
	newBody, _ := json.Marshal(newManifest)

	fake := &fakeStorage{
		objects: map[string][]byte{
			s3.ManifestKey("job1", "20250101000000"): oldBody,
			s3.ManifestKey("job1", "20250201000000"): newBody,
		},
		lists: map[string][]string{
			"manifests/job1/": {
				"manifests/job1/20250101000000.json",
				"manifests/job1/20250201000000.json",
			},
		},
	}
	latestBody, _ := json.Marshal(LatestPointer{Timestamp: "20250201000000", Key: "archives/job1/2025/02/01/backup-h-20250201000000.tar.gz"})
	fake.objects[s3.LatestKey("job1")] = latestBody

	deleted, err := ApplyRetention(ctx, fake, "job1", retention, now)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}
	if _, ok := fake.objects[s3.ManifestKey("job1", "20250101000000")]; ok {
		t.Error("old manifest should be deleted")
	}
	if _, ok := fake.objects["archives/job1/2025/01/01/backup-h-20250101000000.tar.gz"]; ok {
		t.Error("old archive should be deleted")
	}
	if _, ok := fake.objects[s3.ManifestKey("job1", "20250201000000")]; !ok {
		t.Error("new manifest should be kept")
	}
	// Latest was pointing to the new backup, so it should still be there (no update needed).
	if _, ok := fake.objects[s3.LatestKey("job1")]; !ok {
		t.Error("latest pointer should still exist")
	}
}

func TestApplyRetention_UpdatesLatestWhenCurrentDeleted(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)
	retention := &config.RetentionConfig{Days: 30, Weeks: 0, Months: 0}

	oldManifest := Manifest{Job: "j", Timestamp: "20250101000000", Key: "archives/j/2025/01/01/old.tar.gz"}
	keptManifest := Manifest{Job: "j", Timestamp: "20250215000000", Key: "archives/j/2025/02/15/kept.tar.gz"}
	oldBody, _ := json.Marshal(oldManifest)
	keptBody, _ := json.Marshal(keptManifest)

	fake := &fakeStorage{
		objects: map[string][]byte{
			s3.ManifestKey("j", "20250101000000"): oldBody,
			s3.ManifestKey("j", "20250215000000"): keptBody,
		},
		lists: map[string][]string{
			"manifests/j/": {
				"manifests/j/20250101000000.json",
				"manifests/j/20250215000000.json",
			},
		},
	}
	// Latest points to the old (expired) backup - will be deleted
	latestBody, _ := json.Marshal(LatestPointer{Timestamp: "20250101000000", Key: "archives/j/2025/01/01/old.tar.gz"})
	fake.objects[s3.LatestKey("j")] = latestBody

	deleted, err := ApplyRetention(ctx, fake, "j", retention, now)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}
	// Latest should be updated to point to 20250215000000
	latestJSON, ok := fake.objects[s3.LatestKey("j")]
	if !ok {
		t.Fatal("latest pointer should exist (updated)")
	}
	var p LatestPointer
	if err := json.Unmarshal(latestJSON, &p); err != nil {
		t.Fatal(err)
	}
	if p.Timestamp != "20250215000000" || p.Key != "archives/j/2025/02/15/kept.tar.gz" {
		t.Errorf("latest = %+v, want timestamp 20250215000000 and key archives/j/2025/02/15/kept.tar.gz", p)
	}
}

func TestApplyRetention_DeletesLatestWhenAllExpired(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)
	retention := &config.RetentionConfig{Days: 7, Weeks: 0, Months: 0}

	oldManifest := Manifest{Job: "j", Timestamp: "20250101000000", Key: "archives/j/2025/01/01/old.tar.gz"}
	oldBody, _ := json.Marshal(oldManifest)
	fake := &fakeStorage{
		objects: map[string][]byte{
			s3.ManifestKey("j", "20250101000000"): oldBody,
		},
		lists: map[string][]string{
			"manifests/j/": {"manifests/j/20250101000000.json"},
		},
	}
	latestBody, _ := json.Marshal(LatestPointer{Timestamp: "20250101000000", Key: "archives/j/2025/01/01/old.tar.gz"})
	fake.objects[s3.LatestKey("j")] = latestBody

	deleted, err := ApplyRetention(ctx, fake, "j", retention, now)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}
	if _, ok := fake.objects[s3.LatestKey("j")]; ok {
		t.Error("latest pointer should be deleted when no backups remain")
	}
}

// fakeStorage implements Storage for tests.
type fakeStorage struct {
	objects map[string][]byte
	lists   map[string][]string
	listErr error
}

func (f *fakeStorage) ListObjects(_ context.Context, prefix string, _ int32) ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	keys := f.lists[prefix]
	// Return only keys that still exist (so after DeleteObject, list reflects deletions)
	var out []string
	for _, k := range keys {
		if _, ok := f.objects[k]; ok {
			out = append(out, k)
		}
	}
	return out, nil
}

func (f *fakeStorage) GetObject(_ context.Context, key string) (io.ReadCloser, error) {
	b, ok := f.objects[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (f *fakeStorage) DeleteObject(_ context.Context, key string) error {
	delete(f.objects, key)
	return nil
}

func (f *fakeStorage) PutObject(_ context.Context, key string, body io.Reader, _ int64) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if f.objects == nil {
		f.objects = make(map[string][]byte)
	}
	f.objects[key] = b
	return nil
}
