package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"VelBackuper/internal/collector"
	"VelBackuper/internal/config"
	archiveEngine "VelBackuper/internal/engine/archive"
	incrEngine "VelBackuper/internal/engine/incremental"
	"VelBackuper/internal/notifier"
	"VelBackuper/internal/s3"

	"github.com/spf13/cobra"
)

var runJob string
var runAll bool

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVar(&runJob, "job", "", "Run only this job by name")
	runCmd.Flags().BoolVar(&runAll, "all", false, "Run all enabled jobs")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run backup (optionally for one job or all jobs)",
	Long:  "Run backup for the given job or all enabled jobs. Use --job <name> for a single job, or --all for all enabled jobs. If neither is set, no jobs are run.",
	RunE:  runRun,
}

func runRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

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
	if cfg.S3 == nil {
		return fmt.Errorf("s3 configuration is required")
	}

	s3Client, err := s3.New(ctx, s3.Options{
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
	if err != nil {
		return err
	}

	var jobs []config.JobConfig
	if runAll {
		for _, j := range cfg.Jobs {
			if j.Enabled {
				jobs = append(jobs, j)
			}
		}
		if len(jobs) == 0 {
			cmd.Println("No enabled jobs to run")
			return nil
		}
	} else if runJob != "" {
		for _, j := range cfg.Jobs {
			if j.Name == runJob {
				if !j.Enabled {
					return fmt.Errorf("job %q is disabled", runJob)
				}
				jobs = append(jobs, j)
				break
			}
		}
		if len(jobs) == 0 {
			return fmt.Errorf("job %q not found", runJob)
		}
	} else {
		return fmt.Errorf("specify --job <name> or --all")
	}

	notif := NotifierFromConfig(cfg, func(msg string) { cmd.PrintErrln("Warning:", msg) })

	host, _ := os.Hostname()
	if host == "" {
		host = "localhost"
	}

	for i, job := range jobs {
		cmd.Printf("[%d/%d] Running job %q ...\n", i+1, len(jobs), job.Name)

		c := collector.CollectorFromJobConfig(&job)
		if c == nil {
			cmd.Printf("  Skipped: no sources (mysql/presets/paths) configured for job %q\n", job.Name)
			continue
		}

		start := time.Now()
		err := runOneJob(ctx, cmd, cfg.Mode, &job, c, s3Client, notif, host, start)
		duration := time.Since(start)
		if err != nil {
			cmd.Printf("  Failed after %s: %v\n", duration.Round(time.Second), err)
			return err
		}
		cmd.Printf("  OK in %s\n", duration.Round(time.Second))
	}

	cmd.Println("All jobs completed successfully.")
	return nil
}

func runOneJob(ctx context.Context, cmd *cobra.Command, mode string, job *config.JobConfig, c *collector.CompositeCollector, client *s3.Client, notif notifier.Notifier, host string, start time.Time) error {
	if notif != nil {
		_ = notif.NotifyStart(ctx, job.Name, "")
	}

	switch mode {
	case config.ModeArchive:
		return runArchiveJob(ctx, cmd, job, c, client, notif, host, start)
	case config.ModeIncremental:
		return runIncrementalJob(ctx, cmd, job, c, client, notif, start)
	default:
		return config.ErrInvalidMode
	}
}

func runArchiveJob(ctx context.Context, cmd *cobra.Command, job *config.JobConfig, c *collector.CompositeCollector, client *s3.Client, notif notifier.Notifier, host string, start time.Time) error {
	stream, err := archiveEngine.Stream(ctx, c, job.Name, archiveEngine.FormatGzip, 6)
	if err != nil {
		if notif != nil {
			_ = notif.NotifyError(ctx, job.Name, "", err)
		}
		return fmt.Errorf("stream: %w", err)
	}

	cmd.Printf("  Uploading archive ...\n")
	archiveKey, backupID, err := archiveEngine.Upload(ctx, client, job.Name, archiveEngine.FormatGzip, stream, archiveEngine.UploadOptions{PartSizeMB: 5})
	if err != nil {
		if notif != nil {
			_ = notif.NotifyError(ctx, job.Name, backupID, err)
		}
		return fmt.Errorf("upload: %w", err)
	}

	if err := archiveEngine.WriteManifest(ctx, client, archiveEngine.Manifest{
		Job: job.Name, Timestamp: backupID, Key: archiveKey, Size: 0, Host: host, Format: "tar.gz",
	}); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	if err := archiveEngine.WriteLatest(ctx, client, job.Name, backupID, archiveKey); err != nil {
		return fmt.Errorf("write latest: %w", err)
	}

	if notif != nil {
		_ = notif.NotifySuccess(ctx, job.Name, backupID, time.Since(start), 0)
	}
	return nil
}

func runIncrementalJob(ctx context.Context, cmd *cobra.Command, job *config.JobConfig, c *collector.CompositeCollector, client *s3.Client, notif notifier.Notifier, start time.Time) error {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		if err := c.Collect(ctx, job.Name, pw); err != nil {
			_ = pw.CloseWithError(err)
		}
	}()

	cmd.Printf("  Chunking and uploading ...\n")
	opts := incrEngine.RunOptions{
		ChunkSize:     incrEngine.ChunkSizeMin,
		Concurrency:   4,
		HashPrefixLen: incrEngine.DefaultHashPrefixLen,
		Notifier:      notif,
		StrictNotify:  false,
	}
	lockTTL := 30 * time.Minute
	backupID, _, _, err := incrEngine.RunWithS3Lock(ctx, client, job.Name, pr, opts, lockTTL)
	if err != nil {
		return fmt.Errorf("incremental: %w", err)
	}
	_ = backupID
	return nil
}
