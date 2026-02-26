package collector

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var defaultExcludeSystem = []string{"information_schema", "performance_schema", "sys"}

type MySQLOpts struct {
	DumpAll           bool
	ExcludeSystem     bool
	OneFilePerDB      bool
	Socket            string // optional; auto-detect if empty
	DefaultsFile      string // e.g. ~/.my.cnf
	SingleTransaction bool
	Routines          bool
	Events            bool
	Timeout           time.Duration
}

type MySQLCollector struct {
	opts MySQLOpts
}

func NewMySQLCollector(opts MySQLOpts) *MySQLCollector {
	return &MySQLCollector{opts: opts}
}

func (c *MySQLCollector) Collect(ctx context.Context, jobName string, w io.Writer) error {
	mysqldump, err := exec.LookPath("mysqldump")
	if err != nil {
		return fmt.Errorf("mysqldump not found: %w", err)
	}

	runCtx := ctx
	if c.opts.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, c.opts.Timeout)
		defer cancel()
	}

	var databases []string
	if c.opts.DumpAll && c.opts.ExcludeSystem {
		databases, err = c.listDatabases(runCtx)
		if err != nil {
			return fmt.Errorf("list databases: %w", err)
		}
		if len(databases) == 0 {
			return nil // no user databases to dump
		}
	}

	args := c.buildArgs(databases)
	cmd := exec.CommandContext(runCtx, mysqldump, args...)
	cmd.Stdout = w
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return runCtx.Err()
		}
		return fmt.Errorf("mysqldump: %w", err)
	}
	return nil
}

func (c *MySQLCollector) buildArgs(databases []string) []string {
	var args []string

	if c.opts.DefaultsFile != "" {
		args = append(args, "--defaults-extra-file="+expandHome(c.opts.DefaultsFile))
	}
	if socket := c.socket(); socket != "" {
		args = append(args, "--socket="+socket)
	}
	if c.opts.SingleTransaction {
		args = append(args, "--single-transaction")
	}
	if c.opts.Routines {
		args = append(args, "--routines")
	}
	if c.opts.Events {
		args = append(args, "--events")
	}
	args = append(args, "--no-tablespaces")

	if c.opts.DumpAll {
		if len(databases) > 0 {
			args = append(args, "--databases")
			args = append(args, databases...)
		} else {
			args = append(args, "--all-databases")
		}
	}

	return args
}

func (c *MySQLCollector) listDatabases(ctx context.Context) ([]string, error) {
	mysql, err := exec.LookPath("mysql")
	if err != nil {
		return nil, fmt.Errorf("mysql not found: %w", err)
	}
	var args []string
	if c.opts.DefaultsFile != "" {
		args = append(args, "--defaults-extra-file="+expandHome(c.opts.DefaultsFile))
	}
	if socket := c.socket(); socket != "" {
		args = append(args, "--socket="+socket)
	}
	args = append(args, "-N", "-e", "SELECT schema_name FROM information_schema.schemata")
	cmd := exec.CommandContext(ctx, mysql, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	exclude := make(map[string]bool)
	for _, db := range defaultExcludeSystem {
		exclude[db] = true
	}
	var list []string
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		db := strings.TrimSpace(sc.Text())
		if db != "" && !exclude[db] {
			list = append(list, db)
		}
	}
	return list, sc.Err()
}

func expandHome(path string) string {
	if path == "" || !strings.HasPrefix(path, "~/") {
		return path
	}
	if dir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(dir, path[2:])
	}
	return path
}

func (c *MySQLCollector) socket() string {
	if c.opts.Socket != "" {
		return c.opts.Socket
	}
	for _, p := range []string{
		"/var/run/mysqld/mysqld.sock",
		"/tmp/mysql.sock",
		"/var/lib/mysql/mysql.sock",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if s := os.Getenv("MYSQL_UNIX_PORT"); s != "" {
		if !filepath.IsAbs(s) {
			s = filepath.Join("/tmp", s)
		}
		return s
	}
	return ""
}

var _ Collector = (*MySQLCollector)(nil)
