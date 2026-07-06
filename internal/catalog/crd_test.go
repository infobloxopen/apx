package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

const crdManifest = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: appcontracts.appkit.infoblox.dev
spec:
  group: appkit.infoblox.dev
  names:
    kind: AppContract
    listKind: AppContractList
    plural: appcontracts
    singular: appcontract
  scope: Namespaced
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
`

func TestDetectAPIIdentity_CRD(t *testing.T) {
	apiID, domain, line := detectAPIIdentity(
		filepath.FromSlash("crd/appkit.infoblox.dev/appcontract/v1alpha1/appcontract.yaml"))
	if apiID != "crd/appkit.infoblox.dev/appcontract/v1alpha1" {
		t.Errorf("apiID: got %q", apiID)
	}
	if domain != "appkit.infoblox.dev" {
		t.Errorf("domain: got %q", domain)
	}
	if line != "v1alpha1" {
		t.Errorf("line: got %q", line)
	}
}

func TestParseReleaseTag_CRD(t *testing.T) {
	apiID, version := ParseReleaseTag("crd/appkit.infoblox.dev/appcontract/v1alpha1/v1.0.0-alpha.1")
	if apiID != "crd/appkit.infoblox.dev/appcontract/v1alpha1" {
		t.Errorf("apiID: got %q", apiID)
	}
	if version != "v1.0.0-alpha.1" {
		t.Errorf("version: got %q", version)
	}
}

func TestScanDirectory_CRDEnrichment(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "crd", "appkit.infoblox.dev", "appcontract", "v1alpha1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "appcontract.yaml"), []byte(crdManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	g := NewGenerator(filepath.Join(root, "catalog.yaml"))
	modules, err := g.ScanDirectory(root)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(modules))
	}
	m := modules[0]
	if m.ID != "crd/appkit.infoblox.dev/appcontract/v1alpha1" {
		t.Errorf("id: got %q", m.ID)
	}
	if m.Format != "crd" {
		t.Errorf("format: got %q", m.Format)
	}
	if m.CRDGroup != "appkit.infoblox.dev" || m.CRDKind != "AppContract" {
		t.Errorf("GVK: group=%q kind=%q", m.CRDGroup, m.CRDKind)
	}
	if m.StorageVersion != "v1alpha1" {
		t.Errorf("storage version: got %q", m.StorageVersion)
	}
	if len(m.ServedVersions) != 1 || m.ServedVersions[0] != "v1alpha1" {
		t.Errorf("served versions: got %v", m.ServedVersions)
	}
}
