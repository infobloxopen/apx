package validator

import (
	"strings"
	"testing"
)

func TestAvroValidator_Lint(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewAvroValidator(resolver)

	tests := []struct {
		name       string
		path       string
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid avro schema",
			path:    "testdata/avro/valid.avsc",
			wantErr: false,
		},
		{
			name:       "invalid avro schema (missing name)",
			path:       "testdata/avro/invalid.avsc",
			wantErr:    true,
			errContain: "missing required 'name' field",
		},
		{
			name:       "duplicate field names",
			path:       "testdata/avro/duplicate_fields.avsc",
			wantErr:    true,
			errContain: "duplicate field name",
		},
		{
			name:       "empty fields array",
			path:       "testdata/avro/empty_fields.avsc",
			wantErr:    true,
			errContain: "empty 'fields' array",
		},
		{
			name:       "bad field name (not camelCase)",
			path:       "testdata/avro/bad_field_name.avsc",
			wantErr:    true,
			errContain: "camelCase",
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
					t.Errorf("Lint(%q) error = %q, want substring %q", tt.path, err, tt.errContain)
				}
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
		errContain        string
	}{
		{
			name:              "backward compatible — new field with nullable union default",
			compatibilityMode: "BACKWARD",
			path:              "testdata/avro/v2_backward.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           false,
		},
		{
			name:              "backward compatible — new field with nullable union (explicit)",
			compatibilityMode: "BACKWARD",
			path:              "testdata/avro/v2_nullable_union.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           false,
		},
		{
			name:              "backward incompatible — new required field without default",
			compatibilityMode: "BACKWARD",
			path:              "testdata/avro/v2_breaking.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           true,
			errContain:        "added to new schema without a default",
		},
		{
			name:              "backward — field removed is safe (reader ignores unknown writer fields)",
			compatibilityMode: "BACKWARD",
			path:              "testdata/avro/v2_field_removed.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           false,
		},
		{
			name:              "backward — type change is breaking",
			compatibilityMode: "BACKWARD",
			path:              "testdata/avro/v2_type_change.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           true,
			errContain:        "type changed",
		},
		{
			name:              "forward — new required field is safe (old reader ignores unknown writer fields)",
			compatibilityMode: "FORWARD",
			path:              "testdata/avro/v2_breaking.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           false,
		},
		{
			name:              "forward — field removed is breaking (old reader expects it)",
			compatibilityMode: "FORWARD",
			path:              "testdata/avro/v2_field_removed.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           true,
			errContain:        "added to new schema without a default",
		},
		{
			name:              "full — type change breaks both directions",
			compatibilityMode: "FULL",
			path:              "testdata/avro/v2_type_change.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           true,
			errContain:        "type changed",
		},
		{
			name:              "NONE mode always passes",
			compatibilityMode: "NONE",
			path:              "testdata/avro/v2_breaking.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           false,
		},
		{
			name:              "multiple violations reported together",
			compatibilityMode: "BACKWARD",
			path:              "testdata/avro/v2_multi_violation.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           true,
			errContain:        "type changed",
		},
		{
			name:              "unknown compatibility mode",
			compatibilityMode: "INVALID",
			path:              "testdata/avro/v2_backward.avsc",
			against:           "testdata/avro/v1.avsc",
			wantErr:           true,
			errContain:        "unknown compatibility mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v.SetCompatibilityMode(tt.compatibilityMode)
			err := v.Breaking(tt.path, tt.against)
			if (err != nil) != tt.wantErr {
				t.Errorf("Breaking() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errContain != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("Breaking() error = %q, want substring %q", err, tt.errContain)
				}
			}
		})
	}
}

func TestAvroValidator_Breaking_MultiViolationCount(t *testing.T) {
	resolver := &ToolchainResolver{}
	v := NewAvroValidator(resolver)
	v.SetCompatibilityMode("BACKWARD")

	// v2_multi_violation has type change (id: string→long) AND new field without default (phone)
	err := v.Breaking("testdata/avro/v2_multi_violation.avsc", "testdata/avro/v1.avsc")
	if err == nil {
		t.Fatal("expected error for multiple violations")
	}
	msg := err.Error()
	if !strings.Contains(msg, "type changed") {
		t.Errorf("expected 'type changed' violation, got: %s", msg)
	}
	if !strings.Contains(msg, "without a default") {
		t.Errorf("expected 'without a default' violation, got: %s", msg)
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
