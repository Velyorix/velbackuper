# VelBackuper

CLI backup tool for databases, files, and web configs to S3-compatible storage (MinIO, AWS S3). Supports **archive** (streaming tar) and **incremental** (chunked, deduplicated) modes, Discord notifications, and systemd automation.

**Requirements:** Go 1.22+, Linux (Debian 11+, Ubuntu 20.04+) with systemd.

## Install

```bash
go build -o velbackuper .
# or: go install
```

Config path: `/etc/velbackuper/config.yaml` (override with `VELBACKUPER_CONFIG`). Recommended permissions: `0600`.

## Quick start

1. Create config: `velbackuper init`
2. Validate and run: `velbackuper validate` then `velbackuper run --all`
3. Install systemd (Linux): `velbackuper install-systemd` then `systemctl daemon-reload` and `systemctl enable velbackuper-*.timer`

## Configuration

Exactly one mode per config: **`archive`** (streaming tar → multipart S3) or **`incremental`** (chunked, BLAKE3, dedup).

Example (archive):

```yaml
mode: archive
s3:
  endpoint: "https://minio.example.com:9000"
  region: "us-east-1"
  access_key: ""
  secret_key: ""
  bucket: "mybucket"
  prefix: "backups"
jobs:
  - name: web
    enabled: true
    presets:
      nginx: true
      letsencrypt: true
    schedule:
      period: day
      times: 2
      jitter_minutes: 15
    retention:
      days: 7
      weeks: 4
      months: 12
```

S3 layout: archive uses `prefix/archives/<job>/YYYY/MM/DD/`, `prefix/manifests/<job>/`, `prefix/latest/<job>.json`. Incremental uses `prefix/objects/`, `prefix/snapshots/<job>/`, `prefix/indexes/<job>/`, `prefix/locks/`.

Jobs can use **mysql** (mysqldump), **presets** (nginx/apache/letsencrypt), and **paths** (include/exclude).

## Commands

| Command | Description |
|---------|-------------|
| `init` | Interactive wizard: mode, S3, jobs, systemd |
| `validate` | Validate configuration file |
| `run [--job name \| --all]` | Run backup |
| `list` | List backups or snapshots |
| `restore --job name --point id --target dir` | Restore from backup/snapshot |
| `prune [--job name \| --all] [--dry-run]` | Apply retention |
| `status` | Last run, next run, job state |
| `doctor` | Diagnose config, S3, locks, disk |
| `install-systemd` / `uninstall-systemd` | Install or remove systemd units |
| `enable job <name>` / `disable job <name>` | Enable or disable a job |
| `add job [--template web\|mysql\|files] [--name name]` | Add a job |

## Exit codes

| Code | Meaning |
|------|--------|
| 0 | Success |
| 1 | Config invalid |
| 2 | S3 error |
| 3 | MySQL error |
| 4 | Filesystem error |
| 5 | Lock error |
| 6 | Restore error |
| 7 | Prune error |

See [docs/exit-codes.md](docs/exit-codes.md) for details.

## Integration tests (MinIO)

```bash
docker compose -f docker-compose.test.yml up -d
go test -tags=integration ./test/integration/... -v
```

See [test/integration/README.md](test/integration/README.md) for env vars.
