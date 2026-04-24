package marketnameregistry

import (
	"slices"
	"sync"
)

// Manages active market names with thread-safe access.
// Encapsulates a map and its mutex so embedding types need not add their own.
type MarketNameRegistry struct {
	mu    sync.RWMutex
	names map[string]struct{}
}

// Constructs from a copy of the given slice so callers cannot mutate internal state afterward.
// Duplicate entries in initial are collapsed to a single key per name.
func New(initial []string) *MarketNameRegistry {
	r := &MarketNameRegistry{names: make(map[string]struct{}, len(initial))}
	for _, name := range initial {
		r.names[name] = struct{}{}
	}
	return r
}

// Records a market name. Duplicate adds are ignored (set semantics).
func (r *MarketNameRegistry) Add(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.names == nil {
		r.names = make(map[string]struct{})
	}
	r.names[name] = struct{}{}
}

// Deletes a market name from the set. Reports whether it was present.
func (r *MarketNameRegistry) Remove(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.names[name]; !ok {
		return false
	}
	delete(r.names, name)
	return true
}

// Reports whether the given market name is present.
func (r *MarketNameRegistry) Contains(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.names[name]
	return ok
}

// Produces a sorted copy of the current names. Safe to iterate without holding locks.
func (r *MarketNameRegistry) Snapshot() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.names) == 0 {
		return nil
	}
	out := make([]string, 0, len(r.names))
	for k := range r.names {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// Reports how many names are held.
func (r *MarketNameRegistry) len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.names)
}
