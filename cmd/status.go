package cmd

import (
	"fmt"

	"VelBackuper/internal/config"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show backup status (last run, next run, job state)",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	v, err := config.Load(true)
	if err != nil {
		return err
	}
	cfg, err := config.Unmarshal(v)
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	cmd.Println("Mode:", cfg.Mode)
	cmd.Println("Jobs:")
	for _, j := range cfg.Jobs {
		state := "disabled"
		if j.Enabled {
			state = "enabled"
		}

		ret := ""
		if j.Retention != nil {
			ret = fmt.Sprintf("retention=%dd/%dw/%dm", j.Retention.Days, j.Retention.Weeks, j.Retention.Months)
		}
		cmd.Printf("  - %s: %s %s\n", j.Name, state, ret)
	}
	return nil
}
