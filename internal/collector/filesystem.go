package collector

import (
	"archive/tar"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type PathsOpts struct {
	Include        []string
	Exclude        []string
	FollowSymlinks bool
}

type FilesystemCollector struct {
	opts PathsOpts
}

func NewFilesystemCollector(opts PathsOpts) *FilesystemCollector {
	return &FilesystemCollector{opts: opts}
}

func (c *FilesystemCollector) Collect(ctx context.Context, jobName string, w io.Writer) error {
	if len(c.opts.Include) == 0 {
		return nil
	}
	tw := tar.NewWriter(w)
	defer tw.Close()

	for _, root := range c.opts.Include {
		root = filepath.Clean(root)
		if root == "" || root == "." {
			continue
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return err
		}
		if err := c.walk(ctx, tw, absRoot, absRoot); err != nil {
			return err
		}
	}
	return nil
}

func (c *FilesystemCollector) walk(ctx context.Context, tw *tar.Writer, root, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		full := filepath.Join(dir, e.Name())
		rel, err := filepath.Rel(root, full)
		if err != nil {
			return err
		}
		tarName := filepath.ToSlash(rel)
		if strings.HasPrefix(tarName, "..") {
			continue
		}
		if c.excluded(full) {
			if e.IsDir() {
				continue
			}
			continue
		}

		info, err := e.Info()
		if err != nil {
			return err
		}
		mode := info.Mode()

		if mode&os.ModeSymlink != 0 {
			if !c.opts.FollowSymlinks {
				link, err := os.Readlink(full)
				if err != nil {
					return err
				}
				hdr, err := tar.FileInfoHeader(info, link)
				if err != nil {
					return err
				}
				hdr.Name = tarName
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}
				continue
			}

			target, err := filepath.EvalSymlinks(full)
			if err != nil {
				return err
			}
			info, err = os.Stat(target)
			if err != nil {
				return err
			}
			if info.IsDir() {
				hdr, err := tar.FileInfoHeader(info, "")
				if err != nil {
					return err
				}
				hdr.Name = tarName + "/"
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}
				if err := c.walk(ctx, tw, root, full); err != nil {
					return err
				}
				continue
			}
			mode = info.Mode()
		}

		if info.IsDir() {
			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			hdr.Name = tarName + "/"
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			if err := c.walk(ctx, tw, root, full); err != nil {
				return err
			}
			continue
		}

		if !info.Mode().IsRegular() {
			continue
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = tarName
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		f, err := os.Open(full)
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, f)
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *FilesystemCollector) excluded(path string) bool {
	path = filepath.Clean(path)
	for _, ex := range c.opts.Exclude {
		ex = filepath.Clean(ex)
		if ex == "" {
			continue
		}
		absEx, err := filepath.Abs(ex)
		if err != nil {
			continue
		}
		if path == absEx || strings.HasPrefix(path, absEx+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

var _ Collector = (*FilesystemCollector)(nil)
