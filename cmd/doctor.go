package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose config, S3 connectivity, locks, and disk",
	RunE:  runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	// Stub: will implement in later commit
	return nil
}
