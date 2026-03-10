package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractSchema_ProtoDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "service.proto"), []byte(`syntax = "proto3";
package test.v1;
message Foo { string id = 1; }
`), 0o644)

	detail := ExtractSchema(dir, "proto")
	require.NotNil(t, detail)
	require.Len(t, detail.Files, 1)
	assert.Equal(t, "service.proto", detail.Files[0].Filename)
	assert.NotNil(t, detail.Files[0].Proto)
	assert.Equal(t, "proto3", detail.Files[0].Proto.Syntax)
}

func TestExtractSchema_AvroDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "event.avsc"), []byte(`{
		"type": "record", "name": "Event",
		"fields": [{"name": "id", "type": "string"}]
	}`), 0o644)

	detail := ExtractSchema(dir, "avro")
	require.NotNil(t, detail)
	require.Len(t, detail.Files, 1)
	assert.NotNil(t, detail.Files[0].Avro)
	assert.Equal(t, "Event", detail.Files[0].Avro.Name)
}

func TestExtractSchema_ParquetDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "data.parquet"), []byte(`message data {
  required binary id (STRING);
}
`), 0o644)

	detail := ExtractSchema(dir, "parquet")
	require.NotNil(t, detail)
	require.Len(t, detail.Files, 1)
	assert.NotNil(t, detail.Files[0].Parquet)
}

func TestExtractSchema_OpenAPIDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "api.yaml"), []byte(`openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
paths:
  /items:
    get:
      summary: List
      responses:
        "200":
          description: OK
`), 0o644)

	detail := ExtractSchema(dir, "openapi")
	require.NotNil(t, detail)
	require.Len(t, detail.Files, 1)
	assert.NotNil(t, detail.Files[0].OpenAPI)
}

func TestExtractSchema_JSONSchemaDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{
		"type": "object",
		"properties": {"name": {"type": "string"}}
	}`), 0o644)

	detail := ExtractSchema(dir, "jsonschema")
	require.NotNil(t, detail)
	require.Len(t, detail.Files, 1)
	assert.NotNil(t, detail.Files[0].JSONSchema)
}

func TestExtractSchema_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	detail := ExtractSchema(dir, "proto")
	assert.Nil(t, detail)
}

func TestExtractSchema_NonexistentDirectory(t *testing.T) {
	detail := ExtractSchema("/nonexistent/path", "proto")
	assert.Nil(t, detail)
}

func TestExtractSchema_EmptyFormat(t *testing.T) {
	detail := ExtractSchema("/some/path", "")
	assert.Nil(t, detail)
}

func TestExtractSchema_UnknownFormat(t *testing.T) {
	detail := ExtractSchema("/some/path", "graphql")
	assert.Nil(t, detail)
}

func TestExtractSchema_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.proto"), []byte(`syntax = "proto3";
package a.v1;
message A { string id = 1; }
`), 0o644)
	os.WriteFile(filepath.Join(dir, "b.proto"), []byte(`syntax = "proto3";
package b.v1;
message B { string id = 1; }
`), 0o644)

	detail := ExtractSchema(dir, "proto")
	require.NotNil(t, detail)
	assert.Len(t, detail.Files, 2)
}

func TestExtractSchema_IgnoresNonMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte(`# Readme`), 0o644)
	os.WriteFile(filepath.Join(dir, "service.proto"), []byte(`syntax = "proto3";
message Foo { string id = 1; }
`), 0o644)

	detail := ExtractSchema(dir, "proto")
	require.NotNil(t, detail)
	require.Len(t, detail.Files, 1)
	assert.Equal(t, "service.proto", detail.Files[0].Filename)
}
