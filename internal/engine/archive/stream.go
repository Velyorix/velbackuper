package archive

import (
	"context"
	"io"

	"VelBackuper/internal/collector"
)

func CollectToStream(ctx context.Context, c collector.Collector, jobName string) (io.Reader, error) {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		if err := c.Collect(ctx, jobName, pw); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()
	return pr, nil
}

func Stream(ctx context.Context, c collector.Collector, jobName string, format CompressionFormat, compressionLevel int) (io.Reader, error) {
	raw, err := CollectToStream(ctx, c, jobName)
	if err != nil {
		return nil, err
	}
	return NewCompressReader(raw, format, compressionLevel)
}
