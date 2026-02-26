package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(uninstallSystemdCmd)
}

var uninstallSystemdCmd = &cobra.Command{
	Use:   "uninstall-systemd",
	Short: "Remove systemd service and timer units",
	RunE:  runUninstallSystemd,
}

func runUninstallSystemd(cmd *cobra.Command, args []string) error {
	return nil
}
