package publisher

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/infobloxopen/apx/pkg/githubauth"
)

// ErrNoReleaseDiff signals that the prepared snapshot is byte-identical to the
// canonical repo: after staging the snapshot there is nothing to commit, so
// there is nothing to submit. Callers should treat this as a clean "nothing to
// release" outcome rather than a failure — creating a PR from an empty diff is
// what produces GitHub's opaque HTTP 422.
var ErrNoReleaseDiff = errors.New("no diff between prepared snapshot and canonical repo")

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
	HTMLURL string `json:"html_url"`
	State   string `json:"state"`
}

// CreatePR opens a pull request on the canonical repo via the GitHub REST API.
//
//   - canonicalNWO is "owner/repo" for the canonical repo (e.g. "acme/apis").
//   - head is the branch (or fork:branch) containing the changes.
//   - base is the target branch (usually "main").
//   - title / body are the PR metadata.
func CreatePR(client *githubauth.Client, canonicalNWO, head, base, title, body string) (*PRResponse, error) {
	reqBody := map[string]string{
		"title": title,
		"head":  head,
		"base":  base,
		"body":  body,
	}

	respBody, status, err := client.Post(fmt.Sprintf("/repos/%s/pulls", canonicalNWO), reqBody)
	if err != nil {
		return nil, fmt.Errorf("PR create failed: %w", err)
	}
	if status >= 400 {
		return nil, fmt.Errorf("PR create failed: HTTP %d: %s", status, respBody)
	}

	var resp PRResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parsing PR create response: %w", err)
	}
	return &resp, nil
}

// FindExistingPR checks whether an open PR already exists for the given
// branch on the canonical repo. Returns nil if no PR is found.
func FindExistingPR(client *githubauth.Client, canonicalNWO, branch string) (*PRResponse, error) {
	// GitHub requires "owner:branch" format for the head parameter when
	// querying PRs, otherwise the filter is ignored and unrelated PRs may
	// be returned.
	owner := strings.SplitN(canonicalNWO, "/", 2)[0]
	qualifiedHead := owner + ":" + branch
	endpoint := fmt.Sprintf("/repos/%s/pulls?head=%s&state=open", canonicalNWO, qualifiedHead)
	respBody, status, err := client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("PR list failed: %w", err)
	}
	if status >= 400 {
		return nil, fmt.Errorf("PR list failed: HTTP %d: %s", status, respBody)
	}

	var prs []PRResponse
	if err := json.Unmarshal(respBody, &prs); err != nil {
		return nil, fmt.Errorf("parsing PR list response: %w", err)
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
// baseBranch is the canonical-repo branch the PR targets (ARCH-271). An empty
// value defaults to "main" so callers that do not route by branch keep the
// historical behavior.
func SubmitReleaseWithPR(
	client *githubauth.Client,
	manifest *ReleaseManifest,
	snapshotDir string,
	prBodyExtra string,
	baseBranch string,
) (*PRResponse, error) {
	if baseBranch == "" {
		baseBranch = "main"
	}
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

	// Clone the resolved base branch — not the repo's default branch (ARCH-271,
	// apx#34). A shallow "clone --depth=1 <url>" fetches only the default branch
	// (main), so a develop publish would otherwise diff the snapshot against
	// main's tree and cut the release branch from main: that produces a false
	// ErrNoReleaseDiff when main holds identical content, and a wrong merge base
	// once develop has diverged. "--branch <baseBranch>" checks out that branch
	// so the release branch is cut from — and diffed against — the correct base.
	if out, cloneErr := runGit("clone", "--branch", baseBranch, "--depth=1", cloneURL, cloneDir); cloneErr != nil {
		return nil, fmt.Errorf("cloning canonical repo (base branch %q): %s", baseBranch, out)
	}

	// ── 2. Create release branch ─────────────────────────────────────
	branch := ComputeReleaseBranchName(manifest.APIID, manifest.RequestedVersion)

	if out, branchErr := runGitIn(cloneDir, "checkout", "-b", branch); branchErr != nil {
		return nil, fmt.Errorf("creating branch: %s", out)
	}

	// ── 3–4. Stage snapshot files + generate go.mod ──────────────────
	if err := stageRelease(cloneDir, manifest, snapshotDir); err != nil {
		return nil, err
	}

	// ── 5. git add + commit ──────────────────────────────────────────
	if out, addErr := runGitIn(cloneDir, "add", "--all"); addErr != nil {
		return nil, fmt.Errorf("git add: %s", out)
	}

	// Detect the empty-PR / no-diff case before pushing an empty branch and
	// hitting GitHub's opaque HTTP 422 at PR creation. "git diff --cached
	// --quiet" exits 0 (nil error) when nothing is staged.
	if _, diffErr := runGitIn(cloneDir, "diff", "--cached", "--quiet"); diffErr == nil {
		return nil, ErrNoReleaseDiff
	}

	commitMsg := fmt.Sprintf("release: %s@%s\n\nCreated by apx release submit",
		manifest.APIID, manifest.RequestedVersion)
	if out, commitErr := runGitIn(cloneDir, "commit", "-m", commitMsg); commitErr != nil {
		if !strings.Contains(out, "nothing to commit") {
			return nil, fmt.Errorf("git commit: %s", out)
		}
	}

	// ── 6. git push --force (force for retry safety) ─────────────────
	if out, pushErr := runGitIn(cloneDir, "push", "--force", "origin", branch); pushErr != nil {
		return nil, fmt.Errorf("git push: %s", out)
	}

	// ── 7. Check for existing PR (idempotency) ───────────────────────
	existing, findErr := FindExistingPR(client, canonicalNWO, branch)
	if findErr != nil {
		_ = findErr // Non-fatal: if we can't check, try creating a new one
	}
	if existing != nil {
		return existing, nil
	}

	// ── 8. Create PR ─────────────────────────────────────────────────
	title := fmt.Sprintf("release: %s@%s", manifest.APIID, manifest.RequestedVersion)
	prBody := fmt.Sprintf(
		"Automated release submission of API `%s` at version `%s`.\n\n"+
			"- **Tag**: `%s`\n"+
			"- **Source**: `%s/%s`\n\n"+
			"Created by `apx release submit`.",
		manifest.APIID, manifest.RequestedVersion,
		manifest.Tag,
		manifest.SourceRepo, manifest.SourcePath,
	)
	if prBodyExtra != "" {
		prBody += "\n\n" + prBodyExtra
	}

	pr, createErr := CreatePR(client, canonicalNWO, branch, baseBranch, title, prBody)
	if createErr != nil {
		return nil, createErr
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

// stageRelease materializes a prepared snapshot into the canonical clone at the
// manifest's canonical path and, for Go modules, generates the module's go.mod
// in the correct semantic-import-versioning location (see goModTargetDir).
//
// It is the single place a release's files are written into the clone, so the
// "every release PR touches only its own version subtree" invariant is
// exercised — and testable — in one spot. Concurrent multi-version releases of
// one family stage into disjoint file sets and never share a family-root
// go.mod (apx#27).
func stageRelease(cloneDir string, manifest *ReleaseManifest, snapshotDir string) error {
	destDir := filepath.Join(cloneDir, filepath.FromSlash(manifest.CanonicalPath))
	if err := copyDir(snapshotDir, destDir); err != nil {
		return fmt.Errorf("copying snapshot files: %w", err)
	}

	goCoords, ok := manifest.Languages["go"]
	if !ok || goCoords.Module == "" {
		return nil
	}

	goModDir := goModTargetDir(destDir, goCoords.Module)
	goModPath := filepath.Join(goModDir, "go.mod")

	// Write only if absent so a go.mod already carried in the snapshot, or one
	// already present in the canonical repo (e.g. a re-release), is preserved.
	if _, statErr := os.Stat(goModPath); !os.IsNotExist(statErr) {
		return nil
	}

	content, genErr := GenerateGoMod(goCoords.Module, "1.21")
	if genErr != nil {
		return fmt.Errorf("generating go.mod: %w", genErr)
	}
	if mkErr := os.MkdirAll(goModDir, 0o755); mkErr != nil {
		return fmt.Errorf("creating go.mod dir: %w", mkErr)
	}
	if writeErr := os.WriteFile(goModPath, content, 0o644); writeErr != nil {
		return fmt.Errorf("writing go.mod: %w", writeErr)
	}
	return nil
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
