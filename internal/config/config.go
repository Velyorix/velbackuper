package config

import "github.com/spf13/viper"

const (
	ModeArchive     = "archive"
	ModeIncremental = "incremental"
)

type Config struct {
	Mode          string               `mapstructure:"mode"`
	S3            *S3Config            `mapstructure:"s3"`
	Jobs          []JobConfig          `mapstructure:"jobs"`
	Notifications *NotificationsConfig `mapstructure:"notifications"`
}

type S3Config struct {
	Endpoint  string     `mapstructure:"endpoint"`
	Region    string     `mapstructure:"region"`
	AccessKey string     `mapstructure:"access_key"`
	SecretKey string     `mapstructure:"secret_key"`
	Bucket    string     `mapstructure:"bucket"`
	Prefix    string     `mapstructure:"prefix"`
	TLS       *TLSConfig `mapstructure:"tls"`
}

type TLSConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
	CAFile             string `mapstructure:"ca_file"`
}

type JobConfig struct {
	Name    string `mapstructure:"name"`
	Enabled bool   `mapstructure:"enabled"`
}

type NotificationsConfig struct {
	Discord *DiscordConfig `mapstructure:"discord"`
}

type DiscordConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

func Unmarshal(v *viper.Viper) (*Config, error) {
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}
	return &c, nil
}
