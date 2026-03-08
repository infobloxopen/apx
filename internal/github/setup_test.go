package github

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// ---------------------------------------------------------------------------
// CheckGHScopes tests
// ---------------------------------------------------------------------------

func TestCheckGHScopes_HasAdminOrg(t *testing.T) {
	origGHRun := GHRun
	t.Cleanup(func() { GHRun = origGHRun })

	GHRun = func(args ...string) (string, error) {
		return "github.com\n  Token: gho_xxx\n  Token scopes: 'admin:org', 'repo', 'read:org'", nil
	}

	assert.NoError(t, CheckGHScopes())
}

func TestCheckGHScopes_MissingAdminOrg(t *testing.T) {
	origGHRun := GHRun
	t.Cleanup(func() { GHRun = origGHRun })

	GHRun = func(args ...string) (string, error) {
		return "github.com\n  Token: gho_xxx\n  Token scopes: 'repo', 'read:org'", nil
	}

	err := CheckGHScopes()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "admin:org")
	assert.Contains(t, err.Error(), "gh auth refresh")
}

// ---------------------------------------------------------------------------
// App ID cache tests
// ---------------------------------------------------------------------------

func TestCacheAndGetAppID(t *testing.T) {
	tmp := t.TempDir()
	origPemCacheDir := pemCacheDirFn
	pemCacheDirFn = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { pemCacheDirFn = origPemCacheDir })

	// No cached ID yet
	assert.Equal(t, "", GetCachedAppID("acme"))

	// Cache it
	require.NoError(t, CacheAppID("acme", "12345"))

	// Read it back
	assert.Equal(t, "12345", GetCachedAppID("acme"))

	// File has 0600 perms
	info, err := os.Stat(filepath.Join(tmp, "acme-app-id"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// ---------------------------------------------------------------------------
// CachePEMFromContents tests
// ---------------------------------------------------------------------------

func TestCachePEMFromContents(t *testing.T) {
	tmp := t.TempDir()
	origPemCacheDir := pemCacheDirFn
	pemCacheDirFn = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { pemCacheDirFn = origPemCacheDir })

	require.NoError(t, CachePEMFromContents("acme", "pem-data-here"))

	data, err := os.ReadFile(filepath.Join(tmp, "acme-app.pem"))
	require.NoError(t, err)
	assert.Equal(t, "pem-data-here", string(data))

	info, err := os.Stat(filepath.Join(tmp, "acme-app.pem"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// ---------------------------------------------------------------------------
// CreateAppViaManifest tests
// ---------------------------------------------------------------------------

func TestCreateAppViaManifest_ExchangesCode(t *testing.T) {
	origGHRun := GHRun
	origBrowser := openBrowserFn
	t.Cleanup(func() {
		GHRun = origGHRun
		openBrowserFn = origBrowser
	})

	GHRun = func(args ...string) (string, error) {
		// auth status check
		if len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
			return "Logged in", nil
		}
		// manifest code exchange
		if len(args) >= 2 && args[0] == "api" && strings.HasPrefix(args[1], "app-manifests/") {
			return `{"id": 99999, "slug": "apx-apis-acme", "pem": "-----BEGIN RSA PRIVATE KEY-----\nfake\n-----END RSA PRIVATE KEY-----"}`, nil
		}
		// installations check (for EnsureAppInstalled polling)
		if len(args) >= 2 && args[0] == "api" && strings.Contains(args[1], "/installations") {
			return `[{"app_id": 99999}]`, nil
		}
		return "", fmt.Errorf("unexpected gh call: %v", args)
	}

	// Instead of actually opening browser, hit the callback directly
	openBrowserFn = func(url string) error {
		// If it's an install URL, just return (already installed per mock above)
		if strings.Contains(url, "/installations/new") {
			return nil
		}
		// Parse the port from the URL and hit /callback?code=testcode
		go func() {
			time.Sleep(100 * time.Millisecond)
			// Extract port from http://localhost:<port>/
			port := strings.TrimPrefix(url, "http://localhost:")
			port = strings.TrimSuffix(port, "/")
			resp, err := http.Get(fmt.Sprintf("http://localhost:%s/callback?code=testcode", port))
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}

	appID, slug, pem, err := CreateAppViaManifest("acme", "apis")
	require.NoError(t, err)
	assert.Equal(t, "99999", appID)
	assert.Equal(t, "apx-apis-acme", slug)
	assert.Contains(t, pem, "BEGIN RSA PRIVATE KEY")
}
