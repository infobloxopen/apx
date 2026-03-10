package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractParquet_BasicMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "user.parquet")
	os.WriteFile(path, []byte(`message user {
  required binary id (STRING);
  required binary name (STRING);
  optional int32 age;
}
`), 0o644)

	schema, err := ExtractParquet(path)
	require.NoError(t, err)

	assert.Equal(t, "user", schema.MessageName)
	require.Len(t, schema.Columns, 3)

	assert.Equal(t, "id", schema.Columns[0].Name)
	assert.Equal(t, "binary", schema.Columns[0].PhysType)
	assert.Equal(t, "required", schema.Columns[0].Repetition)
	assert.Equal(t, "STRING", schema.Columns[0].Annotation)

	assert.Equal(t, "age", schema.Columns[2].Name)
	assert.Equal(t, "int32", schema.Columns[2].PhysType)
	assert.Equal(t, "optional", schema.Columns[2].Repetition)
	assert.Equal(t, "", schema.Columns[2].Annotation)
}

func TestExtractParquet_AllAnnotations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "event.parquet")
	os.WriteFile(path, []byte(`message event {
  required binary id (STRING);
  required int64 timestamp (TIMESTAMP_MILLIS);
  optional int32 event_date (DATE);
  optional double amount;
}
`), 0o644)

	schema, err := ExtractParquet(path)
	require.NoError(t, err)

	assert.Equal(t, "event", schema.MessageName)
	require.Len(t, schema.Columns, 4)
	assert.Equal(t, "TIMESTAMP_MILLIS", schema.Columns[1].Annotation)
	assert.Equal(t, "DATE", schema.Columns[2].Annotation)
	assert.Equal(t, "double", schema.Columns[3].PhysType)
}

func TestExtractParquet_WithComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.parquet")
	os.WriteFile(path, []byte(`// Order data schema
message order {
  // Primary key
  required binary id (STRING);
  required double total;
}
`), 0o644)

	schema, err := ExtractParquet(path)
	require.NoError(t, err)

	assert.Equal(t, "order", schema.MessageName)
	require.Len(t, schema.Columns, 2)
}

func TestExtractParquet_NoMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.parquet")
	os.WriteFile(path, []byte(`// just a comment`), 0o644)

	_, err := ExtractParquet(path)
	assert.Error(t, err)
}

func TestExtractParquet_FileNotFound(t *testing.T) {
	_, err := ExtractParquet("/nonexistent/path.parquet")
	assert.Error(t, err)
}
