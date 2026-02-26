package collector

import (
	"context"
	"io"
)

type Collector interface {
	Collect(ctx context.Context, jobName string, w io.Writer) error
}
