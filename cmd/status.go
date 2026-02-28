package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"VelBackuper/internal/config"
	archiveEngine "VelBackuper/internal/engine/archive"
	"VelBackuper/internal/s3"
	"VelBackuper/internal/schedule"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "List jobs with status (last run, next run, enabled)",
	Long:  "Shows each job: enabled/disabled, last backup time (from S3), next scheduled run, and whether a run is in progress (lock).",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	v, err := config.Load(false)
	if err != nil {
		return err
	}
	cfg, err := config.Unmarshal(v)
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	cmd.Println("Mode:", cfg.Mode)
	cmd.Println()

	now := time.Now()
	var s3Client *s3.Client
	if cfg.S3 != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		s3Client, err = s3.New(ctx, s3.Options{
			Endpoint:                cfg.S3.Endpoint,
			Region:                  cfg.S3.Region,
			AccessKey:               cfg.S3.AccessKey,
			SecretKey:               cfg.S3.SecretKey,
			Bucket:                  cfg.S3.Bucket,
			Prefix:                  cfg.S3.Prefix,
			PathStyle:               config.S3PathStyle(cfg.S3),
			DisableRequestChecksums: config.S3DisableRequestChecksums(cfg.S3),
			InsecureSkipVerify:      cfg.S3.TLS != nil && cfg.S3.TLS.InsecureSkipVerify,
		})
		cancel()
		if err != nil {
			cmd.Printf("Warning: could not connect to S3 (last run will be unknown): %v\n\n", err)
		}
	}

	for _, j := range cfg.Jobs {
		state := "disabled"
		if j.Enabled {
			state = "enabled"
		}

		lastRun := "-"
		if s3Client != nil && j.Enabled {
			if ts, _, err := archiveEngine.ReadLatest(context.Background(), s3Client, j.Name); err == nil && ts != "" {
				if t, err := time.Parse("20060102150405", ts); err == nil {
					lastRun = t.Format("2006-01-02 15:04")
				} else {
					lastRun = ts
				}
			} else if cfg.Mode == config.ModeIncremental {
				if ts := latestIncrementalSnapshot(context.Background(), s3Client, j.Name); ts != "" {
					if t, err := time.Parse("20060102150405", ts); err == nil {
						lastRun = t.Format("2006-01-02 15:04")
					} else {
						lastRun = ts
					}
				}
			}
		}

		nextRun := "-"
		if j.Enabled && j.Schedule != nil {
			next, desc := schedule.NextRun(j.Schedule, now)
			if !next.IsZero() {
				nextRun = next.Format("2006-01-02 15:04") + " (" + desc + ")"
			}
		}

		inProgress := ""
		if j.Enabled && lockFileExists(j.Name) {
			inProgress = " [running]"
		}

		cmd.Printf("  %s: %s  last=%s  next=%s%s\n", j.Name, state, lastRun, nextRun, inProgress)
	}
	return nil
}

func latestIncrementalSnapshot(ctx context.Context, client *s3.Client, job string) string {
	prefix := s3.SnapshotsPrefixForJob(job)
	keys, err := client.ListObjects(ctx, prefix, 50)
	if err != nil || len(keys) == 0 {
		return ""
	}
	var best string
	for _, k := range keys {
		// key like snapshots/job/20250226120000.json
		ts := strings.TrimSuffix(filepath.Base(k), ".json")
		if len(ts) == 14 && (best == "" || ts > best) {
			best = ts
		}
	}
	return best
}

func lockFileExists(jobName string) bool {
	dir := "/var/run/velbackuper"
	if d := os.Getenv("VELBACKUPER_LOCK_DIR"); d != "" {
		dir = d
	}
	path := filepath.Join(dir, jobName+".lock")
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	// If we can open it, check if it's locked (optional: flock would tell us)
	// On Unix, we could try flock; for simplicity we just report "running" if file exists and is recent (e.g. modified < 30 min)
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < 35*time.Minute
}
