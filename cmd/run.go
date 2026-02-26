package cmd

import (
	"github.com/spf13/cobra"
)

var runJob string
var runAll bool

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVar(&runJob, "job", "", "Run only this job by name")
	runCmd.Flags().BoolVar(&runAll, "all", false, "Run all enabled jobs")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run backup (optionally for one job or all jobs)",
	RunE:  runRun,
}

func runRun(cmd *cobra.Command, args []string) error {
	return nil
}
