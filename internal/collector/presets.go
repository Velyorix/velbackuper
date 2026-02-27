package collector

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

var (
	PresetPathNginx       = "/etc/nginx"
	PresetPathApache      = "/etc/apache2"
	PresetPathApacheAlt   = "/etc/httpd"
	PresetPathLetsEncrypt = "/etc/letsencrypt"
)

type PresetsOpts struct {
	Nginx       bool
	Apache      bool
	LetsEncrypt bool
}

type PresetsCollector struct {
	opts PresetsOpts
}

func NewPresetsCollector(opts PresetsOpts) *PresetsCollector {
	return &PresetsCollector{opts: opts}
}

func (c *PresetsCollector) Collect(ctx context.Context, jobName string, w io.Writer) error {
	include := c.includedPaths()
	if len(include) == 0 {
		return nil
	}
	fs := NewFilesystemCollector(PathsOpts{
		Include:        include,
		Exclude:        nil,
		FollowSymlinks: false,
	})
	return fs.Collect(ctx, jobName, w)
}

func (c *PresetsCollector) includedPaths() []string {
	var out []string
	if c.opts.Nginx {
		if p := PresetPathNginx; pathExists(p) {
			out = append(out, p)
		}
	}
	if c.opts.Apache {
		if pathExists(PresetPathApache) {
			out = append(out, PresetPathApache)
		} else if pathExists(PresetPathApacheAlt) {
			out = append(out, PresetPathApacheAlt)
		}
	}
	if c.opts.LetsEncrypt {
		if p := PresetPathLetsEncrypt; pathExists(p) {
			out = append(out, p)
		}
	}
	return out
}

func pathExists(p string) bool {
	p = filepath.Clean(p)
	_, err := os.Stat(p)
	return err == nil
}

var _ Collector = (*PresetsCollector)(nil)
