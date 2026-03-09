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

// FindExistingPR checks whether an open PR already exists for the given
// branch on the canonical repo. Returns nil if no PR is found.
func FindExistingPR(canonicalNWO, branch string) (*PRResponse, error) {
	out, err := GHRun(
		"pr", "list",
		"--repo", canonicalNWO,
		"--head", branch,
		"--state", "open",
		"--json", "number,url,state",
	)
	if err != nil {
		return nil, fmt.Errorf("gh pr list failed: %s", out)
	}

	out = strings.TrimSpace(out)
	if out == "" || out == "[]" {
		return nil, nil
	}

	var prs []PRResponse
	if jsonErr := json.Unmarshal([]byte(out), &prs); jsonErr != nil {
		return nil, fmt.Errorf("parsing PR list response: %w", jsonErr)
	}
	if len(prs) == 0 {
		return nil, nil
	}
	return &prs[0], nil
}

// ComputeReleaseBranchName returns a deterministic branch name for a release
// submission: apx/release/<normalized-api-id>/<version>
func ComputeReleaseBranchName(apiID, version string) string {
	safe := strings.ReplaceAll(apiID, "/", "-")
	return fmt.Sprintf("apx/release/%s/%s", safe, version)
}

// ---------------------------------------------------------------------------
// Full PR-based release submit flow
// ---------------------------------------------------------------------------

// SubmitReleaseWithPR performs a PR-based release submission of a prepared
// snapshot into the canonical repository.
//
// Flow:
//  1. Shallow-clone the canonical repo to a temp directory.
//  2. Create a release branch named apx/release/<api-id-normalized>/<version>.
//  3. Copy snapshot files from snapshotDir into <clone>/<canonicalPath>.
//  4. Optionally generate go.mod for Go modules.
//  5. git add + commit.
//  6. git push --force the release branch (force for retry safety).
//  7. Check for an existing PR on the branch; if found, return it.
//  8. gh pr create with release metadata.
//
// The prBodyExtra parameter allows callers to append CI provenance or other
// metadata to the PR body.
func SubmitReleaseWithPR(
	manifest *ReleaseManifest,
	snapshotDir string,
	prBodyExtra string,
) (*PRResponse, error) {
	// ── 0. Parse canonical NWO ───────────────────────────────────────
	canonicalNWO, err := ParseCanonicalNWO(manifest.CanonicalRepo)
	if err != nil {
		return nil, fmt.Errorf("parsing canonical repo: %w", err)
	}

	// ── 1. Shallow-clone canonical repo ──────────────────────────────
	tmpDir, err := os.MkdirTemp("", "apx-release-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cloneURL := fmt.Sprintf("https://github.com/%s.git", canonicalNWO)
	cloneDir := filepath.Join(tmpDir, "canonical")

	if out, cloneErr := runGit("clone", "--depth=1", cloneURL, cloneDir); cloneErr != nil {
		return nil, fmt.Errorf("cloning canonical repo: %s", out)
	}

	// ── 2. Create release branch ─────────────────────────────────────
	branch := ComputeReleaseBranchName(manifest.APIID, manifest.RequestedVersion)

	if out, branchErr := runGitIn(cloneDir, "checkout", "-b", branch); branchErr != nil {
		return nil, fmt.Errorf("creating branch: %s", out)
	}

	// ── 3. Copy snapshot files ───────────────────────────────────────
	destDir := filepath.Join(cloneDir, filepath.FromSlash(manifest.CanonicalPath))
	if err := copyDir(snapshotDir, destDir); err != nil {
		return nil, fmt.Errorf("copying snapshot files: %w", err)
	}

	// ── 4. Generate go.mod if needed ─────────────────────────────────
	if manifest.GoModule != "" {
		// go.mod lives one level up from the line dir (e.g. proto/payments/ledger/)
		goModDir := filepath.Dir(destDir)
		goModPath := filepath.Join(goModDir, "go.mod")

		if _, statErr := os.Stat(goModPath); os.IsNotExist(statErr) {
			content, genErr := GenerateGoMod(manifest.GoModule, "1.21")
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

	commitMsg := fmt.Sprintf("release: %s@%s\n\nCreated by apx release submit",
		manifest.APIID, manifest.RequestedVersion)
	if out, commitErr := runGitIn(cloneDir, "commit", "-m", commitMsg); commitErr != nil {
		if strings.Contains(out, "nothing to commit") {
			// Content already matches — still need to push for branch creation
			// and check for existing PR. This is a no-change retry scenario.
		} else {
			return nil, fmt.Errorf("git commit: %s", out)
		}
	}

	// ── 6. git push --force (force for retry safety) ─────────────────
	if out, pushErr := runGitIn(cloneDir, "push", "--force", "origin", branch); pushErr != nil {
		return nil, fmt.Errorf("git push: %s", out)
	}

	// ── 7. Check for existing PR (idempotency) ───────────────────────
	existing, findErr := FindExistingPR(canonicalNWO, branch)
	if findErr != nil {
		// Non-fatal: if we can't check, try creating a new one
		_ = findErr
	}
	if existing != nil {
		return existing, nil
	}

	// ── 8. Create PR ─────────────────────────────────────────────────
	title := fmt.Sprintf("release: %s@%s", manifest.APIID, manifest.RequestedVersion)
	body := fmt.Sprintf(
		"Automated release submission of API `%s` at version `%s`.\n\n"+
			"- **Tag**: `%s`\n"+
			"- **Source**: `%s/%s`\n\n"+
			"Created by `apx release submit`.",
		manifest.APIID, manifest.RequestedVersion,
		manifest.Tag,
		manifest.SourceRepo, manifest.SourcePath,
	)
	if prBodyExtra != "" {
		body += "\n\n" + prBodyExtra
	}

	pr, createErr := CreatePR(canonicalNWO, branch, "main", title, body)
	if createErr != nil {
		return nil, createErr
	}

	// Populate branch on result for manifest storage
	if pr.HTMLURL != "" && pr.Number == 0 {
		// URL-only response: try to extract PR number from URL is not reliable,
		// so leave Number as 0
	}

	return pr, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// runGitFn and runGitInFn are function variables for git operations,
// allowing tests to stub them out.
var runGitFn = runGitReal
var runGitInFn = runGitInReal

// runGit runs a git command inheriting the caller's working directory.
func runGit(args ...string) (string, error) {
	return runGitFn(args...)
}

func runGitReal(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// runGitIn runs a git command inside a specific directory.
func runGitIn(dir string, args ...string) (string, error) {
	return runGitInFn(dir, args...)
}

func runGitInReal(dir string, args ...string) (string, error) {
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
