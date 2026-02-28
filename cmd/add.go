package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"VelBackuper/internal/config"

	"github.com/spf13/cobra"
)

var addJobTemplate string
var addJobName string

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.AddCommand(addJobCmd)
	addJobCmd.Flags().StringVar(&addJobTemplate, "template", "", "Job template: web, mysql, or files")
	addJobCmd.Flags().StringVar(&addJobName, "name", "", "Job name (required with --template)")
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
	if addJobTemplate != "" {
		return runAddJobTemplate(cmd)
	}
	return runAddJobInteractive(cmd)
}

func runAddJobTemplate(cmd *cobra.Command) error {
	if addJobName == "" {
		return fmt.Errorf("--name is required when using --template")
	}
	job := config.JobTemplate(addJobTemplate, addJobName)
	if job == nil {
		return fmt.Errorf("unknown template %q (use: web, mysql, files)", addJobTemplate)
	}
	return addJobToConfig(cmd, job)
}

func runAddJobInteractive(cmd *cobra.Command) error {
	reader := bufio.NewReader(os.Stdin)
	jobName := prompt(reader, "Job name", "my-backup")
	if jobName == "" {
		return fmt.Errorf("job name is required")
	}
	for _, j := range mustLoadConfig(cmd).Jobs {
		if j.Name == jobName {
			return fmt.Errorf("job %q already exists", jobName)
		}
	}
	fmt.Println("Available templates: web (nginx+letsencrypt), mysql, files")
	tpl := strings.ToLower(strings.TrimSpace(prompt(reader, "Template (web/mysql/files) or Enter for custom", "web")))
	if tpl == "" {
		tpl = "web"
	}
	job := config.JobTemplate(tpl, jobName)
	if job == nil {
		job = &config.JobConfig{
			Name:    jobName,
			Enabled: true,
			Paths: &config.PathsConfig{
				Include: strings.Split(strings.TrimSpace(prompt(reader, "Paths to include (comma-separated)", "/var/backup")), ","),
			},
			Schedule:  &config.ScheduleConfig{Period: "day", Times: 1, JitterMinutes: 15},
			Retention: &config.RetentionConfig{Days: 7},
		}
		for i, p := range job.Paths.Include {
			job.Paths.Include[i] = strings.TrimSpace(p)
		}
	}
	return addJobToConfig(cmd, job)
}

func addJobToConfig(cmd *cobra.Command, job *config.JobConfig) error {
	cfg, err := loadConfigForEdit()
	if err != nil {
		return err
	}
	for _, j := range cfg.Jobs {
		if j.Name == job.Name {
			return fmt.Errorf("job %q already exists", job.Name)
		}
	}
	cfg.Jobs = append(cfg.Jobs, *job)
	if err := config.Validate(cfg); err != nil {
		return err
	}
	path := config.ResolveConfigPath()
	if err := config.Write(cfg, path); err != nil {
		return err
	}
	cmd.Printf("Job %q added\n", job.Name)
	return nil
}

func loadConfigForEdit() (*config.Config, error) {
	v, err := config.Load(false)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Unmarshal(v)
	if err != nil {
		return nil, err
	}
	return cfg, config.Validate(cfg)
}

func mustLoadConfig(cmd *cobra.Command) *config.Config {
	cfg, err := loadConfigForEdit()
	if err != nil {
		return &config.Config{Jobs: nil}
	}
	return cfg
}

func prompt(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := reader.ReadString('\n')
	s := strings.TrimSpace(strings.TrimSuffix(line, "\n"))
	if s == "" && defaultVal != "" {
		return defaultVal
	}
	return s
}
