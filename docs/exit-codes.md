# Exit codes

VelBackuper uses the following exit codes so scripts and automation can detect failure type.

| Code | Meaning | When |
|------|--------|------|
| **0** | Success | Command completed without error. |
| **1** | Config invalid | Configuration file missing, unreadable, or validation failed (e.g. invalid mode, missing S3). |
| **2** | S3 error | S3/MinIO connection, upload, download, list, or delete failed. |
| **3** | MySQL error | mysqldump or MySQL collector failed (e.g. connection, dump error). |
| **4** | Filesystem error | Reading source paths, creating temp files, or writing restore target failed. |
| **5** | Lock error | Failed to acquire or release local or S3 lock (e.g. another run in progress, lock dir not writable). |
| **6** | Restore error | Restore failed (e.g. backup not found, extract error, target not writable). |
| **7** | Prune error | Retention or GC failed (e.g. S3 delete error during prune). |

All CLI commands must exit with one of these codes so that callers (e.g. systemd, cron, scripts) can react appropriately (retry, alert, log).
