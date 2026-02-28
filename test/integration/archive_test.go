//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"VelBackuper/internal/collector"
	"VelBackuper/internal/config"
	archiveEngine "VelBackuper/internal/engine/archive"
	"VelBackuper/internal/restore"
	"VelBackuper/internal/s3"
)

func TestMinIO_ArchiveUploadListRestorePrune(t *testing.T) {
	endpoint, accessKey, secretKey, bucket := getMinIOEnv()
	prefix := "integration-test/archive-" + time.Now().Format("20060102150405")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := s3.New(ctx, s3.Options{
		Endpoint:           endpoint,
		Region:             "us-east-1",
		AccessKey:          accessKey,
		SecretKey:          secretKey,
		Bucket:             bucket,
		Prefix:             prefix,
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("s3.New: %v", err)
	}
	if err := client.CreateBucket(ctx); err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "hello.txt"), []byte("hello world"), 0644); err != nil {
		t.Fatalf("write hello.txt: %v", err)
	}
	subDir := filepath.Join(srcDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0644); err != nil {
		t.Fatalf("write nested.txt: %v", err)
	}

	jobName := "it-job"
	c := collector.NewFilesystemCollector(collector.PathsOpts{Include: []string{srcDir}})
	stream, err := archiveEngine.Stream(ctx, c, jobName, archiveEngine.FormatGzip, 6)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	archiveKey, backupID, err := archiveEngine.Upload(ctx, client, jobName, archiveEngine.FormatGzip, stream, archiveEngine.UploadOptions{PartSizeMB: 5})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if archiveKey == "" || backupID == "" {
		t.Fatal("Upload returned empty key or backupID")
	}

	host, _ := os.Hostname()
	if host == "" {
		host = "localhost"
	}
	if err := archiveEngine.WriteManifest(ctx, client, archiveEngine.Manifest{
		Job: jobName, Timestamp: backupID, Key: archiveKey, Host: host, Format: "tar.gz",
	}); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	if err := archiveEngine.WriteLatest(ctx, client, jobName, backupID, archiveKey); err != nil {
		t.Fatalf("WriteLatest: %v", err)
	}

	keys, err := client.ListObjects(ctx, s3.ArchivesPrefix, 100)
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}
	var found bool
	for _, k := range keys {
		if k == archiveKey {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListObjects: archive key %q not in %v", archiveKey, keys)
	}

	restoreDir := t.TempDir()
	if err := restore.RestoreArchive(ctx, client, archiveKey, restoreDir, restore.ArchiveRestoreOptions{}); err != nil {
		t.Fatalf("RestoreArchive: %v", err)
	}
	helloPath := filepath.Join(restoreDir, filepath.Base(srcDir), "hello.txt")
	data, err := os.ReadFile(helloPath)
	if err != nil {
		t.Fatalf("ReadFile restored hello.txt: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("restored hello.txt = %q, want hello world", data)
	}
	nestedPath := filepath.Join(restoreDir, filepath.Base(srcDir), "sub", "nested.txt")
	data2, err := os.ReadFile(nestedPath)
	if err != nil {
		t.Fatalf("ReadFile restored nested.txt: %v", err)
	}
	if string(data2) != "nested" {
		t.Errorf("restored nested.txt = %q, want nested", data2)
	}

	oldTS := "20000101120000"
	oldArchiveKey := s3.ArchiveObjectKey(jobName, "2000", "01", "01", "backup-old-20000101120000.tar.gz")
	if err := archiveEngine.WriteManifest(ctx, client, archiveEngine.Manifest{
		Job: jobName, Timestamp: oldTS, Key: oldArchiveKey, Host: host, Format: "tar.gz",
	}); err != nil {
		t.Fatalf("WriteManifest old: %v", err)
	}
	now := time.Now().UTC()
	retention := &config.RetentionConfig{Days: 7}
	deleted, err := archiveEngine.ApplyRetention(ctx, client, jobName, retention, now)
	if err != nil {
		t.Fatalf("ApplyRetention: %v", err)
	}
	if deleted < 1 {
		t.Errorf("ApplyRetention: deleted = %d, want at least 1 (old manifest)", deleted)
	}
}
