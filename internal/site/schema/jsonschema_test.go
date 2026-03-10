package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractJSONSchema_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "App Config",
		"description": "Application configuration schema",
		"type": "object",
		"properties": {
			"name": {"type": "string", "description": "Application name"},
			"port": {"type": "integer", "description": "Listen port"},
			"debug": {"type": "boolean"}
		},
		"required": ["name", "port"]
	}`), 0o644)

	schema, err := ExtractJSONSchema(path)
	require.NoError(t, err)

	assert.Equal(t, "App Config", schema.Title)
	assert.Equal(t, "Application configuration schema", schema.Description)
	assert.Equal(t, "object", schema.Type)
	assert.Contains(t, schema.SchemaURI, "draft-07")

	require.Len(t, schema.Properties, 3)

	// Properties are sorted by name.
	assert.Equal(t, "debug", schema.Properties[0].Name)
	assert.Equal(t, "boolean", schema.Properties[0].Type)
	assert.False(t, schema.Properties[0].Required)

	assert.Equal(t, "name", schema.Properties[1].Name)
	assert.Equal(t, "string", schema.Properties[1].Type)
	assert.True(t, schema.Properties[1].Required)
	assert.Equal(t, "Application name", schema.Properties[1].Description)

	assert.Equal(t, "port", schema.Properties[2].Name)
	assert.True(t, schema.Properties[2].Required)
}

func TestExtractJSONSchema_NestedObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested.json")
	os.WriteFile(path, []byte(`{
		"type": "object",
		"properties": {
			"address": {
				"type": "object",
				"properties": {
					"city": {"type": "string"},
					"zip": {"type": "string"}
				},
				"required": ["city"]
			}
		}
	}`), 0o644)

	schema, err := ExtractJSONSchema(path)
	require.NoError(t, err)
	require.Len(t, schema.Properties, 1)

	addr := schema.Properties[0]
	assert.Equal(t, "address", addr.Name)
	assert.Equal(t, "object", addr.Type)
	require.Len(t, addr.Properties, 2)
	assert.Equal(t, "city", addr.Properties[0].Name)
	assert.True(t, addr.Properties[0].Required)
	assert.Equal(t, "zip", addr.Properties[1].Name)
	assert.False(t, addr.Properties[1].Required)
}

func TestExtractJSONSchema_ArrayType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "list.json")
	os.WriteFile(path, []byte(`{
		"type": "object",
		"properties": {
			"tags": {
				"type": "array",
				"items": {"type": "string"}
			}
		}
	}`), 0o644)

	schema, err := ExtractJSONSchema(path)
	require.NoError(t, err)
	require.Len(t, schema.Properties, 1)
	assert.Equal(t, "array<string>", schema.Properties[0].Type)
}

func TestExtractJSONSchema_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte(`not json`), 0o644)

	_, err := ExtractJSONSchema(path)
	assert.Error(t, err)
}
