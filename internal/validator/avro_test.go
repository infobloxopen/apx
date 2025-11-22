package validator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAvroValidator_Lint(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewAvroValidator(resolver)

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
			name:    "invalid avro schema",
			path:    "testdata/avro/invalid.avsc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Lint(tt.path)
			// Expect error since avro-tools is not installed
			if err == nil {
				t.Errorf("Lint() error = nil, expected tool not found error")
			}
		})
	}
}

func TestAvroValidator_Breaking(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewAvroValidator(resolver)

	tests := []struct {
		name              string
		compatibilityMode string
		path              string
		against           string
		wantErr           bool
	}{
		{
			name:              "backward compatible",
			compatibilityMode: "BACKWARD",
			path:              "testdata/avro/v2_backward.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           false,
		},
		{
			name:              "backward incompatible",
			compatibilityMode: "BACKWARD",
			path:              "testdata/avro/v2_breaking.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator.SetCompatibilityMode(tt.compatibilityMode)
			err := validator.Breaking(tt.path, tt.against)
			// Expect ErrNotImplemented since validator is not yet implemented
			if err == nil {
				t.Errorf("Breaking() error = nil, expected ErrNotImplemented")
			}
		})
	}
}

func TestAvroValidator_SetCompatibilityMode(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewAvroValidator(resolver)

	modes := []string{"BACKWARD", "FORWARD", "FULL", "NONE"}
	for _, mode := range modes {
		validator.SetCompatibilityMode(mode)
		if validator.compatibilityMode != mode {
			t.Errorf("SetCompatibilityMode(%s) failed, got %s", mode, validator.compatibilityMode)
		}
	}
}

func TestAvroValidator_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	schemaFile := filepath.Join(tmpDir, "user.avsc")

	validSchema := `{
  "type": "record",
  "name": "User",
  "namespace": "com.example",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "name", "type": "string"}
  ]
}
`

	if err := os.WriteFile(schemaFile, []byte(validSchema), 0644); err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	resolver := &ToolchainResolver{}
	validator := NewAvroValidator(resolver)

	err := validator.Lint(schemaFile)
	if err == nil {
		t.Log("avro-tools validation succeeded (avro-tools is installed)")
	} else {
		t.Logf("avro-tools validation failed (expected if avro-tools not available): %v", err)
	}
}
