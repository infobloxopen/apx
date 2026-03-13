// Package githubauth provides GitHub OAuth device-flow authentication,
// token caching, and an authenticated REST client for the GitHub API.
//
// It is designed to be imported by both apx (CLI) and dk (data-kit)
// so any project in the ecosystem can obtain a user-scoped GitHub token
// without depending on the gh CLI.
package githubauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// GitHubBaseURL is the base URL for GitHub OAuth endpoints.
// Override in tests to point at httptest.Server.
var GitHubBaseURL = "https://github.com"

// OpenBrowserFn, if set, is called to open the device verification URL
// in the user's browser during DeviceFlowLogin. The githubauth package
// has no dependency on platform-specific browser code, so the main
// binary wires this up at startup.
var OpenBrowserFn func(url string) error

// deviceCodeResponse is the first-leg response from the device flow.
type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// DeviceFlowLogin performs the GitHub OAuth device flow:
//  1. Requests a device code from GitHub
//  2. Prints the verification URL + user code to stderr
//  3. Polls for the access token until the user authorizes or it times out
//
// The returned Token can be persisted with SaveToken.
func DeviceFlowLogin(clientID string) (*Token, error) {
	if clientID == "" {
		return nil, fmt.Errorf("client_id is required for device flow login")
	}

	// Step 1: request device code.
	dc, err := requestDeviceCode(clientID)
	if err != nil {
		return nil, fmt.Errorf("device code request failed: %w", err)
	}

	// Step 2: print instructions and open browser if possible.
	fmt.Fprintf(os.Stderr, "\n  To authenticate, open: %s\n", dc.VerificationURI)
	fmt.Fprintf(os.Stderr, "  and enter code: %s\n\n", dc.UserCode)
	if OpenBrowserFn != nil {
		_ = OpenBrowserFn(dc.VerificationURI)
	}

	// Step 3: poll for access token.
	interval := time.Duration(dc.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("device flow timed out after %d seconds", dc.ExpiresIn)
		}
		time.Sleep(interval)

		tok, retry, err := pollAccessToken(clientID, dc.DeviceCode)
		if err != nil {
			return nil, err
		}
		if retry {
			continue
		}
		tok.CreatedAt = time.Now().UTC()
		return tok, nil
	}
}

// requestDeviceCode calls POST /login/device/code.
func requestDeviceCode(clientID string) (*deviceCodeResponse, error) {
	data := url.Values{
		"client_id": {clientID},
	}

	req, err := http.NewRequest("POST", GitHubBaseURL+"/login/device/code", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	var dc deviceCodeResponse
	if err := json.Unmarshal(body, &dc); err != nil {
		return nil, fmt.Errorf("failed to parse device code response: %w", err)
	}
	if dc.DeviceCode == "" || dc.UserCode == "" {
		return nil, fmt.Errorf("GitHub returned empty device/user code")
	}
	return &dc, nil
}

// pollAccessToken calls POST /login/oauth/access_token once.
// Returns (token, false, nil) on success, (nil, true, nil) if we should retry,
// or (nil, false, err) on fatal error.
func pollAccessToken(clientID, deviceCode string) (*Token, bool, error) {
	data := url.Values{
		"client_id":   {clientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}

	req, err := http.NewRequest("POST", GitHubBaseURL+"/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, false, fmt.Errorf("failed to parse token response: %w", err)
	}

	switch result.Error {
	case "":
		// Success.
		if result.AccessToken == "" {
			return nil, false, fmt.Errorf("GitHub returned empty access token")
		}
		return &Token{
			AccessToken: result.AccessToken,
			TokenType:   result.TokenType,
			Scope:       result.Scope,
		}, false, nil
	case "authorization_pending":
		return nil, true, nil // keep polling
	case "slow_down":
		return nil, true, nil // keep polling (caller already waits interval)
	case "expired_token":
		return nil, false, fmt.Errorf("device code expired — please try again")
	case "access_denied":
		return nil, false, fmt.Errorf("user denied the authorization request")
	default:
		return nil, false, fmt.Errorf("unexpected OAuth error: %s", result.Error)
	}
}

// ErrDeviceFlowDisabled is returned when the GitHub App does not have
// device flow enabled. Callers can check for this with errors.Is().
var ErrDeviceFlowDisabled = fmt.Errorf("device flow is not enabled on the GitHub App")

// IsDeviceFlowDisabled returns true if the error is due to device flow
// not being enabled on the GitHub App.
func IsDeviceFlowDisabled(err error) bool {
	return err != nil && strings.Contains(err.Error(), "device_flow_disabled")
}

// gitRemoteRe matches GitHub org from a remote URL.
// Handles both HTTPS and SSH forms.
var gitRemoteRe = regexp.MustCompile(`github\.com[:/]([^/]+)/`)

// DetectOrg attempts to determine the GitHub org from the current git
// repository's origin remote.
func DetectOrg() (string, error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get git remote: %w", err)
	}
	remote := strings.TrimSpace(string(out))
	m := gitRemoteRe.FindStringSubmatch(remote)
	if len(m) < 2 {
		return "", fmt.Errorf("could not parse GitHub org from remote %q", remote)
	}
	return m[1], nil
}
