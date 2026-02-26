package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List backups or snapshots",
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	return nil
}
