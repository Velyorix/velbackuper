package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show backup status (last run, next run, job state)",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	return nil
}
