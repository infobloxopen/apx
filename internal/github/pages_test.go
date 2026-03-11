package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/infobloxopen/apx/pkg/githubauth"
)

// ---------------------------------------------------------------------------
// EnsureGitHubPages tests
// ---------------------------------------------------------------------------

func TestEnsureGitHubPages_AlreadyEnabled(t *testing.T) {
	client, cleanup := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GET repos/org/repo/pages succeeds → already enabled.
		w.WriteHeader(200)
		fmt.Fprint(w, `{"status":"built"}`)
	}))
	defer cleanup()

	res := &SetupResult{}
	err := EnsureGitHubPages(client, "myorg", "apis", res)
	assert.NoError(t, err)
	assert.Contains(t, res.Skipped, "GitHub Pages")
	assert.Empty(t, res.Created)
}

func TestEnsureGitHubPages_NewlyEnabled(t *testing.T) {
	callCount := 0
	client, cleanup := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			w.WriteHeader(404) // not enabled yet
			return
		}
		// POST to enable succeeds.
		w.WriteHeader(201)
		fmt.Fprint(w, `{"status":"built"}`)
	}))
	defer cleanup()

	res := &SetupResult{}
	err := EnsureGitHubPages(client, "myorg", "apis", res)
	assert.NoError(t, err)
	assert.Contains(t, res.Created, "GitHub Pages (Actions deployment)")
}

func TestEnsureGitHubPages_409AlreadyExists(t *testing.T) {
	callCount := 0
	client, cleanup := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			w.WriteHeader(404) // not enabled yet
			return
		}
		// POST returns 409 conflict.
		w.WriteHeader(409)
	}))
	defer cleanup()

	res := &SetupResult{}
	err := EnsureGitHubPages(client, "myorg", "apis", res)
	assert.NoError(t, err)
	assert.Contains(t, res.Skipped, "GitHub Pages")
}

// ---------------------------------------------------------------------------
// ConfigurePagesVisibility tests
// ---------------------------------------------------------------------------

func TestConfigurePagesVisibility_PrivateRepo(t *testing.T) {
	putCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]bool{"private": true})
			return
		}
		if r.Method == "PUT" {
			putCalled = true
			w.WriteHeader(204)
			return
		}
	}))
	defer server.Close()

	orig := githubauth.APIBaseURL
	githubauth.APIBaseURL = server.URL
	defer func() { githubauth.APIBaseURL = orig }()
	client := githubauth.NewClient("test-token")

	res := &SetupResult{}
	err := ConfigurePagesVisibility(client, "myorg", "apis", res)
	assert.NoError(t, err)
	assert.True(t, putCalled, "should have called PUT to set visibility")
	assert.Contains(t, res.Created, "GitHub Pages visibility: private")
}

func TestConfigurePagesVisibility_PublicRepo(t *testing.T) {
	client, cleanup := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{"private": false})
	}))
	defer cleanup()

	res := &SetupResult{}
	err := ConfigurePagesVisibility(client, "myorg", "apis", res)
	assert.NoError(t, err)
	assert.Empty(t, res.Created)
	assert.Empty(t, res.Skipped)
}

// ---------------------------------------------------------------------------
// ConfigurePagesDomain tests
// ---------------------------------------------------------------------------

func TestConfigurePagesDomain(t *testing.T) {
	var capturedBody map[string]string
	client, cleanup := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(204)
	}))
	defer cleanup()

	res := &SetupResult{}
	err := ConfigurePagesDomain(client, "myorg", "apis", "apis.internal.infoblox.dev", res)
	assert.NoError(t, err)
	assert.Contains(t, res.Created, "GitHub Pages custom domain: apis.internal.infoblox.dev")
	assert.Equal(t, "apis.internal.infoblox.dev", capturedBody["cname"])
}

// ---------------------------------------------------------------------------
// CheckDNSForPages tests
// ---------------------------------------------------------------------------

func TestCheckDNSForPages_CorrectCNAME(t *testing.T) {
	origLookup := dnsLookupCNAME
	defer func() { dnsLookupCNAME = origLookup }()

	dnsLookupCNAME = func(host string) (string, error) {
		return "myorg.github.io.", nil
	}

	err := CheckDNSForPages("myorg", "apis.example.com")
	assert.NoError(t, err)
}

func TestCheckDNSForPages_CaseInsensitive(t *testing.T) {
	origLookup := dnsLookupCNAME
	defer func() { dnsLookupCNAME = origLookup }()

	dnsLookupCNAME = func(host string) (string, error) {
		return "MyOrg.github.io.", nil
	}

	err := CheckDNSForPages("MyOrg", "apis.example.com")
	assert.NoError(t, err)
}

func TestCheckDNSForPages_WrongCNAME(t *testing.T) {
	origLookup := dnsLookupCNAME
	defer func() { dnsLookupCNAME = origLookup }()

	dnsLookupCNAME = func(host string) (string, error) {
		return "other-org.github.io.", nil
	}

	err := CheckDNSForPages("myorg", "apis.example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CNAME points to other-org.github.io")
	assert.Contains(t, err.Error(), "expected myorg.github.io")
}

func TestCheckDNSForPages_LookupFailure(t *testing.T) {
	origLookup := dnsLookupCNAME
	defer func() { dnsLookupCNAME = origLookup }()

	dnsLookupCNAME = func(host string) (string, error) {
		return "", fmt.Errorf("no such host")
	}

	err := CheckDNSForPages("myorg", "apis.example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DNS lookup failed")
	assert.Contains(t, err.Error(), "myorg.github.io")
}
