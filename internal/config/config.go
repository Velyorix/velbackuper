package config

import "github.com/spf13/viper"

const (
	ModeArchive     = "archive"
	ModeIncremental = "incremental"
)

type Config struct {
	Mode          string               `mapstructure:"mode" yaml:"mode"`
	S3            *S3Config            `mapstructure:"s3" yaml:"s3,omitempty"`
	Jobs          []JobConfig          `mapstructure:"jobs" yaml:"jobs"`
	Notifications *NotificationsConfig `mapstructure:"notifications" yaml:"notifications,omitempty"`
}

type S3Config struct {
	Endpoint  string     `mapstructure:"endpoint" yaml:"endpoint"`
	Region    string     `mapstructure:"region" yaml:"region"`
	AccessKey string     `mapstructure:"access_key" yaml:"access_key"`
	SecretKey string     `mapstructure:"secret_key" yaml:"secret_key"`
	Bucket    string     `mapstructure:"bucket" yaml:"bucket"`
	Prefix    string     `mapstructure:"prefix" yaml:"prefix"`
	TLS       *TLSConfig `mapstructure:"tls" yaml:"tls,omitempty"`
}

type TLSConfig struct {
	Enabled            bool   `mapstructure:"enabled" yaml:"enabled"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	CAFile             string `mapstructure:"ca_file" yaml:"ca_file"`
}

type JobConfig struct {
	Name      string           `mapstructure:"name" yaml:"name"`
	Enabled   bool             `mapstructure:"enabled" yaml:"enabled"`
	MySQL     *MySQLJobConfig  `mapstructure:"mysql" yaml:"mysql,omitempty"`
	Presets   *PresetsConfig   `mapstructure:"presets" yaml:"presets,omitempty"`
	Paths     *PathsConfig     `mapstructure:"paths" yaml:"paths,omitempty"`
	Schedule  *ScheduleConfig  `mapstructure:"schedule" yaml:"schedule,omitempty"`
	Retention *RetentionConfig `mapstructure:"retention" yaml:"retention,omitempty"`
}

type MySQLJobConfig struct {
	Enabled       bool              `mapstructure:"enabled" yaml:"enabled"`
	DumpAll       bool              `mapstructure:"dump_all" yaml:"dump_all"`
	ExcludeSystem bool              `mapstructure:"exclude_system" yaml:"exclude_system"`
	OneFilePerDB  bool              `mapstructure:"one_file_per_db" yaml:"one_file_per_db"`
	Options       *MySQLDumpOptions `mapstructure:"options" yaml:"options,omitempty"`
}

type MySQLDumpOptions struct {
	SingleTransaction bool `mapstructure:"single_transaction" yaml:"single_transaction"`
	Routines          bool `mapstructure:"routines" yaml:"routines"`
	Events            bool `mapstructure:"events" yaml:"events"`
}

type PresetsConfig struct {
	Nginx       bool `mapstructure:"nginx" yaml:"nginx"`
	Apache      bool `mapstructure:"apache" yaml:"apache"`
	LetsEncrypt bool `mapstructure:"letsencrypt" yaml:"letsencrypt"`
}

type PathsConfig struct {
	Include        []string `mapstructure:"include" yaml:"include"`
	Exclude        []string `mapstructure:"exclude" yaml:"exclude"`
	FollowSymlinks bool     `mapstructure:"follow_symlinks" yaml:"follow_symlinks"`
}

type ScheduleConfig struct {
	Period        string `mapstructure:"period" yaml:"period"` // day | week | month
	Times         int    `mapstructure:"times" yaml:"times"`   // 1-5 per period
	JitterMinutes int    `mapstructure:"jitter_minutes" yaml:"jitter_minutes"`
}

type RetentionConfig struct {
	Days   int `mapstructure:"days" yaml:"days"`
	Weeks  int `mapstructure:"weeks" yaml:"weeks"`
	Months int `mapstructure:"months" yaml:"months"`
}

type NotificationsConfig struct {
	Discord *DiscordConfig `mapstructure:"discord" yaml:"discord,omitempty"`
}

type DiscordConfig struct {
	Enabled        bool             `mapstructure:"enabled" yaml:"enabled"`
	WebhookURL     string           `mapstructure:"webhook_url" yaml:"webhook_url"`
	Level          string           `mapstructure:"level" yaml:"level"` // e.g. "all"
	Events         []string         `mapstructure:"events" yaml:"events"`
	Mentions       *DiscordMentions `mapstructure:"mentions" yaml:"mentions,omitempty"`
	TimeoutSeconds int              `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
	Retry          *DiscordRetry    `mapstructure:"retry" yaml:"retry,omitempty"`
}

type DiscordMentions struct {
	OnError string `mapstructure:"on_error" yaml:"on_error"`
}

type DiscordRetry struct {
	Attempts  int `mapstructure:"attempts" yaml:"attempts"`
	BackoffMs int `mapstructure:"backoff_ms" yaml:"backoff_ms"`
}

func Unmarshal(v *viper.Viper) (*Config, error) {
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}
	return &c, nil
}
