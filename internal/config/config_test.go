package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestUnmarshal_ModeOnly(t *testing.T) {
	v := viper.New()
	v.Set("mode", "archive")
	cfg, err := Unmarshal(v)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if cfg.Mode != "archive" {
		t.Errorf("mode = %q, want archive", cfg.Mode)
	}
}

func TestUnmarshal_IncrementalMode(t *testing.T) {
	v := viper.New()
	v.Set("mode", "incremental")
	cfg, err := Unmarshal(v)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if cfg.Mode != "incremental" {
		t.Errorf("mode = %q, want incremental", cfg.Mode)
	}
}

func TestUnmarshal_S3AndJobs(t *testing.T) {
	v := viper.New()
	v.Set("mode", "archive")
	v.Set("s3.endpoint", "http://minio:9000")
	v.Set("s3.bucket", "mybucket")
	v.Set("s3.prefix", "backup/db")
	v.Set("jobs", []map[string]interface{}{
		{"name": "web", "enabled": true},
	})
	cfg, err := Unmarshal(v)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if cfg.S3 == nil {
		t.Fatal("S3 should be set")
	}
	if cfg.S3.Endpoint != "http://minio:9000" {
		t.Errorf("s3.endpoint = %q", cfg.S3.Endpoint)
	}
	if cfg.S3.Bucket != "mybucket" {
		t.Errorf("s3.bucket = %q", cfg.S3.Bucket)
	}
	if cfg.S3.Prefix != "backup/db" {
		t.Errorf("s3.prefix = %q", cfg.S3.Prefix)
	}
	if len(cfg.Jobs) != 1 {
		t.Fatalf("len(jobs) = %d, want 1", len(cfg.Jobs))
	}
	if cfg.Jobs[0].Name != "web" {
		t.Errorf("jobs[0].name = %q", cfg.Jobs[0].Name)
	}
	if !cfg.Jobs[0].Enabled {
		t.Error("jobs[0].enabled should be true")
	}
}

func TestWrite_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		Mode: ModeArchive,
		S3: &S3Config{
			Endpoint:  "https://127.0.0.1:9000",
			Bucket:    "test",
			Prefix:    "backups",
			AccessKey: "key",
			SecretKey: "secret",
		},
		Jobs: []JobConfig{
			{
				Name:      "web",
				Enabled:   true,
				Presets:   &PresetsConfig{Nginx: true, LetsEncrypt: true},
				Schedule:  &ScheduleConfig{Period: "day", Times: 2, JitterMinutes: 15},
				Retention: &RetentionConfig{Days: 7},
			},
		},
	}
	if err := Write(cfg, path); err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("config file is empty")
	}
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("ReadInConfig: %v", err)
	}
	loaded, err := Unmarshal(v)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if loaded.Mode != cfg.Mode {
		t.Errorf("mode = %q, want %q", loaded.Mode, cfg.Mode)
	}
	if loaded.S3 == nil || loaded.S3.Bucket != cfg.S3.Bucket {
		t.Errorf("s3.bucket = %v", loaded.S3)
	}
	if len(loaded.Jobs) != 1 || loaded.Jobs[0].Name != "web" {
		t.Errorf("jobs = %v", loaded.Jobs)
	}
}

func TestJobTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		jobName  string
		wantNil  bool
	}{
		{"web", "web", "myweb", false},
		{"mysql", "mysql", "db", false},
		{"files", "files", "data", false},
		{"unknown", "invalid", "x", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := JobTemplate(tt.template, tt.jobName)
			if (job == nil) != tt.wantNil {
				t.Errorf("JobTemplate(%q, %q) = %v, wantNil=%v", tt.template, tt.jobName, job, tt.wantNil)
			}
			if job != nil && job.Name != tt.jobName {
				t.Errorf("job.Name = %q, want %q", job.Name, tt.jobName)
			}
		})
	}
}
