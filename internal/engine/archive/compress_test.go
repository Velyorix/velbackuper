package archive

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestNewCompressReader_Tar_Identity(t *testing.T) {
	input := []byte("hello tar")
	r, err := NewCompressReader(bytes.NewReader(input), FormatTar, 0)
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, input) {
		t.Errorf("FormatTar should pass through: got %q", out)
	}
}

func TestNewCompressReader_Gzip_Roundtrip(t *testing.T) {
	input := []byte("hello gzip world")
	r, err := NewCompressReader(bytes.NewReader(input), FormatGzip, 6)
	if err != nil {
		t.Fatal(err)
	}
	compressed, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(compressed) == 0 {
		t.Fatal("compressed output empty")
	}
	gr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	decompressed, err := io.ReadAll(gr)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decompressed, input) {
		t.Errorf("gzip roundtrip: got %q, want %q", decompressed, input)
	}
}

func TestNewCompressReader_Zstd_Roundtrip(t *testing.T) {
	input := []byte("hello zstd world")
	r, err := NewCompressReader(bytes.NewReader(input), FormatZstd, 0)
	if err != nil {
		t.Fatal(err)
	}
	compressed, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(compressed) == 0 {
		t.Fatal("compressed output empty")
	}
	zr, err := zstd.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	decompressed, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decompressed, input) {
		t.Errorf("zstd roundtrip: got %q, want %q", decompressed, input)
	}
}

func TestNewCompressReader_Unknown_Identity(t *testing.T) {
	input := []byte("unknown format")
	r, err := NewCompressReader(bytes.NewReader(input), CompressionFormat("unknown"), 0)
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, input) {
		t.Errorf("unknown format should pass through: got %q", out)
	}
}
