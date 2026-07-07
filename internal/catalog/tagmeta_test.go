package catalog

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitAndUnionTags(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c"}, SplitTags(" a, b ,c ,, "))
	assert.Nil(t, SplitTags("   "))
	// Union is sorted + de-duplicated.
	assert.Equal(t, []string{"audience:device", "family:ddi", "product:ddi"},
		UnionTags([]string{"product:ddi", "audience:device"}, []string{"family:ddi", "product:ddi"}))
}

func TestParseAnnotationMeta(t *testing.T) {
	body := "Lifecycle: deprecated\nTags: audience:device, product:universal-ddi\nSource: github.com/org/repo/openapi/x"
	lc, tags := parseAnnotationMeta(body)
	assert.Equal(t, "deprecated", lc)
	assert.Equal(t, []string{"audience:device", "product:universal-ddi"}, tags)

	lc, tags = parseAnnotationMeta("") // lightweight tag — no metadata
	assert.Equal(t, "", lc)
	assert.Nil(t, tags)
}

// TestGenerateFromTagRecords_RecordedLifecycle is the F-32 unit regression: a
// module whose current stable release recorded `deprecated` in its annotation
// surfaces as deprecated, NOT the semver-derived `stable`.
func TestGenerateFromTagRecords_RecordedLifecycle(t *testing.T) {
	records := []TagRecord{
		{Tag: "openapi/csp.infoblox.com/hostapp/v1.0.0", Lifecycle: "deprecated",
			Tags: []string{"audience:device"}},
	}
	cat := GenerateFromTagRecords(records, "org", "repo")
	require.Len(t, cat.Modules, 1)
	m := cat.Modules[0]
	assert.Equal(t, "openapi/csp.infoblox.com/hostapp/v1", m.ID)
	assert.Equal(t, "v1.0.0", m.Version)
	assert.Equal(t, "deprecated", m.Lifecycle, "recorded lifecycle must override the semver-derived 'stable'")
	assert.Equal(t, []string{"audience:device"}, m.Tags)
}

// A tag with no recorded lifecycle still derives from semver (backward compat).
func TestGenerateFromTagRecords_DerivedFallback(t *testing.T) {
	cat := GenerateFromTagRecords([]TagRecord{
		{Tag: "openapi/csp.infoblox.com/probe/v2/v2.0.0"},       // stable
		{Tag: "openapi/csp.infoblox.com/widgets/v1.0.0-beta.1"}, // beta prerelease
	}, "org", "repo")
	byID := map[string]Module{}
	for _, m := range cat.Modules {
		byID[m.ID] = m
	}
	assert.Equal(t, "stable", byID["openapi/csp.infoblox.com/probe/v2"].Lifecycle)
	assert.Equal(t, "beta", byID["openapi/csp.infoblox.com/widgets/v1"].Lifecycle)
}

// TestPreserveCuratedFields covers the in-place-deprecate persistence (F-32) and
// tag/owner preservation (F-33/G6): a regenerated catalog keeps a further-along
// lifecycle and curated fields from the committed catalog.
func TestPreserveCuratedFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")

	// Committed catalog: module deprecated in place, with curated tags/owners.
	existing := &Catalog{
		Version: 1, Org: "org", Repo: "repo",
		Modules: []Module{{
			ID: "openapi/csp.infoblox.com/hostapp/v1", Format: "openapi",
			Version: "v1.0.0", Lifecycle: "deprecated",
			Tags: []string{"audience:device"}, Owners: []string{"team-infra"},
			Description: "Legacy host plane",
		}},
	}
	require.NoError(t, NewGenerator(path).Save(existing))

	// Freshly tag-derived catalog: same version but lifecycle re-derived to stable.
	fresh := GenerateFromTags([]string{"openapi/csp.infoblox.com/hostapp/v1.0.0"}, "org", "repo")
	require.Equal(t, "stable", fresh.Modules[0].Lifecycle)

	PreserveCuratedFields(fresh, path)
	m := fresh.Modules[0]
	assert.Equal(t, "deprecated", m.Lifecycle, "in-place deprecation must survive regeneration")
	assert.Equal(t, []string{"audience:device"}, m.Tags)
	assert.Equal(t, []string{"team-infra"}, m.Owners)
	assert.Equal(t, "Legacy host plane", m.Description)
}

// A stable release must NOT be regressed by a stale earlier lifecycle.
func TestPreserveCuratedFields_DoesNotRegress(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")
	require.NoError(t, NewGenerator(path).Save(&Catalog{
		Version: 1,
		Modules: []Module{{ID: "openapi/x/y/v1", Lifecycle: "beta"}},
	}))
	fresh := &Catalog{Modules: []Module{{ID: "openapi/x/y/v1", Lifecycle: "stable"}}}
	PreserveCuratedFields(fresh, path)
	assert.Equal(t, "stable", fresh.Modules[0].Lifecycle)
}
