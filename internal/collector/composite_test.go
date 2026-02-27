package collector

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestCompositeCollector_RunsAll(t *testing.T) {
	var runOrder []int
	makeCollector := func(id int) Collector {
		return &funcCollector{
			fn: func(ctx context.Context, jobName string, w io.Writer) error {
				runOrder = append(runOrder, id)
				_, _ = w.Write([]byte{byte('0' + id)})
				return nil
			},
		}
	}
	comp := NewCompositeCollector(
		makeCollector(1),
		makeCollector(2),
		makeCollector(3),
	)
	var buf bytes.Buffer
	err := comp.Collect(context.Background(), "job1", &buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(runOrder) != 3 {
		t.Errorf("expected 3 collectors run, got %d", len(runOrder))
	}
	if runOrder[0] != 1 || runOrder[1] != 2 || runOrder[2] != 3 {
		t.Errorf("run order = %v", runOrder)
	}
	if buf.String() != "123" {
		t.Errorf("output = %q, want 123", buf.String())
	}
}

func TestCompositeCollector_SkipsNil(t *testing.T) {
	comp := NewCompositeCollector(nil, noopCollector{}, nil)
	var buf bytes.Buffer
	err := comp.Collect(context.Background(), "job", &buf)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCompositeCollector_EmptyNoOp(t *testing.T) {
	comp := NewCompositeCollector()
	err := comp.Collect(context.Background(), "job", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
}

type funcCollector struct {
	fn func(context.Context, string, io.Writer) error
}

func (f *funcCollector) Collect(ctx context.Context, jobName string, w io.Writer) error {
	return f.fn(ctx, jobName, w)
}
