package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultCacheTTL is the default time-to-live for cached catalog data.
const DefaultCacheTTL = 5 * time.Minute

// CachedSource wraps a CatalogSource with local disk caching.
// If the inner source fails but a stale cache exists, it returns the
// stale data rather than failing. This provides offline resilience.
type CachedSource struct {
	Inner    CatalogSource
	CacheDir string        // e.g. ~/.cache/apx/catalogs/acme/apis
	TTL      time.Duration // default: DefaultCacheTTL

	// NowFn overrides time.Now for testing.
	NowFn func() time.Time
}

// cacheMeta stores freshness metadata alongside the cached catalog.
type cacheMeta struct {
	FetchedAt time.Time `json:"fetched_at"`
	Source    string    `json:"source"`
}

// Load returns the catalog, using the cache when fresh and falling back
// to stale cache when the inner source is unreachable.
func (c *CachedSource) Load() (*Catalog, error) {
	ttl := c.TTL
	if ttl == 0 {
		ttl = DefaultCacheTTL
	}

	// Try cached copy first
	meta, cat, cacheErr := c.readCache()
	if cacheErr == nil && cat != nil {
		age := c.now().Sub(meta.FetchedAt)
		if age < ttl {
			return cat, nil // cache is fresh
		}
	}

	// Cache is stale or missing — try the inner source
	freshCat, fetchErr := c.Inner.Load()
	if fetchErr == nil && freshCat != nil {
		// Update cache with fresh data
		_ = c.writeCache(freshCat)
		return freshCat, nil
	}

	// Inner source failed — fall back to stale cache
	if cat != nil {
		return cat, nil
	}

	// Nothing available
	if fetchErr != nil {
		return nil, fmt.Errorf("fetch from %s failed and no cached copy available: %w", c.Inner.Name(), fetchErr)
	}
	return &Catalog{Version: 1, Modules: []Module{}}, nil
}

// Name returns the inner source's name with a "(cached)" suffix.
func (c *CachedSource) Name() string {
	return c.Inner.Name() + " (cached)"
}

// ---------------------------------------------------------------------------
// Cache I/O
// ---------------------------------------------------------------------------

func (c *CachedSource) catalogPath() string {
	return filepath.Join(c.CacheDir, "catalog.yaml")
}

func (c *CachedSource) metaPath() string {
	return filepath.Join(c.CacheDir, "meta.json")
}

func (c *CachedSource) readCache() (*cacheMeta, *Catalog, error) {
	// Read metadata
	metaData, err := os.ReadFile(c.metaPath())
	if err != nil {
		return nil, nil, err
	}
	var meta cacheMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return nil, nil, err
	}

	// Read catalog
	catData, err := os.ReadFile(c.catalogPath())
	if err != nil {
		return nil, nil, err
	}
	var cat Catalog
	if err := yaml.Unmarshal(catData, &cat); err != nil {
		return nil, nil, err
	}

	return &meta, &cat, nil
}

func (c *CachedSource) writeCache(cat *Catalog) error {
	if err := os.MkdirAll(c.CacheDir, 0o755); err != nil {
		return err
	}

	// Write catalog YAML
	catData, err := yaml.Marshal(cat)
	if err != nil {
		return err
	}
	if err := os.WriteFile(c.catalogPath(), catData, 0o644); err != nil {
		return err
	}

	// Write metadata
	meta := cacheMeta{
		FetchedAt: c.now(),
		Source:    c.Inner.Name(),
	}
	metaData, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(c.metaPath(), metaData, 0o644)
}

func (c *CachedSource) now() time.Time {
	if c.NowFn != nil {
		return c.NowFn()
	}
	return time.Now()
}

// ---------------------------------------------------------------------------
// CacheDir helpers
// ---------------------------------------------------------------------------

// DefaultCacheDir returns the default catalog cache directory for a given
// org and repo: ~/.cache/apx/catalogs/<org>/<repo>
func DefaultCacheDir(org, repo string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".cache", "apx", "catalogs", org, repo)
}
