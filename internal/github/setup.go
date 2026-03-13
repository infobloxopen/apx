// Package github provides idempotent helpers for configuring GitHub
// repositories. Every operation follows a check-then-act pattern so
// that `apx init --setup-github` can be re-run safely.
//
// All GitHub API calls go through *githubauth.Client — there is no
// dependency on the gh CLI.
package github

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/crypto/nacl/box"

	"github.com/infobloxopen/apx/internal/ui"
	"github.com/infobloxopen/apx/pkg/githubauth"
)

// SetupResult tracks what happened during setup so the caller can print
// a summary.
type SetupResult struct {
	Created  []string // things that were created
	Skipped  []string // things that already existed
	Warnings []string // things that need manual action
}

// Add appends an entry to the appropriate list.
func (r *SetupResult) Add(kind string, name string) {
	switch kind {
	case "created":
		r.Created = append(r.Created, name)
	case "skipped":
		r.Skipped = append(r.Skipped, name)
	case "warning":
		r.Warnings = append(r.Warnings, name)
	}
}

// Print outputs a human-readable summary.
func (r *SetupResult) Print() {
	for _, s := range r.Skipped {
		ui.Info("  ✓ Already configured: %s", s)
	}
	for _, c := range r.Created {
		ui.Success("  ✓ Created: %s", c)
	}
	for _, w := range r.Warnings {
		ui.Warning("  ⚠ Requires admin: %s", w)
	}
}

// ---------------------------------------------------------------------------
// Org secrets (NaCl sealed-box encryption via REST API)
// ---------------------------------------------------------------------------

// orgPublicKey holds the org's public key for secret encryption.
type orgPublicKey struct {
	KeyID string `json:"key_id"`
	Key   string `json:"key"` // base64-encoded
}

// orgSecretExists checks whether an org-level Actions secret exists.
func orgSecretExists(client *githubauth.Client, org, name string) bool {
	body, status, err := client.Get(fmt.Sprintf("/orgs/%s/actions/secrets/%s", org, name))
	if err != nil || status >= 400 {
		_ = body
		return false
	}
	return true
}

// SetOrgSecret sets (or skips) an org-level Actions secret using the
// GitHub REST API with NaCl sealed-box encryption.
func SetOrgSecret(client *githubauth.Client, org, name, value, visibility string, res *SetupResult) error {
	if orgSecretExists(client, org, name) {
		res.Add("skipped", fmt.Sprintf("org secret %s", name))
		return nil
	}

	// 1. Get the org's public key.
	keyBody, status, err := client.Get(fmt.Sprintf("/orgs/%s/actions/secrets/public-key", org))
	if err != nil {
		return fmt.Errorf("failed to get org public key: %w", err)
	}
	if status >= 400 {
		errMsg := strings.TrimSpace(string(keyBody))
		if strings.Contains(errMsg, "403") || strings.Contains(errMsg, "not have") {
			res.Add("warning", fmt.Sprintf("org secret %s (requires org admin permissions)", name))
			return nil
		}
		return fmt.Errorf("failed to get org public key: HTTP %d: %s", status, errMsg)
	}

	var pk orgPublicKey
	if err := json.Unmarshal(keyBody, &pk); err != nil {
		return fmt.Errorf("failed to parse org public key: %w", err)
	}

	// 2. Encrypt the value with NaCl sealed box.
	encrypted, err := sealSecret(pk.Key, value)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// 3. PUT the encrypted secret.
	reqBody := map[string]string{
		"encrypted_value": encrypted,
		"key_id":          pk.KeyID,
		"visibility":      visibility,
	}
	respBody, status, err := client.Put(fmt.Sprintf("/orgs/%s/actions/secrets/%s", org, name), reqBody)
	if err != nil {
		return fmt.Errorf("failed to set org secret %s: %w", name, err)
	}
	if status >= 400 {
		errMsg := strings.TrimSpace(string(respBody))
		if strings.Contains(errMsg, "403") || strings.Contains(errMsg, "not have") {
			res.Add("warning", fmt.Sprintf("org secret %s (requires org admin permissions)", name))
			return nil
		}
		return fmt.Errorf("failed to set org secret %s: HTTP %d: %s", name, status, errMsg)
	}

	res.Add("created", fmt.Sprintf("org secret %s", name))
	return nil
}

// sealSecret encrypts a plaintext value using NaCl sealed box with the
// given base64-encoded public key. Returns the base64-encoded ciphertext.
func sealSecret(publicKeyB64, plaintext string) (string, error) {
	pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return "", fmt.Errorf("invalid public key encoding: %w", err)
	}
	if len(pubKeyBytes) != 32 {
		return "", fmt.Errorf("public key must be 32 bytes, got %d", len(pubKeyBytes))
	}

	var recipientKey [32]byte
	copy(recipientKey[:], pubKeyBytes)

	encrypted, err := box.SealAnonymous(nil, []byte(plaintext), &recipientKey, rand.Reader)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// ---------------------------------------------------------------------------
// Branch protection
// ---------------------------------------------------------------------------

type branchProtection struct {
	URL string `json:"url"`
}

// EnsureBranchProtection creates or verifies branch protection on main.
func EnsureBranchProtection(client *githubauth.Client, owner, repo string, res *SetupResult) error {
	endpoint := fmt.Sprintf("/repos/%s/%s/branches/main/protection", owner, repo)
	out, status, err := client.Get(endpoint)
	if err == nil && status < 400 {
		var bp branchProtection
		if json.Unmarshal(out, &bp) == nil && bp.URL != "" {
			res.Add("skipped", "branch protection on main")
			return nil
		}
	}

	body := map[string]interface{}{
		"required_status_checks": map[string]interface{}{
			"strict":   true,
			"contexts": []string{"validate"},
		},
		"enforce_admins": false,
		"required_pull_request_reviews": map[string]interface{}{
			"required_approving_review_count": 1,
			"require_code_owner_reviews":      true,
			"dismiss_stale_reviews":           true,
		},
		"restrictions": nil,
	}

	respBody, status, err := client.Put(endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to set branch protection: %w", err)
	}
	if status >= 400 {
		errMsg := strings.TrimSpace(string(respBody))
		if strings.Contains(errMsg, "403") || strings.Contains(errMsg, "Not Found") {
			res.Add("warning", "branch protection on main (requires admin access)")
			return nil
		}
		return fmt.Errorf("failed to set branch protection: %s", errMsg)
	}
	res.Add("created", "branch protection on main")
	return nil
}

// ---------------------------------------------------------------------------
// Tag protection (rulesets)
// ---------------------------------------------------------------------------

// EnsureTagProtection creates tag protection rulesets.
func EnsureTagProtection(client *githubauth.Client, owner, repo string, res *SetupResult) error {
	endpoint := fmt.Sprintf("/repos/%s/%s/rulesets", owner, repo)
	out, status, err := client.Get(endpoint)

	if err == nil && status < 400 {
		var rulesets []struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(out, &rulesets) == nil {
			for _, rs := range rulesets {
				if rs.Name == "apx-tag-protection" {
					res.Add("skipped", "tag protection ruleset")
					return nil
				}
			}
		}
	}

	body := map[string]interface{}{
		"name":        "apx-tag-protection",
		"target":      "tag",
		"enforcement": "active",
		"conditions": map[string]interface{}{
			"ref_name": map[string]interface{}{
				"include": []string{"~ALL"},
				"exclude": []string{},
			},
		},
		"rules": []map[string]string{
			{"type": "creation"},
			{"type": "deletion"},
		},
		"bypass_actors": []map[string]interface{}{
			{
				"actor_type":  "OrganizationAdmin",
				"actor_id":    1,
				"bypass_mode": "always",
			},
		},
	}

	respBody, status, err := client.Post(endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to create tag protection: %w", err)
	}
	if status >= 400 {
		errMsg := strings.TrimSpace(string(respBody))
		if strings.Contains(errMsg, "Name must be unique") {
			res.Add("skipped", "tag protection ruleset")
			return nil
		}
		if strings.Contains(errMsg, "403") || strings.Contains(errMsg, "422") {
			res.Add("warning", fmt.Sprintf("tag protection ruleset: %s", errMsg))
			return nil
		}
		return fmt.Errorf("failed to create tag protection: %s", errMsg)
	}
	res.Add("created", "tag protection ruleset")
	return nil
}

// ---------------------------------------------------------------------------
// High-level setup orchestrators
// ---------------------------------------------------------------------------

// SetupCanonicalRepo runs the full idempotent setup sequence for a
// canonical API repository.
func SetupCanonicalRepo(client *githubauth.Client, org, repo, appID, pemPath, siteURL string) (*SetupResult, error) {
	res := &SetupResult{}

	// 1. Cache PEM locally
	pemContents, err := CachePEM(org, pemPath)
	if err != nil {
		return nil, fmt.Errorf("PEM setup failed: %w", err)
	}

	// 2. Org secrets
	if err := SetOrgSecret(client, org, "APX_APP_ID", appID, "all", res); err != nil {
		return nil, err
	}
	if err := SetOrgSecret(client, org, "APX_APP_PRIVATE_KEY", pemContents, "all", res); err != nil {
		return nil, err
	}

	// 3. Branch protection
	if err := EnsureBranchProtection(client, org, repo, res); err != nil {
		return nil, err
	}

	// 4. Tag protection
	if err := EnsureTagProtection(client, org, repo, res); err != nil {
		return nil, err
	}

	// 5. GitHub Pages
	if err := EnsureGitHubPages(client, org, repo, res); err != nil {
		return nil, err
	}
	if err := ConfigurePagesVisibility(client, org, repo, res); err != nil {
		return nil, err
	}

	// 6. Custom domain (when configured)
	if siteURL != "" {
		if dnsErr := CheckDNSForPages(org, siteURL); dnsErr != nil {
			ui.Warning("DNS: %v", dnsErr)
		}
		if err := ConfigurePagesDomain(client, org, repo, siteURL, res); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// ---------------------------------------------------------------------------
// GitHub Pages
// ---------------------------------------------------------------------------

// dnsLookupCNAME is the function used for DNS CNAME lookups. Package-level
// variable so tests can replace it with a stub.
var dnsLookupCNAME = net.LookupCNAME

// EnsureGitHubPages enables GitHub Pages with Actions-based deployment.
// If Pages is already enabled (409 response), it is treated as success.
func EnsureGitHubPages(client *githubauth.Client, org, repo string, res *SetupResult) error {
	endpoint := fmt.Sprintf("/repos/%s/%s/pages", org, repo)

	// Check if Pages is already enabled.
	_, status, err := client.Get(endpoint)
	if err == nil && status < 400 {
		res.Add("skipped", "GitHub Pages")
		return nil
	}

	// Enable Pages with Actions workflow deployment.
	_, status, err = client.Post(endpoint, map[string]string{
		"build_type": "workflow",
	})
	if err != nil {
		res.Add("warning", fmt.Sprintf("GitHub Pages: %v", err))
		return nil
	}
	if status == 409 {
		res.Add("skipped", "GitHub Pages")
		return nil
	}
	if status >= 400 {
		res.Add("warning", fmt.Sprintf("GitHub Pages: HTTP %d", status))
		return nil
	}

	res.Add("created", "GitHub Pages (Actions deployment)")
	return nil
}

// ConfigurePagesVisibility sets GitHub Pages visibility to private if the
// repository is private.
func ConfigurePagesVisibility(client *githubauth.Client, org, repo string, res *SetupResult) error {
	endpoint := fmt.Sprintf("/repos/%s/%s", org, repo)

	out, status, err := client.Get(endpoint)
	if err != nil || status >= 400 {
		res.Add("warning", fmt.Sprintf("could not check repo visibility: %v", err))
		return nil
	}

	var repoInfo struct {
		Private bool `json:"private"`
	}
	if err := json.Unmarshal(out, &repoInfo); err != nil || !repoInfo.Private {
		return nil
	}

	// Private repo — set Pages to private.
	pagesEndpoint := fmt.Sprintf("/repos/%s/%s/pages", org, repo)
	_, status, err = client.Put(pagesEndpoint, map[string]bool{"public": false})
	if err != nil || status >= 400 {
		res.Add("warning", fmt.Sprintf("GitHub Pages visibility: %v", err))
		return nil
	}

	res.Add("created", "GitHub Pages visibility: private")
	return nil
}

// ConfigurePagesDomain sets a custom domain (CNAME) for GitHub Pages.
func ConfigurePagesDomain(client *githubauth.Client, org, repo, domain string, res *SetupResult) error {
	endpoint := fmt.Sprintf("/repos/%s/%s/pages", org, repo)

	_, status, err := client.Put(endpoint, map[string]string{"cname": domain})
	if err != nil || status >= 400 {
		res.Add("warning", fmt.Sprintf("GitHub Pages custom domain: %v", err))
		return nil
	}

	res.Add("created", fmt.Sprintf("GitHub Pages custom domain: %s", domain))
	return nil
}

// CheckDNSForPages performs a CNAME lookup on the custom domain and checks
// that it points to the expected GitHub Pages host ({org}.github.io).
func CheckDNSForPages(org, domain string) error {
	expected := strings.ToLower(org) + ".github.io."

	cname, err := dnsLookupCNAME(domain)
	if err != nil {
		return fmt.Errorf("%s: DNS lookup failed — ensure a CNAME record points to %s",
			domain, strings.TrimSuffix(expected, "."))
	}

	if !strings.EqualFold(cname, expected) {
		return fmt.Errorf("%s: CNAME points to %s, expected %s",
			domain, strings.TrimSuffix(cname, "."), strings.TrimSuffix(expected, "."))
	}

	return nil
}

// SetupAppRepo runs the idempotent setup for an app repository.
// It verifies org secrets exist (does not create them) and sets branch
// protection appropriate for app repos.
func SetupAppRepo(client *githubauth.Client, org, repo string) (*SetupResult, error) {
	res := &SetupResult{}

	// Verify org secrets exist
	if orgSecretExists(client, org, "APX_APP_ID") {
		res.Add("skipped", "org secret APX_APP_ID")
	} else {
		res.Add("warning", "org secret APX_APP_ID not found — run `apx init canonical --setup-github` first")
	}
	if orgSecretExists(client, org, "APX_APP_PRIVATE_KEY") {
		res.Add("skipped", "org secret APX_APP_PRIVATE_KEY")
	} else {
		res.Add("warning", "org secret APX_APP_PRIVATE_KEY not found — run `apx init canonical --setup-github` first")
	}

	// Branch protection
	if err := EnsureBranchProtection(client, org, repo, res); err != nil {
		return nil, err
	}

	return res, nil
}

// ---------------------------------------------------------------------------
// Browser helper
// ---------------------------------------------------------------------------

// openBrowserFn opens a URL in the default browser. Variable for testing.
var openBrowserFn = openBrowserReal

// OpenBrowser opens a URL in the default browser.
func OpenBrowser(url string) error {
	return openBrowserFn(url)
}

func openBrowserReal(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	default:
		return fmt.Errorf("unsupported platform %s — open the URL manually", runtime.GOOS)
	}
}

// ---------------------------------------------------------------------------
// GitHub App manifest flow
// ---------------------------------------------------------------------------

// ManifestExchangeURL is the URL template for exchanging a manifest code.
// Override in tests to point at httptest.Server.
var ManifestExchangeURL = "https://api.github.com/app-manifests/%s/conversions"

// AppCredentials holds the result of a GitHub App creation via manifest.
type AppCredentials struct {
	ID       int    `json:"id"`
	Slug     string `json:"slug"`
	PEM      string `json:"pem"`
	ClientID string `json:"client_id"`
}

// CreateAppViaManifest creates a GitHub App using the manifest flow.
// It starts a temporary local HTTP server, opens the browser to GitHub's
// app creation page with a pre-filled manifest, receives the callback
// code, and exchanges it for the app credentials.
//
// No existing authentication is needed — the manifest code exchange is
// an unauthenticated endpoint.
func CreateAppViaManifest(org, appName string, permissions map[string]string) (*AppCredentials, error) {
	// Start listener to discover an available port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://localhost:%d/callback", port)

	// Build the App manifest.
	manifest := map[string]interface{}{
		"name":         appName,
		"url":          fmt.Sprintf("https://github.com/organizations/%s", org),
		"redirect_url": callbackURL,
		"hook_attributes": map[string]interface{}{
			"url":    fmt.Sprintf("https://github.com/organizations/%s", org),
			"active": false,
		},
		"public":              false,
		"default_permissions": permissions,
		"default_events":      []string{},
		// Not in the official manifest schema but GitHub may honor it.
		// If ignored, the init flow catches device_flow_disabled and
		// guides the user to enable it manually in the App settings UI.
		"device_flow_enabled": true,
	}
	manifestJSON, _ := json.Marshal(manifest)

	// Channels for the callback handler.
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()

	// /callback — receives the temporary code from GitHub.
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			errCh <- fmt.Errorf("GitHub redirect missing code parameter")
			return
		}
		codeCh <- code //nolint:errcheck
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body>
<h2>&#10003; GitHub App created!</h2>
<p>You can close this tab and return to the terminal.</p>
</body></html>`)
	})

	// / — landing page that auto-submits the manifest form to GitHub.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		b64 := base64.StdEncoding.EncodeToString(manifestJSON)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html><html><body>
<p>Redirecting to GitHub to create the App&#8230;</p>
<form id="mf" method="post" action="https://github.com/organizations/%s/settings/apps/new">
<input type="hidden" id="manifest" name="manifest" value="">
</form>
<script>
document.getElementById('manifest').value = atob('%s');
document.getElementById('mf').submit();
</script>
</body></html>`, org, b64)
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener) //nolint:errcheck
	defer server.Close()

	startURL := fmt.Sprintf("http://localhost:%d/", port)
	ui.Info("Opening browser to create GitHub App %q for org %q...", appName, org)
	ui.Info("If the browser doesn't open, visit: %s", startURL)
	_ = openBrowserFn(startURL)

	// Wait for the callback or timeout.
	var code string
	select {
	case code = <-codeCh:
		// success
	case e := <-errCh:
		return nil, e
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("timed out waiting for GitHub App creation (5 minutes)")
	}

	// Exchange the temporary code for app credentials via direct HTTP POST.
	// This endpoint is unauthenticated — the code is the credential.
	exchangeURL := fmt.Sprintf(ManifestExchangeURL, code)
	resp, err := http.Post(exchangeURL, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange manifest code: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("manifest code exchange failed: HTTP %d: %s", resp.StatusCode, respBody)
	}

	var creds AppCredentials
	if err := json.Unmarshal(respBody, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse app creation response: %w", err)
	}
	if creds.ID == 0 || creds.PEM == "" {
		return nil, fmt.Errorf("GitHub returned incomplete app data (id=%d, pem_len=%d)", creds.ID, len(creds.PEM))
	}

	return &creds, nil
}

// InstallApp opens the browser to install a GitHub App on an org and
// polls until the installation is confirmed.
func InstallApp(client *githubauth.Client, org string, appID int, appSlug string) error {
	installURL := fmt.Sprintf("https://github.com/apps/%s/installations/new", appSlug)
	ui.Info("\nApp created! Now it must be installed on the %q organization.", org)
	ui.Info("Opening browser to install the App...")
	ui.Info("If the browser doesn't open, visit: %s", installURL)
	_ = openBrowserFn(installURL)

	// Poll until the installation appears (up to 3 minutes).
	ui.Info("Waiting for installation to complete...")
	for i := 0; i < 36; i++ { // 36 × 5s = 3 min
		time.Sleep(5 * time.Second)
		if CheckAppInstalled(client, org, appID) {
			ui.Success("App installed on %q!", org)
			return nil
		}
	}

	ui.Warning("Could not verify installation — make sure you installed the App on %q.", org)
	ui.Warning("Install URL: %s", installURL)
	return nil
}

// CheckAppInstalled checks whether the GitHub App (by ID) has an
// installation on the given org.
func CheckAppInstalled(client *githubauth.Client, org string, appID int) bool {
	items, err := client.GetPaginated(fmt.Sprintf("/orgs/%s/installations", org))
	if err != nil {
		return false
	}
	for _, raw := range items {
		var inst struct {
			AppID int `json:"app_id"`
		}
		if json.Unmarshal(raw, &inst) == nil && inst.AppID == appID {
			return true
		}
	}
	return false
}

// EnsureAppInstalled verifies the App is installed on the org. If not,
// it opens the browser to the installation page and polls until done.
func EnsureAppInstalled(client *githubauth.Client, org string, appID int, appSlug string) error {
	if CheckAppInstalled(client, org, appID) {
		return nil
	}

	installURL := fmt.Sprintf("https://github.com/apps/%s/installations/new", appSlug)
	ui.Info("App is not installed on %q. Opening browser to install...", org)
	ui.Info("If the browser doesn't open, visit: %s", installURL)
	_ = openBrowserFn(installURL)

	ui.Info("Waiting for installation to complete...")
	for i := 0; i < 36; i++ {
		time.Sleep(5 * time.Second)
		if CheckAppInstalled(client, org, appID) {
			ui.Success("App installed on %q!", org)
			return nil
		}
	}
	return fmt.Errorf("app not installed on %q after 3 minutes; install manually at %s", org, installURL)
}

// ---------------------------------------------------------------------------
// App ID cache (CI app — existing pattern)
// ---------------------------------------------------------------------------

// CacheAppID writes the CI App ID to ~/.config/apx/<org>-app-id.
func CacheAppID(org, appID string) error {
	dir, err := pemCacheDirFn()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, org+"-app-id"), []byte(appID), 0600)
}

// GetCachedAppID returns the cached CI App ID for an org, or "" if not cached.
func GetCachedAppID(org string) string {
	dir, err := pemCacheDirFn()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, org+"-app-id"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// CacheAppSlug writes the CI App slug to ~/.config/apx/<org>-app-slug.
func CacheAppSlug(org, slug string) error {
	dir, err := pemCacheDirFn()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, org+"-app-slug"), []byte(slug), 0600)
}

// GetCachedAppSlug returns the cached CI App slug for an org, or "" if not cached.
func GetCachedAppSlug(org string) string {
	dir, err := pemCacheDirFn()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, org+"-app-slug"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// ---------------------------------------------------------------------------
// User app cache
// ---------------------------------------------------------------------------

// CacheUserAppClientID writes the user app's client_id.
func CacheUserAppClientID(org, clientID string) error {
	return githubauth.WriteCache(org, "user-app-client-id", clientID)
}

// GetCachedUserAppClientID returns the cached user app client_id, or "".
func GetCachedUserAppClientID(org string) string {
	v, _ := githubauth.ReadCache(org, "user-app-client-id")
	return v
}

// CacheUserAppID writes the user app's numeric ID.
func CacheUserAppID(org, appID string) error {
	return githubauth.WriteCache(org, "user-app-id", appID)
}

// GetCachedUserAppID returns the cached user app ID, or "".
func GetCachedUserAppID(org string) string {
	v, _ := githubauth.ReadCache(org, "user-app-id")
	return v
}

// CacheUserAppSlug writes the user app's slug.
func CacheUserAppSlug(org, slug string) error {
	return githubauth.WriteCache(org, "user-app-slug", slug)
}

// GetCachedUserAppSlug returns the cached user app slug, or "".
func GetCachedUserAppSlug(org string) string {
	v, _ := githubauth.ReadCache(org, "user-app-slug")
	return v
}

// ---------------------------------------------------------------------------
// User app manifest permissions
// ---------------------------------------------------------------------------

// UserAppPermissions are the least-privilege permissions for the
// user-facing GitHub App used for daily operations: device-flow auth,
// catalog discovery, releases, and pull requests.
var UserAppPermissions = map[string]string{
	"contents":      "write",
	"pull_requests": "write",
	"metadata":      "read",
	"packages":      "read",
}

// AdminAppPermissions are the elevated permissions needed only during
// `apx init canonical --setup-github` for one-time repo configuration
// (branch/tag protection, org secrets, GitHub Pages). This is a
// separate app so that daily-use tokens don't carry admin scopes.
var AdminAppPermissions = map[string]string{
	"contents":                    "write",
	"pull_requests":               "write",
	"metadata":                    "read",
	"packages":                    "read",
	"administration":              "write",
	"pages":                       "write",
	"organization_administration": "read",
	"organization_secrets":        "write",
}

// CIAppPermissions returns the permissions for the CI GitHub App.
var CIAppPermissions = map[string]string{
	"contents":      "write",
	"pull_requests": "write",
	"metadata":      "read",
}

// UserAppName returns the well-known name for the user app: apx-{org}-user.
func UserAppName(org string) string {
	return fmt.Sprintf("apx-%s-user", org)
}

// AdminAppName returns the name for the admin app: apx-{org}-admin.
func AdminAppName(org string) string {
	return fmt.Sprintf("apx-%s-admin", org)
}

// CacheAdminAppClientID writes the admin app's client_id.
func CacheAdminAppClientID(org, clientID string) error {
	return githubauth.WriteCache(org, "admin-app-client-id", clientID)
}

// GetCachedAdminAppClientID returns the cached admin app client_id, or "".
func GetCachedAdminAppClientID(org string) string {
	v, _ := githubauth.ReadCache(org, "admin-app-client-id")
	return v
}

// CacheAdminAppID writes the admin app's numeric ID.
func CacheAdminAppID(org, appID string) error {
	return githubauth.WriteCache(org, "admin-app-id", appID)
}

// GetCachedAdminAppID returns the cached admin app ID, or "".
func GetCachedAdminAppID(org string) string {
	v, _ := githubauth.ReadCache(org, "admin-app-id")
	return v
}

// CacheAdminAppSlug writes the admin app's slug.
func CacheAdminAppSlug(org, slug string) error {
	return githubauth.WriteCache(org, "admin-app-slug", slug)
}

// GetCachedAdminAppSlug returns the cached admin app slug, or "".
func GetCachedAdminAppSlug(org string) string {
	v, _ := githubauth.ReadCache(org, "admin-app-slug")
	return v
}

// CIAppName returns the name for the CI app: apx-{repo}-{org}.
func CIAppName(repo, org string) string {
	return fmt.Sprintf("apx-%s-%s", repo, org)
}

// ---------------------------------------------------------------------------
// PEM cache
// ---------------------------------------------------------------------------

// pemCacheDirFn returns ~/.config/apx, creating it if needed.
// It is a variable so tests can override the cache directory.
var pemCacheDirFn = pemCacheDirReal

func pemCacheDirReal() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".config", "apx")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}
	return dir, nil
}

// PEMCachePath returns the expected path for the cached PEM for an org.
func PEMCachePath(org string) (string, error) {
	dir, err := pemCacheDirFn()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("%s-app.pem", org)), nil
}

// CachePEM ensures the PEM is cached at ~/.config/apx/<org>-app.pem (0600).
// If the file already exists it is read and returned. If pemPath is
// provided and the cache is missing, the file is copied into the cache.
func CachePEM(org, pemPath string) (string, error) {
	cachePath, err := PEMCachePath(org)
	if err != nil {
		return "", err
	}

	// Already cached?
	if data, err := os.ReadFile(cachePath); err == nil {
		ui.Info("Using cached PEM: %s", cachePath)
		return string(data), nil
	}

	// Need a source
	if pemPath == "" {
		return "", fmt.Errorf("no cached PEM found at %s; provide --app-pem-file=<path>", cachePath)
	}

	data, err := os.ReadFile(pemPath)
	if err != nil {
		return "", fmt.Errorf("cannot read PEM file %s: %w", pemPath, err)
	}

	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		return "", fmt.Errorf("cannot write PEM cache %s: %w", cachePath, err)
	}
	ui.Success("Cached PEM at %s (mode 0600)", cachePath)

	return string(data), nil
}

// CachePEMFromContents writes PEM contents directly to the cache.
// Used after the manifest flow returns the PEM as a string.
func CachePEMFromContents(org, contents string) error {
	cachePath, err := PEMCachePath(org)
	if err != nil {
		return err
	}
	if err := os.WriteFile(cachePath, []byte(contents), 0600); err != nil {
		return fmt.Errorf("cannot write PEM cache %s: %w", cachePath, err)
	}
	ui.Success("Cached PEM at %s (mode 0600)", cachePath)
	return nil
}
