package collector

import (
	"sync"
)

type Registry struct {
	mu     sync.RWMutex
	byName map[string]Collector
}

func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]Collector)}
}

func (r *Registry) Register(name string, c Collector) {
	if c == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byName[name] = c
}

func (r *Registry) Get(name string) (Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.byName[name]
	return c, ok
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.byName))
	for n := range r.byName {
		names = append(names, n)
	}
	return names
}
