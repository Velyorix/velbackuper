package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(disableCmd)
	disableCmd.AddCommand(disableJobCmd)
}

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable a job or feature",
}

var disableJobCmd = &cobra.Command{
	Use:   "job [name]",
	Short: "Disable a job by name",
	Args:  cobra.ExactArgs(1),
	RunE:  runDisableJob,
}

func runDisableJob(cmd *cobra.Command, args []string) error {
	_ = args[0] // job name
	return nil
}
