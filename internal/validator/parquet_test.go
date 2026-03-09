package validator

import (
	"testing"
)

func TestParquetValidator_Lint(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewParquetValidator(resolver)

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
			name:    "invalid parquet schema (unknown type)",
			path:    "testdata/parquet/invalid.parquet",
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

func TestParquetValidator_Lint_MissingFile(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewParquetValidator(resolver)
	if err := v.Lint("testdata/parquet/nonexistent.parquet"); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestParquetValidator_Breaking(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewParquetValidator(resolver)

	tests := []struct {
		name                      string
		allowAdditiveNullableOnly bool
		path                      string
		against                   string
		wantErr                   bool
	}{
		{
			name:                      "additive optional column — no breaking change",
			allowAdditiveNullableOnly: true,
			path:                      "testdata/parquet/v2_additive.parquet",
			against:                   "testdata/parquet/v1.parquet",
			wantErr:                   false,
		},
		{
			name:                      "new required column — breaking change",
			allowAdditiveNullableOnly: true,
			path:                      "testdata/parquet/v2_breaking.parquet",
			against:                   "testdata/parquet/v1.parquet",
			wantErr:                   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v.SetAdditiveNullableOnlyPolicy(tt.allowAdditiveNullableOnly)
			err := v.Breaking(tt.path, tt.against)
			if (err != nil) != tt.wantErr {
				t.Errorf("Breaking() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParquetValidator_SetAdditiveNullableOnlyPolicy(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewParquetValidator(resolver)

	if !v.allowAdditiveNullableOnly {
		t.Error("expected default allowAdditiveNullableOnly to be true")
	}

	v.SetAdditiveNullableOnlyPolicy(false)
	if v.allowAdditiveNullableOnly {
		t.Error("SetAdditiveNullableOnlyPolicy(false) failed")
	}

	v.SetAdditiveNullableOnlyPolicy(true)
	if !v.allowAdditiveNullableOnly {
		t.Error("SetAdditiveNullableOnlyPolicy(true) failed")
	}
}
