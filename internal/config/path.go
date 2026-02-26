package config

import (
	"os"
	"path/filepath"
)

const (
	DefaultConfigDir  = "/etc/velbackuper"
	DefaultConfigName = "config.yaml"
)

const EnvConfigPath = "VELBACKUPER_CONFIG"

func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir, DefaultConfigName)
}

func ResolveConfigPath() string {
	if p := os.Getenv(EnvConfigPath); p != "" {
		return p
	}
	return DefaultConfigPath()
}
