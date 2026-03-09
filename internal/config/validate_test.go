package config

import (
	"os"
	"path/filepath"
	"testing"
)

// ---- T016: Valid minimal file, valid full file, missing required fields ----

func TestValidateFile_ValidMinimal(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "apx.yaml")
	writeFile(t, p, `version: 1
org: myorg
repo: myrepo
`)

	result, err := ValidateFile(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", fmtErrs(result.Errors))
	}
}

func TestValidateFile_ValidFullFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "apx.yaml")

	// Use a complete config that mirrors apx.example.yaml
	writeFile(t, p, `version: 1
org: your-org-name
repo: apis
module_roots:
  - proto
  - openapi
  - avro
  - jsonschema
  - parquet
language_targets:
  go:
    enabled: true
    plugins:
      - name: protoc-gen-go
        version: v1.64.0
      - name: protoc-gen-go-grpc
        version: v1.5.0
  python:
    enabled: true
    tool: grpcio-tools
    version: 1.64.0
  java:
    enabled: true
    plugins:
      - name: protoc-gen-grpc-java
        version: 1.68.1
policy:
  forbidden_proto_options:
    - "^gorm\\."
  allowed_proto_plugins:
    - protoc-gen-go
    - protoc-gen-go-grpc
  openapi:
    spectral_ruleset: ".spectral.yaml"
  avro:
    compatibility: "BACKWARD"
  jsonschema:
    breaking_mode: "strict"
  parquet:
    allow_additive_nullable_only: true
release:
  tag_format: "{subdir}/v{version}"
  ci_only: true
tools:
  buf:
    version: v1.45.0
  oasdiff:
    version: v1.9.6
  spectral:
    version: v6.11.0
  avrotool:
    version: "1.11.3"
  jsonschemadiff:
    version: "0.3.0"
execution:
  mode: "local"
  container_image: ""
`)

	result, err := ValidateFile(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", fmtErrs(result.Errors))
	}
	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", fmtErrs(result.Warnings))
	}
}

func TestValidateFile_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		expectFields []string
	}{
		{
			name:         "missing org",
			yaml:         "version: 1\nrepo: myrepo\n",
			expectFields: []string{"org"},
		},
		{
			name:         "missing repo",
			yaml:         "version: 1\norg: myorg\n",
			expectFields: []string{"repo"},
		},
		{
			name:         "missing org and repo",
			yaml:         "version: 1\n",
			expectFields: []string{"org", "repo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateBytes([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Valid {
				t.Fatal("expected invalid result")
			}
			for _, field := range tt.expectFields {
				if !hasErrorForField(result.Errors, field) {
					t.Errorf("expected error for field '%s', got: %v", field, fmtErrs(result.Errors))
				}
			}
		})
	}
}

// ---- T017: Unknown keys, wrong type, invalid enum ----

func TestValidateFile_UnknownTopLevelKey(t *testing.T) {
	result, err := ValidateBytes([]byte("version: 1\norg: myorg\nrepo: myrepo\nfoobar: true\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result")
	}
	if !hasErrorForField(result.Errors, "foobar") {
		t.Errorf("expected unknown key error for 'foobar', got: %v", fmtErrs(result.Errors))
	}
	if !hasErrorKind(result.Errors, ErrUnknownKey) {
		t.Error("expected error kind 'unknown_key'")
	}
}

func TestValidateFile_UnknownNestedKey(t *testing.T) {
	yaml := `version: 1
org: myorg
repo: myrepo
policy:
  openapi:
    unknown_field: true
`
	result, err := ValidateBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result")
	}
	if !hasErrorForField(result.Errors, "policy.openapi.unknown_field") {
		t.Errorf("expected unknown key error for 'policy.openapi.unknown_field', got: %v", fmtErrs(result.Errors))
	}
}

func TestValidateFile_WrongType(t *testing.T) {
	yaml := `version: "one"
org: myorg
repo: myrepo
`
	result, err := ValidateBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The version is a string, not int — should fail
	if result.Valid {
		t.Fatal("expected invalid result for wrong type")
	}
}

func TestValidateFile_InvalidEnumValue(t *testing.T) {
	yaml := `version: 1
org: myorg
repo: myrepo
execution:
  mode: "docker"
`
	result, err := ValidateBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result for bad enum")
	}
	if !hasErrorForField(result.Errors, "execution.mode") {
		t.Errorf("expected error for 'execution.mode', got: %v", fmtErrs(result.Errors))
	}
	if !hasErrorKind(result.Errors, ErrInvalidValue) {
		t.Error("expected error kind 'invalid_value'")
	}
}

// ---- T018: Empty file, whitespace-only, unsupported version, future version ----

func TestValidateFile_EmptyFile(t *testing.T) {
	result, err := ValidateBytes([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result for empty file")
	}
}

func TestValidateFile_WhitespaceOnly(t *testing.T) {
	result, err := ValidateBytes([]byte("   \n  \n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result for whitespace-only file")
	}
}

func TestValidateFile_UnsupportedVersion(t *testing.T) {
	result, err := ValidateBytes([]byte("version: 0\norg: x\nrepo: y\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result for version 0")
	}
	if !hasErrorForField(result.Errors, "version") {
		t.Errorf("expected error for 'version', got: %v", fmtErrs(result.Errors))
	}
}

func TestValidateFile_FutureVersion(t *testing.T) {
	result, err := ValidateBytes([]byte("version: 99\norg: x\nrepo: y\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result for future version")
	}
	found := false
	for _, e := range result.Errors {
		if e.Field == "version" && e.Kind == ErrInvalidValue {
			found = true
		}
	}
	if !found {
		t.Errorf("expected invalid_value error for future version, got: %v", fmtErrs(result.Errors))
	}
}

// ---- T019: Deprecated field emits warning, pattern validation on tag_format ----

func TestValidateFile_DeprecatedField(t *testing.T) {
	// Temporarily add a deprecated field to the v1 schema for testing
	v1 := Registry.Versions[1]
	v1.Fields["old_field"] = FieldDef{
		Name:            "old_field",
		Type:            TypeString,
		Description:     "Deprecated field",
		DeprecatedSince: 1,
		Replacement:     "new_field",
	}
	Registry.Versions[1] = v1
	defer func() {
		delete(v1.Fields, "old_field")
		Registry.Versions[1] = v1
	}()

	yaml := `version: 1
org: myorg
repo: myrepo
old_field: "test"
`
	result, err := ValidateBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be valid (deprecated = warning, not error)
	if !result.Valid {
		t.Errorf("expected valid (deprecated is a warning), got errors: %v", fmtErrs(result.Errors))
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected at least one warning for deprecated field")
	}
	found := false
	for _, w := range result.Warnings {
		if w.Kind == ErrDeprecated && w.Field == "old_field" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected deprecated warning for 'old_field', got: %v", fmtErrs(result.Warnings))
	}
}

func TestValidateFile_PatternValidation_TagFormat(t *testing.T) {
	yaml := `version: 1
org: myorg
repo: myrepo
release:
  tag_format: "v{bad}"
`
	result, err := ValidateBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result for bad tag_format")
	}
	if !hasErrorForField(result.Errors, "release.tag_format") {
		t.Errorf("expected error for 'release.tag_format', got: %v", fmtErrs(result.Errors))
	}
}

func TestValidateFile_PatternValidation_TagFormatValid(t *testing.T) {
	yaml := `version: 1
org: myorg
repo: myrepo
release:
  tag_format: "{subdir}/v{version}"
`
	result, err := ValidateBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", fmtErrs(result.Errors))
	}
}

// ---- T022: Validate apx.example.yaml passes ----

func TestValidateFile_ExampleFile(t *testing.T) {
	// apx.example.yaml is at the repository root
	examplePath := filepath.Join("..", "..", "apx.example.yaml")
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		t.Skip("apx.example.yaml not found (not running from expected directory)")
	}

	result, err := ValidateFile(examplePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("apx.example.yaml should be valid, got errors: %v", fmtErrs(result.Errors))
	}
}

func TestValidateFile_CmdApxYaml(t *testing.T) {
	// cmd/apx/apx.yaml is also in the repository
	yamlPath := filepath.Join("..", "..", "cmd", "apx", "apx.yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		t.Skip("cmd/apx/apx.yaml not found")
	}

	result, err := ValidateFile(yamlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("cmd/apx/apx.yaml should be valid, got errors: %v", fmtErrs(result.Errors))
	}
}

// ---- Helpers ----

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func hasErrorForField(errs []*ValidationError, field string) bool {
	for _, e := range errs {
		if e.Field == field {
			return true
		}
	}
	return false
}

func hasErrorKind(errs []*ValidationError, kind ErrorKind) bool {
	for _, e := range errs {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

func fmtErrs(errs []*ValidationError) string {
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, e.Error())
	}
	return "[" + joinStrs(msgs) + "]"
}

func joinStrs(s []string) string {
	result := ""
	for i, str := range s {
		if i > 0 {
			result += ", "
		}
		result += str
	}
	return result
}
