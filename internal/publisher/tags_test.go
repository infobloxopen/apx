package publisher

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initBareGitRepo creates a temporary git repo with some commits and tags.
func initBareGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", c, out)
	}

	// Create a file and commit
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0644))
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, out)
	}
	run("add", ".")
	run("commit", "-m", "init")

	// Create some tags
	run("tag", "proto/payments/ledger/v1/v1.0.0", "-m", "v1.0.0")
	run("tag", "proto/payments/ledger/v1/v1.1.0", "-m", "v1.1.0")
	run("tag", "proto/payments/ledger/v1/v1.0.1", "-m", "v1.0.1")
	run("tag", "proto/billing/invoices/v1/v1.0.0", "-m", "v1.0.0")
	run("tag", "proto/payments/ledger/v2/v2.0.0-alpha.1", "-m", "v2 alpha")

	return dir
}

func TestListTags(t *testing.T) {
	dir := initBareGitRepo(t)
	tm := NewTagManager(dir, "")

	// List all tags
	tags, err := tm.ListTags("")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tags), 5)

	// List with a pattern
	tags, err = tm.ListTags("proto/payments/ledger/v1/*")
	require.NoError(t, err)
	assert.Len(t, tags, 3)
}

func TestListVersionsForAPI(t *testing.T) {
	dir := initBareGitRepo(t)
	tm := NewTagManager(dir, "")

	versions, err := tm.ListVersionsForAPI("proto/payments/ledger/v1")
	require.NoError(t, err)
	assert.Len(t, versions, 3)
	assert.Contains(t, versions, "v1.0.0")
	assert.Contains(t, versions, "v1.1.0")
	assert.Contains(t, versions, "v1.0.1")

	// Different API
	versions, err = tm.ListVersionsForAPI("proto/billing/invoices/v1")
	require.NoError(t, err)
	assert.Len(t, versions, 1)
	assert.Contains(t, versions, "v1.0.0")

	// Non-existent API
	versions, err = tm.ListVersionsForAPI("proto/users/auth/v1")
	require.NoError(t, err)
	assert.Empty(t, versions)
}

func TestListVersionsForAPI_V2(t *testing.T) {
	dir := initBareGitRepo(t)
	tm := NewTagManager(dir, "")

	versions, err := tm.ListVersionsForAPI("proto/payments/ledger/v2")
	require.NoError(t, err)
	assert.Len(t, versions, 1)
	assert.Contains(t, versions, "v2.0.0-alpha.1")
}

func TestTagManager_ValidateVersion(t *testing.T) {
	tm := NewTagManager(".", "")

	assert.NoError(t, tm.ValidateVersion("v1.0.0"))
	assert.NoError(t, tm.ValidateVersion("v1.0.0-alpha.1"))
	assert.NoError(t, tm.ValidateVersion("v1.0.0-beta.1+build.123"))
	assert.Error(t, tm.ValidateVersion("1.0.0"))
	assert.Error(t, tm.ValidateVersion("latest"))
	assert.Error(t, tm.ValidateVersion(""))
}
