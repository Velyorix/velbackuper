package config

import (
	"errors"
	"fmt"
)

var ErrInvalidMode = errors.New("invalid mode: must be exactly 'archive' or 'incremental'")

func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	switch cfg.Mode {
	case ModeArchive, ModeIncremental:
		if cfg.S3 != nil {
			cfg.S3.Prefix = NormalizePrefix(cfg.S3.Prefix)
		}
		return nil
	case "":
		return fmt.Errorf("%w (mode is required)", ErrInvalidMode)
	default:
		return fmt.Errorf("%w: got %q", ErrInvalidMode, cfg.Mode)
	}
}
