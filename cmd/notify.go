package cmd

import (
	"VelBackuper/internal/config"
	"VelBackuper/internal/notifier"
)

// NotifierFromConfig builds a Notifier from cfg. If notifications are disabled or Discord is misconfigured,
// returns nil. When Discord is configured but invalid (e.g. missing webhook_url), warn is called with the error message.
func NotifierFromConfig(cfg *config.Config, warn func(string)) notifier.Notifier {
	if cfg == nil || !config.NotificationsEnabled(cfg.Notifications) {
		return nil
	}
	if cfg.Notifications.Discord == nil {
		return nil
	}
	n, err := notifier.NewDiscordNotifier(cfg.Notifications.Discord)
	if err != nil {
		if warn != nil {
			warn("discord notification: " + err.Error())
		}
		return nil
	}
	return n
}
