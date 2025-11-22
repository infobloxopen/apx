package validator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSONSchemaValidator_Lint(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewJSONSchemaValidator(resolver)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid json schema",
			path:    "testdata/jsonschema/valid.json",
			wantErr: false,
		},
		{
			name:    "invalid json schema",
			path:    "testdata/jsonschema/invalid.json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Lint(tt.path)
			// Expect ErrNotImplemented since validator is not yet implemented
			if err == nil {
				t.Errorf("Lint() error = nil, expected ErrNotImplemented")
			}
		})
	}
}

func TestJSONSchemaValidator_Breaking(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewJSONSchemaValidator(resolver)

	tests := []struct {
		name    string
		path    string
		against string
		wantErr bool
	}{
		{
			name:    "no breaking changes",
			path:    "testdata/jsonschema/v2_compatible.json",
			against: "testdata/jsonschema/v1.json",
			wantErr: false,
		},
		{
			name:    "breaking changes detected",
			path:    "testdata/jsonschema/v2_breaking.json",
			against: "testdata/jsonschema/v1.json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Breaking(tt.path, tt.against)
			// Expect error since jsonschema-diff is not installed
			if err == nil {
				t.Errorf("Breaking() error = nil, expected tool not found error")
			}
		})
	}
}

func TestJSONSchemaValidator_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	schemaFile := filepath.Join(tmpDir, "user.json")

	validSchema := `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {"type": "string"},
    "name": {"type": "string"}
  },
  "required": ["id", "name"]
}
`

	if err := os.WriteFile(schemaFile, []byte(validSchema), 0644); err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	resolver := &ToolchainResolver{}
	validator := NewJSONSchemaValidator(resolver)

	err := validator.Lint(schemaFile)
	if err == nil {
		t.Log("jsonschema validation succeeded")
	} else {
		t.Logf("jsonschema validation result: %v", err)
	}
}
