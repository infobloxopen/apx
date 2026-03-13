package catalog

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestResolveSource_ExplicitRegistries(t *testing.T) {
	cfg := &config.Config{
		Org:  "acme",
		Repo: "apis",
		CatalogRegistries: []config.CatalogRegistry{
			{Org: "acme", Repo: "apis"},
		},
	}
	globalCfg := &config.GlobalConfig{
		Version: 1,
		Orgs:    []config.KnownOrg{{Name: "other", Repos: []string{"schemas"}}},
	}

	src := ResolveSourceWithGlobal(cfg, globalCfg)
	agg, ok := src.(*AggregateSource)
	assert.True(t, ok, "should return AggregateSource for explicit registries")
	assert.Len(t, agg.Sources, 1)
}

func TestResolveSource_GlobalConfigFallback(t *testing.T) {
	cfg := &config.Config{} // no org, no registries, no catalog_url

	globalCfg := &config.GlobalConfig{
		Version: 1,
		Orgs: []config.KnownOrg{
			{Name: "acme", Repos: []string{"apis"}},
			{Name: "bigcorp", Repos: []string{"schemas"}},
		},
	}

	// Mock discovery to return nothing (so step 2 falls through)
	origDiscover := ghRunDiscover
	ghRunDiscover = func(org string) ([]byte, error) {
		return []byte("[]"), nil
	}
	defer func() { ghRunDiscover = origDiscover }()

	src := ResolveSourceWithGlobal(cfg, globalCfg)
	agg, ok := src.(*AggregateSource)
	assert.True(t, ok, "should return AggregateSource from global config")
	assert.Len(t, agg.Sources, 2, "should have one source per repo across all orgs")
}

func TestResolveSource_GlobalConfigPrioritizesLocalOrg(t *testing.T) {
	cfg := &config.Config{Org: "acme"} // has org but no registries

	globalCfg := &config.GlobalConfig{
		Version: 1,
		Orgs: []config.KnownOrg{
			{Name: "bigcorp", Repos: []string{"schemas"}},
			{Name: "acme", Repos: []string{"apis"}},
		},
	}

	// Mock discovery to return nothing
	origDiscover := ghRunDiscover
	ghRunDiscover = func(org string) ([]byte, error) {
		return []byte("[]"), nil
	}
	defer func() { ghRunDiscover = origDiscover }()

	src := ResolveSourceWithGlobal(cfg, globalCfg)
	agg, ok := src.(*AggregateSource)
	assert.True(t, ok)
	assert.Len(t, agg.Sources, 2)

	// First source should be acme (local org gets priority)
	first := agg.Sources[0].(*CachedSource)
	assert.Contains(t, first.Inner.Name(), "acme")
}

func TestResolveSource_NilGlobalConfig(t *testing.T) {
	cfg := &config.Config{CatalogURL: "https://example.com/catalog.yaml"}
	src := ResolveSourceWithGlobal(cfg, nil)
	_, ok := src.(*HTTPSource)
	assert.True(t, ok, "should fall through to catalog_url when global config is nil")
}

func TestResolveSource_LocalFallback(t *testing.T) {
	cfg := &config.Config{}
	src := ResolveSourceWithGlobal(cfg, nil)
	local, ok := src.(*LocalSource)
	assert.True(t, ok, "should fall back to local file")
	assert.Equal(t, "catalog/catalog.yaml", local.Path)
}

func TestResolveSource_EmptyGlobalConfig(t *testing.T) {
	cfg := &config.Config{}
	globalCfg := &config.GlobalConfig{Version: 1}
	src := ResolveSourceWithGlobal(cfg, globalCfg)
	_, ok := src.(*LocalSource)
	assert.True(t, ok, "empty global config should fall through to local")
}

func TestResolveSource_BackwardCompatible(t *testing.T) {
	// ResolveSource (without global) should still work
	cfg := &config.Config{CatalogURL: "https://example.com/catalog.yaml"}
	src := ResolveSource(cfg)
	_, ok := src.(*HTTPSource)
	assert.True(t, ok, "ResolveSource should still work without global config")
}
