package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractAvro_SimpleRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "user.avsc")
	os.WriteFile(path, []byte(`{
		"type": "record",
		"name": "User",
		"namespace": "com.acme.identity",
		"doc": "A user account",
		"fields": [
			{"name": "id", "type": "string"},
			{"name": "name", "type": "string"},
			{"name": "age", "type": "int"}
		]
	}`), 0o644)

	schema, err := ExtractAvro(path)
	require.NoError(t, err)

	assert.Equal(t, "record", schema.Type)
	assert.Equal(t, "User", schema.Name)
	assert.Equal(t, "com.acme.identity", schema.Namespace)
	assert.Equal(t, "A user account", schema.Doc)
	require.Len(t, schema.Fields, 3)
	assert.Equal(t, "id", schema.Fields[0].Name)
	assert.Equal(t, "string", schema.Fields[0].Type)
	assert.Equal(t, "age", schema.Fields[2].Name)
	assert.Equal(t, "int", schema.Fields[2].Type)
}

func TestExtractAvro_UnionType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "event.avsc")
	os.WriteFile(path, []byte(`{
		"type": "record",
		"name": "Event",
		"fields": [
			{"name": "id", "type": "string"},
			{"name": "metadata", "type": ["null", "string"], "default": null}
		]
	}`), 0o644)

	schema, err := ExtractAvro(path)
	require.NoError(t, err)
	require.Len(t, schema.Fields, 2)
	assert.Equal(t, "union<null, string>", schema.Fields[1].Type)
	assert.Equal(t, "null", schema.Fields[1].Default)
}

func TestExtractAvro_NestedRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.avsc")
	os.WriteFile(path, []byte(`{
		"type": "record",
		"name": "Order",
		"fields": [
			{"name": "id", "type": "string"},
			{"name": "address", "type": {"type": "record", "name": "Address", "fields": [{"name": "city", "type": "string"}]}}
		]
	}`), 0o644)

	schema, err := ExtractAvro(path)
	require.NoError(t, err)
	require.Len(t, schema.Fields, 2)
	assert.Equal(t, "record<Address>", schema.Fields[1].Type)
}

func TestExtractAvro_ArrayType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tags.avsc")
	os.WriteFile(path, []byte(`{
		"type": "record",
		"name": "Tagged",
		"fields": [
			{"name": "tags", "type": {"type": "array", "items": "string"}}
		]
	}`), 0o644)

	schema, err := ExtractAvro(path)
	require.NoError(t, err)
	require.Len(t, schema.Fields, 1)
	assert.Equal(t, "array<string>", schema.Fields[0].Type)
}

func TestExtractAvro_EnumSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "status.avsc")
	os.WriteFile(path, []byte(`{
		"type": "enum",
		"name": "Status",
		"symbols": ["PENDING", "ACTIVE", "CLOSED"]
	}`), 0o644)

	schema, err := ExtractAvro(path)
	require.NoError(t, err)
	assert.Equal(t, "enum", schema.Type)
	assert.Equal(t, "Status", schema.Name)
	assert.Equal(t, []string{"PENDING", "ACTIVE", "CLOSED"}, schema.Symbols)
}

func TestExtractAvro_DocOnFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "event.avsc")
	os.WriteFile(path, []byte(`{
		"type": "record",
		"name": "Event",
		"fields": [
			{"name": "timestamp", "type": "long", "doc": "Event timestamp in millis"}
		]
	}`), 0o644)

	schema, err := ExtractAvro(path)
	require.NoError(t, err)
	require.Len(t, schema.Fields, 1)
	assert.Equal(t, "Event timestamp in millis", schema.Fields[0].Doc)
}

func TestExtractAvro_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.avsc")
	os.WriteFile(path, []byte(`not json`), 0o644)

	_, err := ExtractAvro(path)
	assert.Error(t, err)
}

func TestExtractAvro_FileNotFound(t *testing.T) {
	_, err := ExtractAvro("/nonexistent/path.avsc")
	assert.Error(t, err)
}
