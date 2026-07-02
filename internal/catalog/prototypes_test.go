package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractResourceTypes_SingleResource(t *testing.T) {
	src := `
syntax = "proto3";
package iam.v1;

message User {
  option (google.api.resource) = {
    type: "iam.example.com/User"
    pattern: "users/{user}"
  };
  string name = 1;
}
`
	got := extractResourceTypes(src)
	assert.Equal(t, []string{"iam.example.com/User"}, got)
}

func TestExtractResourceTypes_SameLine(t *testing.T) {
	src := `message Role { option (google.api.resource) = { type: "iam.example.com/Role" pattern: "roles/{role}" }; }`
	got := extractResourceTypes(src)
	assert.Equal(t, []string{"iam.example.com/Role"}, got)
}

func TestExtractResourceTypes_MultipleInOneFile(t *testing.T) {
	src := `
message User {
  option (google.api.resource) = { type: "iam.example.com/User" };
}
message Group {
  option (google.api.resource) = {
    type: "iam.example.com/Group"
  };
}
`
	got := extractResourceTypes(src)
	assert.ElementsMatch(t, []string{"iam.example.com/User", "iam.example.com/Group"}, got)
}

func TestExtractResourceTypes_NoAnnotation(t *testing.T) {
	src := `
message Plain {
  string name = 1;
}
`
	assert.Empty(t, extractResourceTypes(src))
}

func TestExtractResourceTypes_IgnoresLineComment(t *testing.T) {
	src := `
// option (google.api.resource) = { type: "iam.example.com/Ghost" };
message User {
  option (google.api.resource) = { type: "iam.example.com/User" };
}
`
	got := extractResourceTypes(src)
	assert.Equal(t, []string{"iam.example.com/User"}, got)
}

func TestExtractResourceTypes_IgnoresBlockComment(t *testing.T) {
	src := `
/*
message Ghost {
  option (google.api.resource) = { type: "iam.example.com/Ghost" };
}
*/
message User {
  option (google.api.resource) = { type: "iam.example.com/User" };
}
`
	got := extractResourceTypes(src)
	assert.Equal(t, []string{"iam.example.com/User"}, got)
}

func TestExtractResourceTypes_SpacingVariants(t *testing.T) {
	src := `
message A {
  option ( google.api.resource ) = {
    type:"svc.example.com/A"
  };
}
`
	got := extractResourceTypes(src)
	assert.Equal(t, []string{"svc.example.com/A"}, got)
}

func TestScanResourceTypes_MissingDir(t *testing.T) {
	got, err := ScanResourceTypes(filepath.Join(t.TempDir(), "does-not-exist"))
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestScanResourceTypes_DedupesAndSorts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "user.proto"), `
message User { option (google.api.resource) = { type: "iam.example.com/User" }; }
`)
	writeFile(t, filepath.Join(dir, "group.proto"), `
message Group { option (google.api.resource) = { type: "iam.example.com/Group" }; }
// duplicate declaration in another file should collapse
message User2 { option (google.api.resource) = { type: "iam.example.com/User" }; }
`)
	writeFile(t, filepath.Join(dir, "notes.txt"), `option (google.api.resource) = { type: "should.be/Ignored" };`)

	got, err := ScanResourceTypes(dir)
	require.NoError(t, err)
	// sorted, deduped, only from .proto files
	assert.Equal(t, []string{"iam.example.com/Group", "iam.example.com/User"}, got)
}

func TestScanResourceTypes_Recursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "v1")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	writeFile(t, filepath.Join(sub, "user.proto"), `
message User { option (google.api.resource) = { type: "iam.example.com/User" }; }
`)
	got, err := ScanResourceTypes(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"iam.example.com/User"}, got)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
