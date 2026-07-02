package catalog

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func servedModule(id, domain, line, version, lifecycle string, types ...string) Module {
	return Module{
		ID:            id,
		Format:        "proto",
		Domain:        domain,
		APILine:       line,
		Version:       version,
		Lifecycle:     lifecycle,
		ResourceTypes: types,
	}
}

func TestResolveType_HappyPath(t *testing.T) {
	cat := &Catalog{
		Version: 1,
		Modules: []Module{
			servedModule("proto/iam/v1", "iam", "v1", "v1.2.3", "stable", "iam.example.com/User"),
			servedModule("proto/billing/v1", "billing", "v1", "v1.0.0", "stable", "billing.example.com/Invoice"),
		},
	}
	res, err := ResolveType(cat, "iam.example.com/User")
	require.NoError(t, err)
	assert.Equal(t, "iam.example.com/User", res.Type)
	assert.Equal(t, "proto/iam/v1", res.ModuleID)
	assert.Equal(t, "iam", res.Domain)
	assert.Equal(t, "v1", res.APILine)
	assert.Equal(t, "v1.2.3", res.Version)
	assert.Equal(t, "stable", res.Lifecycle)
	assert.Empty(t, res.Warning)
}

func TestResolveType_Unknown(t *testing.T) {
	cat := &Catalog{Modules: []Module{
		servedModule("proto/iam/v1", "iam", "v1", "v1.0.0", "stable", "iam.example.com/User"),
	}}
	res, err := ResolveType(cat, "iam.example.com/Nope")
	assert.Nil(t, res)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnresolved))
	assert.Contains(t, err.Error(), "iam.example.com/Nope")
}

func TestResolveType_EmptyType(t *testing.T) {
	_, err := ResolveType(&Catalog{}, "   ")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnresolved))
}

func TestResolveType_Ambiguous(t *testing.T) {
	cat := &Catalog{Modules: []Module{
		servedModule("proto/iam/v1", "iam", "v1", "v1.0.0", "stable", "shared.example.com/Thing"),
		servedModule("proto/other/v1", "other", "v1", "v1.0.0", "stable", "shared.example.com/Thing"),
	}}
	res, err := ResolveType(cat, "shared.example.com/Thing")
	assert.Nil(t, res)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAmbiguous))
	assert.Contains(t, err.Error(), "proto/iam/v1")
	assert.Contains(t, err.Error(), "proto/other/v1")
}

func TestResolveType_SameModuleTwiceIsNotAmbiguous(t *testing.T) {
	// The same module ID appearing twice (e.g. merged from two sources) is one claimant.
	m := servedModule("proto/iam/v1", "iam", "v1", "v1.0.0", "stable", "iam.example.com/User")
	cat := &Catalog{Modules: []Module{m, m}}
	res, err := ResolveType(cat, "iam.example.com/User")
	require.NoError(t, err)
	assert.Equal(t, "proto/iam/v1", res.ModuleID)
}

func TestResolveType_LifecycleSurfaced(t *testing.T) {
	cat := &Catalog{Modules: []Module{
		servedModule("proto/legacy/v1", "legacy", "v1", "v1.0.0", "deprecated", "legacy.example.com/Thing"),
	}}
	res, err := ResolveType(cat, "legacy.example.com/Thing")
	require.NoError(t, err)
	assert.Equal(t, "deprecated", res.Lifecycle)
}

func TestResolveType_ExternalResolvesToManagingModule(t *testing.T) {
	cat := &Catalog{Modules: []Module{
		{
			ID:            "proto/thirdparty/v1",
			Format:        "proto",
			Domain:        "thirdparty",
			APILine:       "v1",
			Version:       "v1.0.0",
			Lifecycle:     "stable",
			Origin:        "forked",
			ManagedRepo:   "github.com/acme/apis",
			UpstreamRepo:  "github.com/upstream/apis",
			ResourceTypes: []string{"thirdparty.example.com/Widget"},
		},
	}}
	res, err := ResolveType(cat, "thirdparty.example.com/Widget")
	require.NoError(t, err)
	assert.Equal(t, "forked", res.Origin)
	assert.Equal(t, "github.com/acme/apis", res.ManagedRepo)
	assert.Empty(t, res.Warning)
}

func TestResolveType_UnservedTypeWarns(t *testing.T) {
	// A schema-only module: declares the type but has no released serving surface.
	cat := &Catalog{Modules: []Module{
		{
			ID:            "proto/shared/v1",
			Format:        "proto",
			Domain:        "shared",
			APILine:       "v1",
			ResourceTypes: []string{"shared.example.com/Message"},
			// no Version / LatestStable / LatestPrerelease → no serving surface
		},
	}}
	res, err := ResolveType(cat, "shared.example.com/Message")
	require.NoError(t, err) // success, not an error
	assert.Equal(t, "proto/shared/v1", res.ModuleID)
	assert.NotEmpty(t, res.Warning)
	assert.Contains(t, res.Warning, "no serving surface")
}

func TestBuildTypeIndex_SkipsEmptyTypes(t *testing.T) {
	cat := &Catalog{Modules: []Module{
		{ID: "proto/iam/v1", ResourceTypes: []string{"", "iam.example.com/User"}},
	}}
	idx := BuildTypeIndex(cat)
	assert.Len(t, idx, 1)
	assert.Len(t, idx["iam.example.com/User"], 1)
}

// --- US3: parity across sources (published OCI catalog vs local) ------------

// TestResolveType_RegistryParity proves the type index rides inside the
// serialized Catalog: a catalog round-tripped through the OCI layer format
// (createTarGz -> extractCatalog, the exact path RegistrySource uses) resolves a
// type to the same module as the in-memory (Local) catalog. No production code
// change is needed for parity — this test guards it.
func TestResolveType_RegistryParity(t *testing.T) {
	catalogYAML := `version: 1
org: acme
repo: apis
modules:
  - id: proto/iam/v1
    format: proto
    domain: iam
    api_line: v1
    version: v1.2.3
    lifecycle: stable
    path: proto/iam/v1
    resource_types:
      - iam.example.com/User
`
	// Local resolution.
	var localCat Catalog
	require.NoError(t, yaml.Unmarshal([]byte(catalogYAML), &localCat))
	localRes, err := ResolveType(&localCat, "iam.example.com/User")
	require.NoError(t, err)

	// Registry-style round-trip through the OCI layer.
	layer, err := createTarGz("catalog.yaml", []byte(catalogYAML))
	require.NoError(t, err)
	r := &RegistrySource{}
	regCat, err := r.extractCatalog(layer, "application/vnd.oci.image.layer.v1.tar+gzip")
	require.NoError(t, err)
	regRes, err := ResolveType(regCat, "iam.example.com/User")
	require.NoError(t, err)

	assert.Equal(t, localRes, regRes, "registry resolution must equal local resolution")
}

func TestResolveType_AggregateCrossCatalogCollisionIsAmbiguous(t *testing.T) {
	catA := `version: 1
org: acme
repo: apis
modules:
  - id: proto/iam/v1
    format: proto
    version: v1.0.0
    lifecycle: stable
    resource_types: [shared.example.com/Thing]
`
	catB := `version: 1
org: beta
repo: apis
modules:
  - id: proto/other/v1
    format: proto
    version: v1.0.0
    lifecycle: stable
    resource_types: [shared.example.com/Thing]
`
	agg := &AggregateSource{Sources: []CatalogSource{
		&inlineSource{yaml: catA},
		&inlineSource{yaml: catB},
	}}
	merged, err := agg.Load()
	require.NoError(t, err)
	_, err = ResolveType(merged, "shared.example.com/Thing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAmbiguous))
}

// inlineSource is a test CatalogSource backed by an inline YAML string.
type inlineSource struct{ yaml string }

func (s *inlineSource) Load() (*Catalog, error) {
	var cat Catalog
	if err := yaml.Unmarshal([]byte(s.yaml), &cat); err != nil {
		return nil, err
	}
	return &cat, nil
}
func (s *inlineSource) Name() string { return "inline" }
