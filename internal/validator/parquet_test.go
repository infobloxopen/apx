package validator

import (
	"strings"
	"testing"
)

func TestParquetValidator_Lint(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewParquetValidator(resolver)

	tests := []struct {
		name       string
		path       string
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid parquet schema",
			path:    "testdata/parquet/valid.parquet",
			wantErr: false,
		},
		{
			name:       "invalid parquet schema (unknown type)",
			path:       "testdata/parquet/invalid.parquet",
			wantErr:    true,
			errContain: "unknown physical type",
		},
		{
			name:       "duplicate column names",
			path:       "testdata/parquet/duplicate_columns.parquet",
			wantErr:    true,
			errContain: "duplicate column name",
		},
		{
			name:       "empty message (no columns)",
			path:       "testdata/parquet/empty_message.parquet",
			wantErr:    true,
			errContain: "no columns",
		},
		{
			name:       "unrecognized line",
			path:       "testdata/parquet/bad_line.parquet",
			wantErr:    true,
			errContain: "unrecognized column definition",
		},
		{
			name:       "bad column name (not snake_case)",
			path:       "testdata/parquet/bad_column_name.parquet",
			wantErr:    true,
			errContain: "should be snake_case",
		},
		{
			name:       "invalid annotation",
			path:       "testdata/parquet/bad_annotation.parquet",
			wantErr:    true,
			errContain: "unknown logical type annotation",
		},
		{
			name:    "nested groups (valid)",
			path:    "testdata/parquet/nested_groups.parquet",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Lint(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lint(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
			if tt.errContain != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("Lint(%q) error = %q, want containing %q", tt.path, err.Error(), tt.errContain)
				}
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
		name       string
		path       string
		against    string
		wantErr    bool
		errContain string
	}{
		{
			name:    "additive optional column — no breaking change",
			path:    "testdata/parquet/v2_additive.parquet",
			against: "testdata/parquet/v1.parquet",
			wantErr: false,
		},
		{
			name:       "new required column — breaking change",
			path:       "testdata/parquet/v2_breaking.parquet",
			against:    "testdata/parquet/v1.parquet",
			wantErr:    true,
			errContain: "added as required",
		},
		{
			name:       "column removed — breaking change",
			path:       "testdata/parquet/v2_removed.parquet",
			against:    "testdata/parquet/v1.parquet",
			wantErr:    true,
			errContain: "removed",
		},
		{
			name:       "type change — breaking change",
			path:       "testdata/parquet/v2_type_change.parquet",
			against:    "testdata/parquet/v1.parquet",
			wantErr:    true,
			errContain: "physical type changed",
		},
		{
			name:       "annotation change — breaking change",
			path:       "testdata/parquet/v2_annotation_change.parquet",
			against:    "testdata/parquet/v1.parquet",
			wantErr:    true,
			errContain: "annotation changed",
		},
		{
			name:       "optional to required — breaking change",
			path:       "testdata/parquet/v2_optional_to_required.parquet",
			against:    "testdata/parquet/v2_additive.parquet",
			wantErr:    true,
			errContain: "optional to required",
		},
		{
			name:    "required to optional — safe (relaxing constraint)",
			path:    "testdata/parquet/v2_required_to_optional.parquet",
			against: "testdata/parquet/v1.parquet",
			wantErr: false,
		},
		{
			name:    "multiple violations reported",
			path:    "testdata/parquet/v2_multi_violation.parquet",
			against: "testdata/parquet/v1.parquet",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v.SetAdditiveNullableOnlyPolicy(true)
			err := v.Breaking(tt.path, tt.against)
			if (err != nil) != tt.wantErr {
				t.Errorf("Breaking(%q, %q) error = %v, wantErr %v", tt.path, tt.against, err, tt.wantErr)
			}
			if tt.errContain != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("Breaking() error = %q, want containing %q", err.Error(), tt.errContain)
				}
			}
		})
	}
}

func TestParquetValidator_Breaking_MultiViolationCount(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewParquetValidator(resolver)
	v.SetAdditiveNullableOnlyPolicy(true)

	err := v.Breaking("testdata/parquet/v2_multi_violation.parquet", "testdata/parquet/v1.parquet")
	if err == nil {
		t.Fatal("expected error for multiple violations")
	}

	msg := err.Error()
	// v2_multi_violation removes timestamp, changes type's physical type, adds required source
	violations := 0
	if strings.Contains(msg, "physical type changed") {
		violations++
	}
	if strings.Contains(msg, "removed") {
		violations++
	}
	if strings.Contains(msg, "added as required") {
		violations++
	}
	if violations < 2 {
		t.Errorf("expected at least 2 violations in error, got %d: %s", violations, msg)
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
