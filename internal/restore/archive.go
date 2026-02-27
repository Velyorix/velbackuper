package restore

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"VelBackuper/internal/s3"

	"github.com/klauspost/compress/zstd"
)

type ArchiveRestoreOptions struct {
	MysqlOnly bool
	DryRun    bool
}

func RestoreArchive(ctx context.Context, client *s3.Client, key, targetDir string, opts ArchiveRestoreOptions) error {
	rc, err := client.GetObject(ctx, key)
	if err != nil {
		return fmt.Errorf("get archive %s: %w", key, err)
	}
	defer rc.Close()

	r, err := decompressStream(rc, key)
	if err != nil {
		return fmt.Errorf("decompress %s: %w", key, err)
	}
	tr := tar.NewReader(r)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar %s: %w", key, err)
		}
		if err := restoreTarEntry(tr, hdr, targetDir, opts); err != nil {
			return err
		}
	}
	return nil
}

func decompressStream(r io.Reader, key string) (io.Reader, error) {
	lower := strings.ToLower(key)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"), strings.HasSuffix(lower, ".gz"):
		gr, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		return gr, nil
	case strings.HasSuffix(lower, ".tar.zst"), strings.HasSuffix(lower, ".zst"):
		zr, err := zstd.NewReader(r)
		if err != nil {
			return nil, err
		}
		return zr, nil
	default:
		return r, nil
	}
}

func restoreTarEntry(tr *tar.Reader, hdr *tar.Header, targetDir string, opts ArchiveRestoreOptions) error {
	name := cleanTarName(hdr.Name)
	if name == "" {
		return nil
	}
	if opts.MysqlOnly && !strings.HasPrefix(name, "mysql/") {
		return nil
	}

	dstPath := filepath.Join(targetDir, filepath.FromSlash(name))

	switch hdr.Typeflag {
	case tar.TypeDir:
		if opts.DryRun {
			return nil
		}
		return os.MkdirAll(dstPath, os.FileMode(hdr.Mode))
	case tar.TypeReg, tar.TypeRegA:
		if opts.DryRun {
			_, _ = io.Copy(io.Discard, tr)
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			_ = f.Close()
			return err
		}
		return f.Close()
	case tar.TypeSymlink:
		if opts.DryRun {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}

		_ = os.Remove(dstPath)
		return os.Symlink(hdr.Linkname, dstPath)
	default:
		return nil
	}
}

func cleanTarName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = path.Clean(name)
	name = strings.TrimLeft(name, "/")
	if name == "" || strings.HasPrefix(name, "..") {
		return ""
	}
	return name
}
