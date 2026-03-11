package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/infobloxopen/apx/pkg/githubauth"
)

// remoteConfig is a minimal struct for parsing only the import_root field
// from a remote apx.yaml, avoiding full schema validation.
type remoteConfig struct {
	ImportRoot string `yaml:"import_root"`
}

// ghContentsResponse is the shape of the GitHub Contents API response.
type ghContentsResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

// httpGetFn and ghAPIFn are function variables for testability,
// following the runGitFn pattern in internal/publisher/pr.go.
var httpGetFn = httpGetReal
var ghAPIFn = ghAPIReal

// FetchRemoteImportRoot attempts to resolve import_root for the given org/repo.
//
// Resolution order:
//  1. raw.githubusercontent.com/{org}/{repo}/HEAD/apx.yaml (public repos)
//  2. GitHub REST API repos/{org}/{repo}/contents/apx.yaml (private repos, via cached token)
//  3. Cached catalog at ~/.cache/apx/catalogs/{org}/{repo}/catalog.yaml
//
// Returns "" on any failure — never surfaces errors to the caller.
func FetchRemoteImportRoot(org, repo string) string {
	// 1. Try raw.githubusercontent.com (public repos, no auth)
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/HEAD/apx.yaml", org, repo)
	if data, err := httpGetFn(url); err == nil {
		var rc remoteConfig
		if err := yaml.Unmarshal(data, &rc); err == nil && rc.ImportRoot != "" {
			return rc.ImportRoot
		}
	}

	// 2. Try GitHub REST API (private repos, uses cached token if available)
	endpoint := fmt.Sprintf("repos/%s/%s/contents/apx.yaml", org, repo)
	if data, err := ghAPIFn(endpoint); err == nil {
		var resp ghContentsResponse
		if err := json.Unmarshal(data, &resp); err == nil && resp.Encoding == "base64" {
			// GitHub returns base64 with embedded newlines
			clean := strings.ReplaceAll(resp.Content, "\n", "")
			if decoded, err := base64.StdEncoding.DecodeString(clean); err == nil {
				var rc remoteConfig
				if err := yaml.Unmarshal(decoded, &rc); err == nil && rc.ImportRoot != "" {
					return rc.ImportRoot
				}
			}
		}
	}

	// 3. Try cached catalog
	if ir := importRootFromCachedCatalog(org, repo); ir != "" {
		return ir
	}

	return ""
}

// httpGetReal performs an HTTP GET with a short timeout.
func httpGetReal(url string) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url) //nolint:gosec // URL is constructed from user-provided org/repo
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// ghAPIReal uses a cached githubauth token (if available) to call the
// GitHub REST API. It does NOT trigger the device flow — if no token is
// cached, it silently fails so callers can fall through to the next
// resolution method.
func ghAPIReal(endpoint string) ([]byte, error) {
	// Try to detect the org from git remote for token lookup.
	org, err := githubauth.DetectOrg()
	if err != nil {
		return nil, fmt.Errorf("no org detected: %w", err)
	}

	tok, err := githubauth.LoadToken(org)
	if err != nil || tok == nil {
		return nil, fmt.Errorf("no cached token for org %q", org)
	}

	client := githubauth.NewClient(tok.AccessToken)
	body, status, err := client.Get("/" + endpoint)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("HTTP %d", status)
	}
	return body, nil
}

// importRootFromCachedCatalog reads import_root from a locally cached catalog.
// Uses the same cache directory as the catalog subsystem:
// ~/.cache/apx/catalogs/{org}/{repo}/catalog.yaml
func importRootFromCachedCatalog(org, repo string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	catalogPath := filepath.Join(home, ".cache", "apx", "catalogs", org, repo, "catalog.yaml")
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return ""
	}

	// Parse only the import_root field to avoid importing the catalog package
	// (which would create a circular dependency).
	var cat struct {
		ImportRoot string `yaml:"import_root"`
	}
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return ""
	}
	return cat.ImportRoot
}
