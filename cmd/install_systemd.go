package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"VelBackuper/internal/config"
	"VelBackuper/internal/systemd"

	"github.com/spf13/cobra"
)

var installSystemdUnitDir string
var installSystemdBinary string
var installSystemdHardening bool

func init() {
	rootCmd.AddCommand(installSystemdCmd)
	installSystemdCmd.Flags().StringVar(&installSystemdUnitDir, "unit-dir", systemd.DefaultUnitDir, "Directory for systemd unit files")
	installSystemdCmd.Flags().StringVar(&installSystemdBinary, "binary", systemd.DefaultBinary, "Path to velbackuper binary")
	installSystemdCmd.Flags().BoolVar(&installSystemdHardening, "hardening", true, "Enable systemd hardening options")
}

var installSystemdCmd = &cobra.Command{
	Use:   "install-systemd",
	Short: "Install systemd service and timer units",
	RunE:  runInstallSystemd,
}

func runInstallSystemd(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("install-systemd is only supported on Linux")
	}

	v, err := config.Load(true)
	if err != nil {
		return err
	}
	cfg, err := config.Unmarshal(v)
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	configPath := config.ResolveConfigPath()
	opts := systemd.GeneratorOptions{
		Binary:     installSystemdBinary,
		ConfigPath: configPath,
		UnitDir:    installSystemdUnitDir,
		Hardening:  installSystemdHardening,
	}

	var installed []string
	for _, job := range cfg.Jobs {
		if !job.Enabled || job.Schedule == nil {
			continue
		}
		units, err := systemd.Generate(job, job.Schedule, opts)
		if err != nil {
			return fmt.Errorf("generate units for job %s: %w", job.Name, err)
		}
		svcName, timerName := systemd.UnitFileNames(job.Name)
		svcPath := filepath.Join(installSystemdUnitDir, svcName)
		timerPath := filepath.Join(installSystemdUnitDir, timerName)

		if err := os.WriteFile(svcPath, []byte(units.Service), 0644); err != nil {
			return fmt.Errorf("write %s: %w", svcPath, err)
		}
		if err := os.WriteFile(timerPath, []byte(units.Timer), 0644); err != nil {
			_ = os.Remove(svcPath)
			return fmt.Errorf("write %s: %w", timerPath, err)
		}
		installed = append(installed, job.Name)
		cmd.Printf("Installed %s and %s for job %s\n", svcName, timerName, job.Name)
	}

	if len(installed) == 0 {
		cmd.Println("No jobs with schedule to install")
		return nil
	}

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	cmd.Println("Reloaded systemd daemon")

	for _, jobName := range installed {
		_, timerName := systemd.UnitFileNames(jobName)
		if err := exec.Command("systemctl", "enable", timerName).Run(); err != nil {
			return fmt.Errorf("systemctl enable %s: %w", timerName, err)
		}
		cmd.Printf("Enabled timer %s\n", timerName)
	}

	return nil
}
