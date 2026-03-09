package catalog

import (
	"fmt"
	"strings"
)

// AggregateSource loads catalogs from multiple sources and merges them.
// Modules are deduplicated by a composite key of (org + repo + module ID).
// When duplicates exist, the first source (leftmost) wins.
type AggregateSource struct {
	Sources []CatalogSource
}

// Load fetches catalogs from all sources, merges, and returns a unified catalog.
// Individual source errors are collected but do not prevent other sources
// from being tried. Returns an error only if ALL sources fail.
func (a *AggregateSource) Load() (*Catalog, error) {
	if len(a.Sources) == 0 {
		return &Catalog{Version: 1, Modules: []Module{}}, nil
	}

	var (
		allModules []Module
		seen       = make(map[string]bool) // dedupe key: "org/repo/moduleID"
		errs       []string
		anySuccess bool
	)

	for _, src := range a.Sources {
		cat, err := src.Load()
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", src.Name(), err))
			continue
		}
		if cat == nil {
			continue
		}
		anySuccess = true

		for _, m := range cat.Modules {
			key := cat.Org + "/" + cat.Repo + "/" + m.ID
			if seen[key] {
				continue
			}
			seen[key] = true
			allModules = append(allModules, m)
		}
	}

	if !anySuccess && len(errs) > 0 {
		return nil, fmt.Errorf("all catalog sources failed: %s", strings.Join(errs, "; "))
	}

	return &Catalog{
		Version: 1,
		Modules: allModules,
	}, nil
}

// Name returns a summary of all source names.
func (a *AggregateSource) Name() string {
	names := make([]string, len(a.Sources))
	for i, s := range a.Sources {
		names[i] = s.Name()
	}
	return "aggregate[" + strings.Join(names, ", ") + "]"
}
