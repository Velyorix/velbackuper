package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pruneCmd)
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Apply retention and remove old backups or orphan chunks",
	RunE:  runPrune,
}

func runPrune(cmd *cobra.Command, args []string) error {
	return nil
}
