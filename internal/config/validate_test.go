package config

import (
	"errors"
	"testing"
)

func TestValidate_NilConfig(t *testing.T) {
	err := Validate(nil)
	if err == nil {
		t.Fatal("Validate(nil) should return error")
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestValidate_EmptyMode(t *testing.T) {
	cfg := &Config{Mode: ""}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate(empty mode) should return error")
	}
	if !errors.Is(err, ErrInvalidMode) {
		t.Errorf("expected ErrInvalidMode, got %v", err)
	}
}

func TestValidate_InvalidMode(t *testing.T) {
	cfg := &Config{Mode: "invalid"}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate(invalid mode) should return error")
	}
	if !errors.Is(err, ErrInvalidMode) {
		t.Errorf("expected ErrInvalidMode, got %v", err)
	}
	if err.Error() == "" {
		t.Error("error should mention the invalid value")
	}
}

func TestValidate_ValidArchive(t *testing.T) {
	cfg := &Config{Mode: ModeArchive}
	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate(archive) should succeed: %v", err)
	}
}

func TestValidate_ValidIncremental(t *testing.T) {
	cfg := &Config{Mode: ModeIncremental}
	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate(incremental) should succeed: %v", err)
	}
}

func TestValidate_NormalizesS3Prefix(t *testing.T) {
	cfg := &Config{
		Mode: ModeArchive,
		S3:   &S3Config{Prefix: "/backup//database/"},
	}
	err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate should succeed: %v", err)
	}
	if cfg.S3.Prefix != "backup/database" {
		t.Errorf("S3.Prefix should be normalized to %q, got %q", "backup/database", cfg.S3.Prefix)
	}
}

func TestValidate_NilS3NoPanic(t *testing.T) {
	cfg := &Config{Mode: ModeArchive, S3: nil}
	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate with nil S3 should succeed: %v", err)
	}
}
