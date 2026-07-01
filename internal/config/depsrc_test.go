package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// --- DependencyLock YAML round-trip + IsOverride truth table ---------------

func TestDependencyLock_YAMLRoundTrip_Overrides(t *testing.T) {
	in := DependencyLock{
		Repo:    "github.com/acme/apis",
		Ref:     "override",
		Modules: []string{"openapi/billing/invoices/v2"},
		Path:    "../billing-api",
		Git:     "github.com/acme/apis",
		GitRef:  "feature-branch",
	}

	data, err := yaml.Marshal(in)
	require.NoError(t, err)
	assert.Contains(t, string(data), "path: ../billing-api")
	assert.Contains(t, string(data), "git: github.com/acme/apis")
	assert.Contains(t, string(data), "git_ref: feature-branch")

	var out DependencyLock
	require.NoError(t, yaml.Unmarshal(data, &out))
	assert.Equal(t, in, out)
}

func TestDependencyLock_OverrideFieldsOmittedWhenEmpty(t *testing.T) {
	// A released dependency (no override) must marshal byte-identically to the
	// pre-Phase-3 shape: no path/git/git_ref keys.
	in := DependencyLock{
		Repo:    "github.com/acme/apis",
		Ref:     "v1.2.3",
		Modules: []string{"openapi/billing/invoices/v2"},
	}
	data, err := yaml.Marshal(in)
	require.NoError(t, err)
	s := string(data)
	assert.NotContains(t, s, "path:")
	assert.NotContains(t, s, "git:")
	assert.NotContains(t, s, "git_ref:")
}

func TestDependencyLock_IsOverride(t *testing.T) {
	cases := []struct {
		name string
		dep  DependencyLock
		want bool
	}{
		{"released version", DependencyLock{Ref: "v1.2.3"}, false},
		{"empty", DependencyLock{}, false},
		{"path override", DependencyLock{Path: "../x"}, true},
		{"git override", DependencyLock{Git: "github.com/o/r", GitRef: "br"}, true},
		{"both path and git", DependencyLock{Path: "../x", Git: "github.com/o/r"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.dep.IsOverride())
		})
	}
}

// --- MaterializeSpec: local path override ----------------------------------

// writeOpenAPISpecTree lays out an api-id directory beneath root that
// ResolveAPIPath can find, containing an OpenAPI spec.
//
// Layout: <root>/<apiID>/<name>.openapi.yaml
func writeOpenAPISpecTree(t *testing.T, root, apiID, specName string) string {
	t.Helper()
	apiDir := filepath.Join(root, filepath.FromSlash(apiID))
	require.NoError(t, os.MkdirAll(apiDir, 0o755))
	spec := filepath.Join(apiDir, specName)
	require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.0.0\ninfo:\n  title: x\n  version: 1.0.0\npaths: {}\n"), 0o644))
	return spec
}

func TestMaterializeSpec_LocalPath_ResolvesSpecInApiDir(t *testing.T) {
	root := t.TempDir()
	root, _ = filepath.EvalSymlinks(root)
	apiID := "openapi/billing/invoices/v2"
	wantSpec := writeOpenAPISpecTree(t, root, apiID, "invoices.openapi.yaml")

	dep := DependencyLock{Path: root, Ref: "override"}
	got, cleanup, err := MaterializeSpec(dep, apiID)
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	require.NoError(t, cleanup())

	gotAbs, _ := filepath.EvalSymlinks(got)
	assert.Equal(t, wantSpec, gotAbs)
}

func TestMaterializeSpec_LocalPath_RepoRootOpenAPIConvention(t *testing.T) {
	// When the api-id directory has no spec, fall back to the producer's
	// repo-root convention: <root>/openapi/*.openapi.yaml.
	root := t.TempDir()
	root, _ = filepath.EvalSymlinks(root)
	apiID := "openapi/billing/invoices/v2"
	// Create the api-id dir but leave it empty of specs.
	require.NoError(t, os.MkdirAll(filepath.Join(root, filepath.FromSlash(apiID)), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "openapi"), 0o755))
	wantSpec := filepath.Join(root, "openapi", "invoices.openapi.yaml")
	require.NoError(t, os.WriteFile(wantSpec, []byte("openapi: 3.0.0\n"), 0o644))

	dep := DependencyLock{Path: root}
	got, _, err := MaterializeSpec(dep, apiID)
	require.NoError(t, err)
	gotAbs, _ := filepath.EvalSymlinks(got)
	assert.Equal(t, wantSpec, gotAbs)
}

func TestMaterializeSpec_LocalPath_NotFound(t *testing.T) {
	root := t.TempDir()
	dep := DependencyLock{Path: root}
	_, _, err := MaterializeSpec(dep, "openapi/billing/invoices/v2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no OpenAPI spec found")
}

func TestMaterializeSpec_LocalPath_MissingDir(t *testing.T) {
	dep := DependencyLock{Path: filepath.Join(t.TempDir(), "does-not-exist")}
	_, _, err := MaterializeSpec(dep, "openapi/billing/invoices/v2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not an existing directory")
}

func TestMaterializeSpec_NoOverride(t *testing.T) {
	_, _, err := MaterializeSpec(DependencyLock{Ref: "v1.0.0"}, "openapi/x/y/v1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no override")
}

// --- MaterializeSpec: git override (offline, local bare repo) --------------

func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// Deterministic, hermetic identity so `git commit` succeeds without a
	// user's global config.
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=apx-test", "GIT_AUTHOR_EMAIL=apx@test.local",
		"GIT_COMMITTER_NAME=apx-test", "GIT_COMMITTER_EMAIL=apx@test.local",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, string(out))
}

func TestMaterializeSpec_Git_LocalBareRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	base := t.TempDir()
	bare := filepath.Join(base, "apis.git")
	work := filepath.Join(base, "work")
	require.NoError(t, os.MkdirAll(bare, 0o755))
	require.NoError(t, os.MkdirAll(work, 0o755))

	// Bare origin.
	gitCmd(t, bare, "init", "--bare", "-b", "main")

	// Work clone: create the api-id dir + spec, commit, push a feature branch.
	gitCmd(t, base, "clone", bare, work)
	apiID := "openapi/billing/invoices/v2"
	writeOpenAPISpecTree(t, work, apiID, "invoices.openapi.yaml")
	gitCmd(t, work, "add", "-A")
	gitCmd(t, work, "commit", "-m", "add invoices spec")
	gitCmd(t, work, "checkout", "-b", "feature-x")
	gitCmd(t, work, "push", "origin", "feature-x")

	// Redirect the depsrc cache to a temp dir so we don't touch ~/.cache.
	cache := filepath.Join(base, "cache")
	t.Setenv(depSrcCacheEnv, cache)

	dep := DependencyLock{Git: "file://" + bare, GitRef: "feature-x"}
	got, cleanup, err := MaterializeSpec(dep, apiID)
	require.NoError(t, err)
	require.NoError(t, cleanup())

	assert.FileExists(t, got)
	assert.Contains(t, got, filepath.FromSlash(apiID))
	// The clone lives under the redirected cache.
	assert.Contains(t, got, cache)

	// Second call reuses the cached checkout (still resolves).
	got2, _, err := MaterializeSpec(dep, apiID)
	require.NoError(t, err)
	assert.Equal(t, got, got2)
}

func TestMaterializeSpec_Git_CommitSHAFallback(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	base := t.TempDir()
	bare := filepath.Join(base, "apis.git")
	work := filepath.Join(base, "work")
	require.NoError(t, os.MkdirAll(bare, 0o755))

	gitCmd(t, bare, "init", "--bare", "-b", "main")
	gitCmd(t, base, "clone", bare, work)
	apiID := "openapi/billing/invoices/v2"
	writeOpenAPISpecTree(t, work, apiID, "invoices.openapi.yaml")
	gitCmd(t, work, "add", "-A")
	gitCmd(t, work, "commit", "-m", "add invoices spec")
	gitCmd(t, work, "push", "origin", "main")

	// Capture the commit SHA — a --branch clone rejects it, exercising the
	// full-clone + checkout fallback path.
	sha := gitRevParse(t, work)

	cache := filepath.Join(base, "cache")
	t.Setenv(depSrcCacheEnv, cache)

	dep := DependencyLock{Git: "file://" + bare, GitRef: sha}
	got, _, err := MaterializeSpec(dep, apiID)
	require.NoError(t, err)
	assert.FileExists(t, got)
}

func gitRevParse(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.Output()
	require.NoError(t, err)
	return string([]byte(out[:len(out)-1])) // strip trailing newline
}

func TestMaterializeSpec_Git_MissingRef(t *testing.T) {
	dep := DependencyLock{Git: "file:///nonexistent"}
	_, _, err := MaterializeSpec(dep, "openapi/x/y/v1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no git_ref")
}

func TestSanitizeForPath(t *testing.T) {
	cases := map[string]string{
		"github.com/acme/apis":     "github.com_acme_apis",
		"github.com/acme/apis.git": "github.com_acme_apis",
		"feature/x":                "feature_x",
		"v1.2.3":                   "v1.2.3",
		// Traversal is neutralized: no path separator survives, so the result is
		// always a single safe path segment (dots are kept but cannot escape).
		"../../etc/passwd": ".._.._etc_passwd",
	}
	for in, want := range cases {
		got := sanitizeForPath(in)
		assert.Equal(t, want, got, "input %q", in)
		assert.NotContains(t, got, "/", "sanitized %q must not contain a path separator", in)
	}
}
