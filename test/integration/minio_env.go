//go:build integration

package integration

import (
	"os"
	"strings"
)

func getMinIOEnv() (endpoint, accessKey, secretKey, bucket string) {
	endpoint = os.Getenv("VELBACKUPER_MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:9000"
	}
	accessKey = os.Getenv("VELBACKUPER_MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}
	secretKey = os.Getenv("VELBACKUPER_MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin"
	}
	bucket = os.Getenv("VELBACKUPER_MINIO_BUCKET")
	if bucket == "" {
		bucket = "velbackuper-test"
	}
	return strings.TrimSuffix(endpoint, "/"), accessKey, secretKey, bucket
}
