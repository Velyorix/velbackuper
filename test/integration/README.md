# Integration tests (MinIO)

These tests require a running MinIO (S3-compatible) server. They are built only with the `integration` build tag.

## Run MinIO

```bash
docker compose -f docker-compose.test.yml up -d
```

Wait a few seconds for MinIO to be ready.

## Run tests

```bash
go test -tags=integration ./test/integration/... -v
```

## Environment (optional)

| Variable | Default |
|----------|---------|
| `VELBACKUPER_MINIO_ENDPOINT` | `http://localhost:9000` |
| `VELBACKUPER_MINIO_ACCESS_KEY` | `minioadmin` |
| `VELBACKUPER_MINIO_SECRET_KEY` | `minioadmin` |
| `VELBACKUPER_MINIO_BUCKET` | `velbackuper-test` |

The bucket is created by the tests if it does not exist.
