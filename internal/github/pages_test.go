package github

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// EnsureGitHubPages tests
// ---------------------------------------------------------------------------

func TestEnsureGitHubPages_AlreadyEnabled(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	GHRun = func(args ...string) (string, error) {
		// GET repos/org/repo/pages succeeds → already enabled.
		if args[0] == "api" && strings.Contains(args[1], "/pages") && len(args) == 3 {
			return `{"status":"built"}`, nil
		}
		return "", nil
	}

	res := &SetupResult{}
	err := EnsureGitHubPages("myorg", "apis", res)
	assert.NoError(t, err)
	assert.Contains(t, res.Skipped, "GitHub Pages")
	assert.Empty(t, res.Created)
}

func TestEnsureGitHubPages_NewlyEnabled(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	callCount := 0
	GHRun = func(args ...string) (string, error) {
		callCount++
		if callCount == 1 {
			// GET check fails → not enabled yet.
			return "", fmt.Errorf("HTTP 404")
		}
		// POST to enable succeeds.
		return `{"status":"built"}`, nil
	}

	res := &SetupResult{}
	err := EnsureGitHubPages("myorg", "apis", res)
	assert.NoError(t, err)
	assert.Contains(t, res.Created, "GitHub Pages (Actions deployment)")
}

func TestEnsureGitHubPages_409AlreadyExists(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	callCount := 0
	GHRun = func(args ...string) (string, error) {
		callCount++
		if callCount == 1 {
			// GET check fails.
			return "", fmt.Errorf("HTTP 404")
		}
		// POST returns 409 conflict.
		return "", fmt.Errorf("HTTP 409: Conflict")
	}

	res := &SetupResult{}
	err := EnsureGitHubPages("myorg", "apis", res)
	assert.NoError(t, err)
	assert.Contains(t, res.Skipped, "GitHub Pages")
}

// ---------------------------------------------------------------------------
// ConfigurePagesVisibility tests
// ---------------------------------------------------------------------------

func TestConfigurePagesVisibility_PrivateRepo(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	var putCalled bool
	GHRun = func(args ...string) (string, error) {
		if args[0] == "api" && !strings.Contains(args[1], "/pages") {
			return "true", nil // repo is private
		}
		if args[0] == "api" && strings.Contains(args[1], "/pages") {
			putCalled = true
			return "", nil
		}
		return "", nil
	}

	res := &SetupResult{}
	err := ConfigurePagesVisibility("myorg", "apis", res)
	assert.NoError(t, err)
	assert.True(t, putCalled, "should have called PUT to set visibility")
	assert.Contains(t, res.Created, "GitHub Pages visibility: private")
}

func TestConfigurePagesVisibility_PublicRepo(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	GHRun = func(args ...string) (string, error) {
		return "false", nil // repo is public
	}

	res := &SetupResult{}
	err := ConfigurePagesVisibility("myorg", "apis", res)
	assert.NoError(t, err)
	assert.Empty(t, res.Created)
	assert.Empty(t, res.Skipped)
}

// ---------------------------------------------------------------------------
// ConfigurePagesDomain tests
// ---------------------------------------------------------------------------

func TestConfigurePagesDomain(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	var capturedArgs []string
	GHRun = func(args ...string) (string, error) {
		capturedArgs = args
		return "", nil
	}

	res := &SetupResult{}
	err := ConfigurePagesDomain("myorg", "apis", "apis.internal.infoblox.dev", res)
	assert.NoError(t, err)
	assert.Contains(t, res.Created, "GitHub Pages custom domain: apis.internal.infoblox.dev")

	// Verify the gh api call includes the domain.
	joined := strings.Join(capturedArgs, " ")
	assert.Contains(t, joined, "cname=apis.internal.infoblox.dev")
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
