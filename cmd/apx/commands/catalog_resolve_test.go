package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infobloxopen/apx/internal/ui"
)

const resolveTestCatalog = `version: 1
org: acme
repo: apis
modules:
  - id: proto/iam/v1
    format: proto
    domain: iam
    api_line: v1
    version: v1.2.3
    lifecycle: stable
    path: proto/iam/v1
    resource_types:
      - iam.example.com/User
  - id: proto/dupe-a/v1
    format: proto
    version: v1.0.0
    lifecycle: stable
    resource_types: [shared.example.com/Thing]
  - id: proto/dupe-b/v1
    format: proto
    version: v1.0.0
    lifecycle: stable
    resource_types: [shared.example.com/Thing]
`

func writeResolveCatalog(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")
	if err := os.WriteFile(path, []byte(resolveTestCatalog), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
	return path
}

func runResolve(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out strings.Builder
	ui.SetOutput(&out)
	ui.SetErrorOutput(&out)
	defer ui.SetOutput(os.Stdout)
	defer ui.SetErrorOutput(os.Stderr)

	cmd := NewRootCmd("test")
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestCatalogResolve_KnownType(t *testing.T) {
	path := writeResolveCatalog(t)
	out, err := runResolve(t, "catalog", "resolve", "--catalog", path, "iam.example.com/User")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	for _, want := range []string{"iam.example.com/User", "proto/iam/v1", "v1.2.3", "stable"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nGot:\n%s", want, out)
		}
	}
}

func TestCatalogResolve_KnownType_JSON(t *testing.T) {
	path := writeResolveCatalog(t)
	out, err := runResolve(t, "--json", "catalog", "resolve", "--catalog", path, "iam.example.com/User")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	for _, want := range []string{`"type": "iam.example.com/User"`, `"module_id": "proto/iam/v1"`} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON output missing %q\nGot:\n%s", want, out)
		}
	}
}

func TestCatalogResolve_UnknownType_FailsLoud(t *testing.T) {
	path := writeResolveCatalog(t)
	_, err := runResolve(t, "catalog", "resolve", "--catalog", path, "iam.example.com/Nope")
	if err == nil {
		t.Fatalf("expected a fail-loud error for unknown type, got nil")
	}
	if !strings.Contains(err.Error(), "iam.example.com/Nope") {
		t.Errorf("error should name the type, got: %v", err)
	}
}

func TestCatalogResolve_AmbiguousType_FailsLoud(t *testing.T) {
	path := writeResolveCatalog(t)
	_, err := runResolve(t, "catalog", "resolve", "--catalog", path, "shared.example.com/Thing")
	if err == nil {
		t.Fatalf("expected a fail-loud error for ambiguous type, got nil")
	}
	if !strings.Contains(err.Error(), "proto/dupe-a/v1") || !strings.Contains(err.Error(), "proto/dupe-b/v1") {
		t.Errorf("ambiguous error should list claimants, got: %v", err)
	}
}
