package incremental

import (
	"bytes"
	"encoding/hex"
	"io"
	"testing"
)

func TestReadChunks_ClampsAndCoversAllData(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 10*1024*1024) // 10 MB
	var total int64
	var chunks int

	err := ReadChunks(bytes.NewReader(data), 1, func(chunk []byte) error {
		chunks++
		total += int64(len(chunk))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != int64(len(data)) {
		t.Errorf("total = %d, want %d", total, len(data))
	}
	if chunks < 1 {
		t.Errorf("expected at least 1 chunk, got %d", chunks)
	}
}

func TestHashChunkHex_ConsistentWithBlake3(t *testing.T) {
	data := []byte("hello blake3")
	h1 := HashChunkHex(data)
	h2 := HashChunkHex(data)
	if h1 != h2 {
		t.Fatalf("hashes differ: %s vs %s", h1, h2)
	}
	if len(h1) != hex.EncodedLen(32) {
		t.Errorf("hash len = %d, want %d", len(h1), hex.EncodedLen(32))
	}
}

func TestObjectKeyPrefix(t *testing.T) {
	hash := "abcd1234"
	if got := ObjectKeyPrefix(hash, 2); got != "ab" {
		t.Errorf("ObjectKeyPrefix(%q,2) = %q, want ab", hash, got)
	}
	if got := ObjectKeyPrefix(hash, 10); got != hash {
		t.Errorf("ObjectKeyPrefix(%q,10) = %q, want full hash", hash, got)
	}
	if got := ObjectKeyPrefix(hash, 0); got != hash {
		t.Errorf("ObjectKeyPrefix(%q,0) = %q, want full hash", hash, got)
	}
}

func TestReadChunks_StopsOnError(t *testing.T) {
	r := bytes.NewReader([]byte("some data"))
	calls := 0
	err := ReadChunks(r, ChunkSizeMin, func(chunk []byte) error {
		calls++
		return io.ErrUnexpectedEOF
	})
	if err != io.ErrUnexpectedEOF {
		t.Fatalf("err=%v, want io.ErrUnexpectedEOF", err)
	}
	if calls != 1 {
		t.Errorf("calls=%d, want 1", calls)
	}
}
