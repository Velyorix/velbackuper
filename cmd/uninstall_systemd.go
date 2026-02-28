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

var uninstallSystemdUnitDir string

func init() {
	rootCmd.AddCommand(uninstallSystemdCmd)
	uninstallSystemdCmd.Flags().StringVar(&uninstallSystemdUnitDir, "unit-dir", systemd.DefaultUnitDir, "Directory for systemd unit files")
}

var uninstallSystemdCmd = &cobra.Command{
	Use:   "uninstall-systemd",
	Short: "Remove systemd service and timer units",
	RunE:  runUninstallSystemd,
}

func runUninstallSystemd(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("uninstall-systemd is only supported on Linux")
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

	var removed []string
	for _, job := range cfg.Jobs {
		if !job.Enabled || job.Schedule == nil {
			continue
		}
		svcName, timerName := systemd.UnitFileNames(job.Name)
		svcPath := filepath.Join(uninstallSystemdUnitDir, svcName)
		timerPath := filepath.Join(uninstallSystemdUnitDir, timerName)

		_ = exec.Command("systemctl", "disable", "--now", timerName).Run()

		if err := os.Remove(timerPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", timerPath, err)
		}
		if err := os.Remove(svcPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", svcPath, err)
		}
		removed = append(removed, job.Name)
		cmd.Printf("Removed %s and %s for job %s\n", svcName, timerName, job.Name)
	}

	if len(removed) == 0 {
		cmd.Println("No jobs with schedule to uninstall")
		return nil
	}

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	cmd.Println("Reloaded systemd daemon")

	return nil
}
