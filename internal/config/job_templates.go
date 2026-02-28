package config

func JobTemplate(name, jobName string) *JobConfig {
	switch name {
	case "web":
		return &JobConfig{
			Name:    jobName,
			Enabled: true,
			Presets: &PresetsConfig{Nginx: true, Apache: false, LetsEncrypt: true},
			Schedule: &ScheduleConfig{
				Period:        "day",
				Times:         2,
				JitterMinutes: 15,
			},
			Retention: &RetentionConfig{Days: 7, Weeks: 0, Months: 0},
		}
	case "mysql":
		return &JobConfig{
			Name:    jobName,
			Enabled: true,
			MySQL: &MySQLJobConfig{
				Enabled:       true,
				DumpAll:       true,
				ExcludeSystem: true,
				OneFilePerDB:  false,
			},
			Schedule: &ScheduleConfig{
				Period:        "day",
				Times:         1,
				JitterMinutes: 30,
			},
			Retention: &RetentionConfig{Days: 7, Weeks: 0, Months: 0},
		}
	case "files":
		return &JobConfig{
			Name:    jobName,
			Enabled: true,
			Paths: &PathsConfig{
				Include:        []string{"/var/backup"},
				Exclude:        nil,
				FollowSymlinks: false,
			},
			Schedule: &ScheduleConfig{
				Period:        "day",
				Times:         1,
				JitterMinutes: 15,
			},
			Retention: &RetentionConfig{Days: 7, Weeks: 0, Months: 0},
		}
	default:
		return nil
	}
}

func JobTemplateNames() []string {
	return []string{"web", "mysql", "files"}
}
