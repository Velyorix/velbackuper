package cmd

import (
	"fmt"

	"VelBackuper/internal/config"

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
	jobName := args[0]
	v, err := config.Load(false)
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
	found := false
	for i := range cfg.Jobs {
		if cfg.Jobs[i].Name == jobName {
			cfg.Jobs[i].Enabled = true
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("job %q not found", jobName)
	}
	path := config.ResolveConfigPath()
	if err := config.Write(cfg, path); err != nil {
		return err
	}
	cmd.Printf("Job %q enabled\n", jobName)
	return nil
}
