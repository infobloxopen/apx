package publisher

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// goModTargetDir — go.mod placement follows Go semantic-import-versioning
// ---------------------------------------------------------------------------

func TestGoModTargetDir(t *testing.T) {
	base := filepath.FromSlash("/canon/openapi/csp.infoblox.com/iam-identity")

	tests := []struct {
		name    string
		destDir string
		module  string
		want    string
	}{
		{
			name:    "v1 has no version suffix — rooted at family root",
			destDir: filepath.Join(base, "v1"),
			module:  "github.com/acme/apis/openapi/csp.infoblox.com/iam-identity",
			want:    base,
		},
		{
			name:    "v2 carries /v2 suffix — rooted in its version directory",
			destDir: filepath.Join(base, "v2"),
			module:  "github.com/acme/apis/openapi/csp.infoblox.com/iam-identity/v2",
			want:    filepath.Join(base, "v2"),
		},
		{
			name:    "v3 carries /v3 suffix — rooted in its version directory",
			destDir: filepath.Join(base, "v3"),
			module:  "github.com/acme/apis/openapi/csp.infoblox.com/iam-identity/v3",
			want:    filepath.Join(base, "v3"),
		},
		{
			name:    "v10 (double-digit major) — rooted in its version directory",
			destDir: filepath.Join(base, "v10"),
			module:  "github.com/acme/apis/openapi/csp.infoblox.com/iam-identity/v10",
			want:    filepath.Join(base, "v10"),
		},
		{
			name:    "internal v2 module — rooted in its own version directory",
			destDir: filepath.FromSlash("/canon/openapi/csp.infoblox.com/iam-identity-internal/v2"),
			module:  "github.com/acme/apis/openapi/csp.infoblox.com/iam-identity-internal/v2",
			want:    filepath.FromSlash("/canon/openapi/csp.infoblox.com/iam-identity-internal/v2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, goModTargetDir(tt.destDir, tt.module))
		})
	}
}

// ---------------------------------------------------------------------------
// stageRelease — file-writing path (no git, no network)
// ---------------------------------------------------------------------------

// makeSnapshot writes a minimal prepared snapshot (one schema file, no go.mod)
// into a fresh temp dir and returns its path.
func makeSnapshot(t *testing.T, filename string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte("schema"), 0o644))
	return dir
}

// stagedFiles stages the manifest's snapshot into a fresh clone dir and returns
// the sorted set of created file paths, relative to the clone (slash form).
func stagedFiles(t *testing.T, manifest *ReleaseManifest, snapshotDir string) (cloneDir string, rels []string) {
	t.Helper()
	cloneDir = t.TempDir()
	require.NoError(t, stageRelease(cloneDir, manifest, snapshotDir))

	err := filepath.Walk(cloneDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(cloneDir, path)
		if relErr != nil {
			return relErr
		}
		rels = append(rels, filepath.ToSlash(rel))
		return nil
	})
	require.NoError(t, err)
	sort.Strings(rels)
	return cloneDir, rels
}

// familyManifest builds a manifest for a released Go surface, deriving the
// module path per Go's SIV rules (no suffix for v0/v1, /vN for v2+) exactly as
// the language plugin does, so the fixture matches real releases.
func familyManifest(t *testing.T, name, line string) *ReleaseManifest {
	t.Helper()
	major, err := config.LineMajor(line)
	require.NoError(t, err)

	repoPrefix := "github.com/acme/apis"
	familyPath := "openapi/csp.infoblox.com/" + name // repo-relative family root
	canonicalPath := familyPath + "/" + line         // version directory holds the sources

	module := repoPrefix + "/" + familyPath
	if major >= 2 {
		module += "/v" + line[1:]
	}

	return &ReleaseManifest{
		APIID:            "openapi/csp.infoblox.com/" + name + "/" + line,
		Format:           "openapi",
		Domain:           "csp.infoblox.com",
		Name:             name,
		Line:             line,
		RequestedVersion: line + ".0.0",
		CanonicalRepo:    "github.com/acme/apis",
		CanonicalPath:    canonicalPath,
		Languages:        map[string]config.LanguageCoords{"go": {Module: module}},
	}
}

// TestStageRelease_ConcurrentMultiVersion_NonOverlapping is the apx#27 repro
// and its permanent regression guard. Two release PRs for different major
// versions of the same family, cut from the same base before either merges,
// must stage into DISJOINT file sets — otherwise the second PR becomes an
// add/add conflict on a shared family-root go.mod once the first merges.
//
// It FAILS against the pre-fix code, which wrote every version's go.mod to
// filepath.Dir(destDir) (the shared family root), so v1 and v2 both produced
// ".../iam-identity/go.mod".
func TestStageRelease_ConcurrentMultiVersion_NonOverlapping(t *testing.T) {
	v1 := familyManifest(t, "iam-identity", "v1")
	v2 := familyManifest(t, "iam-identity", "v2")

	_, v1Files := stagedFiles(t, v1, makeSnapshot(t, "openapi.yaml"))
	_, v2Files := stagedFiles(t, v2, makeSnapshot(t, "openapi.yaml"))

	// Order-insensitive: no path is written by both releases.
	v1Set := map[string]bool{}
	for _, f := range v1Files {
		v1Set[f] = true
	}
	var overlap []string
	for _, f := range v2Files {
		if v1Set[f] {
			overlap = append(overlap, f)
		}
	}
	assert.Empty(t, overlap, "concurrent v1 and v2 releases must not write any shared path (got overlap %v)", overlap)

	// The concrete failure mode: the family-root go.mod must belong to exactly
	// one release (v1), never both.
	familyGoMod := "openapi/csp.infoblox.com/iam-identity/go.mod"
	assert.Contains(t, v1Files, familyGoMod, "v0/v1 module is rooted at the family root")
	assert.NotContains(t, v2Files, familyGoMod, "v2 release must not write the shared family-root go.mod")

	// v2 writes its go.mod inside its own version subtree.
	assert.Contains(t, v2Files, "openapi/csp.infoblox.com/iam-identity/v2/go.mod")
}

// TestStageRelease_ModulePathMatchesDirectory asserts the SIV correctness
// property: a generated go.mod declaring "module M" lives at the directory Go
// expects for M — i.e. M's path relative to the canonical repo equals the
// go.mod's directory relative to the clone. This holds for v0/v1 (family root)
// and v2+ (version dir) alike, and FAILS against the pre-fix code for v2+.
func TestStageRelease_ModulePathMatchesDirectory(t *testing.T) {
	for _, line := range []string{"v1", "v2", "v3"} {
		t.Run(line, func(t *testing.T) {
			m := familyManifest(t, "iam-identity", line)
			cloneDir, files := stagedFiles(t, m, makeSnapshot(t, "openapi.yaml"))

			// Locate the single generated go.mod.
			var goModRel string
			for _, f := range files {
				if strings.HasSuffix(f, "/go.mod") || f == "go.mod" {
					require.Empty(t, goModRel, "exactly one go.mod expected, found a second: %s", f)
					goModRel = f
				}
			}
			require.NotEmpty(t, goModRel, "a go.mod must be generated")

			content, err := os.ReadFile(filepath.Join(cloneDir, filepath.FromSlash(goModRel)))
			require.NoError(t, err)
			module, err := ParseGoModModule(content)
			require.NoError(t, err)

			goModDirRel := filepath.ToSlash(filepath.Dir(goModRel))
			moduleRelDir := strings.TrimPrefix(module, "github.com/acme/apis/")

			assert.Equal(t, moduleRelDir, goModDirRel,
				"go.mod for %q must live at the directory its module path implies", module)
		})
	}
}

// TestStageRelease_PreservesSnapshotGoMod verifies the "write only if absent"
// guard: a go.mod already carried in the prepared snapshot is preserved, not
// clobbered by generation.
func TestStageRelease_PreservesSnapshotGoMod(t *testing.T) {
	m := familyManifest(t, "iam-identity", "v2")

	snap := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(snap, "openapi.yaml"), []byte("schema"), 0o644))
	custom := "module github.com/acme/apis/openapi/csp.infoblox.com/iam-identity/v2\n\ngo 1.23\n"
	require.NoError(t, os.WriteFile(filepath.Join(snap, "go.mod"), []byte(custom), 0o644))

	cloneDir := t.TempDir()
	require.NoError(t, stageRelease(cloneDir, m, snap))

	got, err := os.ReadFile(filepath.Join(cloneDir,
		filepath.FromSlash("openapi/csp.infoblox.com/iam-identity/v2/go.mod")))
	require.NoError(t, err)
	assert.Equal(t, custom, string(got), "snapshot-provided go.mod must be preserved")
}

// TestStageRelease_NonGoSurfaceSkipsGoMod verifies a surface without Go
// language coordinates gets no generated go.mod.
func TestStageRelease_NonGoSurfaceSkipsGoMod(t *testing.T) {
	m := familyManifest(t, "iam-identity", "v1")
	m.Languages = nil // e.g. a TypeScript-only or CRD surface

	_, files := stagedFiles(t, m, makeSnapshot(t, "openapi.yaml"))
	for _, f := range files {
		assert.NotContains(t, f, "go.mod", "no go.mod for a non-Go surface")
	}
}
