package client

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = make(map[string]Generator)
)

// Register adds a generator to the global registry. It is called from init()
// in each generator file. It panics if a generator with the same Name() is
// already registered.
func Register(g Generator) {
	mu.Lock()
	defer mu.Unlock()
	name := g.Name()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("client generator %q already registered", name))
	}
	registry[name] = g
}

// Get returns the generator registered under the given name, or nil if none.
func Get(name string) Generator {
	mu.RLock()
	defer mu.RUnlock()
	return registry[name]
}

// Names returns the names of all registered generators, sorted.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// resetForTesting saves the current registry and replaces it with an empty one.
// Returns a restore function that must be called (typically via defer) to
// restore the original registry. Only for use in tests.
func resetForTesting() func() {
	mu.Lock()
	defer mu.Unlock()
	saved := registry
	registry = make(map[string]Generator)
	return func() {
		mu.Lock()
		defer mu.Unlock()
		registry = saved
	}
}
