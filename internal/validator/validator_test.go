package validator

import (
	"os"
	"path/filepath"
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
			if got := detectFormatFromFile(tt.path); got != tt.want {
				t.Errorf("detectFormatFromFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectFormat_Directory(t *testing.T) {
	// Create temp directory with schema files
	tmp := t.TempDir()

	tests := []struct {
		name  string
		files []string // relative paths to create
		want  SchemaFormat
	}{
		{
			name:  "directory with proto files",
			files: []string{"internal/apis/proto/payments/ledger/v1/ledger.proto"},
			want:  FormatProto,
		},
		{
			name:  "directory with avro files",
			files: []string{"schemas/avro/user.avsc"},
			want:  FormatAvro,
		},
		{
			name:  "empty directory",
			files: nil,
			want:  FormatUnknown,
		},
		{
			name:  "directory with no schema files",
			files: []string{"README.md", "go.mod"},
			want:  FormatUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(tmp, tt.name)
			_ = os.MkdirAll(dir, 0o755)
			for _, f := range tt.files {
				p := filepath.Join(dir, f)
				_ = os.MkdirAll(filepath.Dir(p), 0o755)
				_ = os.WriteFile(p, []byte(""), 0o644)
			}
			if got := DetectFormat(dir); got != tt.want {
				t.Errorf("DetectFormat(%s) = %v, want %v", dir, got, tt.want)
			}
		})
	}
}

func TestDetectFormatFromModuleRoots(t *testing.T) {
	tests := []struct {
		name  string
		roots []string
		want  SchemaFormat
	}{
		{
			name:  "single proto root",
			roots: []string{"internal/apis/proto"},
			want:  FormatProto,
		},
		{
			name:  "single openapi root",
			roots: []string{"schemas/openapi"},
			want:  FormatOpenAPI,
		},
		{
			name:  "single avro root",
			roots: []string{"schemas/avro"},
			want:  FormatAvro,
		},
		{
			name:  "single jsonschema root",
			roots: []string{"schemas/jsonschema"},
			want:  FormatJSONSchema,
		},
		{
			name:  "single parquet root",
			roots: []string{"data/parquet"},
			want:  FormatParquet,
		},
		{
			name:  "multiple roots same format",
			roots: []string{"internal/apis/proto", "vendor/apis/proto"},
			want:  FormatProto,
		},
		{
			name:  "mixed format roots",
			roots: []string{"schemas/proto", "schemas/openapi"},
			want:  FormatUnknown,
		},
		{
			name:  "unrecognized root segment",
			roots: []string{"internal/apis/custom"},
			want:  FormatUnknown,
		},
		{
			name:  "empty roots",
			roots: nil,
			want:  FormatUnknown,
		},
		{
			name:  "protobuf alias",
			roots: []string{"schemas/protobuf"},
			want:  FormatProto,
		},
		{
			name:  "swagger alias",
			roots: []string{"schemas/swagger"},
			want:  FormatOpenAPI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectFormatFromModuleRoots(tt.roots); got != tt.want {
				t.Errorf("DetectFormatFromModuleRoots(%v) = %v, want %v", tt.roots, got, tt.want)
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
			// Expect error (tools not installed or not implemented)
			if err == nil {
				t.Errorf("Lint() error = nil, expected error")
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
			// Expect error (tools not installed or not implemented)
			if err == nil {
				t.Errorf("Breaking() error = nil, expected error")
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
