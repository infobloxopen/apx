package catalog

import (
	"github.com/infobloxopen/apx/internal/config"
)

// ResolveSource builds a CatalogSource from configuration.
// Resolution order:
//  1. catalog_registries in config → AggregateSource of CachedSources
//  2. Auto-discover from org → query GHCR packages API for *-catalog
//  3. catalog_url in config → HTTPSource
//  4. Local catalog/catalog.yaml → LocalSource
func ResolveSource(cfg *config.Config) CatalogSource {
	// 1. Explicit catalog_registries
	if len(cfg.CatalogRegistries) > 0 {
		var sources []CatalogSource
		for _, reg := range cfg.CatalogRegistries {
			src := &RegistrySource{
				Org:  reg.Org,
				Repo: reg.Repo,
			}
			sources = append(sources, &CachedSource{
				Inner:    src,
				CacheDir: DefaultCacheDir(reg.Org, reg.Repo),
			})
		}
		return &AggregateSource{Sources: sources}
	}

	// 2. Auto-discover from org
	if cfg.Org != "" {
		discovered := DiscoverRegistries(cfg.Org)
		if len(discovered) > 0 {
			return &AggregateSource{Sources: discovered}
		}
	}

	// 3. catalog_url → HTTP or local source
	if cfg.CatalogURL != "" {
		return SourceFor(cfg.CatalogURL)
	}

	// 4. Local fallback
	return &LocalSource{Path: "catalog/catalog.yaml"}
}
