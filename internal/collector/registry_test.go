package collector

import (
	"context"
	"io"
	"testing"
)

type noopCollector struct{}

func (n noopCollector) Collect(ctx context.Context, jobName string, w io.Writer) error {
	return nil
}

func TestRegistry_RegisterGet(t *testing.T) {
	r := NewRegistry()
	c := noopCollector{}
	r.Register("noop", c)
	got, ok := r.Get("noop")
	if !ok || got == nil {
		t.Fatal("Get(noop) should return the collector")
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get("missing")
	if ok {
		t.Error("Get(missing) should return false")
	}
}

func TestRegistry_Names(t *testing.T) {
	r := NewRegistry()
	r.Register("a", noopCollector{})
	r.Register("b", noopCollector{})
	names := r.Names()
	if len(names) != 2 {
		t.Fatalf("Names() len = %d, want 2", len(names))
	}
	seen := make(map[string]bool)
	for _, n := range names {
		seen[n] = true
	}
	if !seen["a"] || !seen["b"] {
		t.Errorf("Names() = %v", names)
	}
}

func TestRegistry_RegisterNilIgnored(t *testing.T) {
	r := NewRegistry()
	r.Register("nil", nil)
	_, ok := r.Get("nil")
	if ok {
		t.Error("Register(nil) should not store")
	}
}

func TestRegistry_Overwrite(t *testing.T) {
	r := NewRegistry()
	r.Register("x", noopCollector{})
	r.Register("x", noopCollector{})
	_, ok := r.Get("x")
	if !ok {
		t.Error("second Register should overwrite")
	}
	if len(r.Names()) != 1 {
		t.Errorf("Names() len = %d, want 1", len(r.Names()))
	}
}
