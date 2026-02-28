package cmd

import (
	"context"
	"fmt"
	"time"

	"VelBackuper/internal/config"
	archiveEngine "VelBackuper/internal/engine/archive"
	incrEngine "VelBackuper/internal/engine/incremental"
	"VelBackuper/internal/s3"

	"github.com/spf13/cobra"
)

var pruneJob string
var pruneAll bool
var pruneDryRun bool

func init() {
	rootCmd.AddCommand(pruneCmd)
	pruneCmd.Flags().StringVar(&pruneJob, "job", "", "Prune only this job by name")
	pruneCmd.Flags().BoolVar(&pruneAll, "all", false, "Prune all enabled jobs")
	pruneCmd.Flags().BoolVar(&pruneDryRun, "dry-run", false, "Show what would be pruned without deleting")
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Apply retention and remove old backups or orphan chunks",
	RunE:  runPrune,
}

func runPrune(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	v, err := config.Load(true)
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

	notif := NotifierFromConfig(cfg, func(msg string) { cmd.PrintErrln("Warning:", msg) })
	now := time.Now().UTC()

	var jobs []config.JobConfig
	if pruneAll {
		for _, j := range cfg.Jobs {
			if j.Enabled {
				jobs = append(jobs, j)
			}
		}
	} else if pruneJob != "" {
		for _, j := range cfg.Jobs {
			if j.Name == pruneJob {
				jobs = append(jobs, j)
				break
			}
		}
		if len(jobs) == 0 {
			return fmt.Errorf("job %q not found", pruneJob)
		}
	} else {
		return fmt.Errorf("either --job or --all must be specified")
	}

	for _, job := range jobs {
		if job.Retention == nil {
			continue
		}
		switch cfg.Mode {
		case config.ModeArchive:
			if pruneDryRun {
				cmd.Printf("Would apply archive retention for job %s\n", job.Name)
				continue
			}
			deleted, err := archiveEngine.ApplyRetention(ctx, s3Client, job.Name, job.Retention, now)
			if err != nil {
				return fmt.Errorf("archive prune for job %s: %w", job.Name, err)
			}
			cmd.Printf("Pruned %d archive backups for job %s\n", deleted, job.Name)
			if notif != nil && deleted > 0 {
				_ = notif.NotifyPrune(ctx, job.Name, 0, deleted)
			}
		case config.ModeIncremental:
			if pruneDryRun {
				cmd.Printf("Would prune incremental snapshots/objects for job %s\n", job.Name)
				continue
			}
			res, err := incrEngine.Prune(ctx, s3Client, job.Name, job.Retention, now, incrEngine.DefaultHashPrefixLen)
			if err != nil {
				return fmt.Errorf("incremental prune for job %s: %w", job.Name, err)
			}
			deleted := res.DeletedSnapshots + res.DeletedIndexes + res.DeletedObjects
			cmd.Printf("Pruned job %s: %d snapshots, %d indexes, %d objects\n", job.Name, res.DeletedSnapshots, res.DeletedIndexes, res.DeletedObjects)
			if notif != nil && deleted > 0 {
				_ = notif.NotifyPrune(ctx, job.Name, 0, deleted)
			}
		default:
			return config.ErrInvalidMode
		}
	}

	return nil
}
