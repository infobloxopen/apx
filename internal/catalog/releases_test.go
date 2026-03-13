package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateFromReleases_Basic(t *testing.T) {
	dir := t.TempDir()

	// Create release manifests
	apiDir := filepath.Join(dir, "proto", "google", "api", "v1")
	os.MkdirAll(apiDir, 0o755)

	os.WriteFile(filepath.Join(apiDir, "v1.0.0.yaml"), []byte(`
id: proto/google/api/v1
version: v1.0.0
format: proto
source_repo: github.com/infobloxopen/third-party-apis-google
source_ref: proto/google/api/v1/v1.0.0
source_path: google/api
import_mode: preserve
released_at: "2026-01-01T00:00:00Z"
`), 0o644)

	os.WriteFile(filepath.Join(apiDir, "v1.0.1.yaml"), []byte(`
id: proto/google/api/v1
version: v1.0.1
format: proto
source_repo: github.com/infobloxopen/third-party-apis-google
source_ref: proto/google/api/v1/v1.0.1
source_path: google/api
import_mode: preserve
released_at: "2026-02-01T00:00:00Z"
`), 0o644)

	rpcDir := filepath.Join(dir, "proto", "google", "rpc", "v1")
	os.MkdirAll(rpcDir, 0o755)

	os.WriteFile(filepath.Join(rpcDir, "v1.0.0.yaml"), []byte(`
id: proto/google/rpc/v1
version: v1.0.0
format: proto
source_repo: github.com/infobloxopen/third-party-apis-google
source_ref: proto/google/rpc/v1/v1.0.0
source_path: google/rpc
import_mode: preserve
`), 0o644)

	cat, err := GenerateFromReleases(dir, "infobloxopen", "apis")
	if err != nil {
		t.Fatal(err)
	}

	if len(cat.Modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(cat.Modules))
	}

	// Should be sorted: api before rpc
	api := cat.Modules[0]
	if api.ID != "proto/google/api/v1" {
		t.Fatalf("expected api first, got %s", api.ID)
	}
	if api.Version != "v1.0.1" {
		t.Fatalf("expected latest version v1.0.1, got %s", api.Version)
	}
	if api.Lifecycle != "stable" {
		t.Fatalf("expected stable, got %s", api.Lifecycle)
	}
	if api.Origin != "sourced" {
		t.Fatalf("expected sourced, got %s", api.Origin)
	}
	if api.ManagedRepo != "github.com/infobloxopen/third-party-apis-google" {
		t.Fatalf("expected managed repo, got %s", api.ManagedRepo)
	}
	if api.ImportMode != "preserve" {
		t.Fatalf("expected preserve, got %s", api.ImportMode)
	}
	if api.Path != "google/api" {
		t.Fatalf("expected google/api path, got %s", api.Path)
	}

	rpc := cat.Modules[1]
	if rpc.ID != "proto/google/rpc/v1" {
		t.Fatalf("expected rpc, got %s", rpc.ID)
	}
	if rpc.Version != "v1.0.0" {
		t.Fatalf("expected v1.0.0, got %s", rpc.Version)
	}
}

func TestGenerateFromReleases_Prerelease(t *testing.T) {
	dir := t.TempDir()

	apiDir := filepath.Join(dir, "proto", "infoblox", "authz", "v1")
	os.MkdirAll(apiDir, 0o755)

	os.WriteFile(filepath.Join(apiDir, "v1.0.0-alpha.1.yaml"), []byte(`
id: proto/infoblox/authz/v1
version: v1.0.0-alpha.1
format: proto
source_repo: github.com/Infoblox-CTO/ngp.authz
source_ref: proto/infoblox/authz/v1/v1.0.0-alpha.1
source_path: dbapiserver/pkg/pb
import_mode: preserve
`), 0o644)

	cat, err := GenerateFromReleases(dir, "infobloxopen", "apis")
	if err != nil {
		t.Fatal(err)
	}

	if len(cat.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(cat.Modules))
	}

	m := cat.Modules[0]
	if m.Lifecycle != "experimental" {
		t.Fatalf("expected experimental for alpha, got %s", m.Lifecycle)
	}
	if m.Version != "v1.0.0-alpha.1" {
		t.Fatalf("expected alpha version, got %s", m.Version)
	}
}

func TestGenerateFromReleases_Empty(t *testing.T) {
	dir := t.TempDir()
	cat, err := GenerateFromReleases(dir, "test", "apis")
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Modules) != 0 {
		t.Fatalf("expected 0 modules, got %d", len(cat.Modules))
	}
}

func TestGenerateFromReleases_InvalidManifest(t *testing.T) {
	dir := t.TempDir()
	apiDir := filepath.Join(dir, "proto", "x", "y", "v1")
	os.MkdirAll(apiDir, 0o755)

	// Missing id
	os.WriteFile(filepath.Join(apiDir, "v1.0.0.yaml"), []byte(`
version: v1.0.0
format: proto
`), 0o644)

	_, err := GenerateFromReleases(dir, "test", "apis")
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestGenerateFromReleases_StableOverridesPrerelease(t *testing.T) {
	dir := t.TempDir()
	apiDir := filepath.Join(dir, "proto", "test", "svc", "v1")
	os.MkdirAll(apiDir, 0o755)

	os.WriteFile(filepath.Join(apiDir, "v1.0.0-beta.1.yaml"), []byte(`
id: proto/test/svc/v1
version: v1.0.0-beta.1
format: proto
source_repo: github.com/test/repo
source_ref: proto/test/svc/v1/v1.0.0-beta.1
source_path: proto/svc
import_mode: preserve
`), 0o644)

	os.WriteFile(filepath.Join(apiDir, "v1.0.0.yaml"), []byte(`
id: proto/test/svc/v1
version: v1.0.0
format: proto
source_repo: github.com/test/repo
source_ref: proto/test/svc/v1/v1.0.0
source_path: proto/svc
import_mode: preserve
`), 0o644)

	cat, err := GenerateFromReleases(dir, "test", "apis")
	if err != nil {
		t.Fatal(err)
	}

	m := cat.Modules[0]
	if m.Version != "v1.0.0" {
		t.Fatalf("stable should be the version, got %s", m.Version)
	}
	if m.Lifecycle != "stable" {
		t.Fatalf("expected stable, got %s", m.Lifecycle)
	}
	if m.LatestPrerelease != "v1.0.0-beta.1" {
		t.Fatalf("expected prerelease tracked, got %s", m.LatestPrerelease)
	}
}
