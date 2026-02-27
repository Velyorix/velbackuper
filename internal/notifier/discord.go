package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"VelBackuper/internal/config"
)

type DiscordNotifier struct {
	webhookURL string
	timeout    time.Duration
	retry      *config.DiscordRetry
	mentions   *config.DiscordMentions
	level      string
	events     map[string]struct{}
	host       string
	client     *http.Client
}

type discordEmbed struct {
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Color       int            `json:"color,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
	Fields      []discordField `json:"fields,omitempty"`
	Footer      *discordFooter `json:"footer,omitempty"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordFooter struct {
	Text string `json:"text,omitempty"`
}

type discordPayload struct {
	Content string         `json:"content,omitempty"`
	Embeds  []discordEmbed `json:"embeds,omitempty"`
}

func NewDiscordNotifier(cfg *config.DiscordConfig) (*DiscordNotifier, error) {
	if cfg == nil || !cfg.Enabled || cfg.WebhookURL == "" {
		return nil, fmt.Errorf("discord notifier disabled or missing webhook_url")
	}
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown"
	}
	timeout := 10 * time.Second
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}
	events := make(map[string]struct{})
	for _, e := range cfg.Events {
		events[e] = struct{}{}
	}
	return &DiscordNotifier{
		webhookURL: cfg.WebhookURL,
		timeout:    timeout,
		retry:      cfg.Retry,
		mentions:   cfg.Mentions,
		level:      cfg.Level,
		events:     events,
		host:       host,
		client:     &http.Client{Timeout: timeout},
	}, nil
}

func (d *DiscordNotifier) allowed(event string) bool {
	if len(d.events) == 0 {
		return true
	}
	_, ok := d.events[event]
	return ok
}

func (d *DiscordNotifier) send(ctx context.Context, embed discordEmbed, mention string) error {
	if d.webhookURL == "" {
		return nil
	}
	payload := discordPayload{
		Content: mention,
		Embeds:  []discordEmbed{embed},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	attempts := 1
	delay := 0 * time.Millisecond
	if d.retry != nil && d.retry.Attempts > 1 {
		attempts = d.retry.Attempts
		delay = time.Duration(d.retry.BackoffMs) * time.Millisecond
	}
	for i := 0; i < attempts; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := d.client.Do(req)
		if err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			_ = resp.Body.Close()
			return nil
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		if delay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return fmt.Errorf("discord webhook failed after %d attempts", attempts)
}

func (d *DiscordNotifier) NotifyStart(ctx context.Context, jobName, backupID string) error {
	if !d.allowed("start") {
		return nil
	}
	embed := discordEmbed{
		Title:     "Backup started",
		Color:     0x3498db,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Fields: []discordField{
			{Name: "Host", Value: d.host, Inline: true},
			{Name: "Job", Value: jobName, Inline: true},
			{Name: "Backup ID", Value: backupID, Inline: true},
		},
	}
	return d.send(ctx, embed, "")
}

func (d *DiscordNotifier) NotifySuccess(ctx context.Context, jobName, backupID string, duration time.Duration, size int64) error {
	if !d.allowed("success") {
		return nil
	}
	embed := discordEmbed{
		Title:     "Backup success",
		Color:     0x2ecc71,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Fields: []discordField{
			{Name: "Host", Value: d.host, Inline: true},
			{Name: "Job", Value: jobName, Inline: true},
			{Name: "Backup ID", Value: backupID, Inline: true},
			{Name: "Duration", Value: duration.String(), Inline: true},
			{Name: "Size", Value: fmt.Sprintf("%d bytes", size), Inline: true},
		},
	}
	return d.send(ctx, embed, "")
}

func (d *DiscordNotifier) NotifyWarning(ctx context.Context, jobName, backupID, message string) error {
	if !d.allowed("warning") {
		return nil
	}
	mention := ""
	if d.mentions != nil && d.mentions.OnError != "" {
		mention = d.mentions.OnError
	}
	embed := discordEmbed{
		Title:       "Backup warning",
		Description: message,
		Color:       0xf1c40f,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Fields: []discordField{
			{Name: "Host", Value: d.host, Inline: true},
			{Name: "Job", Value: jobName, Inline: true},
			{Name: "Backup ID", Value: backupID, Inline: true},
		},
	}
	return d.send(ctx, embed, mention)
}

func (d *DiscordNotifier) NotifyError(ctx context.Context, jobName, backupID string, err error) error {
	if !d.allowed("error") {
		return nil
	}
	mention := ""
	if d.mentions != nil && d.mentions.OnError != "" {
		mention = d.mentions.OnError
	}
	embed := discordEmbed{
		Title:       "Backup failed",
		Description: err.Error(),
		Color:       0xe74c3c,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Fields: []discordField{
			{Name: "Host", Value: d.host, Inline: true},
			{Name: "Job", Value: jobName, Inline: true},
			{Name: "Backup ID", Value: backupID, Inline: true},
		},
	}
	return d.send(ctx, embed, mention)
}

func (d *DiscordNotifier) NotifyPrune(ctx context.Context, jobName string, retained, deleted int) error {
	if !d.allowed("prune") {
		return nil
	}
	embed := discordEmbed{
		Title:     "Prune completed",
		Color:     0x9b59b6,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Fields: []discordField{
			{Name: "Host", Value: d.host, Inline: true},
			{Name: "Job", Value: jobName, Inline: true},
			{Name: "Retained", Value: fmt.Sprintf("%d", retained), Inline: true},
			{Name: "Deleted", Value: fmt.Sprintf("%d", deleted), Inline: true},
		},
	}
	return d.send(ctx, embed, "")
}

func (d *DiscordNotifier) NotifyRestore(ctx context.Context, jobName, pointID, targetDir string) error {
	if !d.allowed("restore") {
		return nil
	}
	embed := discordEmbed{
		Title:     "Restore completed",
		Color:     0x1abc9c,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Fields: []discordField{
			{Name: "Host", Value: d.host, Inline: true},
			{Name: "Job", Value: jobName, Inline: true},
			{Name: "Point", Value: pointID, Inline: true},
			{Name: "Target", Value: targetDir, Inline: false},
		},
	}
	return d.send(ctx, embed, "")
}
