package collector

import (
	"time"

	"VelBackuper/internal/config"
)

// CollectorFromJobConfig builds a CompositeCollector from a job config (MySQL + Presets + Paths).
// Returns nil if the job has no sources configured.
func CollectorFromJobConfig(job *config.JobConfig) *CompositeCollector {
	var collectors []Collector

	if job.MySQL != nil && job.MySQL.Enabled {
		opts := MySQLOpts{
			DumpAll:           job.MySQL.DumpAll,
			ExcludeSystem:     job.MySQL.ExcludeSystem,
			OneFilePerDB:      job.MySQL.OneFilePerDB,
			SingleTransaction: true,
			Routines:          true,
			Events:            true,
			Timeout:           30 * time.Minute,
		}
		if job.MySQL.Options != nil {
			opts.SingleTransaction = job.MySQL.Options.SingleTransaction
			opts.Routines = job.MySQL.Options.Routines
			opts.Events = job.MySQL.Options.Events
		}
		collectors = append(collectors, NewMySQLCollector(opts))
	}

	if job.Presets != nil && (job.Presets.Nginx || job.Presets.Apache || job.Presets.LetsEncrypt) {
		collectors = append(collectors, NewPresetsCollector(PresetsOpts{
			Nginx:       job.Presets.Nginx,
			Apache:      job.Presets.Apache,
			LetsEncrypt: job.Presets.LetsEncrypt,
		}))
	}

	if job.Paths != nil && len(job.Paths.Include) > 0 {
		collectors = append(collectors, NewFilesystemCollector(PathsOpts{
			Include:        job.Paths.Include,
			Exclude:        job.Paths.Exclude,
			FollowSymlinks: job.Paths.FollowSymlinks,
		}))
	}

	if len(collectors) == 0 {
		return nil
	}
	return NewCompositeCollector(collectors...)
}
