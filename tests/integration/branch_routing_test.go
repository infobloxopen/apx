package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// End-to-end tests for the develop/main pre-release publishing model (ARCH-271,
// apx#30): source-branch → base-branch routing (B), pre-release version mechanics
// with a commit-hash-encoded version (A / AC-2), and the fail-closed ratchet
// (AC-1). They drive the real `apx release prepare` CLI against a git repo whose
// tags stand in for the canonical catalog's released versions.
//
// The scenarios are table-driven so a new branch/channel (e.g. a "staging"
// pre-release branch) is one row, not a new test.

// preparedManifest is the subset of .apx-release.yaml these tests assert on.
type preparedManifest struct {
	APIID            string `yaml:"api_id"`
	RequestedVersion string `yaml:"requested_version"`
	Tag              string `yaml:"tag"`
	BaseBranch       string `yaml:"base_branch"`
	Line             string `yaml:"line"`
}

// setupBranchRoutingRepo builds a git repo with an apx.yaml (custom
// branch_targets), one openapi module, and a GA tag (v1.1.1) for the module line
// so the ratchet has a floor to compare against. It returns the repo dir.
func setupBranchRoutingRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// A branch_targets map that both matches the default (main/master/develop)
	// AND adds a non-default entry (staging → develop) to prove the mapping is
	// honored and tweakable.
	apxYAML := `version: 1
org: acme
repo: apis
module_roots:
  - openapi
branch_targets:
  main: main
  master: main
  develop: develop
  staging: develop
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "apx.yaml"), []byte(apxYAML), 0o644))

	moduleDir := filepath.Join(dir, "openapi", "users", "v1")
	require.NoError(t, os.MkdirAll(moduleDir, 0o755))
	spec := "openapi: 3.0.0\ninfo:\n  title: Users\n  version: 1.1.1\npaths: {}\n"
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "users.yaml"), []byte(spec), 0o644))

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %s: %s", strings.Join(args, " "), string(out))
	}
	runGit("init")
	runGit("config", "user.name", "Test User")
	runGit("config", "user.email", "test@example.com")
	runGit("add", "-A")
	runGit("commit", "-m", "initial")
	// GA release tag for the openapi/users v1 line (tag prefix collapses v1).
	runGit("tag", "-a", "openapi/users/v1.1.1", "-m", "GA v1.1.1")

	return dir
}

// runPrepare runs `apx release prepare` in dir and returns combined output + err.
func runPrepare(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	apxBinary, err := filepath.Abs(getRelativeBinaryPath())
	require.NoError(t, err)

	full := append([]string{"release", "prepare"}, args...)
	cmd := exec.Command(apxBinary, full...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), ciEnv, noColorEnv, disableTTY, "GITHUB_REF_NAME=")
	out, runErr := cmd.CombinedOutput()
	return string(out), runErr
}

func readPreparedManifest(t *testing.T, dir string) preparedManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, ".apx-release.yaml"))
	require.NoError(t, err)
	var m preparedManifest
	require.NoError(t, yaml.Unmarshal(data, &m))
	return m
}

func TestBranchRouting_BaseBranchResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e prepare test in short mode")
	}
	const apiID = "openapi/users/v1"

	tests := []struct {
		name         string
		sourceBranch string
		version      string
		lifecycle    string
		encodeHash   bool
		wantBase     string
		wantPre      bool // expect a pre-release (hash-carrying) tag
	}{
		{
			name:         "main publishes stable to apis main",
			sourceBranch: "main",
			version:      "v1.2.0",
			lifecycle:    "stable",
			wantBase:     "main",
		},
		{
			name:         "master publishes stable to apis main",
			sourceBranch: "master",
			version:      "v1.2.0",
			lifecycle:    "stable",
			wantBase:     "main",
		},
		{
			name:         "develop publishes prerelease to apis develop",
			sourceBranch: "develop",
			version:      "v1.2.0-beta.1",
			lifecycle:    "beta",
			encodeHash:   true,
			wantBase:     "develop",
			wantPre:      true,
		},
		{
			name:         "non-default staging mapping is honored",
			sourceBranch: "staging",
			version:      "v1.2.0-beta.1",
			lifecycle:    "beta",
			encodeHash:   true,
			wantBase:     "develop",
			wantPre:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupBranchRoutingRepo(t)
			args := []string{
				apiID,
				"--version", tt.version,
				"--lifecycle", tt.lifecycle,
				"--source-branch", tt.sourceBranch,
				"--canonical-repo", "github.com/acme/apis",
				"--canonical-dir", dir,
			}
			if tt.encodeHash {
				args = append(args, "--encode-commit-hash")
			}
			out, err := runPrepare(t, dir, args...)
			require.NoErrorf(t, err, "prepare failed: %s", out)

			m := readPreparedManifest(t, dir)
			assert.Equal(t, tt.wantBase, m.BaseBranch, "resolved base branch")

			if tt.wantPre {
				// AC-2: the pre-release version carries a g-prefixed commit hash in
				// the pre-release segment, and both the version and its tag are
				// valid SemVer AND valid Go module versions.
				assert.Regexp(t, `-beta\.1\.g[0-9a-f]{7,12}$`, m.RequestedVersion,
					"prerelease must carry a g-prefixed commit hash")
				assertValidGoModuleVersion(t, m.RequestedVersion)
				assert.Contains(t, m.Tag, "openapi/users/"+m.RequestedVersion)
			} else {
				assert.NotContains(t, m.RequestedVersion, "-", "stable release has no prerelease")
				assert.Equal(t, "openapi/users/"+tt.version, m.Tag)
			}
		})
	}
}

// TestBranchRouting_RatchetFailsClosed covers AC-1: a pre-release at or below the
// line's highest GA is rejected; one strictly above it is accepted.
func TestBranchRouting_RatchetFailsClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e prepare test in short mode")
	}
	const apiID = "openapi/users/v1"

	t.Run("prerelease at existing GA is rejected", func(t *testing.T) {
		dir := setupBranchRoutingRepo(t) // GA v1.1.1 exists
		out, err := runPrepare(t, dir,
			apiID,
			"--version", "v1.1.1-beta.1",
			"--lifecycle", "beta",
			"--source-branch", "develop",
			"--canonical-repo", "github.com/acme/apis",
			"--canonical-dir", dir,
		)
		require.Error(t, err, "ratchet must fail closed: %s", out)
		assert.Contains(t, out, "v1.1.2-beta.1", "error should state the next legal version")
	})

	t.Run("prerelease above GA is accepted", func(t *testing.T) {
		dir := setupBranchRoutingRepo(t)
		out, err := runPrepare(t, dir,
			apiID,
			"--version", "v1.1.2-beta.1",
			"--lifecycle", "beta",
			"--source-branch", "develop",
			"--encode-commit-hash",
			"--canonical-repo", "github.com/acme/apis",
			"--canonical-dir", dir,
		)
		require.NoErrorf(t, err, "prerelease above GA must succeed: %s", out)
	})
}

// assertValidGoModuleVersion asserts v is valid SemVer and a valid Go module
// version (go get-resolvable) — the AC-2 constraint that the hash must not break
// Go module resolution.
func assertValidGoModuleVersion(t *testing.T, v string) {
	t.Helper()
	assert.Truef(t, semver.IsValid(v), "version %q must be valid semver", v)
	// A representative v1 module path; module.Check validates the version's
	// pseudo/prerelease form for Go module resolution.
	err := module.Check("github.com/acme/apis/openapi/users", v)
	assert.NoErrorf(t, err, "version %q must be a valid Go module version", v)
}
