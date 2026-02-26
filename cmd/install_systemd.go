package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(installSystemdCmd)
}

var installSystemdCmd = &cobra.Command{
	Use:   "install-systemd",
	Short: "Install systemd service and timer units",
	RunE:  runInstallSystemd,
}

func runInstallSystemd(cmd *cobra.Command, args []string) error {
	return nil
}
