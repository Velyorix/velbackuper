package collector

import (
	"bytes"
	"testing"
)

func TestReadChunks_RespectsSizeBounds(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 10*1024*1024) // 10 MB
	var chunks int
	var total int64
	err := ReadChunks(bytes.NewReader(data), ChunkSizeMin, func(chunk []byte) error {
		chunks++
		total += int64(len(chunk))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if chunks != 3 {
		t.Errorf("expected 3 chunks (4MB+4MB+2MB), got %d", chunks)
	}
	if total != int64(len(data)) {
		t.Errorf("total bytes = %d, want %d", total, len(data))
	}
}

func TestReadChunks_LastChunkSmaller(t *testing.T) {
	// 3 chunks (4MB, 4MB, 2MB)
	data := bytes.Repeat([]byte("x"), 10*1024*1024)
	var sizes []int
	err := ReadChunks(bytes.NewReader(data), ChunkSizeMin, func(chunk []byte) error {
		sizes = append(sizes, len(chunk))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sizes) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(sizes))
	}
	if sizes[0] != ChunkSizeMin || sizes[1] != ChunkSizeMin || sizes[2] != 2*1024*1024 {
		t.Errorf("chunk sizes = %v, want [4MB, 4MB, 2MB]", sizes)
	}
}

func TestReadChunks_ClampsToMinMax(t *testing.T) {
	data := []byte("short")
	var got int
	err := ReadChunks(bytes.NewReader(data), 1, func(chunk []byte) error {
		got += len(chunk)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != 5 {
		t.Errorf("got %d bytes, want 5", got)
	}
}

func TestObjectKeyPrefix(t *testing.T) {
	hash := "abcd1234ef567890"
	if got := ObjectKeyPrefix(hash, 2); got != "ab" {
		t.Errorf("ObjectKeyPrefix(%q, 2) = %q, want ab", hash, got)
	}
	if got := ObjectKeyPrefix(hash, 4); got != "abcd" {
		t.Errorf("ObjectKeyPrefix(%q, 4) = %q, want abcd", hash, got)
	}
	if got := ObjectKeyPrefix("xy", 5); got != "xy" {
		t.Errorf("ObjectKeyPrefix(short, 5) = %q, want xy", got)
	}
	if got := ObjectKeyPrefix(hash, 0); got != hash {
		t.Errorf("ObjectKeyPrefix(%q, 0) should return full hash", hash)
	}
}

func TestChunkSizeConstants(t *testing.T) {
	if ChunkSizeMin != 4*1024*1024 {
		t.Errorf("ChunkSizeMin = %d, want 4MB", ChunkSizeMin)
	}
	if ChunkSizeMax != 16*1024*1024 {
		t.Errorf("ChunkSizeMax = %d, want 16MB", ChunkSizeMax)
	}
}
