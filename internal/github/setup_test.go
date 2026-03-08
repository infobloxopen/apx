package github

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// PEM cache tests
// ---------------------------------------------------------------------------

func TestCachePEM_CopiesAndCaches(t *testing.T) {
	tmp := t.TempDir()
	srcPEM := filepath.Join(tmp, "test.pem")
	require.NoError(t, os.WriteFile(srcPEM, []byte("fake-pem-data"), 0644))

	cacheDir := filepath.Join(tmp, "cache")
	origPemCacheDir := pemCacheDirFn
	pemCacheDirFn = func() (string, error) {
		if err := os.MkdirAll(cacheDir, 0700); err != nil {
			return "", err
		}
		return cacheDir, nil
	}
	t.Cleanup(func() { pemCacheDirFn = origPemCacheDir })

	// First call should copy
	contents, err := CachePEM("testorg", srcPEM)
	require.NoError(t, err)
	assert.Equal(t, "fake-pem-data", contents)

	// Verify the cached file exists with 0600
	cachedPath := filepath.Join(cacheDir, "testorg-app.pem")
	info, err := os.Stat(cachedPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Second call should use cache (even with empty pemPath)
	contents2, err := CachePEM("testorg", "")
	require.NoError(t, err)
	assert.Equal(t, "fake-pem-data", contents2)
}

func TestCachePEM_MissingBoth(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	origPemCacheDir := pemCacheDirFn
	pemCacheDirFn = func() (string, error) {
		if err := os.MkdirAll(cacheDir, 0700); err != nil {
			return "", err
		}
		return cacheDir, nil
	}
	t.Cleanup(func() { pemCacheDirFn = origPemCacheDir })

	_, err := CachePEM("testorg", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no cached PEM found")
}

// ---------------------------------------------------------------------------
// PEMCachePath tests
// ---------------------------------------------------------------------------

func TestPEMCachePath(t *testing.T) {
	tmp := t.TempDir()
	origPemCacheDir := pemCacheDirFn
	pemCacheDirFn = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { pemCacheDirFn = origPemCacheDir })

	path, err := PEMCachePath("acme")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmp, "acme-app.pem"), path)
}

// ---------------------------------------------------------------------------
// SetupResult tests
// ---------------------------------------------------------------------------

func TestSetupResult_Add(t *testing.T) {
	r := &SetupResult{}
	r.Add("created", "thing1")
	r.Add("skipped", "thing2")
	r.Add("warning", "thing3")

	assert.Equal(t, []string{"thing1"}, r.Created)
	assert.Equal(t, []string{"thing2"}, r.Skipped)
	assert.Equal(t, []string{"thing3"}, r.Warnings)
}

// ---------------------------------------------------------------------------
// orgSecretExists tests (with stub)
// ---------------------------------------------------------------------------

func TestOrgSecretExists(t *testing.T) {
	origGHRun := GHRun
	t.Cleanup(func() { GHRun = origGHRun })

	GHRun = func(args ...string) (string, error) {
		return "APX_APP_ID\tUpdated 2025-01-01\nOTHER_SECRET\tUpdated 2025-01-01", nil
	}

	assert.True(t, orgSecretExists("myorg", "APX_APP_ID"))
	assert.False(t, orgSecretExists("myorg", "APX_APP_PRIVATE_KEY"))
}

func TestOrgSecretExists_Error(t *testing.T) {
	origGHRun := GHRun
	t.Cleanup(func() { GHRun = origGHRun })

	GHRun = func(args ...string) (string, error) {
		return "", fmt.Errorf("not authenticated")
	}

	assert.False(t, orgSecretExists("myorg", "APX_APP_ID"))
}
