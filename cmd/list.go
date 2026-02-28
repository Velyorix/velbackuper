package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"VelBackuper/internal/config"
	"VelBackuper/internal/s3"

	"github.com/spf13/cobra"
)

var listJob string

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVar(&listJob, "job", "", "List backups/snapshots for this job only")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List backups or snapshots",
	Long:  "List backup IDs (archive) or snapshot timestamps (incremental) for one job or all jobs. Use --job <name> to list a single job.",
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := s3.New(ctx, s3.Options{
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
	if listJob != "" {
		for _, j := range cfg.Jobs {
			if j.Name == listJob {
				jobs = append(jobs, j)
				break
			}
		}
		if len(jobs) == 0 {
			return fmt.Errorf("job %q not found", listJob)
		}
	} else {
		jobs = cfg.Jobs
	}

	for _, j := range jobs {
		cmd.Printf("Job: %s\n", j.Name)
		if cfg.Mode == config.ModeArchive {
			keys, err := client.ListObjects(ctx, s3.ManifestsPrefix+"/"+j.Name, 100)
			if err != nil {
				cmd.Printf("  error: %v\n", err)
				continue
			}
			var timestamps []string
			for _, k := range keys {
				if strings.HasSuffix(k, ".json") {
					ts := strings.TrimSuffix(k[strings.LastIndex(k, "/")+1:], ".json")
					if len(ts) == 14 {
						timestamps = append(timestamps, ts)
					}
				}
			}
			sort.Sort(sort.Reverse(sort.StringSlice(timestamps)))
			for _, ts := range timestamps {
				if t, err := time.Parse("20060102150405", ts); err == nil {
					cmd.Printf("  %s  %s\n", ts, t.Format("2006-01-02 15:04:05"))
				} else {
					cmd.Printf("  %s\n", ts)
				}
			}
			if len(timestamps) == 0 {
				cmd.Println("  (no backups yet)")
			}
		} else {
			prefix := s3.SnapshotsPrefixForJob(j.Name)
			keys, err := client.ListObjects(ctx, strings.TrimSuffix(prefix, "/"), 100)
			if err != nil {
				cmd.Printf("  error: %v\n", err)
				continue
			}
			var timestamps []string
			for _, k := range keys {
				base := k[strings.LastIndex(k, "/")+1:]
				ts := strings.TrimSuffix(base, ".json")
				if len(ts) == 14 {
					timestamps = append(timestamps, ts)
				}
			}
			sort.Sort(sort.Reverse(sort.StringSlice(timestamps)))
			for _, ts := range timestamps {
				if t, err := time.Parse("20060102150405", ts); err == nil {
					cmd.Printf("  %s  %s\n", ts, t.Format("2006-01-02 15:04:05"))
				} else {
					cmd.Printf("  %s\n", ts)
				}
			}
			if len(timestamps) == 0 {
				cmd.Println("  (no snapshots yet)")
			}
		}
		cmd.Println()
	}
	return nil
}
