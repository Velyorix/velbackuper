package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"VelBackuper/internal/config"

	"github.com/spf13/cobra"
)

var (
	webhookURLFlag       string
	discordEnableFlag    bool
	discordDisableFlag   bool
	notificationsEnable  bool
	notificationsDisable bool
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configWebhooksCmd)
	configWebhooksCmd.Flags().StringVar(&webhookURLFlag, "webhook-url", "", "Discord webhook URL (or set VELBACKUPER_DISCORD_WEBHOOK_URL)")
	configWebhooksCmd.Flags().BoolVar(&discordEnableFlag, "discord-enable", false, "Enable Discord notifications")
	configWebhooksCmd.Flags().BoolVar(&discordDisableFlag, "discord-disable", false, "Disable Discord notifications")
	configWebhooksCmd.Flags().BoolVar(&notificationsEnable, "notifications-on", false, "Enable all notifications")
	configWebhooksCmd.Flags().BoolVar(&notificationsDisable, "notifications-off", false, "Disable all notifications")
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
}

var configWebhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "Configure Discord webhook and notifications",
	Long:  "Show current notification settings and optionally set Discord webhook URL, enable/disable Discord or global notifications. Run without flags for interactive prompts.",
	RunE:  runConfigWebhooks,
}

func runConfigWebhooks(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfigForEdit()
	if err != nil {
		return err
	}
	path := config.ResolveConfigPath()

	// Ensure we have a notifications block to edit
	if cfg.Notifications == nil {
		cfg.Notifications = &config.NotificationsConfig{}
	}
	if cfg.Notifications.Discord == nil {
		cfg.Notifications.Discord = &config.DiscordConfig{}
	}

	hasFlags := webhookURLFlag != "" || discordEnableFlag || discordDisableFlag || notificationsEnable || notificationsDisable

	if hasFlags {
		return applyWebhookFlags(cmd, cfg, path)
	}
	return runConfigWebhooksInteractive(cmd, cfg, path)
}

func applyWebhookFlags(cmd *cobra.Command, cfg *config.Config, path string) error {
	if discordEnableFlag && discordDisableFlag {
		return fmt.Errorf("cannot use both --discord-enable and --discord-disable")
	}
	if notificationsEnable && notificationsDisable {
		return fmt.Errorf("cannot use both --notifications-on and --notifications-off")
	}

	if webhookURLFlag != "" {
		cfg.Notifications.Discord.WebhookURL = strings.TrimSpace(webhookURLFlag)
	}
	if discordEnableFlag {
		cfg.Notifications.Discord.Enabled = true
	}
	if discordDisableFlag {
		cfg.Notifications.Discord.Enabled = false
	}
	if notificationsEnable {
		cfg.Notifications.Enabled = boolPtr(true)
	}
	if notificationsDisable {
		cfg.Notifications.Enabled = boolPtr(false)
	}

	if err := config.Validate(cfg); err != nil {
		return err
	}
	if err := config.Write(cfg, path); err != nil {
		return err
	}
	cmd.Printf("Configuration updated: %s\n", path)
	printWebhookStatus(cmd, cfg)
	return nil
}

func runConfigWebhooksInteractive(cmd *cobra.Command, cfg *config.Config, path string) error {
	reader := bufio.NewReader(os.Stdin)

	cmd.Println("Current notification settings:")
	printWebhookStatus(cmd, cfg)
	cmd.Println()

	// Discord webhook URL
	currentURL := cfg.Notifications.Discord.WebhookURL
	if currentURL == "" && os.Getenv("VELBACKUPER_DISCORD_WEBHOOK_URL") != "" {
		currentURL = "(set via VELBACKUPER_DISCORD_WEBHOOK_URL)"
	}
	promptURL := "Discord webhook URL"
	if currentURL != "" {
		promptURL += " (Enter to keep current)"
	}
	promptURL += ": "
	cmd.Print(promptURL)
	line, _ := reader.ReadString('\n')
	urlInput := strings.TrimSpace(strings.TrimSuffix(line, "\n"))
	if urlInput != "" {
		cfg.Notifications.Discord.WebhookURL = urlInput
	}

	// Enable Discord?
	if confirm(reader, "Enable Discord notifications?", cfg.Notifications.Discord.Enabled) {
		cfg.Notifications.Discord.Enabled = true
	} else {
		cfg.Notifications.Discord.Enabled = false
	}

	// Global notifications on/off
	globOn := config.NotificationsEnabled(cfg.Notifications)
	if confirm(reader, "Enable all notifications (global switch)?", globOn) {
		cfg.Notifications.Enabled = boolPtr(true)
	} else {
		cfg.Notifications.Enabled = boolPtr(false)
	}

	if err := config.Validate(cfg); err != nil {
		return err
	}
	if err := config.Write(cfg, path); err != nil {
		return err
	}
	cmd.Printf("\nConfiguration saved to %s\n", path)
	return nil
}

func printWebhookStatus(cmd *cobra.Command, cfg *config.Config) {
	glob := "on"
	if !config.NotificationsEnabled(cfg.Notifications) {
		glob = "off"
	}
	cmd.Printf("  Notifications (global): %s\n", glob)

	if cfg.Notifications == nil || cfg.Notifications.Discord == nil {
		cmd.Println("  Discord: not configured")
		return
	}
	d := cfg.Notifications.Discord
	cmd.Printf("  Discord: %s\n", onOff(d.Enabled))
	if d.WebhookURL != "" {
		cmd.Printf("    Webhook URL: %s\n", maskWebhookURL(d.WebhookURL))
	} else if os.Getenv("VELBACKUPER_DISCORD_WEBHOOK_URL") != "" {
		cmd.Println("    Webhook URL: (from env)")
	} else {
		cmd.Println("    Webhook URL: (not set)")
	}
}

func maskWebhookURL(s string) string {
	const max = 50
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func onOff(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

func boolPtr(b bool) *bool {
	return &b
}
