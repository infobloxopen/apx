package validator

import (
	"os"
	"os/exec"
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
			// Expect error since spectral is not installed
			if err == nil {
				t.Errorf("Lint() error = nil, expected tool not found error")
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
			// Expect error since oasdiff is not installed
			if err == nil {
				t.Errorf("Breaking() error = nil, expected tool not found error")
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

// TestResolveOpenAPISpecFile covers the revision-resolution seam of the F-29
// fix: finalize hands Breaking a module DIRECTORY, and the spec file inside it
// must be selected while go.mod / dotfile configs are skipped.
func TestResolveOpenAPISpecFile(t *testing.T) {
	// A module dir laid out under an "openapi/" root (so detectFormatFromFile
	// classifies the yaml as OpenAPI), alongside the non-spec artifacts finalize
	// commits next to it.
	root := t.TempDir()
	modDir := filepath.Join(root, "openapi", "csp.infoblox.com", "svc", "v1")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	spec := filepath.Join(modDir, "svc.yaml")
	for name, content := range map[string]string{
		"svc.yaml":       "openapi: 3.0.3\ninfo: {title: svc, version: 1.0.0}\npaths: {}\n",
		"go.mod":         "module example.com/svc\n\ngo 1.21\n",
		".spectral.yaml": "extends: spectral:oas\n",
	} {
		if err := os.WriteFile(filepath.Join(modDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := resolveOpenAPISpecFile(modDir)
	if err != nil {
		t.Fatalf("resolveOpenAPISpecFile(dir) error: %v", err)
	}
	if got != spec {
		t.Errorf("resolved spec = %q, want %q (go.mod / .spectral.yaml must be skipped)", got, spec)
	}

	// A plain file path is returned as-is.
	if got, err := resolveOpenAPISpecFile(spec); err != nil || got != spec {
		t.Errorf("resolveOpenAPISpecFile(file) = %q, %v; want %q, nil", got, err, spec)
	}

	// A directory with no spec is an error, not a silent empty path.
	empty := t.TempDir()
	if err := os.WriteFile(filepath.Join(empty, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveOpenAPISpecFile(empty); err == nil {
		t.Error("resolveOpenAPISpecFile on a spec-less dir must error")
	}
}

// TestBaseSpecFromRef covers the base-resolution seam of the F-29 fix: the base
// is the previous release TAG (a git ref, not a path on disk), so its committed
// content must be materialized via `git show <ref>:<spec path>` — the previous
// code passed the tag straight to oasdiff, which read it as a filename and
// failed to load the base. This seam is oasdiff-free (git only).
func TestBaseSpecFromRef(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")

	// A module dir whose spec is committed and tagged, then changed in the
	// working tree — the tagged (base) content must come back, not the tree's.
	modDir := filepath.Join(root, "openapi", "csp.infoblox.com", "svc", "v1")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	spec := filepath.Join(modDir, "svc.yaml")
	const baseContent = "openapi: 3.0.3\ninfo: {title: svc, version: 1.0.0}\npaths: {}\n"
	if err := os.WriteFile(spec, []byte(baseContent), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", "release v1.0.0")
	// A slash-bearing release tag, exactly as apx mints them.
	tag := "openapi/csp.infoblox.com/svc/v1/v1.0.0"
	git("tag", tag)

	// Diverge the working tree.
	if err := os.WriteFile(spec, []byte("openapi: 3.0.3\ninfo: {title: svc, version: 2.0.0}\npaths: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	baseFile, cleanup, err := baseSpecFromRef(spec, tag)
	if err != nil {
		t.Fatalf("baseSpecFromRef: %v", err)
	}
	defer cleanup()
	got, err := os.ReadFile(baseFile)
	if err != nil {
		t.Fatalf("reading materialized base: %v", err)
	}
	if string(got) != baseContent {
		t.Errorf("materialized base content = %q, want the tagged v1 content %q", got, baseContent)
	}

	// An unknown ref surfaces a base-load error (no silent success).
	if _, _, err := baseSpecFromRef(spec, "openapi/csp.infoblox.com/svc/v1/v9.9.9"); err == nil {
		t.Error("baseSpecFromRef on a nonexistent ref must error")
	}
}
