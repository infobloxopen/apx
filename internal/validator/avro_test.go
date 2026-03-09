package validator

import (
	"testing"
)

func TestAvroValidator_Lint(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewAvroValidator(resolver)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid avro schema",
			path:    "testdata/avro/valid.avsc",
			wantErr: false,
		},
		{
			name:    "invalid avro schema (missing name)",
			path:    "testdata/avro/invalid.avsc",
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

func TestAvroValidator_Lint_MissingFile(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewAvroValidator(resolver)
	if err := v.Lint("testdata/avro/nonexistent.avsc"); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestAvroValidator_Breaking(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewAvroValidator(resolver)

	tests := []struct {
		name              string
		compatibilityMode string
		path              string
		against           string
		wantErr           bool
	}{
		{
			name:              "backward compatible — new field with default",
			compatibilityMode: "BACKWARD",
			path:              "testdata/avro/v2_backward.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           false,
		},
		{
			name:              "backward incompatible — new required field without default",
			compatibilityMode: "BACKWARD",
			path:              "testdata/avro/v2_breaking.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           true,
		},
		{
			name:              "NONE mode always passes",
			compatibilityMode: "NONE",
			path:              "testdata/avro/v2_breaking.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v.SetCompatibilityMode(tt.compatibilityMode)
			err := v.Breaking(tt.path, tt.against)
			if (err != nil) != tt.wantErr {
				t.Errorf("Breaking() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAvroValidator_SetCompatibilityMode(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewAvroValidator(resolver)

	for _, mode := range []string{"BACKWARD", "FORWARD", "FULL", "NONE"} {
		v.SetCompatibilityMode(mode)
		if v.compatibilityMode != mode {
			t.Errorf("SetCompatibilityMode(%s) failed, got %s", mode, v.compatibilityMode)
		}
	}
}
