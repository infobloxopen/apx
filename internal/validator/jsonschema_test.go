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

func TestJSONSchemaValidator_Lint_Directory(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewJSONSchemaValidator(resolver)
	// The testdata/jsonschema/ dir has both valid and invalid files.
	// Lint should fail because invalid.json is present.
	if err := v.Lint("testdata/jsonschema"); err == nil {
		t.Error("expected error when directory contains invalid schema")
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
			name:    "compatible — additive field",
			path:    "testdata/jsonschema/v2_compatible.json",
			against: "testdata/jsonschema/v1.json",
			wantErr: false,
		},
		{
			name:    "breaking — type changed and required added",
			path:    "testdata/jsonschema/v2_breaking.json",
			against: "testdata/jsonschema/v1.json",
			wantErr: true,
		},
		{
			name:    "baseline missing — new schema, no comparison",
			path:    "testdata/jsonschema/v1.json",
			against: "testdata/jsonschema/nonexistent.json",
			wantErr: false,
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
