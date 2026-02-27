package collector

import (
	"context"
	"io"
)

type CompositeCollector struct {
	collectors []Collector
}

func NewCompositeCollector(collectors ...Collector) *CompositeCollector {
	var c []Collector
	for _, col := range collectors {
		if col != nil {
			c = append(c, col)
		}
	}
	return &CompositeCollector{collectors: c}
}

func (c *CompositeCollector) Collect(ctx context.Context, jobName string, w io.Writer) error {
	for _, col := range c.collectors {
		if err := col.Collect(ctx, jobName, w); err != nil {
			return err
		}
	}
	return nil
}

var _ Collector = (*CompositeCollector)(nil)
