package catalog

import (
	"os"

	"github.com/infobloxopen/apx/internal/config"
)

// localFileExists reports whether path exists and is a regular file.
func localFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// ResolveSource builds a CatalogSource from configuration.
// Resolution order:
//  1. catalog_registries in config → AggregateSource of CachedSources
//  2. Auto-discover from org → query GHCR packages API for *-catalog
//  3. catalog_url in config → HTTPSource
//  4. Local catalog/catalog.yaml → LocalSource
func ResolveSource(cfg *config.Config) CatalogSource {
	return ResolveSourceWithGlobal(cfg, nil)
}

// ResolveSourceWithGlobal builds a CatalogSource from local and global config.
// Resolution order:
//  0. Local catalog/catalog.yaml (if it exists on disk — canonical repo)
//  1. catalog_registries in local config → AggregateSource of CachedSources
//  2. Auto-discover from local config org → query GHCR packages API
//  3. Global config known orgs/repos → RegistrySources (no API call needed)
//  4. catalog_url in local config → HTTPSource
//  5. Local catalog/catalog.yaml → LocalSource (fallback even if not on disk)
func ResolveSourceWithGlobal(cfg *config.Config, globalCfg *config.GlobalConfig) CatalogSource {
	// 0. If a local catalog file exists on disk, use it directly.
	// This is the common case inside a canonical repo where
	// `apx catalog generate` has already been run.
	if localFileExists("catalog/catalog.yaml") {
		return &LocalSource{Path: "catalog/catalog.yaml"}
	}

	// 1. Explicit catalog_registries
	if cfg != nil && len(cfg.CatalogRegistries) > 0 {
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

	// 2. Auto-discover from local config org
	if cfg != nil && cfg.Org != "" {
		discovered := DiscoverRegistries(cfg.Org)
		if len(discovered) > 0 {
			return &AggregateSource{Sources: discovered}
		}
	}

	// 3. Global config: build sources from known orgs/repos.
	// This avoids an API call on every search — repos were discovered
	// during `apx auth login` and cached in the global config.
	if globalCfg != nil && len(globalCfg.Orgs) > 0 {
		var sources []CatalogSource
		localOrg := ""
		if cfg != nil {
			localOrg = cfg.Org
		}

		// Prioritize the local org's catalogs first
		for _, org := range globalCfg.Orgs {
			if org.Name == localOrg {
				for _, repo := range org.Repos {
					sources = append(sources, &CachedSource{
						Inner:    &RegistrySource{Org: org.Name, Repo: repo},
						CacheDir: DefaultCacheDir(org.Name, repo),
					})
				}
				break
			}
		}

		// Then add other orgs
		for _, org := range globalCfg.Orgs {
			if org.Name == localOrg {
				continue // already added above
			}
			for _, repo := range org.Repos {
				sources = append(sources, &CachedSource{
					Inner:    &RegistrySource{Org: org.Name, Repo: repo},
					CacheDir: DefaultCacheDir(org.Name, repo),
				})
			}
		}

		if len(sources) > 0 {
			return &AggregateSource{Sources: sources}
		}
	}

	// 4. catalog_url → HTTP or local source
	if cfg != nil && cfg.CatalogURL != "" {
		return SourceFor(cfg.CatalogURL)
	}

	// 5. Local fallback
	return &LocalSource{Path: "catalog/catalog.yaml"}
}
