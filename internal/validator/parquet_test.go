package validator

import (
	"testing"
)

func TestParquetValidator_Lint(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewParquetValidator(resolver)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid parquet schema",
			path:    "testdata/parquet/valid.parquet",
			wantErr: false,
		},
		{
			name:    "invalid parquet schema",
			path:    "testdata/parquet/invalid.parquet",
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

func TestParquetValidator_Breaking(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewParquetValidator(resolver)

	tests := []struct {
		name                      string
		allowAdditiveNullableOnly bool
		path                      string
		against                   string
		wantErr                   bool
	}{
		{
			name:                      "additive nullable allowed",
			allowAdditiveNullableOnly: true,
			path:                      "testdata/parquet/v2_additive.parquet",
			against:                   "testdata/parquet/v1.parquet",
			wantErr:                   false,
		},
		{
			name:                      "breaking change detected",
			allowAdditiveNullableOnly: true,
			path:                      "testdata/parquet/v2_breaking.parquet",
			against:                   "testdata/parquet/v1.parquet",
			wantErr:                   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator.SetAdditiveNullableOnlyPolicy(tt.allowAdditiveNullableOnly)
			err := validator.Breaking(tt.path, tt.against)
			// Expect ErrNotImplemented since validator is not yet implemented
			if err == nil {
				t.Errorf("Breaking() error = nil, expected ErrNotImplemented")
			}
		})
	}
}

func TestParquetValidator_SetAdditiveNullableOnlyPolicy(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewParquetValidator(resolver)

	// Test default
	if !validator.allowAdditiveNullableOnly {
		t.Error("expected default allowAdditiveNullableOnly to be true")
	}

	// Test setter
	validator.SetAdditiveNullableOnlyPolicy(false)
	if validator.allowAdditiveNullableOnly {
		t.Error("SetAdditiveNullableOnlyPolicy(false) failed")
	}

	validator.SetAdditiveNullableOnlyPolicy(true)
	if !validator.allowAdditiveNullableOnly {
		t.Error("SetAdditiveNullableOnlyPolicy(true) failed")
	}
}
