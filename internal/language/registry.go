package language

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = make(map[string]LanguagePlugin)
)

// Register adds a plugin to the global registry. It is called from init()
// in each plugin file. Panics if a plugin with the same Name() is already registered.
func Register(p LanguagePlugin) {
	mu.Lock()
	defer mu.Unlock()
	name := p.Name()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("language plugin %q already registered", name))
	}
	registry[name] = p
}

// Get returns the plugin registered under the given name, or nil if not found.
func Get(name string) LanguagePlugin {
	mu.RLock()
	defer mu.RUnlock()
	return registry[name]
}

// All returns all registered plugins sorted by display order:
// Tier 1 first, then Tier 2 alphabetically by name.
func All() []LanguagePlugin {
	mu.RLock()
	defer mu.RUnlock()
	plugins := make([]LanguagePlugin, 0, len(registry))
	for _, p := range registry {
		plugins = append(plugins, p)
	}
	sort.Slice(plugins, func(i, j int) bool {
		if plugins[i].Tier() != plugins[j].Tier() {
			return plugins[i].Tier() < plugins[j].Tier()
		}
		return plugins[i].Name() < plugins[j].Name()
	})
	return plugins
}

// Available returns plugins that are available for the given context,
// sorted in display order.
func Available(ctx DerivationContext) []LanguagePlugin {
	all := All()
	result := make([]LanguagePlugin, 0, len(all))
	for _, p := range all {
		if p.Available(ctx) {
			result = append(result, p)
		}
	}
	return result
}

// Names returns the names of all registered plugins in display order.
// Useful for generating help text and CLI usage strings.
func Names() []string {
	all := All()
	names := make([]string, len(all))
	for i, p := range all {
		names[i] = p.Name()
	}
	return names
}

// resetForTesting saves the current registry and replaces it with an empty one.
// Returns a restore function that must be called (typically via defer) to
// restore the original registry. Only for use in tests.
func resetForTesting() func() {
	mu.Lock()
	defer mu.Unlock()
	saved := registry
	registry = make(map[string]LanguagePlugin)
	return func() {
		mu.Lock()
		defer mu.Unlock()
		registry = saved
	}
}
