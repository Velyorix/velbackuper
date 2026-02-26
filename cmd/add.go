package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.AddCommand(addJobCmd)
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a resource",
}

var addJobCmd = &cobra.Command{
	Use:   "job",
	Short: "Add a new job (interactive or template)",
	RunE:  runAddJob,
}

func runAddJob(cmd *cobra.Command, args []string) error {
	// Stub: will add a job to config in later commit
	return nil
}
