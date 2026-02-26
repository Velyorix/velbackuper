package cmd

import (
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
	return nil
}
