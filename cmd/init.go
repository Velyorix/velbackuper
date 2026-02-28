package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"VelBackuper/internal/config"
	"VelBackuper/internal/s3"
	"VelBackuper/internal/systemd"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive wizard to create configuration and systemd units",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)
	path := config.ResolveConfigPath()

	if _, err := os.Stat(path); err == nil {
		cmd.Printf("Config already exists at %s\n", path)
		if !confirm(reader, "Overwrite?", false) {
			return nil
		}
	}

	cmd.Println("VelBackuper configuration wizard")
	cmd.Println()

	mode := promptChoice(reader, "Mode", []string{config.ModeArchive, config.ModeIncremental}, config.ModeArchive)

	cmd.Println("\n--- S3 storage ---")
	endpoint := prompt(reader, "S3 endpoint (e.g. https://minio.example.com:9000)", "https://localhost:9000")
	bucket := prompt(reader, "Bucket name", "velbackuper")
	prefix := prompt(reader, "Prefix (optional)", "backups")
	accessKey := prompt(reader, "Access key", "")
	secretKey := prompt(reader, "Secret key", "")
	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("access key and secret key are required")
	}
	insecure := confirm(reader, "Skip TLS verify (for self-signed certs)?", false)

	s3cfg := &config.S3Config{
		Endpoint:  endpoint,
		Region:    "us-east-1",
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucket,
		Prefix:    config.NormalizePrefix(prefix),
	}
	if insecure {
		s3cfg.TLS = &config.TLSConfig{InsecureSkipVerify: true}
	}

	cmd.Println("\nTesting S3 connection...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := s3.New(ctx, s3.Options{
		Endpoint:           s3cfg.Endpoint,
		Region:             s3cfg.Region,
		AccessKey:          s3cfg.AccessKey,
		SecretKey:          s3cfg.SecretKey,
		Bucket:             s3cfg.Bucket,
		Prefix:             s3cfg.Prefix,
		InsecureSkipVerify: s3cfg.TLS != nil && s3cfg.TLS.InsecureSkipVerify,
	})
	if err != nil {
		return fmt.Errorf("s3 client: %w", err)
	}
	testKey := "velbackuper-init-test"
	if err := client.PutObject(ctx, testKey, bytes.NewReader([]byte("ok")), 2); err != nil {
		return fmt.Errorf("s3 test upload failed: %w", err)
	}
	_ = client.DeleteObject(ctx, testKey)
	cmd.Println("S3 connection OK")

	var jobs []config.JobConfig
	hasNginx := pathExists("/etc/nginx")
	hasApache := pathExists("/etc/apache2") || pathExists("/etc/httpd")
	hasLetsEncrypt := pathExists("/etc/letsencrypt")
	hasMySQL := mysqlAvailable()

	if hasNginx || hasApache || hasLetsEncrypt {
		presets := &config.PresetsConfig{}
		if hasNginx {
			presets.Nginx = confirm(reader, "Include nginx config?", true)
		}
		if hasApache {
			presets.Apache = confirm(reader, "Include apache config?", true)
		}
		if hasLetsEncrypt {
			presets.LetsEncrypt = confirm(reader, "Include letsencrypt certs?", true)
		}
		if presets.Nginx || presets.Apache || presets.LetsEncrypt {
			jobName := prompt(reader, "Web job name", "web")
			jobs = append(jobs, config.JobConfig{
				Name:      jobName,
				Enabled:   true,
				Presets:   presets,
				Schedule:  &config.ScheduleConfig{Period: "day", Times: 2, JitterMinutes: 15},
				Retention: &config.RetentionConfig{Days: 7},
			})
		}
	}

	if hasMySQL && confirm(reader, "Add MySQL backup job?", true) {
		jobName := prompt(reader, "MySQL job name", "mysql")
		jobs = append(jobs, *config.JobTemplate("mysql", jobName))
	}

	if len(jobs) == 0 || confirm(reader, "Add a custom files job?", false) {
		jobName := prompt(reader, "Files job name", "files")
		pathsStr := prompt(reader, "Paths to include (comma-separated)", "/var/backup")
		var include []string
		for _, p := range strings.Split(pathsStr, ",") {
			if s := strings.TrimSpace(p); s != "" {
				include = append(include, s)
			}
		}
		if len(include) == 0 {
			include = []string{"/var/backup"}
		}
		jobs = append(jobs, config.JobConfig{
			Name:      jobName,
			Enabled:   true,
			Paths:     &config.PathsConfig{Include: include},
			Schedule:  &config.ScheduleConfig{Period: "day", Times: 1, JitterMinutes: 15},
			Retention: &config.RetentionConfig{Days: 7},
		})
	}

	cfg := &config.Config{
		Mode: mode,
		S3:   s3cfg,
		Jobs: jobs,
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}
	if err := config.Write(cfg, path); err != nil {
		return err
	}
	cmd.Printf("\nConfiguration written to %s\n", path)

	if runtime.GOOS == "linux" && confirm(reader, "Install systemd timer units?", true) {
		opts := systemd.GeneratorOptions{
			Binary:     systemd.DefaultBinary,
			ConfigPath: path,
			UnitDir:    systemd.DefaultUnitDir,
			Hardening:  true,
		}
		for _, job := range cfg.Jobs {
			if !job.Enabled || job.Schedule == nil {
				continue
			}
			units, err := systemd.Generate(job, job.Schedule, opts)
			if err != nil {
				return fmt.Errorf("generate units for %s: %w", job.Name, err)
			}
			svcName, timerName := systemd.UnitFileNames(job.Name)
			svcPath := filepath.Join(opts.UnitDir, svcName)
			timerPath := filepath.Join(opts.UnitDir, timerName)
			if err := os.WriteFile(svcPath, []byte(units.Service), 0644); err != nil {
				return fmt.Errorf("write %s: %w", svcPath, err)
			}
			if err := os.WriteFile(timerPath, []byte(units.Timer), 0644); err != nil {
				_ = os.Remove(svcPath)
				return fmt.Errorf("write %s: %w", timerPath, err)
			}
			cmd.Printf("Installed %s and %s\n", svcName, timerName)
		}
		cmd.Println("Run 'systemctl daemon-reload' and 'systemctl enable velbackuper-*.timer' to activate")
	}

	cmd.Println("\nDone. Run 'velbackuper validate' to verify, then 'velbackuper run' to test.")
	return nil
}

func pathExists(p string) bool {
	_, err := os.Stat(filepath.Clean(p))
	return err == nil
}

func mysqlAvailable() bool {

	if runtime.GOOS != "linux" {
		return false
	}
	if _, err := os.Stat("/usr/bin/mysqldump"); err == nil {
		return true
	}
	if _, err := os.Stat("/var/run/mysqld/mysqld.sock"); err == nil {
		return true
	}
	return false
}

func confirm(reader *bufio.Reader, msg string, defaultYes bool) bool {
	def := "y"
	if !defaultYes {
		def = "n"
	}
	s := strings.ToLower(strings.TrimSpace(prompt(reader, msg+" (y/n)", def)))
	if s == "" {
		return defaultYes
	}
	return s == "y" || s == "yes"
}

func promptChoice(reader *bufio.Reader, label string, choices []string, defaultVal string) string {
	optStr := strings.Join(choices, "|")
	s := strings.ToLower(strings.TrimSpace(prompt(reader, label+" ("+optStr+")", defaultVal)))
	if s == "" {
		return defaultVal
	}
	for _, c := range choices {
		if strings.ToLower(c) == s {
			return c
		}
	}
	return defaultVal
}
