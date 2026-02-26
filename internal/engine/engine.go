package engine

import (
	"context"
)

type Engine interface {
	Run(ctx context.Context, jobName string) (backupID string, err error)
	List(ctx context.Context, jobName string) ([]ListEntry, error)
	Restore(ctx context.Context, jobName, pointID, targetDir string) error
	Prune(ctx context.Context, jobName string) error
}

type ListEntry struct {
	ID        string
	JobName   string
	Timestamp string
	Size      int64
}
