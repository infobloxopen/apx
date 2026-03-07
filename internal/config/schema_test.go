package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T027: config.Init() output passes ValidateFile.
func TestInit_OutputPassesValidateFile(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// Init should create apx.yaml
	require.NoError(t, Init())

	// The generated file must pass strict validation
	result, err := ValidateFile(filepath.Join(dir, "apx.yaml"))
	require.NoError(t, err)
	assert.True(t, result.Valid, "Init() output should pass validation, errors: %v", result.Errors)
	assert.Empty(t, result.Errors)
}

// T028: DefaultConfig() round-trips through MarshalConfig + Load.
func TestDefaultConfig_RoundTrip(t *testing.T) {
	cfg := DefaultConfig()

	data, err := MarshalConfig(cfg)
	require.NoError(t, err)

	// Write to temp file and validate
	dir := t.TempDir()
	path := filepath.Join(dir, "apx.yaml")
	require.NoError(t, os.WriteFile(path, data, 0644))

	result, err := ValidateFile(path)
	require.NoError(t, err)
	assert.True(t, result.Valid, "DefaultConfig round-trip should pass validation, errors: %v", result.Errors)
	assert.Empty(t, result.Errors)

	// Load back and verify key fields survived
	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, cfg.Version, loaded.Version)
	assert.Equal(t, cfg.Org, loaded.Org)
	assert.Equal(t, cfg.Repo, loaded.Repo)
	assert.Equal(t, cfg.ModuleRoots, loaded.ModuleRoots)
	assert.Equal(t, cfg.Publishing.TagFormat, loaded.Publishing.TagFormat)
	assert.Equal(t, cfg.Publishing.CIOnly, loaded.Publishing.CIOnly)
	assert.Equal(t, cfg.Execution.Mode, loaded.Execution.Mode)
	assert.Equal(t, cfg.Tools.Buf.Version, loaded.Tools.Buf.Version)
}

// T028 extra: MarshalConfigString produces readable output with section spacing.
func TestMarshalConfigString_Formatting(t *testing.T) {
	cfg := DefaultConfig()
	s, err := MarshalConfigString(cfg)
	require.NoError(t, err)

	// Should contain blank lines between top-level sections
	assert.True(t, strings.Contains(s, "\n\nmodule_roots:") || strings.Contains(s, "\n\npolicy:"),
		"MarshalConfigString should add blank lines between sections")

	// Should be non-empty and contain version
	assert.Contains(t, s, "version:")
	assert.Contains(t, s, "org:")
	assert.Contains(t, s, "repo:")
}

// T040+T041: GenerateSchemaDoc covers every v1 field.
func TestGenerateSchemaDoc_CoversAllFields(t *testing.T) {
	doc := GenerateSchemaDoc()
	require.NotEmpty(t, doc)

	// Must contain top-level required fields
	for _, field := range []string{"version", "org", "repo"} {
		assert.Contains(t, doc, field, "GenerateSchemaDoc should include field: %s", field)
	}

	// Must contain nested fields
	for _, field := range []string{
		"module_roots", "language_targets", "policy", "publishing", "tools", "execution",
		"tag_format", "ci_only", "mode", "container_image",
		"forbidden_proto_options", "allowed_proto_plugins",
		"spectral_ruleset", "compatibility", "breaking_mode", "allow_additive_nullable_only",
	} {
		assert.Contains(t, doc, field, "GenerateSchemaDoc should include field: %s", field)
	}

	// Must contain Markdown table header
	assert.Contains(t, doc, "| YAML Path")
	assert.Contains(t, doc, "| Type")
}

// T043: Every field name in docs/cli-reference/configuration.md exists in the v1 FieldDef tree.
func TestConfigurationDoc_ParityWithSchema(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "cli-reference", "configuration.md")
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Skipf("docs/cli-reference/configuration.md not found (path: %s): %v", docPath, err)
	}
	docContent := string(data)

	// Collect all field names from the v1 schema
	schema := Registry.Versions[CurrentSchemaVersion]
	require.NotNil(t, schema)

	allFields := collectFieldNames("", schema.Fields)

	// Every field in the schema must appear in the doc
	for _, field := range allFields {
		assert.Contains(t, docContent, field,
			"docs/cli-reference/configuration.md should reference field: %s", field)
	}
}

// collectFieldNames recursively collects all leaf field names from a FieldDef tree.
func collectFieldNames(prefix string, fields map[string]FieldDef) []string {
	var names []string
	for key, fd := range fields {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		names = append(names, path)

		if fd.Type == TypeStruct && len(fd.Children) > 0 {
			names = append(names, collectFieldNames(path, fd.Children)...)
		}
		if fd.Type == TypeMap && fd.ItemDef != nil && fd.ItemDef.Type == TypeStruct {
			mapPrefix := path + ".<key>"
			names = append(names, mapPrefix)
			if len(fd.ItemDef.Children) > 0 {
				names = append(names, collectFieldNames(mapPrefix, fd.ItemDef.Children)...)
			}
		}
	}
	return names
}
