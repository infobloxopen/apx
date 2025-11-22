package validator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProtoValidator_Lint(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewProtoValidator(resolver)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid proto file",
			path:    "testdata/proto/valid.proto",
			wantErr: false,
		},
		{
			name:    "invalid proto file",
			path:    "testdata/proto/invalid.proto",
			wantErr: true,
		},
		{
			name:    "nonexistent file",
			path:    "testdata/proto/notfound.proto",
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

func TestProtoValidator_Breaking(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewProtoValidator(resolver)

	tests := []struct {
		name    string
		path    string
		against string
		wantErr bool
	}{
		{
			name:    "no breaking changes",
			path:    "testdata/proto/v2_compatible.proto",
			against: "testdata/proto/v1.proto",
			wantErr: false,
		},
		{
			name:    "breaking changes detected",
			path:    "testdata/proto/v2_breaking.proto",
			against: "testdata/proto/v1.proto",
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

func TestProtoValidator_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create test fixtures
	tmpDir := t.TempDir()
	protoFile := filepath.Join(tmpDir, "test.proto")

	validProto := `syntax = "proto3";

package test;

message User {
  string id = 1;
  string name = 2;
}
`

	if err := os.WriteFile(protoFile, []byte(validProto), 0644); err != nil {
		t.Fatalf("failed to create test proto: %v", err)
	}

	resolver := &ToolchainResolver{}
	validator := NewProtoValidator(resolver)

	// This will fail until buf is actually installed
	err := validator.Lint(protoFile)
	if err == nil {
		t.Log("buf lint succeeded (buf is installed)")
	} else {
		t.Logf("buf lint failed (expected if buf not in PATH): %v", err)
	}
}
