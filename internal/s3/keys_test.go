package s3

import (
	"testing"
)

func TestArchiveObjectKey(t *testing.T) {
	got := ArchiveObjectKey("web-prod", "2025", "02", "26", "backup-host-123.tar.gz")
	want := "archives/web-prod/2025/02/26/backup-host-123.tar.gz"
	if got != want {
		t.Errorf("ArchiveObjectKey = %q, want %q", got, want)
	}
}

func TestManifestKey(t *testing.T) {
	got := ManifestKey("web-prod", "20250226120000")
	want := "manifests/web-prod/20250226120000.json"
	if got != want {
		t.Errorf("ManifestKey = %q, want %q", got, want)
	}
}

func TestLatestKey(t *testing.T) {
	got := LatestKey("web-prod")
	want := "latest/web-prod.json"
	if got != want {
		t.Errorf("LatestKey = %q, want %q", got, want)
	}
}

func TestSnapshotKey(t *testing.T) {
	got := SnapshotKey("job1", "20250226120000")
	want := "snapshots/job1/20250226120000.json"
	if got != want {
		t.Errorf("SnapshotKey = %q, want %q", got, want)
	}
}

func TestIndexKey(t *testing.T) {
	got := IndexKey("job1", "20250226120000")
	want := "indexes/job1/20250226120000.json"
	if got != want {
		t.Errorf("IndexKey = %q, want %q", got, want)
	}
}

func TestObjectKey(t *testing.T) {
	got := ObjectKey("ab", "abcd1234")
	want := "objects/ab/abcd1234"
	if got != want {
		t.Errorf("ObjectKey = %q, want %q", got, want)
	}
}

func TestLockKey(t *testing.T) {
	got := LockKey("web-prod")
	want := "locks/web-prod.lock"
	if got != want {
		t.Errorf("LockKey = %q, want %q", got, want)
	}
}

func TestParseArchiveKey(t *testing.T) {
	key := "archives/web-prod/2025/02/26/backup-host-123.tar.gz"
	job, yyyy, mm, dd, filename := ParseArchiveKey(key)
	if job != "web-prod" || yyyy != "2025" || mm != "02" || dd != "26" || filename != "backup-host-123.tar.gz" {
		t.Errorf("ParseArchiveKey(%q) = %q,%q,%q,%q,%q", key, job, yyyy, mm, dd, filename)
	}
}

func TestParseArchiveKey_Invalid(t *testing.T) {
	job, yyyy, mm, dd, filename := ParseArchiveKey("other/path")
	if job != "" || yyyy != "" || mm != "" || dd != "" || filename != "" {
		t.Errorf("ParseArchiveKey(invalid) should return empty: %q,%q,%q,%q,%q", job, yyyy, mm, dd, filename)
	}
}

func TestParseArchiveKey_TooShort(t *testing.T) {
	job, yyyy, mm, dd, filename := ParseArchiveKey("archives/job/2025")
	if job != "" || filename != "" {
		t.Errorf("ParseArchiveKey(too short) should return empty job/filename: %q,%q", job, filename)
	}
}

func TestSnapshotsPrefixForJob(t *testing.T) {
	got := SnapshotsPrefixForJob("job1")
	want := "snapshots/job1/"
	if got != want {
		t.Errorf("SnapshotsPrefixForJob = %q, want %q", got, want)
	}
}

func TestArchivesPrefixForJob(t *testing.T) {
	got := ArchivesPrefixForJob("web-prod")
	want := "archives/web-prod/"
	if got != want {
		t.Errorf("ArchivesPrefixForJob = %q, want %q", got, want)
	}
}

func TestClientKey_WithPrefix(t *testing.T) {
	client := &Client{prefix: "backup/db"}
	full := client.Key(ArchiveObjectKey("job1", "2025", "02", "26", "x.tar.gz"))
	want := "backup/db/archives/job1/2025/02/26/x.tar.gz"
	if full != want {
		t.Errorf("Client.Key(ArchiveObjectKey(...)) = %q, want %q", full, want)
	}
}
