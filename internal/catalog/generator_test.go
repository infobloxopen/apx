package catalog

import (
	"os"
	"path/filepath"
	"testing"

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
	assert.Equal(t, "preview", m.Lifecycle)
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
	assert.Equal(t, "preview", m.Lifecycle) // rc → preview lifecycle
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
	assert.Equal(t, "preview", cat.Modules[3].Lifecycle)
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
	assert.Equal(t, "preview", loaded.Modules[0].Lifecycle)
}
