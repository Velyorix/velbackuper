package systemd

import (
	"fmt"
	"strings"

	"VelBackuper/internal/config"
)

const (
	DefaultUnitDir    = "/etc/systemd/system"
	DefaultBinary     = "/usr/bin/velbackuper"
	DefaultConfigPath = "/etc/velbackuper/config.yaml"
)

type GeneratorOptions struct {
	Binary     string
	ConfigPath string
	UnitDir    string
	Hardening  bool
}

type GeneratedUnits struct {
	Service string
	Timer   string
}

func Generate(job config.JobConfig, schedule *config.ScheduleConfig, opts GeneratorOptions) (*GeneratedUnits, error) {
	if schedule == nil {
		return nil, fmt.Errorf("schedule is required")
	}
	if opts.Binary == "" {
		opts.Binary = DefaultBinary
	}
	if opts.ConfigPath == "" {
		opts.ConfigPath = DefaultConfigPath
	}

	execStart := fmt.Sprintf("%s run --job %s", opts.Binary, job.Name)

	service := buildService(job.Name, execStart, opts.ConfigPath, opts.Hardening)
	timer := buildTimer(job.Name, schedule, opts.Hardening)

	return &GeneratedUnits{Service: service, Timer: timer}, nil
}

func buildService(jobName, execStart, configPath string, hardening bool) string {
	var b strings.Builder
	safeName := sanitizeUnitName(jobName)

	b.WriteString("[Unit]\n")
	b.WriteString(fmt.Sprintf("Description=VelBackuper backup for job %s\n", jobName))
	b.WriteString("After=network-online.target\n")
	b.WriteString("Wants=network-online.target\n\n")

	b.WriteString("[Service]\n")
	b.WriteString("Type=oneshot\n")
	b.WriteString(fmt.Sprintf("ExecStart=%s\n", execStart))
	b.WriteString("Environment=VELBACKUPER_CONFIG=" + configPath + "\n")

	if hardening {
		b.WriteString("ProtectSystem=full\n")
		b.WriteString("ProtectHome=read-only\n")
		b.WriteString("PrivateTmp=yes\n")
		b.WriteString("NoNewPrivileges=yes\n")
		b.WriteString("ProtectKernelTunables=yes\n")
		b.WriteString("ProtectKernelModules=yes\n")
		b.WriteString("ProtectControlGroups=yes\n")
		b.WriteString("RestrictRealtime=yes\n")
		b.WriteString("RestrictSUIDSGID=yes\n")
		b.WriteString("LockPersonality=yes\n")
		b.WriteString("PrivateMounts=yes\n")
		b.WriteString("ProtectClock=yes\n")
		b.WriteString("ProtectHostname=yes\n")
		b.WriteString("ProtectKernelLogs=yes\n")
		b.WriteString("ProtectProc=invisible\n")
		b.WriteString("ProcSubset=pid\n")
		b.WriteString("RestrictNamespaces=yes\n")
		b.WriteString("RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6\n")
		b.WriteString("RestrictFileSystems=~cgroup2 ~ext4 ~tmpfs ~squashfs\n")
	}

	b.WriteString("\n[Install]\n")
	b.WriteString("WantedBy=multi-user.target\n")

	_ = safeName
	return b.String()
}

func buildTimer(jobName string, schedule *config.ScheduleConfig, hardening bool) string {
	var b strings.Builder
	safeName := sanitizeUnitName(jobName)

	b.WriteString("[Unit]\n")
	b.WriteString(fmt.Sprintf("Description=VelBackuper timer for job %s\n", jobName))
	b.WriteString("Requires=velbackuper-" + safeName + ".service\n\n")

	b.WriteString("[Timer]\n")
	calendars := buildOnCalendar(schedule)
	for _, c := range calendars {
		b.WriteString("OnCalendar=" + c + "\n")
	}
	jitterSec := schedule.JitterMinutes * 60
	if jitterSec < 0 {
		jitterSec = 0
	}
	if jitterSec > 0 {
		b.WriteString(fmt.Sprintf("RandomizedDelaySec=%d\n", jitterSec))
	}
	b.WriteString("Persistent=yes\n\n")

	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=timers.target\n")

	_ = hardening
	return b.String()
}

func buildOnCalendar(s *config.ScheduleConfig) []string {
	times := s.Times
	if times < 1 {
		times = 1
	}
	if times > 5 {
		times = 5
	}

	switch s.Period {
	case "week":
		// weekdays: Mon=1, Tue=2, Wed=3, Thu=4, Fri=5
		days := [][]int{{1}, {1, 4}, {1, 3, 5}, {1, 2, 4, 5}, {1, 2, 3, 4, 5}}
		idx := times - 1
		var out []string
		for _, d := range days[idx] {
			dayName := []string{"", "Mon", "Tue", "Wed", "Thu", "Fri"}[d]
			out = append(out, fmt.Sprintf("%s *-*-* 02:00:00", dayName))
		}
		return out
	case "month":
		// month days: 1, 7, 14, 21, 28
		days := [][]int{{1}, {1, 15}, {1, 10, 20}, {1, 8, 15, 22}, {1, 7, 14, 21, 28}}
		idx := times - 1
		var out []string
		for _, d := range days[idx] {
			out = append(out, fmt.Sprintf("*-*-%02d 02:00:00", d))
		}
		return out
	default:
		// day: spread across 24h
		hours := [][]int{{2}, {2, 14}, {2, 10, 18}, {2, 8, 14, 20}, {2, 6, 12, 18, 22}}
		idx := times - 1
		var out []string
		for _, h := range hours[idx] {
			out = append(out, fmt.Sprintf("*-*-* %02d:00:00", h))
		}
		return out
	}
}

func sanitizeUnitName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' || r == '.' {
			b.WriteRune('-')
		}
	}
	s := b.String()
	if s == "" {
		return "default"
	}
	return s
}
