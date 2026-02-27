package archive

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"VelBackuper/internal/s3"
)

const (
	PartSizeMinMB = 5
)

type UploadOptions struct {
	PartSizeMB int
}

func ArchiveKey(job string, format CompressionFormat, at time.Time) (string, string) {
	host, _ := os.Hostname()
	if host == "" {
		host = "localhost"
	}
	host = sanitizeFilename(host)
	timestamp := at.UTC().Format("20060102150405")
	ext := formatExtension(format)
	filename := fmt.Sprintf("backup-%s-%s%s", host, timestamp, ext)
	yyyy := at.UTC().Format("2006")
	mm := at.UTC().Format("01")
	dd := at.UTC().Format("02")
	key := s3.ArchiveObjectKey(job, yyyy, mm, dd, filename)
	return key, timestamp
}

func formatExtension(f CompressionFormat) string {
	switch f {
	case FormatGzip:
		return ".tar.gz"
	case FormatZstd:
		return ".tar.zst"
	default:
		return ".tar"
	}
}

var sanitizeRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func sanitizeFilename(s string) string {
	return sanitizeRe.ReplaceAllString(strings.TrimSpace(s), "_")
}

func Upload(ctx context.Context, client *s3.Client, job string, format CompressionFormat, stream io.Reader, opts UploadOptions) (key, backupID string, err error) {
	at := time.Now()
	key, backupID = ArchiveKey(job, format, at)
	partSize := int64(opts.PartSizeMB) * 1024 * 1024
	if partSize < s3.MinPartSizeBytes {
		partSize = s3.MinPartSizeBytes
	}
	if err := client.UploadMultipart(ctx, key, stream, partSize); err != nil {
		return "", "", fmt.Errorf("upload archive: %w", err)
	}
	return key, backupID, nil
}
