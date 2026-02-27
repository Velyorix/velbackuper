package incremental

import (
	"bytes"
	"context"
	"io"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestObjectKeyForHash_DefaultPrefix(t *testing.T) {
	hash := "abcd1234ef567890"
	got := objectKeyForHash(hash, 0)
	want := "objects/ab/abcd1234ef567890"
	if got != want {
		t.Errorf("objectKeyForHash = %q, want %q", got, want)
	}
}

func TestUploadChunk_SkipsExisting(t *testing.T) {
	ctx := context.Background()
	s := newFakeStorage()

	hash := "aaaaaaaa"
	key := objectKeyForHash(hash, DefaultHashPrefixLen)
	s.setObject(key, []byte("exists"))

	uploaded, err := UploadChunk(ctx, s, hash, []byte("newdata"), DefaultHashPrefixLen)
	if err != nil {
		t.Fatal(err)
	}
	if uploaded {
		t.Error("expected uploaded=false for existing object")
	}
}

func TestUploadChunk_UploadsMissing(t *testing.T) {
	ctx := context.Background()
	s := newFakeStorage()

	hash := "bbbbbbbb"
	uploaded, err := UploadChunk(ctx, s, hash, []byte("data"), DefaultHashPrefixLen)
	if err != nil {
		t.Fatal(err)
	}
	if !uploaded {
		t.Error("expected uploaded=true")
	}

	key := objectKeyForHash(hash, DefaultHashPrefixLen)
	if got := string(s.objects[key]); got != "data" {
		t.Errorf("stored = %q, want %q", got, "data")
	}
}

func TestUploadChunks_DedupWithinCall(t *testing.T) {
	ctx := context.Background()
	s := newFakeStorage()

	chunks := []ChunkObject{
		{Hash: "cccc", Data: []byte("1")},
		{Hash: "cccc", Data: []byte("2")},
		{Hash: "dddd", Data: []byte("3")},
	}
	res, err := UploadChunks(ctx, s, chunks, UploadOptions{Concurrency: 2})
	if err != nil {
		t.Fatal(err)
	}
	if res.Uploaded != 2 || res.Skipped != 0 {
		t.Errorf("result = %+v, want Uploaded=2 Skipped=0", res)
	}
}

type fakeStorage struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{objects: make(map[string][]byte)}
}

func (f *fakeStorage) setObject(key string, data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = append([]byte(nil), data...)
}

func (f *fakeStorage) HeadObject(_ context.Context, key string) (*time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.objects[key]; ok {
		tm := time.Unix(0, 0).UTC()
		return &tm, nil
	}
	return nil, nil
}

func (f *fakeStorage) PutObject(_ context.Context, key string, body io.Reader, _ int64) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = b
	return nil
}

func TestUploadChunks_SkippedCountsExisting(t *testing.T) {
	ctx := context.Background()
	s := newFakeStorage()

	hash1 := "eeee"
	hash2 := "ffff"
	s.setObject(objectKeyForHash(hash1, 2), []byte("exists"))

	res, err := UploadChunks(ctx, s, []ChunkObject{
		{Hash: hash1, Data: []byte("x")},
		{Hash: hash2, Data: []byte("y")},
	}, UploadOptions{Concurrency: 4, HashPrefixLen: 2})
	if err != nil {
		t.Fatal(err)
	}
	if res.Uploaded != 1 || res.Skipped != 1 {
		t.Errorf("result = %+v, want Uploaded=1 Skipped=1", res)
	}
}

func TestUploadChunks_StoresUnderPrefix(t *testing.T) {
	ctx := context.Background()
	s := newFakeStorage()

	res, err := UploadChunks(ctx, s, []ChunkObject{
		{Hash: "abcd1234", Data: []byte("z")},
	}, UploadOptions{Concurrency: 1, HashPrefixLen: 2})
	if err != nil {
		t.Fatal(err)
	}
	if res.Uploaded != 1 {
		t.Fatalf("uploaded=%d, want 1", res.Uploaded)
	}

	s.mu.Lock()
	var keys []string
	for k := range s.objects {
		keys = append(keys, k)
	}
	s.mu.Unlock()
	sort.Strings(keys)

	if len(keys) != 1 {
		t.Fatalf("keys=%v, want single key", keys)
	}
	if keys[0] != "objects/ab/abcd1234" {
		t.Errorf("key=%q, want objects/ab/abcd1234", keys[0])
	}
	if !bytes.Equal(s.objects[keys[0]], []byte("z")) {
		t.Errorf("value=%q, want z", s.objects[keys[0]])
	}
}
