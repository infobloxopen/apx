package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/infobloxopen/apx/internal/config"
)

// initCanonicalRepo creates a throwaway git repo that mimics a canonical
// catalog: it writes the given module files under the api-id path, commits
// them, and tags the commit with the module's release tag. It returns the repo
// dir.
func initCanonicalRepo(t *testing.T, apiID, version string, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-q")
	modDir := filepath.Join(repo, filepath.FromSlash(apiID))
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(modDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "-A")
	run("commit", "-q", "-m", "release")
	run("tag", config.DeriveTag(apiID, version))
	return repo
}

// writeLocalModule creates a local module dir with the given files.
func writeLocalModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestComputeDriftStatus(t *testing.T) {
	const apiID = "openapi/csp.infoblox.com/identity/v2"
	const specA = "openapi: 3.0.3\ninfo:\n  title: Identity\n  version: v2\n"
	const specB = "openapi: 3.0.3\ninfo:\n  title: Identity CHANGED\n  version: v2\n"

	t.Run("unchanged ignores generated go.mod", func(t *testing.T) {
		// Canonical carries the spec AND a generated go.mod; local carries only
		// the spec. Schema-scoped hashing must treat these as in sync.
		repo := initCanonicalRepo(t, apiID, "v2.0.0", map[string]string{
			"identity.yaml": specA,
			"go.mod":        "module github.com/acme/apis/openapi/csp.infoblox.com/identity/v2\n\ngo 1.21\n",
		})
		local := writeLocalModule(t, map[string]string{"identity.yaml": specA})

		res, err := computeDriftStatus(apiID, local, repo, "gitea.example.com/acme/apis", "")
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != DriftUnchanged {
			t.Fatalf("status = %q, want unchanged (local=%s published=%s)", res.Status, res.LocalHash, res.PublishedHash)
		}
		if res.PublishedVersion != "v2.0.0" {
			t.Errorf("published version = %q, want v2.0.0", res.PublishedVersion)
		}
	})

	t.Run("changed spec", func(t *testing.T) {
		repo := initCanonicalRepo(t, apiID, "v2.0.0", map[string]string{"identity.yaml": specA})
		local := writeLocalModule(t, map[string]string{"identity.yaml": specB})

		res, err := computeDriftStatus(apiID, local, repo, "", "")
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != DriftChanged {
			t.Fatalf("status = %q, want changed", res.Status)
		}
	})

	t.Run("absent when unpublished", func(t *testing.T) {
		// Canonical publishes a different module; ours has no tags.
		repo := initCanonicalRepo(t, "openapi/csp.infoblox.com/other/v1", "v1.0.0",
			map[string]string{"other.yaml": specA})
		local := writeLocalModule(t, map[string]string{"identity.yaml": specA})

		res, err := computeDriftStatus(apiID, local, repo, "", "")
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != DriftAbsent {
			t.Fatalf("status = %q, want absent", res.Status)
		}
	})

	t.Run("picks newest published version", func(t *testing.T) {
		repo := initCanonicalRepo(t, apiID, "v2.0.0", map[string]string{"identity.yaml": specA})
		// Add a newer tag with changed content on top.
		run := func(args ...string) {
			cmd := exec.Command("git", args...)
			cmd.Dir = repo
			cmd.Env = append(os.Environ(),
				"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
				"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v\n%s", args, err, out)
			}
		}
		if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(apiID), "identity.yaml"), []byte(specB), 0o644); err != nil {
			t.Fatal(err)
		}
		run("add", "-A")
		run("commit", "-q", "-m", "v2.1.0")
		run("tag", config.DeriveTag(apiID, "v2.1.0"))

		local := writeLocalModule(t, map[string]string{"identity.yaml": specB})
		res, err := computeDriftStatus(apiID, local, repo, "", "")
		if err != nil {
			t.Fatal(err)
		}
		if res.PublishedVersion != "v2.1.0" {
			t.Fatalf("published version = %q, want v2.1.0", res.PublishedVersion)
		}
		if res.Status != DriftUnchanged {
			t.Fatalf("status = %q, want unchanged against newest", res.Status)
		}
	})
}

func TestNormalizeCloneURL(t *testing.T) {
	cases := map[string]string{
		"github.com/acme/apis":             "https://github.com/acme/apis.git",
		"github.com/acme/apis.git":         "https://github.com/acme/apis.git",
		"gitea.example.com/acme/apis":      "https://gitea.example.com/acme/apis.git",
		"https://github.com/acme/apis.git": "https://github.com/acme/apis.git",
		"git@github.com:acme/apis.git":     "git@github.com:acme/apis.git",
	}
	for in, want := range cases {
		if got := normalizeCloneURL(in); got != want {
			t.Errorf("normalizeCloneURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsSchemaFile(t *testing.T) {
	// openapi module: yaml/yml/json are schema content.
	for _, f := range []string{"identity.yaml", "identity.openapi.yaml", "v2/identity.json", "sub/thing.yml"} {
		if !isSchemaFile(f, "openapi") {
			t.Errorf("isSchemaFile(%q, openapi) = false, want true", f)
		}
	}
	// Packaging and apx sidecars (incl. .yaml-extension dotfiles) must NOT be
	// schema — else a catalog sidecar in the module dir would force false drift.
	for _, f := range []string{
		"go.mod", "go.sum", "catalog.yaml", "apx.lock",
		".apx-release.yaml", ".apx-release-record.yaml", "README.md", "gen/foo.pb.go",
	} {
		if isSchemaFile(f, "openapi") {
			t.Errorf("isSchemaFile(%q, openapi) = true, want false", f)
		}
	}
	// Format-specific allowlist.
	if !isSchemaFile("ledger.proto", "proto") {
		t.Error("proto: .proto should be schema")
	}
	if isSchemaFile("ledger.proto", "openapi") {
		t.Error("openapi: .proto should not be schema")
	}
	if isSchemaFile("identity.yaml", "proto") {
		t.Error("proto: .yaml should not be schema")
	}
}
