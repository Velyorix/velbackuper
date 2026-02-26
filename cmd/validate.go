package cmd

import (
	"github.com/spf13/cobra"

	"VelBackuper/internal/config"
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
	v, err := config.Load(false)
	if err != nil {
		return err
	}
	cfg, err := config.Unmarshal(v)
	if err != nil {
		return err
	}
	return config.Validate(cfg)
}
