package publisher

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// gh CLI helper (stubable for tests, same pattern as internal/github)
// ---------------------------------------------------------------------------

// GHRun is the function used to invoke the gh CLI.  Tests can replace it
// with a stub.
var GHRun = ghRunReal

func ghRunReal(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// CheckGHCLI verifies that gh is installed and authenticated.
func CheckGHCLI() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found — install from https://cli.github.com")
	}
	out, err := GHRun("auth", "status")
	if err != nil {
		return fmt.Errorf("gh is not authenticated: %s\nRun: gh auth login", out)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Canonical repo URL helpers
// ---------------------------------------------------------------------------

// ParseCanonicalNWO extracts "owner/repo" from a canonical repo URL like
// "github.com/acme/apis" or "https://github.com/acme/apis.git".
func ParseCanonicalNWO(canonicalRepo string) (string, error) {
	s := canonicalRepo
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "github.com/")
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimSuffix(s, "/")

	parts := strings.SplitN(s, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("cannot parse owner/repo from %q", canonicalRepo)
	}
	return parts[0] + "/" + parts[1], nil
}

// ---------------------------------------------------------------------------
// Pull-request creation
// ---------------------------------------------------------------------------

// PRResponse represents the result of creating a pull request.
type PRResponse struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"url"`
	State   string `json:"state"`
}

// CreatePR opens a pull request on the canonical repo using the gh CLI.
//
//   - canonicalNWO is "owner/repo" for the canonical repo (e.g. "acme/apis").
//   - head is the branch (or fork:branch) containing the changes.
//   - base is the target branch (usually "main").
//   - title / body are the PR metadata.
func CreatePR(canonicalNWO, head, base, title, body string) (*PRResponse, error) {
	out, err := GHRun(
		"pr", "create",
		"--repo", canonicalNWO,
		"--head", head,
		"--base", base,
		"--title", title,
		"--body", body,
	)
	if err != nil {
		return nil, fmt.Errorf("gh pr create failed: %s", out)
	}

	// gh pr create prints the PR URL on success.
	// Try to parse JSON first (if --json were used), fall back to URL.
	var resp PRResponse
	if jsonErr := json.Unmarshal([]byte(out), &resp); jsonErr != nil {
		return &PRResponse{HTMLURL: out}, nil
	}
	return &resp, nil
}

// ---------------------------------------------------------------------------
// Full PR-based publish flow
// ---------------------------------------------------------------------------

// PublishModuleWithPR performs a copy-based publish of a module into the
// canonical repository, committing the content on a feature branch and
// opening a pull request via the gh CLI.
//
// Flow:
//  1. Shallow-clone the canonical repo to a temp directory.
//  2. Create a feature branch named apx/publish/<apiID>/<version>.
//  3. Copy files from localModuleDir into <clone>/<targetPath>.
//  4. Optionally generate go.mod for Go modules.
//  5. git add + commit.
//  6. git push the feature branch.
//  7. gh pr create.
func PublishModuleWithPR(
	localModuleDir string,
	canonicalNWO string,
	targetPath string,
	apiID string,
	version string,
	goModulePath string,
) (*PRResponse, error) {
	// ── 0. Preflight ─────────────────────────────────────────────────
	if err := CheckGHCLI(); err != nil {
		return nil, err
	}

	// ── 1. Shallow-clone canonical repo ──────────────────────────────
	tmpDir, err := os.MkdirTemp("", "apx-publish-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cloneURL := fmt.Sprintf("https://github.com/%s.git", canonicalNWO)
	cloneDir := filepath.Join(tmpDir, "canonical")

	if out, cloneErr := runGit("clone", "--depth=1", cloneURL, cloneDir); cloneErr != nil {
		return nil, fmt.Errorf("cloning canonical repo: %s", out)
	}

	// ── 2. Create feature branch ─────────────────────────────────────
	safeBranch := strings.ReplaceAll(apiID, "/", "-")
	branch := fmt.Sprintf("apx/publish/%s/%s", safeBranch, version)

	if out, branchErr := runGitIn(cloneDir, "checkout", "-b", branch); branchErr != nil {
		return nil, fmt.Errorf("creating branch: %s", out)
	}

	// ── 3. Copy module files ─────────────────────────────────────────
	destDir := filepath.Join(cloneDir, filepath.FromSlash(targetPath))
	if err := copyDir(localModuleDir, destDir); err != nil {
		return nil, fmt.Errorf("copying module files: %w", err)
	}

	// ── 4. Generate go.mod if requested ──────────────────────────────
	if goModulePath != "" {
		goModPath := filepath.Join(destDir, "..", "go.mod")
		// Normalise: if targetPath ends in /v1, go.mod lives one level up
		// (at the module root, e.g. proto/payments/ledger/).
		goModDir := filepath.Dir(destDir)
		goModPath = filepath.Join(goModDir, "go.mod")

		if _, statErr := os.Stat(goModPath); os.IsNotExist(statErr) {
			content, genErr := GenerateGoMod(goModulePath, "1.21")
			if genErr != nil {
				return nil, fmt.Errorf("generating go.mod: %w", genErr)
			}
			if mkErr := os.MkdirAll(goModDir, 0o755); mkErr != nil {
				return nil, fmt.Errorf("creating go.mod dir: %w", mkErr)
			}
			if writeErr := os.WriteFile(goModPath, content, 0o644); writeErr != nil {
				return nil, fmt.Errorf("writing go.mod: %w", writeErr)
			}
		}
	}

	// ── 5. git add + commit ──────────────────────────────────────────
	if out, addErr := runGitIn(cloneDir, "add", "--all"); addErr != nil {
		return nil, fmt.Errorf("git add: %s", out)
	}

	commitMsg := fmt.Sprintf("publish: %s@%s\n\nCreated by apx publish --create-pr", apiID, version)
	if out, commitErr := runGitIn(cloneDir, "commit", "-m", commitMsg); commitErr != nil {
		// No changes to commit is acceptable if module already matches
		if strings.Contains(out, "nothing to commit") {
			return nil, fmt.Errorf("no changes to publish — canonical repo already contains this content")
		}
		return nil, fmt.Errorf("git commit: %s", out)
	}

	// ── 6. git push ──────────────────────────────────────────────────
	if out, pushErr := runGitIn(cloneDir, "push", "origin", branch); pushErr != nil {
		return nil, fmt.Errorf("git push: %s", out)
	}

	// ── 7. Create PR ─────────────────────────────────────────────────
	title := fmt.Sprintf("publish: %s@%s", apiID, version)
	body := fmt.Sprintf(
		"Automated publish of API `%s` at version `%s`.\n\nCreated by `apx publish --create-pr`.",
		apiID, version,
	)

	return CreatePR(canonicalNWO, branch, "main", title, body)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// runGit runs a git command inheriting the caller's working directory.
func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// runGitIn runs a git command inside a specific directory.
func runGitIn(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// copyDir recursively copies src to dst, creating dst if it does not exist.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if mkErr := os.MkdirAll(filepath.Dir(target), 0o755); mkErr != nil {
			return mkErr
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
