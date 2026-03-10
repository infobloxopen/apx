// Package github provides idempotent helpers for configuring GitHub
// repositories via the gh CLI. Every operation follows a check-then-act
// pattern so that `apx init --setup-github` can be re-run safely.
package github

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/infobloxopen/apx/internal/ui"
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
// gh CLI helpers
// ---------------------------------------------------------------------------

// GHRun is the function used to invoke the gh CLI. It is a package-level
// variable so tests can replace it with a stub.
var GHRun = ghRunReal

func ghRunReal(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// CheckGHAuth verifies that gh is installed and authenticated.
func CheckGHAuth() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found; install from https://cli.github.com")
	}
	out, err := GHRun("auth", "status")
	if err != nil {
		return fmt.Errorf("gh is not authenticated: %s\nRun: gh auth login", out)
	}
	return nil
}

// CheckGHScopes verifies the gh token has the required OAuth scopes for
// org-level operations. Call this early so users don't hit permission
// errors midway through setup.
func CheckGHScopes() error {
	out, err := GHRun("auth", "status")
	if err != nil {
		return fmt.Errorf("gh is not authenticated: %s", out)
	}

	// gh auth status prints scopes like: Token scopes: 'admin:org', 'repo', ...
	// Check for admin:org which is needed for org secrets.
	if !strings.Contains(out, "admin:org") {
		return fmt.Errorf("gh token is missing the 'admin:org' scope needed for org secrets.\n" +
			"Run: gh auth refresh -h github.com -s admin:org")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Org secrets
// ---------------------------------------------------------------------------

// orgSecretExists checks whether an org-level secret already exists.
func orgSecretExists(org, name string) bool {
	out, err := GHRun("secret", "list", "--org", org)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == name {
			return true
		}
	}
	return false
}

// SetOrgSecret sets (or skips) an org-level Actions secret.
func SetOrgSecret(org, name, value, visibility string, res *SetupResult) error {
	if orgSecretExists(org, name) {
		res.Add("skipped", fmt.Sprintf("org secret %s", name))
		return nil
	}
	cmd := exec.Command("gh", "secret", "set", name,
		"--org", org,
		"--visibility", visibility,
		"--body", value,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(out))
		if strings.Contains(errMsg, "not have the required permissions") ||
			strings.Contains(errMsg, "403") {
			res.Add("warning", fmt.Sprintf("org secret %s (run as org admin: gh secret set %s --org %s --visibility %s)",
				name, name, org, visibility))
			return nil
		}
		return fmt.Errorf("failed to set org secret %s: %s", name, errMsg)
	}
	res.Add("created", fmt.Sprintf("org secret %s", name))
	return nil
}

// ---------------------------------------------------------------------------
// Branch protection
// ---------------------------------------------------------------------------

type branchProtection struct {
	URL string `json:"url"`
}

// EnsureBranchProtection creates or verifies branch protection on main.
func EnsureBranchProtection(owner, repo string, res *SetupResult) error {
	endpoint := fmt.Sprintf("repos/%s/%s/branches/main/protection", owner, repo)
	out, err := GHRun("api", endpoint, "--silent")
	if err == nil && out != "" {
		var bp branchProtection
		if json.Unmarshal([]byte(out), &bp) == nil && bp.URL != "" {
			res.Add("skipped", "branch protection on main")
			return nil
		}
	}

	body := `{
  "required_status_checks": {
    "strict": true,
    "contexts": ["validate"]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "required_approving_review_count": 1,
    "require_code_owner_reviews": true,
    "dismiss_stale_reviews": true
  },
  "restrictions": null
}`

	cmd := exec.Command("gh", "api", endpoint, "--method", "PUT", "--input", "-", "--silent")
	cmd.Stdin = strings.NewReader(body)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(outBytes))
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

// EnsureTagProtection creates tag protection rulesets for schema format
// tag patterns like proto/**/v*.
func EnsureTagProtection(owner, repo string, res *SetupResult) error {
	endpoint := fmt.Sprintf("repos/%s/%s/rulesets", owner, repo)
	out, err := GHRun("api", endpoint)

	if err == nil {
		var rulesets []struct {
			Name string `json:"name"`
		}
		if json.Unmarshal([]byte(out), &rulesets) == nil {
			for _, rs := range rulesets {
				if rs.Name == "apx-tag-protection" {
					res.Add("skipped", "tag protection ruleset")
					return nil
				}
			}
		}
	}

	body := `{
  "name": "apx-tag-protection",
  "target": "tag",
  "enforcement": "active",
  "conditions": {
    "ref_name": {
      "include": [
        "~ALL"
      ],
      "exclude": []
    }
  },
  "rules": [
    {"type": "creation"},
    {"type": "deletion"}
  ],
  "bypass_actors": [
    {
      "actor_type": "OrganizationAdmin",
      "actor_id": 1,
      "bypass_mode": "always"
    }
  ]
}`

	cmd := exec.Command("gh", "api", endpoint, "--method", "POST", "--input", "-")
	cmd.Stdin = strings.NewReader(body)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(outBytes))
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
func SetupCanonicalRepo(org, repo, appID, pemPath, siteURL string) (*SetupResult, error) {
	res := &SetupResult{}

	if err := CheckGHAuth(); err != nil {
		return nil, err
	}
	if err := CheckGHScopes(); err != nil {
		return nil, err
	}

	// 1. Cache PEM locally
	pemContents, err := CachePEM(org, pemPath)
	if err != nil {
		return nil, fmt.Errorf("PEM setup failed: %w", err)
	}

	// 2. Org secrets
	if err := SetOrgSecret(org, "APX_APP_ID", appID, "all", res); err != nil {
		return nil, err
	}
	if err := SetOrgSecret(org, "APX_APP_PRIVATE_KEY", pemContents, "all", res); err != nil {
		return nil, err
	}

	// 3. Branch protection
	if err := EnsureBranchProtection(org, repo, res); err != nil {
		return nil, err
	}

	// 4. Tag protection
	if err := EnsureTagProtection(org, repo, res); err != nil {
		return nil, err
	}

	// 5. GitHub Pages
	if err := EnsureGitHubPages(org, repo, res); err != nil {
		return nil, err
	}
	if err := ConfigurePagesVisibility(org, repo, res); err != nil {
		return nil, err
	}

	// 6. Custom domain (when configured)
	if siteURL != "" {
		if dnsErr := CheckDNSForPages(org, siteURL); dnsErr != nil {
			ui.Warning("DNS: %v", dnsErr)
		}
		if err := ConfigurePagesDomain(org, repo, siteURL, res); err != nil {
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
func EnsureGitHubPages(org, repo string, res *SetupResult) error {
	endpoint := fmt.Sprintf("repos/%s/%s/pages", org, repo)

	// Check if Pages is already enabled.
	_, err := GHRun("api", endpoint, "--silent")
	if err == nil {
		res.Add("skipped", "GitHub Pages")
		return nil
	}

	// Enable Pages with Actions workflow deployment.
	_, err = GHRun("api", endpoint, "-X", "POST",
		"-f", "build_type=workflow")
	if err != nil {
		// 409 means already enabled — treat as success.
		if strings.Contains(err.Error(), "409") {
			res.Add("skipped", "GitHub Pages")
			return nil
		}
		res.Add("warning", fmt.Sprintf("GitHub Pages: %v", err))
		return nil
	}

	res.Add("created", "GitHub Pages (Actions deployment)")
	return nil
}

// ConfigurePagesVisibility sets GitHub Pages visibility to private if the
// repository is private, ensuring internal sites aren't publicly accessible.
func ConfigurePagesVisibility(org, repo string, res *SetupResult) error {
	endpoint := fmt.Sprintf("repos/%s/%s", org, repo)

	out, err := GHRun("api", endpoint, "--jq", ".private")
	if err != nil {
		res.Add("warning", fmt.Sprintf("could not check repo visibility: %v", err))
		return nil
	}

	if strings.TrimSpace(out) != "true" {
		// Public repo — Pages visibility doesn't need to be restricted.
		return nil
	}

	// Private repo — set Pages to private.
	pagesEndpoint := fmt.Sprintf("repos/%s/%s/pages", org, repo)
	_, err = GHRun("api", pagesEndpoint, "-X", "PUT",
		"-F", "public=false")
	if err != nil {
		res.Add("warning", fmt.Sprintf("GitHub Pages visibility: %v", err))
		return nil
	}

	res.Add("created", "GitHub Pages visibility: private")
	return nil
}

// ConfigurePagesDomain sets a custom domain (CNAME) for GitHub Pages.
func ConfigurePagesDomain(org, repo, domain string, res *SetupResult) error {
	endpoint := fmt.Sprintf("repos/%s/%s/pages", org, repo)

	_, err := GHRun("api", endpoint, "-X", "PUT",
		"-f", fmt.Sprintf("cname=%s", domain))
	if err != nil {
		res.Add("warning", fmt.Sprintf("GitHub Pages custom domain: %v", err))
		return nil
	}

	res.Add("created", fmt.Sprintf("GitHub Pages custom domain: %s", domain))
	return nil
}

// CheckDNSForPages performs a CNAME lookup on the custom domain and checks
// that it points to the expected GitHub Pages host ({org}.github.io).
// Returns an error describing the mismatch; callers should treat this as a
// warning, not a fatal error, since DNS may be configured after setup.
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
func SetupAppRepo(org, repo string) (*SetupResult, error) {
	res := &SetupResult{}

	if err := CheckGHAuth(); err != nil {
		return nil, err
	}

	// Verify org secrets exist
	if orgSecretExists(org, "APX_APP_ID") {
		res.Add("skipped", "org secret APX_APP_ID")
	} else {
		res.Add("warning", "org secret APX_APP_ID not found — run `apx init canonical --setup-github` first")
	}
	if orgSecretExists(org, "APX_APP_PRIVATE_KEY") {
		res.Add("skipped", "org secret APX_APP_PRIVATE_KEY")
	} else {
		res.Add("warning", "org secret APX_APP_PRIVATE_KEY not found — run `apx init canonical --setup-github` first")
	}

	// Branch protection
	if err := EnsureBranchProtection(org, repo, res); err != nil {
		return nil, err
	}

	return res, nil
}

// ---------------------------------------------------------------------------
// Browser helper
// ---------------------------------------------------------------------------

// openBrowserFn opens a URL in the default browser. Variable for testing.
var openBrowserFn = openBrowserReal

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

// CreateAppViaManifest creates a GitHub App using the manifest flow.
// It starts a temporary local HTTP server, opens the browser to GitHub's
// app creation page with a pre-filled manifest, receives the callback
// code, and exchanges it for the app credentials (ID + PEM).
func CreateAppViaManifest(org, repo string) (appID, appSlug, pemContents string, err error) {
	if err := CheckGHAuth(); err != nil {
		return "", "", "", err
	}

	// Start listener to discover an available port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://localhost:%d/callback", port)

	// Build the App manifest.
	manifest := map[string]interface{}{
		"name":         fmt.Sprintf("apx-%s-%s", repo, org),
		"url":          fmt.Sprintf("https://github.com/%s/%s", org, repo),
		"redirect_url": callbackURL,
		"hook_attributes": map[string]interface{}{
			"url":    fmt.Sprintf("https://github.com/%s/%s", org, repo),
			"active": false,
		},
		"public": false,
		"default_permissions": map[string]string{
			"contents":      "write",
			"pull_requests": "write",
			"metadata":      "read",
		},
		"default_events": []string{},
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
	// We base64-encode the manifest and decode it in JS to avoid any
	// HTML escaping issues with the JSON content.
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
	ui.Info("Opening browser to create GitHub App for org %q...", org)
	ui.Info("If the browser doesn't open, visit: %s", startURL)
	_ = openBrowserFn(startURL)

	// Wait for the callback or timeout.
	var code string
	select {
	case code = <-codeCh:
		// success
	case e := <-errCh:
		return "", "", "", e
	case <-time.After(5 * time.Minute):
		return "", "", "", fmt.Errorf("timed out waiting for GitHub App creation (5 minutes)")
	}

	// Exchange the temporary code for app credentials.
	out, ghErr := GHRun("api", fmt.Sprintf("app-manifests/%s/conversions", code), "--method", "POST")
	if ghErr != nil {
		return "", "", "", fmt.Errorf("failed to exchange manifest code: %s", out)
	}

	var result struct {
		ID   int    `json:"id"`
		Slug string `json:"slug"`
		PEM  string `json:"pem"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return "", "", "", fmt.Errorf("failed to parse app creation response: %w", err)
	}
	if result.ID == 0 || result.PEM == "" {
		return "", "", "", fmt.Errorf("GitHub returned incomplete app data (id=%d, pem_len=%d)", result.ID, len(result.PEM))
	}

	// The App must be installed on the org before workflows can use it.
	// Open the installation page and wait for the user to complete it.
	installURL := fmt.Sprintf("https://github.com/apps/%s/installations/new", result.Slug)
	ui.Info("\nApp created! Now it must be installed on the %q organization.", org)
	ui.Info("Opening browser to install the App...")
	ui.Info("If the browser doesn't open, visit: %s", installURL)
	_ = openBrowserFn(installURL)

	// Poll until the installation appears (up to 3 minutes).
	ui.Info("Waiting for installation to complete...")
	installed := false
	for i := 0; i < 36; i++ { // 36 × 5s = 3 min
		time.Sleep(5 * time.Second)
		if checkAppInstalled(org, result.ID) {
			installed = true
			break
		}
	}
	if !installed {
		ui.Warning("Could not verify installation — make sure you installed the App on %q.", org)
		ui.Warning("Install URL: %s", installURL)
	} else {
		ui.Success("App installed on %q!", org)
	}

	return fmt.Sprintf("%d", result.ID), result.Slug, result.PEM, nil
}

// checkAppInstalled checks whether the GitHub App (by ID) has an
// installation on the given org. Uses user auth (gh api) to query
// the org's installations list.
func checkAppInstalled(org string, appID int) bool {
	out, err := GHRun("api", fmt.Sprintf("orgs/%s/installations", org),
		"--paginate",
		"--jq", fmt.Sprintf(".installations[] | select(.app_id == %d) | .id", appID))
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// EnsureAppInstalled verifies the App is installed on the org. If not,
// it opens the browser to the installation page and polls until done.
// This is used on subsequent runs when the App was already created but
// may not have been installed.
func EnsureAppInstalled(org string, appID int, appSlug string) error {
	if checkAppInstalled(org, appID) {
		return nil
	}

	installURL := fmt.Sprintf("https://github.com/apps/%s/installations/new", appSlug)
	ui.Info("App is not installed on %q. Opening browser to install...", org)
	ui.Info("If the browser doesn't open, visit: %s", installURL)
	_ = openBrowserFn(installURL)

	ui.Info("Waiting for installation to complete...")
	for i := 0; i < 36; i++ {
		time.Sleep(5 * time.Second)
		if checkAppInstalled(org, appID) {
			ui.Success("App installed on %q!", org)
			return nil
		}
	}
	return fmt.Errorf("app not installed on %q after 3 minutes; install manually at %s", org, installURL)
}

// ---------------------------------------------------------------------------
// App ID cache
// ---------------------------------------------------------------------------

// CacheAppID writes the App ID to ~/.config/apx/<org>-app-id.
func CacheAppID(org, appID string) error {
	dir, err := pemCacheDirFn()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, org+"-app-id"), []byte(appID), 0600)
}

// GetCachedAppID returns the cached App ID for an org, or "" if not cached.
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

// CacheAppSlug writes the App slug to ~/.config/apx/<org>-app-slug.
func CacheAppSlug(org, slug string) error {
	dir, err := pemCacheDirFn()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, org+"-app-slug"), []byte(slug), 0600)
}

// GetCachedAppSlug returns the cached App slug for an org, or "" if not cached.
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
