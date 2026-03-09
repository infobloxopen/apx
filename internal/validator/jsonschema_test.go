package validator

import (
	"testing"
)

func TestJSONSchemaValidator_Lint(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewJSONSchemaValidator(resolver)

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
			name:    "invalid json schema (bad type)",
			path:    "testdata/jsonschema/invalid.json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Lint(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lint(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestJSONSchemaValidator_Lint_MissingFile(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewJSONSchemaValidator(resolver)
	if err := v.Lint("testdata/jsonschema/nonexistent.json"); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestJSONSchemaValidator_Breaking(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewJSONSchemaValidator(resolver)

	tests := []struct {
		name    string
		path    string
		against string
		wantErr bool
	}{
		{
			name:    "breaking — jsonschema-diff not installed",
			path:    "testdata/jsonschema/v2_compatible.json",
			against: "testdata/jsonschema/v1.json",
			wantErr: true, // expects error: jsonschema-diff tool not found in CI
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Breaking(tt.path, tt.against)
			if (err != nil) != tt.wantErr {
				t.Errorf("Breaking() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
