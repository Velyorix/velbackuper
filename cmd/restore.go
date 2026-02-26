package cmd

import (
	"github.com/spf13/cobra"
)

var restoreJob string
var restorePoint string
var restoreTarget string

func init() {
	rootCmd.AddCommand(restoreCmd)
	restoreCmd.Flags().StringVar(&restoreJob, "job", "", "Job name to restore from (required)")
	restoreCmd.Flags().StringVar(&restorePoint, "point", "", "Backup ID or snapshot timestamp to restore (required)")
	restoreCmd.Flags().StringVar(&restoreTarget, "target", "", "Target directory to restore into (required)")
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore from a backup or snapshot",
	RunE:  runRestore,
}

func runRestore(cmd *cobra.Command, args []string) error {
	return nil
}
