package archive

import (
	"compress/gzip"
	"io"

	"github.com/klauspost/compress/zstd"
)

type CompressionFormat string

const (
	FormatTar  CompressionFormat = "tar"
	FormatGzip CompressionFormat = "gz"
	FormatZstd CompressionFormat = "zst"
)

func NewCompressReader(r io.Reader, format CompressionFormat, level int) (io.Reader, error) {
	switch format {
	case FormatTar:
		return r, nil
	case FormatGzip:
		return newGzipReader(r, level)
	case FormatZstd:
		return newZstdReader(r, level)
	default:
		return r, nil
	}
}

func newGzipReader(r io.Reader, level int) (io.Reader, error) {
	if level < 1 {
		level = gzip.DefaultCompression
	}
	if level > 9 {
		level = 9
	}
	pr, pw := io.Pipe()
	go func() {
		gw, err := gzip.NewWriterLevel(pw, level)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_, err = io.Copy(gw, r)
		if err != nil {
			_ = gw.Close()
			_ = pw.CloseWithError(err)
			return
		}
		if err := gw.Close(); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_ = pw.Close()
	}()
	return pr, nil
}

func newZstdReader(r io.Reader, _ int) (io.Reader, error) {
	pr, pw := io.Pipe()
	go func() {
		zw, err := zstd.NewWriter(pw)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_, err = io.Copy(zw, r)
		if err != nil {
			_ = zw.Close()
			_ = pw.CloseWithError(err)
			return
		}
		if err := zw.Close(); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_ = pw.Close()
	}()
	return pr, nil
}
