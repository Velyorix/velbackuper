package archive

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestArchiveKey_LayoutAndTimestamp(t *testing.T) {
	at := time.Date(2025, 2, 26, 12, 0, 5, 0, time.UTC)
	key, ts := ArchiveKey("web-prod", FormatGzip, at)
	if ts != "20250226120005" {
		t.Errorf("timestamp = %q, want 20250226120005", ts)
	}
	// Key: archives/job/YYYY/MM/DD/backup-<host>-<ts>.tar.gz
	layoutRe := regexp.MustCompile(`^archives/web-prod/2025/02/26/backup-[a-zA-Z0-9._-]+-20250226120005\.tar\.gz$`)
	if !layoutRe.MatchString(key) {
		t.Errorf("key %q does not match expected layout archives/job/YYYY/MM/DD/backup-host-ts.tar.gz", key)
	}
}

func TestArchiveKey_Formats(t *testing.T) {
	at := time.Date(2025, 1, 15, 8, 30, 0, 0, time.UTC)
	t.Run("tar", func(t *testing.T) {
		key, _ := ArchiveKey("job1", FormatTar, at)
		if key != "" && !strings.HasSuffix(key, ".tar") {
			t.Errorf("FormatTar key should end with .tar: %q", key)
		}
	})
	t.Run("gz", func(t *testing.T) {
		key, _ := ArchiveKey("job1", FormatGzip, at)
		if key != "" && !strings.HasSuffix(key, ".tar.gz") {
			t.Errorf("FormatGzip key should end with .tar.gz: %q", key)
		}
	})
	t.Run("zst", func(t *testing.T) {
		key, _ := ArchiveKey("job1", FormatZstd, at)
		if key != "" && !strings.HasSuffix(key, ".tar.zst") {
			t.Errorf("FormatZstd key should end with .tar.zst: %q", key)
		}
	})
}

func TestArchiveKey_DateParts(t *testing.T) {
	at := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	key, _ := ArchiveKey("x", FormatTar, at)
	if key == "" {
		t.Fatal("key empty")
	}
	// Must contain YYYY/MM/DD
	if !regexp.MustCompile(`/2026/12/01/`).MatchString(key) {
		t.Errorf("key should contain /2026/12/01/: %q", key)
	}
}
