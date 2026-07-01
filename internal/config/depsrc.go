package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/infobloxopen/apx/pkg/githubauth"
)

// depSrcCacheEnv lets tests (and offline callers) redirect the git clone cache
// away from the user's home directory.
const depSrcCacheEnv = "APX_DEPSRC_CACHE"

// MaterializeSpec resolves the OpenAPI spec for an unreleased dependency
// override to a concrete file path on disk.
//
// It handles two override kinds:
//
//   - Path override (dep.Path != ""): the spec is read from a local checkout
//     rooted at dep.Path. The api-id is resolved to a spec file beneath that
//     directory using the repo-layout conventions in ResolveAPIPath.
//   - Git override (dep.Git != ""): the repo is cloned (shallow when GitRef is
//     a branch/tag; full + checkout when it is a commit SHA) into a persistent
//     cache under ~/.cache/apx/depsrc/, then the api-id is resolved within the
//     clone.
//
// The returned cleanup is always non-nil and safe to call; it is a no-op for
// both kinds (the path override touches nothing; the git cache is persistent
// and reused across runs). It is returned so callers can defer it uniformly and
// so a future temp-dir strategy can be swapped in without changing callers.
//
// Only OpenAPI specs are in scope for this phase: if the api-id does not
// resolve to an OpenAPI spec file, a clear error is returned.
func MaterializeSpec(dep DependencyLock, apiID string) (specPath string, cleanup func() error, err error) {
	noop := func() error { return nil }

	switch {
	case dep.Path != "":
		root, statErr := filepath.Abs(dep.Path)
		if statErr != nil {
			return "", noop, fmt.Errorf("resolving override path %q: %w", dep.Path, statErr)
		}
		if fi, statErr := os.Stat(root); statErr != nil || !fi.IsDir() {
			return "", noop, fmt.Errorf("override path %q is not an existing directory", dep.Path)
		}
		spec, resolveErr := resolveSpecInRoot(root, apiID)
		if resolveErr != nil {
			return "", noop, fmt.Errorf("resolving spec for %q under path %q: %w", apiID, dep.Path, resolveErr)
		}
		return spec, noop, nil

	case dep.Git != "":
		cloneDir, cloneErr := materializeGit(dep, apiID)
		if cloneErr != nil {
			return "", noop, cloneErr
		}
		spec, resolveErr := resolveSpecInRoot(cloneDir, apiID)
		if resolveErr != nil {
			return "", noop, fmt.Errorf("resolving spec for %q in git checkout %s@%s: %w",
				apiID, dep.Git, dep.GitRef, resolveErr)
		}
		return spec, noop, nil

	default:
		return "", noop, fmt.Errorf("dependency has no override (path or git) to materialize")
	}
}

// resolveSpecInRoot finds the OpenAPI spec file for apiID within the override
// root. It tries, in order:
//
//  1. ResolveAPIPath(apiID) evaluated with root as the working directory and as
//     a module_root — this locates the api-id directory (or a file, if the
//     api-id happens to already be a spec-file path). If that yields a file, it
//     is used directly; if it yields a directory, the directory is scanned for
//     an OpenAPI spec (*.openapi.yaml, then *.openapi.yml/*.yaml/*.yml).
//  2. The repo-root producer convention <root>/openapi/*.openapi.yaml, which is
//     where `apx gen` and `resolveClientSpec` expect the emitted spec to live.
//
// It errors clearly if nothing OpenAPI-shaped is found.
func resolveSpecInRoot(root, apiID string) (string, error) {
	// Resolve the api-id relative to the override root without disturbing the
	// caller's cwd: run ResolveAPIPath with root injected as a module_root and
	// chdir'd so its cwd-relative fallbacks (internal/apis, schemas, api) are
	// evaluated under root.
	if resolved, err := resolveUnderRoot(root, apiID); err == nil {
		fi, statErr := os.Stat(resolved)
		if statErr == nil {
			if !fi.IsDir() {
				if isOpenAPISpecFile(resolved) {
					return resolved, nil
				}
				return "", fmt.Errorf("resolved %q to %q, which is not an OpenAPI spec", apiID, resolved)
			}
			if spec := scanDirForSpec(resolved); spec != "" {
				return spec, nil
			}
		}
	}

	// Fall back to the producer's repo-root convention: openapi/*.openapi.yaml.
	if spec := firstSpecMatch(filepath.Join(root, "openapi", "*.openapi.yaml")); spec != "" {
		return spec, nil
	}
	if spec := firstSpecMatch(filepath.Join(root, "openapi", "*.openapi.yml")); spec != "" {
		return spec, nil
	}

	return "", fmt.Errorf("no OpenAPI spec found for %q under %s (looked for the api-id directory and %s)",
		apiID, root, filepath.Join("openapi", "*.openapi.yaml"))
}

// resolveUnderRoot evaluates ResolveAPIPath as though root were the working
// directory, so an api-id like "openapi/billing/invoices/v2" resolves to a
// path beneath root regardless of the caller's cwd.
func resolveUnderRoot(root, apiID string) (string, error) {
	cfg := &Config{ModuleRoots: []string{root}}
	// ResolveAPIPath's cwd-relative fallbacks require chdir; do it briefly and
	// restore. This mirrors how the resolve tests exercise the fallbacks.
	orig, err := os.Getwd()
	if err != nil {
		orig = ""
	}
	if chErr := os.Chdir(root); chErr != nil {
		return "", chErr
	}
	defer func() {
		if orig != "" {
			_ = os.Chdir(orig)
		}
	}()
	return ResolveAPIPath(apiID, cfg)
}

// scanDirForSpec returns the first OpenAPI spec file found directly in dir,
// preferring the *.openapi.yaml naming convention.
func scanDirForSpec(dir string) string {
	for _, pat := range []string{"*.openapi.yaml", "*.openapi.yml", "*.yaml", "*.yml"} {
		if m := firstSpecMatch(filepath.Join(dir, pat)); m != "" {
			return m
		}
	}
	return ""
}

// firstSpecMatch returns the first glob match, or "" when there is none.
func firstSpecMatch(pattern string) string {
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// isOpenAPISpecFile is a cheap name-based check that a file is plausibly an
// OpenAPI/YAML spec. Content validation is left to the generator.
func isOpenAPISpecFile(p string) bool {
	l := strings.ToLower(p)
	return strings.HasSuffix(l, ".yaml") || strings.HasSuffix(l, ".yml") ||
		strings.HasSuffix(l, ".json")
}

// materializeGit clones dep.Git@dep.GitRef into a persistent cache directory and
// returns the checkout path. Branches/tags are fetched with a shallow
// --branch clone; if that fails (e.g. GitRef is a commit SHA, which --branch
// rejects) it falls back to a full clone + checkout. A best-effort token from
// pkg/githubauth is injected into the clone URL for private repos; public
// repos clone without auth.
func materializeGit(dep DependencyLock, apiID string) (string, error) {
	if dep.GitRef == "" {
		return "", fmt.Errorf("git override for %q has no git_ref; set --ref", apiID)
	}

	base, err := depSrcCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, sanitizeForPath(dep.Git), sanitizeForPath(dep.GitRef))

	url := normalizeGitURL(dep.Git)
	// Auth is supplied transiently via `git -c http.extraHeader=...`, which is
	// NOT persisted to the checkout's .git/config — so the token never lands on
	// disk in the (persistent) cache. The remote URL stored by clone is clean.
	auth := gitAuthArgs(url)

	// Cache reuse: a valid existing checkout is reused. For a branch the tip may
	// have moved, so refresh it; commits and tags are immutable and kept as-is.
	if isGitCheckout(dir) {
		if refreshErr := refreshCheckout(dir, dep.GitRef, auth); refreshErr != nil {
			// A refresh failure (e.g. offline) is non-fatal: fall through to use
			// the cached checkout as-is.
			_ = refreshErr
		}
		return dir, nil
	}

	// Stale/partial cache dir: remove before cloning.
	_ = os.RemoveAll(dir)
	if mkErr := os.MkdirAll(filepath.Dir(dir), 0o755); mkErr != nil {
		return "", fmt.Errorf("creating depsrc cache dir: %w", mkErr)
	}

	// Shallow branch/tag clone first. Auth args (if any) precede the git
	// subcommand so `-c http.extraHeader` applies to the transport only.
	shallow := exec.Command("git", append(append([]string{}, auth...),
		"clone", "--depth", "1", "--branch", dep.GitRef, url, dir)...)
	shallow.Env = os.Environ()
	if out, cloneErr := shallow.CombinedOutput(); cloneErr != nil {
		_ = os.RemoveAll(dir)
		// GitRef may be a commit SHA (--branch rejects it): full clone + checkout.
		full := exec.Command("git", append(append([]string{}, auth...), "clone", url, dir)...)
		full.Env = os.Environ()
		if out2, fullErr := full.CombinedOutput(); fullErr != nil {
			return "", fmt.Errorf("git clone %s failed: %w\n%s\n%s",
				dep.Git, fullErr, strings.TrimSpace(string(out)), strings.TrimSpace(string(out2)))
		}
		checkout := exec.Command("git", "-C", dir, "checkout", dep.GitRef)
		checkout.Env = os.Environ()
		if out3, coErr := checkout.CombinedOutput(); coErr != nil {
			_ = os.RemoveAll(dir)
			return "", fmt.Errorf("git checkout %s in %s failed: %w\n%s",
				dep.GitRef, dep.Git, coErr, strings.TrimSpace(string(out3)))
		}
	}

	return dir, nil
}

// isGitCheckout reports whether dir looks like a populated git working tree.
func isGitCheckout(dir string) bool {
	if fi, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return fi.IsDir() || !fi.IsDir() // .git may be a dir (clone) or a file (worktree)
	}
	return false
}

// refreshCheckout best-effort updates a cached branch checkout to the remote
// tip. It is a no-op-on-error for immutable refs (tags/commits) and offline
// runs.
func refreshCheckout(dir, ref string, auth []string) error {
	fetchArgs := append(append([]string{}, auth...), "-C", dir, "fetch", "--depth", "1", "origin", ref)
	fetch := exec.Command("git", fetchArgs...)
	fetch.Env = os.Environ()
	if out, err := fetch.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	reset := exec.Command("git", "-C", dir, "reset", "--hard", "FETCH_HEAD")
	reset.Env = os.Environ()
	if out, err := reset.CombinedOutput(); err != nil {
		return fmt.Errorf("git reset: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// normalizeGitURL turns a short "github.com/org/repo" form into a clonable
// https URL. A URL that already carries a scheme (https://, git@, file://,
// /abs/path) is used verbatim. The returned URL is always token-free so it can
// be safely persisted in the cache checkout's .git/config; auth is supplied
// transiently via gitAuthArgs instead.
func normalizeGitURL(repo string) string {
	url := repo
	if !strings.Contains(url, "://") && !strings.HasPrefix(url, "git@") && !strings.HasPrefix(url, "/") {
		url = "https://" + strings.TrimSuffix(url, ".git") + ".git"
	}
	return url
}

// gitAuthArgs returns git `-c` config args that inject a best-effort GitHub
// token as a transient HTTP Authorization header for github.com https clones of
// private repos. Passing the credential via `-c http.extraHeader` (rather than
// embedding it in the clone URL) keeps the token OUT of the persisted checkout's
// .git/config — it applies only to the single git invocation. Public clones
// succeed without it; a bad/absent token simply fails auth for private repos.
func gitAuthArgs(url string) []string {
	if !strings.HasPrefix(url, "https://github.com/") {
		return nil
	}
	tok := githubTokenBestEffort()
	if tok == "" {
		return nil
	}
	basic := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + tok))
	return []string{"-c", "http.extraHeader=AUTHORIZATION: Basic " + basic}
}

// githubTokenBestEffort returns a GitHub token from the environment or the apx
// credential cache, or "" if none is available. It never triggers an
// interactive device-flow login — override materialization must stay
// non-interactive.
func githubTokenBestEffort() string {
	for _, env := range []string{"APX_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	if org, err := githubauth.DetectOrg(); err == nil && org != "" {
		if tok, err := githubauth.LoadToken(org); err == nil && tok != nil && tok.AccessToken != "" {
			return tok.AccessToken
		}
	}
	return ""
}

// depSrcCacheDir returns the persistent cache root for git overrides,
// ~/.cache/apx/depsrc (overridable via APX_DEPSRC_CACHE for tests).
func depSrcCacheDir() (string, error) {
	if custom := os.Getenv(depSrcCacheEnv); custom != "" {
		if err := os.MkdirAll(custom, 0o755); err != nil {
			return "", fmt.Errorf("creating depsrc cache %q: %w", custom, err)
		}
		return custom, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory for depsrc cache: %w", err)
	}
	dir := filepath.Join(home, ".cache", "apx", "depsrc")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating depsrc cache: %w", err)
	}
	return dir, nil
}

var pathSanitizeRE = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// sanitizeForPath maps an arbitrary repo/ref string to a single safe path
// segment so it can key a cache directory without directory traversal.
func sanitizeForPath(s string) string {
	s = strings.TrimSuffix(s, ".git")
	s = pathSanitizeRE.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		s = "_"
	}
	return s
}
