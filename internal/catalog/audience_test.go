package catalog

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestValidateAudience(t *testing.T) {
	for _, ok := range []string{"", "public", "internal"} {
		assert.NoError(t, ValidateAudience(ok), "audience %q", ok)
	}
	for _, bad := range []string{"Public", "private", "device", " internal"} {
		assert.Error(t, ValidateAudience(bad), "audience %q", bad)
	}
}

func TestParseAnnotationAudience(t *testing.T) {
	tests := map[string]string{
		"Lifecycle: stable\nAudience: internal\nSource: x": "internal",
		"Audience:public": "public",
		"Lifecycle: stable\nTags: audience:internal": "", // a tag is not the audience field
		"Audience: device":                           "", // not a valid audience: ignored
		"":                                           "",
	}
	for body, want := range tests {
		assert.Equal(t, want, parseAnnotationAudience(body), "body %q", body)
	}
}

// The audience of a module comes from the highest version whose release tag
// records one. A module whose tags record none has no audience, and the YAML
// has no audience key for it.
func TestGenerateFromTagRecords_Audience(t *testing.T) {
	cat := GenerateFromTagRecords([]TagRecord{
		{Tag: "openapi/payments/ledger/v1/v1.0.0", Audience: "internal"},
		{Tag: "openapi/payments/ledger/v1/v1.2.0", Audience: "public"},
		{Tag: "openapi/payments/ledger/v1/v1.1.0", Audience: "internal"},
		{Tag: "openapi/payments/ledger/v1/v1.3.0"}, // a later release without the flag
		{Tag: "openapi/payments/wallet/v1/v1.0.0"},
	}, "org", "repo")
	byID := map[string]Module{}
	for _, m := range cat.Modules {
		byID[m.ID] = m
	}
	assert.Equal(t, "public", byID["openapi/payments/ledger/v1"].Audience)
	assert.Equal(t, "", byID["openapi/payments/wallet/v1"].Audience)

	data, err := yaml.Marshal(cat)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(data), "audience:"), "only the module with an audience has the key:\n%s", data)
}

func TestModuleAudience_YAMLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")
	gen := NewGenerator(path)
	require.NoError(t, gen.Save(&Catalog{Version: 1, Modules: []Module{
		{ID: "openapi/a/b/v1", Format: "openapi", Path: "openapi/a/b/v1", Audience: AudienceInternal},
		{ID: "openapi/a/c/v1", Format: "openapi", Path: "openapi/a/c/v1"},
	}}))
	got, err := gen.Load()
	require.NoError(t, err)
	require.Len(t, got.Modules, 2)
	assert.Equal(t, "internal", got.Modules[0].Audience)
	assert.Equal(t, "", got.Modules[1].Audience)
}

// A regenerated catalog keeps the audience of the committed catalog when no
// release tag records one, and a recorded audience wins over the committed one.
func TestPreserveCuratedFields_Audience(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")
	require.NoError(t, NewGenerator(path).Save(&Catalog{Version: 1, Modules: []Module{
		{ID: "openapi/a/b/v1", Format: "openapi", Path: "openapi/a/b/v1", Audience: "internal"},
		{ID: "openapi/a/c/v1", Format: "openapi", Path: "openapi/a/c/v1", Audience: "internal"},
		{ID: "openapi/a/d/v1", Format: "openapi", Path: "openapi/a/d/v1"},
	}}))

	fresh := GenerateFromTagRecords([]TagRecord{
		{Tag: "openapi/a/b/v1/v1.0.0"},
		{Tag: "openapi/a/c/v1/v1.0.0", Audience: "public"},
		{Tag: "openapi/a/d/v1/v1.0.0"},
	}, "org", "repo")
	PreserveCuratedFields(fresh, path)

	byID := map[string]Module{}
	for _, m := range fresh.Modules {
		byID[m.ID] = m
	}
	assert.Equal(t, "internal", byID["openapi/a/b/v1"].Audience, "kept from the committed catalog")
	assert.Equal(t, "public", byID["openapi/a/c/v1"].Audience, "the recorded audience wins")
	assert.Equal(t, "", byID["openapi/a/d/v1"].Audience, "absent stays absent")
}
