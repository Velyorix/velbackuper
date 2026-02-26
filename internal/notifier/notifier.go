package notifier

import (
	"context"
	"time"
)

type Notifier interface {
	NotifyStart(ctx context.Context, jobName, backupID string) error
	NotifySuccess(ctx context.Context, jobName, backupID string, duration time.Duration, size int64) error
	NotifyWarning(ctx context.Context, jobName, backupID, message string) error
	NotifyError(ctx context.Context, jobName, backupID string, err error) error
	NotifyPrune(ctx context.Context, jobName string, retained, deleted int) error
	NotifyRestore(ctx context.Context, jobName, pointID, targetDir string) error
}
