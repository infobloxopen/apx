package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/infobloxopen/apx/pkg/githubauth"
)

// setupTestClient creates a test HTTP server and returns a client + cleanup func.
func setupTestClient(t *testing.T, handler http.Handler) (*githubauth.Client, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	orig := githubauth.APIBaseURL
	githubauth.APIBaseURL = server.URL
	client := githubauth.NewClient("test-token-123")
	return client, func() {
		githubauth.APIBaseURL = orig
		server.Close()
	}
}

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

	// Verify the cached file exists with 0600 (skip on Windows)
	cachedPath := filepath.Join(cacheDir, "testorg-app.pem")
	info, err := os.Stat(cachedPath)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}

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
// orgSecretExists tests (via HTTP mock)
// ---------------------------------------------------------------------------

func TestOrgSecretExists(t *testing.T) {
	client, cleanup := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/APX_APP_ID") {
			w.WriteHeader(200)
			fmt.Fprint(w, `{"name":"APX_APP_ID"}`)
		} else {
			w.WriteHeader(404)
			fmt.Fprint(w, `{"message":"Not Found"}`)
		}
	}))
	defer cleanup()

	assert.True(t, orgSecretExists(client, "myorg", "APX_APP_ID"))
	assert.False(t, orgSecretExists(client, "myorg", "APX_APP_PRIVATE_KEY"))
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

	// File has 0600 perms (skip on Windows)
	info, err := os.Stat(filepath.Join(tmp, "acme-app-id"))
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
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
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
}

// ---------------------------------------------------------------------------
// CreateAppViaManifest tests
// ---------------------------------------------------------------------------

func TestCreateAppViaManifest_ExchangesCode(t *testing.T) {
	origBrowser := openBrowserFn
	t.Cleanup(func() { openBrowserFn = origBrowser })

	// Mock the manifest code exchange endpoint.
	exchangeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/app-manifests/testcode/conversions")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        99999,
			"slug":      "apx-acme-user",
			"pem":       "-----BEGIN RSA PRIVATE KEY-----\nfake\n-----END RSA PRIVATE KEY-----",
			"client_id": "Iv1.abc123",
		})
	}))
	defer exchangeServer.Close()

	origExchangeURL := ManifestExchangeURL
	ManifestExchangeURL = exchangeServer.URL + "/app-manifests/%s/conversions"
	t.Cleanup(func() { ManifestExchangeURL = origExchangeURL })

	// Instead of actually opening browser, hit the callback directly.
	openBrowserFn = func(url string) error {
		if strings.Contains(url, "/installations/new") {
			return nil
		}
		go func() {
			time.Sleep(100 * time.Millisecond)
			port := strings.TrimPrefix(url, "http://localhost:")
			port = strings.TrimSuffix(port, "/")
			resp, err := http.Get(fmt.Sprintf("http://localhost:%s/callback?code=testcode", port))
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}

	creds, err := CreateAppViaManifest("acme", "apx-acme-user", UserAppPermissions)
	require.NoError(t, err)
	assert.Equal(t, 99999, creds.ID)
	assert.Equal(t, "apx-acme-user", creds.Slug)
	assert.Contains(t, creds.PEM, "BEGIN RSA PRIVATE KEY")
	assert.Equal(t, "Iv1.abc123", creds.ClientID)
}

// ---------------------------------------------------------------------------
// EnsureBranchProtection tests (via HTTP mock)
// ---------------------------------------------------------------------------

func TestEnsureBranchProtection_AlreadyExists(t *testing.T) {
	client, cleanup := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"url": "https://api.github.com/repos/acme/apis/branches/main/protection"})
	}))
	defer cleanup()

	res := &SetupResult{}
	err := EnsureBranchProtection(client, "acme", "apis", res)
	require.NoError(t, err)
	assert.Contains(t, res.Skipped, "branch protection on main")
}

func TestEnsureBranchProtection_Creates(t *testing.T) {
	callCount := 0
	client, cleanup := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			w.WriteHeader(404)
			return
		}
		// PUT
		w.WriteHeader(200)
		fmt.Fprint(w, `{"url":"created"}`)
	}))
	defer cleanup()

	res := &SetupResult{}
	err := EnsureBranchProtection(client, "acme", "apis", res)
	require.NoError(t, err)
	assert.Contains(t, res.Created, "branch protection on main")
}

// ---------------------------------------------------------------------------
// CheckAppInstalled tests
// ---------------------------------------------------------------------------

func TestCheckAppInstalled_Found(t *testing.T) {
	client, cleanup := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count": 1,
			"installations": []map[string]interface{}{
				{"app_id": 99999, "id": 1},
			},
		})
	}))
	defer cleanup()

	assert.True(t, CheckAppInstalled(client, "acme", 99999))
}

func TestCheckAppInstalled_NotFound(t *testing.T) {
	client, cleanup := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count":   1,
			"installations": []map[string]interface{}{{"app_id": 11111, "id": 1}},
		})
	}))
	defer cleanup()

	assert.False(t, CheckAppInstalled(client, "acme", 99999))
}

// ---------------------------------------------------------------------------
// User app cache tests
// ---------------------------------------------------------------------------

func TestUserAppCache(t *testing.T) {
	tmp := t.TempDir()
	orig := githubauth.ConfigDir
	githubauth.ConfigDir = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { githubauth.ConfigDir = orig })

	require.NoError(t, CacheUserAppClientID("acme", "Iv1.abc123"))
	assert.Equal(t, "Iv1.abc123", GetCachedUserAppClientID("acme"))

	require.NoError(t, CacheUserAppID("acme", "12345"))
	assert.Equal(t, "12345", GetCachedUserAppID("acme"))

	require.NoError(t, CacheUserAppSlug("acme", "apx-acme-user"))
	assert.Equal(t, "apx-acme-user", GetCachedUserAppSlug("acme"))
}

// ---------------------------------------------------------------------------
// App name tests
// ---------------------------------------------------------------------------

func TestAppNames(t *testing.T) {
	assert.Equal(t, "apx-acme-user", UserAppName("acme"))
	assert.Equal(t, "apx-apis-acme", CIAppName("apis", "acme"))
}
