package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "velbackuper",
	Short: "Backup tool for databases, files, and configs to S3-compatible storage",
	Long:  "Velbackuper backs up MySQL, filesystem paths, and web presets to MinIO/S3 in archive or incremental mode.",
}

func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}
