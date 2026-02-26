package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(validateCmd)
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	RunE:  runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	return nil
}
