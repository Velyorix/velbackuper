package incremental

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"VelBackuper/internal/config"
	"VelBackuper/internal/s3"
)

type fakeS3 struct {
	objects map[string][]byte
}

func newFakeS3() *fakeS3 {
	return &fakeS3{objects: make(map[string][]byte)}
}

func (f *fakeS3) Put(key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f.objects[key] = b
	return nil
}

func (f *fakeS3) Client() *s3.Client {
	// not used in tests; GC works directly with fake via wrapper methods
	return nil
}

func TestPrune_RemovesExpiredSnapshotsAndOrphans(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)
	ret := &config.RetentionConfig{Days: 30}

	mem := newFakeS3()

	job := "job1"
	oldTS := "20250101000000"
	newTS := "20250215000000"

	oldSnapKey := s3.SnapshotKey(job, oldTS)
	newSnapKey := s3.SnapshotKey(job, newTS)
	oldIdxKey := s3.IndexKey(job, oldTS)
	newIdxKey := s3.IndexKey(job, newTS)

	if err := mem.Put(oldSnapKey, Snapshot{Job: job, Timestamp: oldTS, IndexKey: oldIdxKey}); err != nil {
		t.Fatal(err)
	}
	if err := mem.Put(oldIdxKey, Index{
		Job:       job,
		Timestamp: oldTS,
		Chunks: []IndexChunk{
			{Hash: "aaaa", Size: 1},
		},
	}); err != nil {
		t.Fatal(err)
	}

	if err := mem.Put(newSnapKey, Snapshot{Job: job, Timestamp: newTS, IndexKey: newIdxKey}); err != nil {
		t.Fatal(err)
	}
	if err := mem.Put(newIdxKey, Index{
		Job:       job,
		Timestamp: newTS,
		Chunks: []IndexChunk{
			{Hash: "bbbb", Size: 1},
		},
	}); err != nil {
		t.Fatal(err)
	}

	oldObjKey := s3.ObjectKey("aa", "aaaa")
	newObjKey := s3.ObjectKey("bb", "bbbb")
	mem.objects[oldObjKey] = []byte("old")
	mem.objects[newObjKey] = []byte("new")

	wrap := &gcTestClient{mem: mem}

	res, err := Prune(ctx, wrap, job, ret, now, DefaultHashPrefixLen)
	if err != nil {
		t.Fatal(err)
	}
	if res.DeletedSnapshots != 1 {
		t.Errorf("DeletedSnapshots=%d, want 1", res.DeletedSnapshots)
	}
	if res.DeletedIndexes != 1 {
		t.Errorf("DeletedIndexes=%d, want 1", res.DeletedIndexes)
	}
	if res.DeletedObjects != 1 {
		t.Errorf("DeletedObjects=%d, want 1", res.DeletedObjects)
	}

	if _, ok := mem.objects[oldSnapKey]; ok {
		t.Error("old snapshot should be deleted")
	}
	if _, ok := mem.objects[oldIdxKey]; ok {
		t.Error("old index should be deleted")
	}
	if _, ok := mem.objects[oldObjKey]; ok {
		t.Error("unreferenced object should be deleted")
	}
	if _, ok := mem.objects[newObjKey]; !ok {
		t.Error("live object should be retained")
	}
}

type gcTestClient struct {
	mem *fakeS3
}

func (c *gcTestClient) ListObjects(_ context.Context, prefix string, _ int32) ([]string, error) {
	var keys []string
	for k := range c.mem.objects {
		if len(prefix) == 0 || strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (c *gcTestClient) DeleteObject(_ context.Context, key string) error {
	delete(c.mem.objects, key)
	return nil
}

func (c *gcTestClient) GetObject(_ context.Context, key string) (io.ReadCloser, error) {
	b, ok := c.mem.objects[key]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}
