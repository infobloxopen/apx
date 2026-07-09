package publisher

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newGitRepoWithTags creates a temporary git repo with one commit and applies
// the given annotated tags (one per name).
func newGitRepoWithTags(t *testing.T, tags ...string) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, out)
	}

	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0644))
	run("add", ".")
	run("commit", "-m", "init")
	for _, tag := range tags {
		run("tag", tag, "-m", tag)
	}
	return dir
}

// initBareGitRepo creates a temporary git repo with a standard tag set. The tag
// prefix omits the major-version segment for ALL majors (Go tag convention), so
// the v1 and v2 lines share the prefix "proto/payments/ledger" and are
// distinguished only by the version's major.
func initBareGitRepo(t *testing.T) string {
	t.Helper()
	return newGitRepoWithTags(t,
		"proto/payments/ledger/v1.0.0",
		"proto/payments/ledger/v1.1.0",
		"proto/payments/ledger/v1.0.1",
		"proto/billing/invoices/v1.0.0",
		// v2 line: no /v2 subdirectory in the tag; it shares the v1 prefix.
		"proto/payments/ledger/v2.0.0-alpha.1",
	)
}

func TestListTags(t *testing.T) {
	dir := initBareGitRepo(t)
	tm := NewTagManager(dir, "")

	// List all tags
	tags, err := tm.ListTags("")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tags), 5)

	// List with a pattern: the three v1-line tags. The v2 tag begins "v2." so it
	// does not match the "…/v1*" glob.
	tags, err = tm.ListTags("proto/payments/ledger/v1*")
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
	// The v2 tag shares the prefix but must be scoped out of the v1 line.
	assert.NotContains(t, versions, "v2.0.0-alpha.1")

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

// TestListVersionsForAPI_MixedMajorsRatchet proves that when v1 and v2 GAs share
// the (major-version-stripped) tag prefix, each line's version list — and thus
// its pre-release ratchet — is scoped to its own major. A v2 pre-release must
// ratchet only against v2 GAs, never against a higher v1 GA.
func TestListVersionsForAPI_MixedMajorsRatchet(t *testing.T) {
	dir := newGitRepoWithTags(t,
		"proto/payments/ledger/v1.0.0",
		"proto/payments/ledger/v1.5.0", // v1 GA higher than the v2 GA
		"proto/payments/ledger/v2.0.0", // v2 GA
	)
	tm := NewTagManager(dir, "")

	v2versions, err := tm.ListVersionsForAPI("proto/payments/ledger/v2")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"v2.0.0"}, v2versions, "v2 line must see only v2.x tags")

	v1versions, err := tm.ListVersionsForAPI("proto/payments/ledger/v1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"v1.0.0", "v1.5.0"}, v1versions, "v1 line must see only v1.x tags")

	// v2.0.0-beta.1 is <= the v2 GA (v2.0.0), so the ratchet rejects it — proving
	// it did NOT ratchet against the higher v1 GA (v1.5.0), which it would clear.
	require.Error(t, config.AssertPrereleaseRatchet("v2.0.0-beta.1", v2versions, 2))
	// v2.0.1-beta.1 is above the v2 GA, so it is accepted.
	require.NoError(t, config.AssertPrereleaseRatchet("v2.0.1-beta.1", v2versions, 2))
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
