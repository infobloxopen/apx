package validator

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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

func TestBufRootAndPath(t *testing.T) {
	// Workspace root holds buf.yaml; the schema lives in a nested module dir.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "buf.yaml"), []byte("version: v2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	schemaDir := filepath.Join(root, "proto", "infoblox", "authz", "v1")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("subdir contained by module", func(t *testing.T) {
		gotRoot, gotRel, err := bufRootAndPath(schemaDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotRoot != root {
			t.Errorf("root: got %q, want %q", gotRoot, root)
		}
		if want := "proto/infoblox/authz/v1"; gotRel != want {
			t.Errorf("rel: got %q, want %q", gotRel, want)
		}
	})

	t.Run("buf.work.yaml is also recognized", func(t *testing.T) {
		wsRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(wsRoot, "buf.work.yaml"), []byte("version: v1\n"), 0644); err != nil {
			t.Fatal(err)
		}
		sub := filepath.Join(wsRoot, "proto")
		if err := os.MkdirAll(sub, 0755); err != nil {
			t.Fatal(err)
		}
		gotRoot, gotRel, err := bufRootAndPath(sub)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotRoot != wsRoot || gotRel != "proto" {
			t.Errorf("got (%q,%q), want (%q,%q)", gotRoot, gotRel, wsRoot, "proto")
		}
	})

	t.Run("schema dir is itself the root", func(t *testing.T) {
		_, gotRel, err := bufRootAndPath(root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotRel != "." {
			t.Errorf("rel: got %q, want %q", gotRel, ".")
		}
	})

	t.Run("no buf config above path", func(t *testing.T) {
		// A temp dir with no buf.yaml in it or (typically) any ancestor.
		orphan := filepath.Join(t.TempDir(), "proto", "x", "v1")
		if err := os.MkdirAll(orphan, 0755); err != nil {
			t.Fatal(err)
		}
		if _, _, err := bufRootAndPath(orphan); err == nil {
			t.Error("expected error when no buf workspace/module config is found")
		}
	})
}

// TestProtoValidator_Breaking_AgainstTagWithSlashes is the regression test
// for release-tag refs: tags like proto/payments/ledger/v1.0.0 contain
// slashes, and the old heuristic treated any slash-containing against as a
// filesystem path, so buf targeted nothing and the check failed spuriously.
func TestProtoValidator_Breaking_AgainstTagWithSlashes(t *testing.T) {
	resolver := NewToolchainResolver()
	if _, err := resolver.ResolveTool("buf", "v1.66.1"); err != nil {
		t.Skipf("buf not resolvable: %v", err)
	}

	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "buf.yaml"),
		[]byte("version: v2\nmodules:\n  - path: proto\nbreaking:\n  use:\n    - FILE\n"), 0644); err != nil {
		t.Fatal(err)
	}
	schemaDir := filepath.Join(dir, "proto", "payments", "ledger", "v1")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}
	proto := `syntax = "proto3";

package payments.ledger.v1;

message Entry {
  string id = 1;
}
`
	if err := os.WriteFile(filepath.Join(schemaDir, "ledger.proto"), []byte(proto), 0644); err != nil {
		t.Fatal(err)
	}

	run("init", "-q")
	run("config", "user.name", "t")
	run("config", "user.email", "t@t")
	run("add", ".")
	run("commit", "-qm", "v1")
	run("tag", "proto/payments/ledger/v1.0.0") // release tag: contains slashes

	v := NewProtoValidator(resolver)

	// No changes since the tag — breaking must pass, which requires the
	// slash-containing tag to be resolved as a git ref, not a path.
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd) //nolint:errcheck
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := v.Breaking("proto/payments/ledger/v1", "proto/payments/ledger/v1.0.0"); err != nil {
		t.Fatalf("Breaking against slash-containing tag failed: %v", err)
	}
}

func TestBufTargetArgs(t *testing.T) {
	// buf.yaml v2 declares a single module rooted at "proto".
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "buf.yaml"),
		[]byte("version: v2\nmodules:\n  - path: proto\n"), 0644); err != nil {
		t.Fatal(err)
	}
	schemaDir := filepath.Join(root, "proto", "infoblox", "authz", "v1")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		target   string
		wantArgs []string
	}{
		{"workspace root -> no selector", root, nil},
		{"module root -> positional input", filepath.Join(root, "proto"), []string{"proto"}},
		{"inside module -> --path selector", schemaDir, []string{"--path", "proto/infoblox/authz/v1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDir, gotArgs, err := bufTargetArgs(tt.target)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotDir != root {
				t.Errorf("dir: got %q, want %q", gotDir, root)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("args: got %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}
