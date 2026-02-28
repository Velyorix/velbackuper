package systemd

import (
	"strings"
	"testing"

	"VelBackuper/internal/config"
)

func TestGenerate_ServiceAndTimer(t *testing.T) {
	job := config.JobConfig{Name: "web-prod", Enabled: true}
	schedule := &config.ScheduleConfig{
		Period:        "day",
		Times:         2,
		JitterMinutes: 5,
	}
	opts := GeneratorOptions{
		Binary:     "/usr/bin/velbackuper",
		ConfigPath: "/etc/velbackuper/config.yaml",
		Hardening:  true,
	}

	units, err := Generate(job, schedule, opts)
	if err != nil {
		t.Fatal(err)
	}
	if units == nil {
		t.Fatal("units nil")
	}

	if !strings.Contains(units.Service, "[Unit]") {
		t.Error("service missing [Unit]")
	}
	if !strings.Contains(units.Service, "[Service]") {
		t.Error("service missing [Service]")
	}
	if !strings.Contains(units.Service, "ExecStart=/usr/bin/velbackuper run --job web-prod") {
		t.Errorf("service ExecStart wrong: %s", units.Service)
	}
	if !strings.Contains(units.Service, "ProtectSystem=full") {
		t.Error("service missing hardening")
	}
	if !strings.Contains(units.Service, "VELBACKUPER_CONFIG") {
		t.Error("service missing config env")
	}

	if !strings.Contains(units.Timer, "[Timer]") {
		t.Error("timer missing [Timer]")
	}
	if !strings.Contains(units.Timer, "OnCalendar=") {
		t.Error("timer missing OnCalendar")
	}
	if !strings.Contains(units.Timer, "RandomizedDelaySec=300") {
		t.Error("timer missing jitter (5*60=300)")
	}
}

func TestGenerate_NilSchedule_Error(t *testing.T) {
	job := config.JobConfig{Name: "x"}
	_, err := Generate(job, nil, GeneratorOptions{})
	if err == nil {
		t.Error("expected error for nil schedule")
	}
}

func TestBuildOnCalendar_Day(t *testing.T) {
	s := &config.ScheduleConfig{Period: "day", Times: 3}
	cal := buildOnCalendar(s)
	if len(cal) != 3 {
		t.Errorf("day times=3: got %d calendars", len(cal))
	}
}

func TestBuildOnCalendar_Week(t *testing.T) {
	s := &config.ScheduleConfig{Period: "week", Times: 2}
	cal := buildOnCalendar(s)
	if len(cal) != 2 {
		t.Errorf("week times=2: got %d calendars", len(cal))
	}
}

func TestSanitizeUnitName(t *testing.T) {
	if got := sanitizeUnitName("web-prod"); got != "web-prod" {
		t.Errorf("sanitize web-prod = %q", got)
	}
	if got := sanitizeUnitName("my job"); got != "my-job" {
		t.Errorf("sanitize 'my job' = %q", got)
	}
	if got := sanitizeUnitName(""); got != "default" {
		t.Errorf("sanitize empty = %q", got)
	}
}
