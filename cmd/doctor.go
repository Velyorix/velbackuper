package cmd

import (
	"context"
	"fmt"

	"VelBackuper/internal/config"
	"VelBackuper/internal/doctor"

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
	ctx := context.Background()

	v, err := config.Load(true)
	if err != nil {
		cmd.Printf("Config load: ERROR: %v\n", err)
		return err
	}
	cfg, err := config.Unmarshal(v)
	if err != nil {
		cmd.Printf("Config unmarshal: ERROR: %v\n", err)
		return err
	}
	if err := config.Validate(cfg); err != nil {
		cmd.Printf("Config validate: ERROR: %v\n", err)
		return err
	}

	results := doctor.Run(ctx, cfg)
	allOK := true
	for _, r := range results {
		status := "OK"
		if !r.OK {
			status = "ERROR"
			allOK = false
		}
		cmd.Printf("%-12s %s: %s\n", r.Name, status, r.Detail)
	}
	if !allOK {
		return fmt.Errorf("one or more checks failed; see output above")
	}
	return nil
}
