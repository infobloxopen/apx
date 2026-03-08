// Package github provides idempotent helpers for configuring GitHub
// repositories via the gh CLI. Every operation follows a check-then-act
// pattern so that `apx init --setup-github` can be re-run safely.
package github

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	out, err := GHRun("api", endpoint, "--silent")

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
        "refs/tags/proto/**",
        "refs/tags/openapi/**",
        "refs/tags/avro/**",
        "refs/tags/jsonschema/**",
        "refs/tags/parquet/**"
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
      "actor_type": "Integration",
      "actor_id": 0,
      "bypass_mode": "always"
    }
  ]
}`

	cmd := exec.Command("gh", "api", endpoint, "--method", "POST", "--input", "-", "--silent")
	cmd.Stdin = strings.NewReader(body)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(outBytes))
		if strings.Contains(errMsg, "403") {
			res.Add("warning", "tag protection ruleset (requires admin access)")
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
func SetupCanonicalRepo(org, repo, appID, pemPath string) (*SetupResult, error) {
	res := &SetupResult{}

	if err := CheckGHAuth(); err != nil {
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

	return res, nil
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
