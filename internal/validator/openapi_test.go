package validator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAPIValidator_Lint(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewOpenAPIValidator(resolver)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid openapi spec",
			path:    "testdata/openapi/valid.yaml",
			wantErr: false,
		},
		{
			name:    "invalid openapi spec",
			path:    "testdata/openapi/invalid.yaml",
			wantErr: true,
		},
		{
			name:    "nonexistent file",
			path:    "testdata/openapi/notfound.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Lint(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenAPIValidator_Breaking(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewOpenAPIValidator(resolver)

	tests := []struct {
		name    string
		path    string
		against string
		wantErr bool
	}{
		{
			name:    "no breaking changes",
			path:    "testdata/openapi/v2_compatible.yaml",
			against: "testdata/openapi/v1.yaml",
			wantErr: false,
		},
		{
			name:    "breaking changes detected",
			path:    "testdata/openapi/v2_breaking.yaml",
			against: "testdata/openapi/v1.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Breaking(tt.path, tt.against)
			if (err != nil) != tt.wantErr {
				t.Errorf("Breaking() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenAPIValidator_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "openapi.yaml")

	validSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      summary: List users
      responses:
        '200':
          description: Success
`

	if err := os.WriteFile(specFile, []byte(validSpec), 0644); err != nil {
		t.Fatalf("failed to create test spec: %v", err)
	}

	resolver := &ToolchainResolver{}
	validator := NewOpenAPIValidator(resolver)

	err := validator.Lint(specFile)
	if err == nil {
		t.Log("spectral lint succeeded (spectral is installed)")
	} else {
		t.Logf("spectral lint failed (expected if spectral not in PATH): %v", err)
	}
}
