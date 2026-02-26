package config

import (
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
