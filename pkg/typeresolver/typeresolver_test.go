package typeresolver_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/pkg/typeresolver"
)

const consumerCatalog = `version: 1
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
  - id: proto/a/v1
    format: proto
    version: v1.0.0
    resource_types: [shared.example.com/Thing]
  - id: proto/b/v1
    format: proto
    version: v1.0.0
    resource_types: [shared.example.com/Thing]
`

// writeCatalog writes the fixture catalog and returns a LocalSource for it —
// mirroring how the devedge-sdk F041 ReferenceResolver wires apx as its backing
// resolver over a catalog source.
func writeCatalog(t *testing.T) typeresolver.Source {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")
	if err := os.WriteFile(path, []byte(consumerCatalog), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
	return catalog.SourceFor(path)
}

func TestResolve_KnownType(t *testing.T) {
	res, err := typeresolver.Resolve(writeCatalog(t), "iam.example.com/User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ModuleID != "proto/iam/v1" {
		t.Errorf("ModuleID = %q, want proto/iam/v1", res.ModuleID)
	}
	if res.Version != "v1.2.3" || res.Lifecycle != "stable" {
		t.Errorf("coordinates missing: %+v", res)
	}
}

func TestResolve_Unknown_FailLoud(t *testing.T) {
	_, err := typeresolver.Resolve(writeCatalog(t), "iam.example.com/Missing")
	if !errors.Is(err, typeresolver.ErrUnresolved) {
		t.Fatalf("want ErrUnresolved, got %v", err)
	}
}

func TestResolve_Ambiguous_FailLoud(t *testing.T) {
	_, err := typeresolver.Resolve(writeCatalog(t), "shared.example.com/Thing")
	if !errors.Is(err, typeresolver.ErrAmbiguous) {
		t.Fatalf("want ErrAmbiguous, got %v", err)
	}
}
