package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func Load(checkPerms bool) (*viper.Viper, error) {
	path := ResolveConfigPath()
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.AutomaticEnv()

	if checkPerms {
		if err := checkConfigPermissions(path); err != nil {
			return nil, err
		}
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	return v, nil
}

func checkConfigPermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	mode := info.Mode().Perm()

	if mode&0077 != 0 {
		return fmt.Errorf("config file %s has overly permissive mode %s (recommended: 0600)", path, mode)
	}
	return nil
}
