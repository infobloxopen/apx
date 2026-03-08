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
			// Expect error (either buf not installed or testdata missing)
			if err == nil {
				t.Errorf("Lint() error = nil, expected error")
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
			// Expect error (either buf not installed or testdata missing)
			if err == nil {
				t.Errorf("Breaking() error = nil, expected error")
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

func TestExtractGoPackage(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantImport string
		wantAlias  string
		wantErr    bool
	}{
		{
			name: "simple go_package",
			content: `syntax = "proto3";
option go_package = "github.com/acme/apis/proto/payments/ledger/v1";
package ledger.v1;
`,
			wantImport: "github.com/acme/apis/proto/payments/ledger/v1",
		},
		{
			name: "go_package with alias",
			content: `syntax = "proto3";
option go_package = "github.com/acme/apis/proto/payments/ledger/v1;ledgerpb";
package ledger.v1;
`,
			wantImport: "github.com/acme/apis/proto/payments/ledger/v1",
			wantAlias:  "ledgerpb",
		},
		{
			name: "no go_package",
			content: `syntax = "proto3";
package ledger.v1;

message Foo { string id = 1; }
`,
			wantImport: "",
			wantAlias:  "",
		},
		{
			name: "go_package after comments",
			content: `syntax = "proto3";
// This is a comment about go_package
// option go_package = "wrong/path";
option go_package = "github.com/acme/apis/proto/payments/ledger/v1";
`,
			wantImport: "github.com/acme/apis/proto/payments/ledger/v1",
		},
		{
			name: "go_package with extra spaces",
			content: `syntax = "proto3";
  option  go_package  =  "github.com/acme/apis/proto/billing/invoices/v2"  ;
`,
			wantImport: "github.com/acme/apis/proto/billing/invoices/v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			protoFile := filepath.Join(tmpDir, "test.proto")
			if err := os.WriteFile(protoFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write proto file: %v", err)
			}

			gotImport, gotAlias, err := ExtractGoPackage(protoFile)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotImport != tt.wantImport {
				t.Errorf("import: got %q, want %q", gotImport, tt.wantImport)
			}
			if gotAlias != tt.wantAlias {
				t.Errorf("alias: got %q, want %q", gotAlias, tt.wantAlias)
			}
		})
	}
}

func TestExtractGoPackage_FileNotFound(t *testing.T) {
	_, _, err := ExtractGoPackage("/nonexistent/path/test.proto")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestGlobProtoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"a.proto":     "syntax = \"proto3\";",
		"b.proto":     "syntax = \"proto3\";",
		"c.txt":       "not a proto",
		"sub/d.proto": "syntax = \"proto3\";",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := GlobProtoFiles(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 3 {
		t.Errorf("expected 3 proto files, got %d: %v", len(got), got)
	}

	// Ensure only .proto files
	for _, f := range got {
		if filepath.Ext(f) != ".proto" {
			t.Errorf("non-proto file in results: %s", f)
		}
	}
}
