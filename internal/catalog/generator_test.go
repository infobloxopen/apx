package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ParseReleaseTag
// ---------------------------------------------------------------------------

func TestParseReleaseTag_Valid(t *testing.T) {
	tests := []struct {
		tag     string
		wantID  string
		wantVer string
	}{
		{"proto/payments/ledger/v1/v1.0.0", "proto/payments/ledger/v1", "v1.0.0"},
		{"openapi/iam/roles/v2/v2.3.1", "openapi/iam/roles/v2", "v2.3.1"},
		{"avro/analytics/events/v1/v0.1.0-alpha.1", "avro/analytics/events/v1", "v0.1.0-alpha.1"},
		{"jsonschema/config/schema/v3/v3.0.0-rc.1", "jsonschema/config/schema/v3", "v3.0.0-rc.1"},
		{"parquet/data/lake/v1/v1.0.0-beta.2+build.42", "parquet/data/lake/v1", "v1.0.0-beta.2+build.42"},
		{"proto/x/y/v10/v10.0.0", "proto/x/y/v10", "v10.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			id, ver := ParseReleaseTag(tt.tag)
			assert.Equal(t, tt.wantID, id)
			assert.Equal(t, tt.wantVer, ver)
		})
	}
}

// TestParseReleaseTag_LineDropped covers the 4-segment tag shape `finalize`
// mints for ALL Go-module majors (Go's tag convention drops the /vN suffix from
// the tag prefix). The line is recovered from the version's semver major.
// Regression for G3 (v0/v1) and ARCH-271 (v2+).
func TestParseReleaseTag_LineDropped(t *testing.T) {
	tests := []struct {
		tag     string
		wantID  string
		wantVer string
	}{
		// Dotted-domain v1 modules (the brownfield-pilot shape).
		{"openapi/csp.infoblox.com/probe/v1.0.0", "openapi/csp.infoblox.com/probe/v1", "v1.0.0"},
		{"openapi/csp.infoblox.com/probe-internal/v1.0.0", "openapi/csp.infoblox.com/probe-internal/v1", "v1.0.0"},
		// v1 pre-release still resolves to the v1 line.
		{"proto/payments/ledger/v1.0.0-beta.1", "proto/payments/ledger/v1", "v1.0.0-beta.1"},
		// v0 line.
		{"openapi/iam/roles/v0.3.0", "openapi/iam/roles/v0", "v0.3.0"},
		// v2+ omit-form: the line is recovered from the version's major.
		{"openapi/csp.infoblox.com/iam-identity/v2.0.0-beta.1", "openapi/csp.infoblox.com/iam-identity/v2", "v2.0.0-beta.1"},
		{"proto/payments/ledger/v3.1.4", "proto/payments/ledger/v3", "v3.1.4"},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			id, ver := ParseReleaseTag(tt.tag)
			assert.Equal(t, tt.wantID, id)
			assert.Equal(t, tt.wantVer, ver)
		})
	}
}

// TestDeriveTagParseReleaseTagRoundTrip proves DeriveTag and ParseReleaseTag are
// inverses for v1/v2/v3 Go-module APIs: the tag DeriveTag mints (omit-form, no
// /vN subdirectory) parses back to the exact API ID and version. This is the
// invariant that keeps a v2+ release both `go get`-able and catalog-resolvable.
func TestDeriveTagParseReleaseTagRoundTrip(t *testing.T) {
	tests := []struct {
		apiID   string
		version string
		wantTag string
	}{
		{"openapi/csp.infoblox.com/iam-identity/v1", "v1.0.0-beta.1", "openapi/csp.infoblox.com/iam-identity/v1.0.0-beta.1"},
		{"openapi/csp.infoblox.com/iam-identity/v2", "v2.0.0-beta.1", "openapi/csp.infoblox.com/iam-identity/v2.0.0-beta.1"},
		{"proto/payments/ledger/v3", "v3.1.4", "proto/payments/ledger/v3.1.4"},
		{"proto/infoblox/authz/v1", "v1.0.0", "proto/infoblox/authz/v1.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.apiID+"@"+tt.version, func(t *testing.T) {
			tag := config.DeriveTag(tt.apiID, tt.version)
			assert.Equal(t, tt.wantTag, tag, "DeriveTag must omit the /vN subdirectory")

			gotID, gotVer := ParseReleaseTag(tag)
			assert.Equal(t, tt.apiID, gotID, "round-trip API ID")
			assert.Equal(t, tt.version, gotVer, "round-trip version")
		})
	}
}

func TestParseReleaseTag_Invalid(t *testing.T) {
	tests := []struct {
		name string
		tag  string
	}{
		{"wrong_part_count_3", "proto/payments/v1"},
		{"wrong_part_count_6", "proto/payments/ledger/v1/extra/v1.0.0"},
		{"bad_format", "grpc/payments/ledger/v1/v1.0.0"},
		{"bad_line", "proto/payments/ledger/latest/v1.0.0"},
		{"bad_semver_no_v", "proto/payments/ledger/v1/1.0.0"},
		{"bad_semver_garbage", "proto/payments/ledger/v1/vgarbage"},
		{"bad_semver_dots", "proto/payments/ledger/v1/v1.0.0.0"},
		{"empty_tag", ""},
		{"just_slashes", "////"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ver := ParseReleaseTag(tt.tag)
			assert.Empty(t, id, "expected empty apiID for invalid tag %q", tt.tag)
			assert.Empty(t, ver, "expected empty version for invalid tag %q", tt.tag)
		})
	}
}

// ---------------------------------------------------------------------------
// isStableVersion
// ---------------------------------------------------------------------------

func TestIsStableVersion(t *testing.T) {
	assert.True(t, isStableVersion("v1.0.0"))
	assert.True(t, isStableVersion("v2.3.4"))
	assert.True(t, isStableVersion("v0.0.1"))
	assert.False(t, isStableVersion("v1.0.0-alpha.1"))
	assert.False(t, isStableVersion("v1.0.0-beta.2"))
	assert.False(t, isStableVersion("v1.0.0-rc.1"))
	assert.False(t, isStableVersion("v1.0.0-0.pre"))
}

// ---------------------------------------------------------------------------
// GenerateFromTags
// ---------------------------------------------------------------------------

func TestGenerateFromTags_Empty(t *testing.T) {
	cat := GenerateFromTags(nil, "org", "repo")
	assert.Equal(t, 1, cat.Version)
	assert.Equal(t, "org", cat.Org)
	assert.Equal(t, "repo", cat.Repo)
	assert.Empty(t, cat.Modules)
}

func TestGenerateFromTags_SingleStable(t *testing.T) {
	tags := []string{
		"proto/payments/ledger/v1/v1.0.0",
		"proto/payments/ledger/v1/v1.1.0",
		"proto/payments/ledger/v1/v1.0.1",
	}
	cat := GenerateFromTags(tags, "myorg", "apis")

	require.Len(t, cat.Modules, 1)
	m := cat.Modules[0]
	assert.Equal(t, "proto/payments/ledger/v1", m.ID)
	assert.Equal(t, "v1.1.0", m.LatestStable)
	assert.Empty(t, m.LatestPrerelease)
	assert.Equal(t, "v1.1.0", m.Version)
	assert.Equal(t, "stable", m.Lifecycle)
	assert.Equal(t, "proto", m.Format)
	assert.Equal(t, "payments", m.Domain)
	assert.Equal(t, "v1", m.APILine)
}

func TestGenerateFromTags_PrereleaseOnly_Alpha(t *testing.T) {
	tags := []string{
		"openapi/iam/roles/v2/v2.0.0-alpha.1",
		"openapi/iam/roles/v2/v2.0.0-alpha.3",
		"openapi/iam/roles/v2/v2.0.0-alpha.2",
	}
	cat := GenerateFromTags(tags, "org", "repo")

	require.Len(t, cat.Modules, 1)
	m := cat.Modules[0]
	assert.Equal(t, "v2.0.0-alpha.3", m.LatestPrerelease)
	assert.Empty(t, m.LatestStable)
	assert.Equal(t, "v2.0.0-alpha.3", m.Version)
	assert.Equal(t, "experimental", m.Lifecycle)
}

func TestGenerateFromTags_PrereleaseOnly_Beta(t *testing.T) {
	tags := []string{
		"avro/data/events/v1/v1.0.0-beta.1",
		"avro/data/events/v1/v1.0.0-beta.3",
	}
	cat := GenerateFromTags(tags, "org", "repo")

	require.Len(t, cat.Modules, 1)
	m := cat.Modules[0]
	assert.Equal(t, "v1.0.0-beta.3", m.LatestPrerelease)
	assert.Equal(t, "beta", m.Lifecycle)
}

func TestGenerateFromTags_PrereleaseOnly_RC(t *testing.T) {
	tags := []string{
		"proto/svc/auth/v1/v1.0.0-rc.1",
		"proto/svc/auth/v1/v1.0.0-rc.2",
	}
	cat := GenerateFromTags(tags, "org", "repo")

	require.Len(t, cat.Modules, 1)
	m := cat.Modules[0]
	assert.Equal(t, "v1.0.0-rc.2", m.LatestPrerelease)
	assert.Equal(t, "beta", m.Lifecycle) // rc → beta lifecycle
}

func TestGenerateFromTags_StableOverridesPrerelease(t *testing.T) {
	tags := []string{
		"proto/payments/ledger/v1/v1.0.0-beta.1",
		"proto/payments/ledger/v1/v1.0.0-rc.1",
		"proto/payments/ledger/v1/v1.0.0",
		"proto/payments/ledger/v1/v1.1.0-alpha.1",
	}
	cat := GenerateFromTags(tags, "org", "repo")

	require.Len(t, cat.Modules, 1)
	m := cat.Modules[0]
	assert.Equal(t, "v1.0.0", m.LatestStable)
	assert.Equal(t, "v1.1.0-alpha.1", m.LatestPrerelease)
	assert.Equal(t, "v1.0.0", m.Version) // Stable takes precedence as display version
	assert.Equal(t, "stable", m.Lifecycle)
}

func TestGenerateFromTags_MultipleAPIs(t *testing.T) {
	tags := []string{
		"proto/payments/ledger/v1/v1.2.3",
		"proto/payments/wallet/v1/v1.0.0-beta.1",
		"openapi/iam/roles/v2/v2.0.0",
		"proto/payments/ledger/v2/v2.0.0-alpha.1",
		// Invalid tags are silently skipped
		"refs/heads/main",
		"not-a-release-tag",
		"v1.0.0",
	}
	cat := GenerateFromTags(tags, "acme", "apis")

	require.Len(t, cat.Modules, 4)

	// Sorted alphabetically by ID
	assert.Equal(t, "openapi/iam/roles/v2", cat.Modules[0].ID)
	assert.Equal(t, "stable", cat.Modules[0].Lifecycle)

	assert.Equal(t, "proto/payments/ledger/v1", cat.Modules[1].ID)
	assert.Equal(t, "stable", cat.Modules[1].Lifecycle)

	assert.Equal(t, "proto/payments/ledger/v2", cat.Modules[2].ID)
	assert.Equal(t, "experimental", cat.Modules[2].Lifecycle) // alpha

	assert.Equal(t, "proto/payments/wallet/v1", cat.Modules[3].ID)
	assert.Equal(t, "beta", cat.Modules[3].Lifecycle)
}

// TestGenerateFromTags_V1AndV2LineForms is the end-to-end G3 regression: a v1
// module (line-dropped 4-segment tag) and a v2 module (line-present 5-segment
// tag) must BOTH appear in the generated catalog. Before the fix the v1 tag was
// silently dropped by the scanner's exact-5-segment requirement.
func TestGenerateFromTags_V1AndV2LineForms(t *testing.T) {
	tags := []string{
		"openapi/csp.infoblox.com/probe/v1.0.0",    // v1: line dropped (4-seg)
		"openapi/csp.infoblox.com/probe/v2/v2.0.0", // v2: line present (5-seg)
	}
	cat := GenerateFromTags(tags, "devedge-dogfood", "apis-sandbox")
	require.Len(t, cat.Modules, 2, "both the v1 and v2 modules must be discovered")

	byID := map[string]Module{}
	for _, m := range cat.Modules {
		byID[m.ID] = m
	}

	v1, ok := byID["openapi/csp.infoblox.com/probe/v1"]
	require.True(t, ok, "v1 module must appear in the generated catalog")
	assert.Equal(t, "v1", v1.APILine)
	assert.Equal(t, "csp.infoblox.com", v1.Domain)
	assert.Equal(t, "v1.0.0", v1.Version)

	v2, ok := byID["openapi/csp.infoblox.com/probe/v2"]
	require.True(t, ok, "v2 module must appear in the generated catalog")
	assert.Equal(t, "v2", v2.APILine)
	assert.Equal(t, "csp.infoblox.com", v2.Domain)
	assert.Equal(t, "v2.0.0", v2.Version)
}

func TestGenerateFromTags_IgnoresNonMatchingTags(t *testing.T) {
	tags := []string{
		"refs/heads/main",
		"v1.0.0",
		"some/random/tag",
		"proto/payments/ledger/latest/v1.0.0", // invalid line
		"grpc/payments/ledger/v1/v1.0.0",      // invalid format
	}
	cat := GenerateFromTags(tags, "org", "repo")
	assert.Empty(t, cat.Modules)
}

func TestGenerateFromTags_BuildMetadata(t *testing.T) {
	// Build metadata is valid in semver but ignored in precedence
	tags := []string{
		"proto/svc/api/v1/v1.0.0+build.123",
		"proto/svc/api/v1/v1.0.1+build.456",
	}
	cat := GenerateFromTags(tags, "org", "repo")

	require.Len(t, cat.Modules, 1)
	m := cat.Modules[0]
	assert.Equal(t, "stable", m.Lifecycle)
	// Build metadata versions are valid and tracked
	assert.NotEmpty(t, m.LatestStable)
}

// ---------------------------------------------------------------------------
// Generator integration: Save and Load round-trip
// ---------------------------------------------------------------------------

func TestGenerateCatalogSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "catalog", "catalog.yaml")

	gen := NewGenerator(output)

	tags := []string{
		"proto/payments/ledger/v1/v1.0.0",
		"proto/payments/ledger/v1/v1.1.0",
		"openapi/iam/roles/v2/v2.0.0-beta.1",
	}
	cat := GenerateFromTags(tags, "myorg", "apis")

	err := os.MkdirAll(filepath.Dir(output), 0o755)
	require.NoError(t, err)

	err = gen.Save(cat)
	require.NoError(t, err)

	// Read it back
	loaded, err := gen.Load()
	require.NoError(t, err)
	require.Len(t, loaded.Modules, 2)

	assert.Equal(t, "proto/payments/ledger/v1", loaded.Modules[1].ID)
	assert.Equal(t, "v1.1.0", loaded.Modules[1].Version)
	assert.Equal(t, "stable", loaded.Modules[1].Lifecycle)

	assert.Equal(t, "openapi/iam/roles/v2", loaded.Modules[0].ID)
	assert.Equal(t, "v2.0.0-beta.1", loaded.Modules[0].Version)
	assert.Equal(t, "beta", loaded.Modules[0].Lifecycle)
}

func TestCatalogImportRoot_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "catalog.yaml")

	gen := NewGenerator(output)

	cat := &Catalog{
		Version:    1,
		Org:        "acme",
		Repo:       "apis",
		ImportRoot: "go.acme.dev/apis",
		Modules:    []Module{},
	}

	err := gen.Save(cat)
	require.NoError(t, err)

	loaded, err := gen.Load()
	require.NoError(t, err)
	assert.Equal(t, "go.acme.dev/apis", loaded.ImportRoot)
}

func TestCatalogImportRoot_OmittedWhenEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "catalog.yaml")

	gen := NewGenerator(output)

	cat := &Catalog{
		Version: 1,
		Org:     "acme",
		Repo:    "apis",
		Modules: []Module{},
	}

	err := gen.Save(cat)
	require.NoError(t, err)

	// Read raw YAML and verify import_root is not present
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "import_root")
}
