package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(enableCmd)
	enableCmd.AddCommand(enableJobCmd)
}

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable a job or feature",
}

var enableJobCmd = &cobra.Command{
	Use:   "job [name]",
	Short: "Enable a job by name",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnableJob,
}

func runEnableJob(cmd *cobra.Command, args []string) error {
	_ = args[0] // job name
	return nil
}
