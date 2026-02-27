package archive

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"testing"

	"VelBackuper/internal/collector"
)

func TestCollectToStream_ForwardsCollectorOutput(t *testing.T) {
	want := []byte("collector output")
	c := &streamTestCollector{data: want}
	r, err := CollectToStream(context.Background(), c, "job1")
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStream_Tar_Passthrough(t *testing.T) {
	data := []byte("raw tar data")
	c := &streamTestCollector{data: data}
	r, err := Stream(context.Background(), c, "job", FormatTar, 0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("got %q, want %q", got, data)
	}
}

func TestStream_Gzip_Compresses(t *testing.T) {
	data := []byte("hello stream gzip")
	c := &streamTestCollector{data: data}
	r, err := Stream(context.Background(), c, "job", FormatGzip, 6)
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
	got, err := io.ReadAll(gr)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("roundtrip: got %q, want %q", got, data)
	}
}

type streamTestCollector struct {
	data []byte
}

func (s *streamTestCollector) Collect(_ context.Context, _ string, w io.Writer) error {
	_, err := w.Write(s.data)
	return err
}

func init() {
	var _ collector.Collector = (*streamTestCollector)(nil)
}
