package validator

import (
	"testing"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name string
		path string
		want SchemaFormat
	}{
		{
			name: "proto file",
			path: "schemas/user.proto",
			want: FormatProto,
		},
		{
			name: "openapi yaml",
			path: "schemas/openapi.yaml",
			want: FormatOpenAPI,
		},
		{
			name: "swagger yaml",
			path: "schemas/swagger.yaml",
			want: FormatOpenAPI,
		},
		{
			name: "avro schema",
			path: "schemas/user.avsc",
			want: FormatAvro,
		},
		{
			name: "avro in directory",
			path: "schemas/avro/user.json",
			want: FormatAvro,
		},
		{
			name: "jsonschema in directory",
			path: "schemas/jsonschema/user.json",
			want: FormatJSONSchema,
		},
		{
			name: "parquet extension",
			path: "schemas/events.parquet",
			want: FormatParquet,
		},
		{
			name: "unknown format",
			path: "schemas/data.txt",
			want: FormatUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectFormat(tt.path); got != tt.want {
				t.Errorf("DetectFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidator_Lint(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewValidator(resolver)

	tests := []struct {
		name    string
		path    string
		format  SchemaFormat
		wantErr bool
	}{
		{
			name:    "proto format explicit",
			path:    "testdata/proto/valid.proto",
			format:  FormatProto,
			wantErr: false,
		},
		{
			name:    "proto format detected",
			path:    "testdata/proto/valid.proto",
			format:  FormatUnknown,
			wantErr: false,
		},
		{
			name:    "unknown format",
			path:    "testdata/unknown.xyz",
			format:  FormatUnknown,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Lint(tt.path, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_Breaking(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewValidator(resolver)

	tests := []struct {
		name    string
		path    string
		against string
		format  SchemaFormat
		wantErr bool
	}{
		{
			name:    "proto format explicit",
			path:    "testdata/proto/v2.proto",
			against: "testdata/proto/v1.proto",
			format:  FormatProto,
			wantErr: false,
		},
		{
			name:    "unknown format",
			path:    "testdata/unknown.xyz",
			against: "testdata/unknown.xyz",
			format:  FormatUnknown,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Breaking(tt.path, tt.against, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("Breaking() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_SetAvroCompatibilityMode(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewValidator(resolver)

	modes := []string{"BACKWARD", "FORWARD", "FULL", "NONE"}
	for _, mode := range modes {
		validator.SetAvroCompatibilityMode(mode)
		if validator.avroValidator.compatibilityMode != mode {
			t.Errorf("SetAvroCompatibilityMode(%s) failed", mode)
		}
	}
}

func TestValidator_SetParquetAdditiveNullableOnly(t *testing.T) {
	resolver := &ToolchainResolver{}
	validator := NewValidator(resolver)

	validator.SetParquetAdditiveNullableOnly(false)
	if validator.parquetValidator.allowAdditiveNullableOnly {
		t.Error("SetParquetAdditiveNullableOnly(false) failed")
	}

	validator.SetParquetAdditiveNullableOnly(true)
	if !validator.parquetValidator.allowAdditiveNullableOnly {
		t.Error("SetParquetAdditiveNullableOnly(true) failed")
	}
}
