package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// seedProject writes a minimal apx.yaml + apx.lock into dir and chdirs there.
func seedProject(t *testing.T, dir, apxYAML, apxLock string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "apx.yaml"), []byte(apxYAML), 0o644))
	if apxLock != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "apx.lock"), []byte(apxLock), 0o644))
	}
	withWorkdir(t, dir)
}

func readLock(t *testing.T, dir string) config.LockFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "apx.lock"))
	require.NoError(t, err)
	var lf config.LockFile
	require.NoError(t, yaml.Unmarshal(data, &lf))
	return lf
}

// --- apx add override flag parsing -----------------------------------------

func TestAddOverride_Path(t *testing.T) {
	dir := t.TempDir()
	seedProject(t, dir, "dependencies: []\n", "dependencies: {}\n")

	cmd := newAddCmd()
	cmd.SetArgs([]string{"openapi/billing/invoices/v2", "--path", "../billing-api"})
	require.NoError(t, cmd.Execute())

	lf := readLock(t, dir)
	dep := lf.Dependencies["openapi/billing/invoices/v2"]
	assert.Equal(t, "../billing-api", dep.Path)
	assert.True(t, dep.IsOverride())
}

func TestAddOverride_Git(t *testing.T) {
	dir := t.TempDir()
	seedProject(t, dir, "dependencies: []\n", "dependencies: {}\n")

	cmd := newAddCmd()
	cmd.SetArgs([]string{"openapi/orders/v1", "--git", "github.com/o/r", "--ref", "br"})
	require.NoError(t, cmd.Execute())

	lf := readLock(t, dir)
	dep := lf.Dependencies["openapi/orders/v1"]
	assert.Equal(t, "github.com/o/r", dep.Git)
	assert.Equal(t, "br", dep.GitRef)
	assert.True(t, dep.IsOverride())
}

func TestAddOverride_PathAndGitMutuallyExclusive(t *testing.T) {
	dir := t.TempDir()
	seedProject(t, dir, "dependencies: []\n", "dependencies: {}\n")

	cmd := newAddCmd()
	cmd.SetArgs([]string{"openapi/x/v1", "--path", "../y", "--git", "github.com/o/r"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestAddOverride_GitRequiresRef(t *testing.T) {
	dir := t.TempDir()
	seedProject(t, dir, "dependencies: []\n", "dependencies: {}\n")

	cmd := newAddCmd()
	cmd.SetArgs([]string{"openapi/x/v1", "--git", "github.com/o/r"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--git requires --ref")
}

// TestAddOverride_NeitherSetUnchanged proves the no-override path is byte
// identical to a plain version pin (no path/git/git_ref keys written).
func TestAddOverride_NeitherSetUnchanged(t *testing.T) {
	dir := t.TempDir()
	seedProject(t, dir, "dependencies: []\n", "dependencies: {}\n")

	cmd := newAddCmd()
	cmd.SetArgs([]string{"proto/payments/ledger/v1@v1.2.3"})
	require.NoError(t, cmd.Execute())

	data, err := os.ReadFile(filepath.Join(dir, "apx.lock"))
	require.NoError(t, err)
	s := string(data)
	assert.Contains(t, s, "ref: v1.2.3")
	assert.NotContains(t, s, "path:")
	assert.NotContains(t, s, "git:")
	assert.NotContains(t, s, "git_ref:")
}

// --- CI drift gate ---------------------------------------------------------

const lockWithPathOverride = `version: 1
dependencies:
  openapi/billing/invoices/v2:
    repo: github.com/acme/apis
    ref: override
    modules:
      - openapi/billing/invoices/v2
    path: ../billing-api
`

const lockReleasedOnly = `version: 1
dependencies:
  openapi/billing/invoices/v2:
    repo: github.com/acme/apis
    ref: v1.2.3
    modules:
      - openapi/billing/invoices/v2
`

func TestAssertNoUnreleasedOverrides_BlocksOnOverride(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "apx.lock"), []byte(lockWithPathOverride), 0o644))
	withWorkdir(t, dir)

	err := assertNoUnreleasedOverrides()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "release blocked")
	assert.Contains(t, err.Error(), "openapi/billing/invoices/v2@path:../billing-api")
}

func TestAssertNoUnreleasedOverrides_PassesWithoutOverride(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "apx.lock"), []byte(lockReleasedOnly), 0o644))
	withWorkdir(t, dir)
	require.NoError(t, assertNoUnreleasedOverrides())
}

func TestAssertNoUnreleasedOverrides_NoLockFile(t *testing.T) {
	dir := t.TempDir()
	withWorkdir(t, dir)
	require.NoError(t, assertNoUnreleasedOverrides())
}

// The gate must fire at the very start of each release phase, before any
// phase-specific work (e.g. before the missing-manifest checks in submit /
// finalize would otherwise fire).
func TestReleasePhases_BlockedByOverride(t *testing.T) {
	newPhase := map[string]func() error{
		"prepare": func() error {
			cmd := newReleasePrepareCmd()
			cmd.SetArgs([]string{"openapi/billing/invoices/v2", "--version", "v1.0.0"})
			return cmd.Execute()
		},
		"submit": func() error {
			cmd := newReleaseSubmitCmd()
			cmd.SetArgs([]string{})
			return cmd.Execute()
		},
		"finalize": func() error {
			cmd := newReleaseFinalizeCmd()
			cmd.SetArgs([]string{})
			return cmd.Execute()
		},
	}

	for name, run := range newPhase {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "apx.lock"), []byte(lockWithPathOverride), 0o644))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "apx.yaml"), []byte("dependencies: []\n"), 0o644))
			withWorkdir(t, dir)

			err := run()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "release blocked", "phase %s should be blocked by the drift gate", name)
		})
	}
}

// --- client --from resolution ----------------------------------------------

func TestClientFrom_ResolvesLocalPathOverride(t *testing.T) {
	// Producer checkout with an OpenAPI spec at the api-id directory.
	producer := t.TempDir()
	producer, _ = filepath.EvalSymlinks(producer)
	apiID := "openapi/billing/invoices/v2"
	apiDir := filepath.Join(producer, filepath.FromSlash(apiID))
	require.NoError(t, os.MkdirAll(apiDir, 0o755))
	wantSpec := filepath.Join(apiDir, "invoices.openapi.yaml")
	require.NoError(t, os.WriteFile(wantSpec, []byte("openapi: 3.0.0\n"), 0o644))

	// Consumer project pins that dep to the producer path.
	consumer := t.TempDir()
	lock := config.LockFile{
		Version: 1,
		Dependencies: map[string]config.DependencyLock{
			apiID: {Repo: "github.com/acme/apis", Ref: "override", Path: producer, Modules: []string{apiID}},
		},
	}
	lockData, err := yaml.Marshal(lock)
	require.NoError(t, err)
	seedProject(t, consumer, "dependencies:\n  - "+apiID+"\n", string(lockData))

	cmd := newClientGenerateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from", apiID}))
	// resolveClientContext performs the spec resolution without running codegen.
	_, gc, err := resolveClientContext(cmd, nil)
	require.NoError(t, err)

	gotAbs, _ := filepath.EvalSymlinks(gc.SpecPath)
	assert.Equal(t, wantSpec, gotAbs)
}

func TestClientFrom_ReleasedDepOutOfScope(t *testing.T) {
	dir := t.TempDir()
	seedProject(t, dir, "dependencies:\n  - openapi/billing/invoices/v2\n", lockReleasedOnly)

	cmd := newClientGenerateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from", "openapi/billing/invoices/v2"}))
	_, _, err := resolveClientContext(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet supported")
}

func TestClientFrom_UnknownDependency(t *testing.T) {
	dir := t.TempDir()
	seedProject(t, dir, "dependencies: []\n", "dependencies: {}\n")

	cmd := newClientGenerateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from", "openapi/missing/v1"}))
	_, _, err := resolveClientContext(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such dependency")
}

// TestClientFrom_TargetFromField proves a clients: target's from: field is
// honored when no --from flag is given.
func TestClientFrom_TargetFromField(t *testing.T) {
	producer := t.TempDir()
	producer, _ = filepath.EvalSymlinks(producer)
	apiID := "openapi/billing/invoices/v2"
	apiDir := filepath.Join(producer, filepath.FromSlash(apiID))
	require.NoError(t, os.MkdirAll(apiDir, 0o755))
	wantSpec := filepath.Join(apiDir, "invoices.openapi.yaml")
	require.NoError(t, os.WriteFile(wantSpec, []byte("openapi: 3.0.0\n"), 0o644))

	consumer := t.TempDir()
	lock := config.LockFile{
		Version: 1,
		Dependencies: map[string]config.DependencyLock{
			apiID: {Repo: "github.com/acme/apis", Ref: "override", Path: producer, Modules: []string{apiID}},
		},
	}
	lockData, _ := yaml.Marshal(lock)
	apxYAML := "dependencies:\n  - " + apiID + "\nclients:\n  - name: web\n    from: " + apiID + "\n"
	seedProject(t, consumer, apxYAML, string(lockData))

	cmd := newClientGenerateCmd()
	cmd.SetArgs([]string{"web"})
	_, gc, err := resolveClientContext(cmd, []string{"web"})
	require.NoError(t, err)
	gotAbs, _ := filepath.EvalSymlinks(gc.SpecPath)
	assert.Equal(t, wantSpec, gotAbs)
}
