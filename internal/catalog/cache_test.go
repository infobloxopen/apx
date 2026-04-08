package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// stubSource is a simple CatalogSource for testing.
type stubSource struct {
	name string
	cat  *Catalog
	err  error
}

func (s *stubSource) Load() (*Catalog, error) { return s.cat, s.err }
func (s *stubSource) Name() string            { return s.name }

func TestCachedSource_FreshCache(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)

	cat := &Catalog{Version: 1, Org: "acme", Repo: "apis", Modules: []Module{
		{ID: "proto/orders/v1", Format: "proto"},
	}}

	// Pre-populate cache
	inner := &stubSource{name: "test", cat: nil, err: fmt.Errorf("should not be called")}
	cached := &CachedSource{
		Inner:    inner,
		CacheDir: dir,
		TTL:      5 * time.Minute,
		NowFn:    func() time.Time { return now },
	}

	// Write cache manually
	catData, _ := yaml.Marshal(cat)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "catalog.yaml"), catData, 0o644))
	metaJSON := fmt.Sprintf(`{"fetched_at":"%s","source":"test"}`, now.Add(-2*time.Minute).Format(time.RFC3339))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "meta.json"), []byte(metaJSON), 0o644))

	// Should return cached copy (2 min old, TTL is 5 min)
	result, err := cached.Load()
	require.NoError(t, err)
	assert.Equal(t, "acme", result.Org)
	require.Len(t, result.Modules, 1)
}

func TestCachedSource_StaleCache_RefreshSuccess(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)

	staleCat := &Catalog{Version: 1, Org: "old", Modules: []Module{{ID: "old/api/v1"}}}
	freshCat := &Catalog{Version: 1, Org: "fresh", Modules: []Module{{ID: "fresh/api/v1"}}}

	// Pre-populate stale cache (10 min old, TTL is 5 min)
	catData, _ := yaml.Marshal(staleCat)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "catalog.yaml"), catData, 0o644))
	metaJSON := fmt.Sprintf(`{"fetched_at":"%s","source":"test"}`, now.Add(-10*time.Minute).Format(time.RFC3339))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "meta.json"), []byte(metaJSON), 0o644))

	inner := &stubSource{name: "test", cat: freshCat}
	cached := &CachedSource{
		Inner:    inner,
		CacheDir: dir,
		TTL:      5 * time.Minute,
		NowFn:    func() time.Time { return now },
	}

	result, err := cached.Load()
	require.NoError(t, err)
	assert.Equal(t, "fresh", result.Org, "should return fresh data from inner source")
}

func TestCachedSource_StaleCache_RefreshFailure(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)

	staleCat := &Catalog{Version: 1, Org: "stale", Modules: []Module{{ID: "stale/api/v1"}}}

	// Pre-populate stale cache
	catData, _ := yaml.Marshal(staleCat)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "catalog.yaml"), catData, 0o644))
	metaJSON := fmt.Sprintf(`{"fetched_at":"%s","source":"test"}`, now.Add(-10*time.Minute).Format(time.RFC3339))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "meta.json"), []byte(metaJSON), 0o644))

	// Inner source fails
	inner := &stubSource{name: "test", err: fmt.Errorf("network unreachable")}
	cached := &CachedSource{
		Inner:    inner,
		CacheDir: dir,
		TTL:      5 * time.Minute,
		NowFn:    func() time.Time { return now },
	}

	// Should fall back to stale cache
	result, err := cached.Load()
	require.NoError(t, err)
	assert.Equal(t, "stale", result.Org, "should return stale cache when inner fails")
}

func TestCachedSource_NoCache_FetchSuccess(t *testing.T) {
	dir := t.TempDir()
	freshCat := &Catalog{Version: 1, Org: "fresh", Modules: []Module{{ID: "fresh/api/v1"}}}

	inner := &stubSource{name: "test", cat: freshCat}
	cached := &CachedSource{
		Inner:    inner,
		CacheDir: filepath.Join(dir, "new-cache"),
		TTL:      5 * time.Minute,
	}

	result, err := cached.Load()
	require.NoError(t, err)
	assert.Equal(t, "fresh", result.Org)

	// Verify cache was written
	_, statErr := os.Stat(filepath.Join(dir, "new-cache", "catalog.yaml"))
	assert.NoError(t, statErr, "catalog.yaml should be cached to disk")
	_, statErr = os.Stat(filepath.Join(dir, "new-cache", "meta.json"))
	assert.NoError(t, statErr, "meta.json should be cached to disk")
}

func TestCachedSource_NoCache_FetchFailure(t *testing.T) {
	dir := t.TempDir()
	inner := &stubSource{name: "test-source", err: fmt.Errorf("network error")}
	cached := &CachedSource{
		Inner:    inner,
		CacheDir: filepath.Join(dir, "empty"),
		TTL:      5 * time.Minute,
	}

	_, err := cached.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no cached copy available")
	assert.Contains(t, err.Error(), "test-source")
}

func TestCachedSource_Name(t *testing.T) {
	inner := &stubSource{name: "ghcr.io/acme/apis/catalog:latest"}
	cached := &CachedSource{Inner: inner}
	assert.Equal(t, "ghcr.io/acme/apis/catalog:latest (cached)", cached.Name())
}

func TestDefaultCacheDir(t *testing.T) {
	dir := DefaultCacheDir("acme", "apis")
	assert.Contains(t, dir, filepath.Join("apx", "catalogs", "acme", "apis"))
}
