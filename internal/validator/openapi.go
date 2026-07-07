package validator

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// OpenAPIValidator handles OpenAPI schema validation
type OpenAPIValidator struct {
	resolver *ToolchainResolver
}

// NewOpenAPIValidator creates a new OpenAPI validator
func NewOpenAPIValidator(resolver *ToolchainResolver) *OpenAPIValidator {
	return &OpenAPIValidator{resolver: resolver}
}

// Lint runs spectral lint on OpenAPI specs
func (v *OpenAPIValidator) Lint(path string) error {
	spectralPath, err := v.resolver.ResolveTool("spectral", "v6.15.0")
	if err != nil {
		return fmt.Errorf("failed to resolve spectral: %w", err)
	}

	// finalize passes the module DIRECTORY; spectral does not glob a bare
	// directory ("No files found to lint"), so resolve it to the spec file
	// first (WS-035 G4 directory-glob shim).
	specPath, err := resolveOpenAPISpecFile(path)
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(specPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	cmd := exec.Command(spectralPath, "lint", absPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("spectral lint failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// Breaking runs oasdiff to detect breaking changes between the base spec
// (against) and the revision (revPath).
//
// oasdiff compares two spec FILES and — unlike buf — cannot read a git ref. At
// finalize the revision is handed in as the module DIRECTORY and the base is the
// PREVIOUS RELEASE TAG (e.g. openapi/csp.infoblox.com/ddi-wapi/v3/v3.0.0). So the
// revision is first resolved to the OpenAPI spec inside the module dir, and —
// when the base is a git ref rather than a path on disk — the spec's committed
// content at that ref is materialized to a temp file (git show <ref>:<spec path>)
// so the comparison actually runs. Without this, oasdiff treats the tag as a
// filesystem path and fails to "load base spec" (WS-035 F-29), which the caller
// misreads as "breaking detected" and which blocks every second release on a
// module line. The comparison RESULT is left unchanged (the no-op-severity
// behavior is the separate G7 issue).
func (v *OpenAPIValidator) Breaking(revPath, against string) error {
	oasdiffPath, err := v.resolver.ResolveTool("oasdiff", "v1.9.6")
	if err != nil {
		return fmt.Errorf("failed to resolve oasdiff: %w", err)
	}

	// Resolve the revision to the actual spec file (finalize passes a directory).
	revFile, err := resolveOpenAPISpecFile(revPath)
	if err != nil {
		return err
	}
	absRev, err := filepath.Abs(revFile)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Resolve the base. When against is a readable path use it directly;
	// otherwise treat it as a git ref and materialize its committed spec content.
	baseArg := against
	if _, statErr := os.Stat(against); statErr != nil {
		baseFile, cleanup, mErr := baseSpecFromRef(absRev, against)
		if mErr != nil {
			return mErr
		}
		defer cleanup()
		baseArg = baseFile
	}

	// --fail-on ERR is load-bearing: without it oasdiff exits 0 even when it
	// reports breaking changes, so the process exit code alone is a no-op gate
	// (WS-035 G7). With it, oasdiff exits non-zero when a breaking (ERR-level)
	// change is present, so a removed path or removed required field actually
	// fails the check and drives the semver bump.
	cmd := exec.Command(oasdiffPath, "breaking", "--fail-on", "ERR", baseArg, absRev)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("oasdiff breaking failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// resolveOpenAPISpecFile returns the OpenAPI spec file at p. When p is a module
// directory (as finalize passes it), it selects the single spec file inside,
// skipping non-spec artifacts (go.mod, go.sum, dotfiles such as .spectral.yaml).
func resolveOpenAPISpecFile(p string) (string, error) {
	info, err := os.Stat(p)
	if err != nil {
		return "", fmt.Errorf("locating OpenAPI spec: %w", err)
	}
	if !info.IsDir() {
		return p, nil
	}
	entries, err := os.ReadDir(p)
	if err != nil {
		return "", fmt.Errorf("reading module dir %s: %w", p, err)
	}
	var fallback string
	for _, e := range entries {
		if e.IsDir() || !isSpecCandidate(e.Name()) {
			continue
		}
		full := filepath.Join(p, e.Name())
		if detectFormatFromFile(full) == FormatOpenAPI {
			return full, nil
		}
		if fallback == "" {
			fallback = full
		}
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("no OpenAPI spec file found in %s", p)
}

// isSpecCandidate reports whether a filename could be an OpenAPI spec (a
// .yaml/.yml/.json file that is not a Go module file or a dotfile config).
func isSpecCandidate(name string) bool {
	if strings.HasPrefix(name, ".") || name == "go.mod" || name == "go.sum" {
		return false
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".yaml", ".yml", ".json":
		return true
	}
	return false
}

// baseSpecFromRef materializes the OpenAPI spec's committed content at a git ref
// (a release tag) into a temp file so oasdiff can load it as the base. It uses
// the revision spec's repo-relative path; if that path is absent at the ref (the
// spec was renamed between releases) it falls back to the first spec file in the
// same module directory at the ref. The returned cleanup removes the temp file.
func baseSpecFromRef(revFile, ref string) (string, func(), error) {
	fail := func(err error) (string, func(), error) {
		return "", nil, fmt.Errorf("loading base spec from %q: %w", ref, err)
	}
	root, err := gitTopLevel(filepath.Dir(revFile))
	if err != nil {
		return fail(err)
	}
	// git reports a symlink-resolved top level (e.g. /var -> /private/var on
	// macOS); canonicalize the revision path the same way so the relative path
	// used for `git show` is not a spurious ../.. escape.
	if resolved, e := filepath.EvalSymlinks(revFile); e == nil {
		revFile = resolved
	}
	rel, err := filepath.Rel(root, revFile)
	if err != nil {
		return fail(err)
	}
	rel = filepath.ToSlash(rel)

	content, err := gitShow(root, ref, rel)
	if err != nil {
		if alt, aErr := firstSpecAtRef(root, ref, path.Dir(rel)); aErr == nil && alt != rel {
			content, err = gitShow(root, ref, alt)
		}
	}
	if err != nil {
		return fail(err)
	}

	tmp, err := os.CreateTemp("", "apx-base-*"+filepath.Ext(revFile))
	if err != nil {
		return fail(err)
	}
	cleanup := func() { _ = os.Remove(tmp.Name()) }
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		cleanup()
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fail(err)
	}
	return tmp.Name(), cleanup, nil
}

// gitTopLevel returns the git working-tree root at or above dir.
func gitTopLevel(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --show-toplevel in %s: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// gitShow returns the committed content of relPath at ref.
func gitShow(root, ref, relPath string) ([]byte, error) {
	out, err := exec.Command("git", "-C", root, "show", ref+":"+relPath).Output()
	if err != nil {
		return nil, fmt.Errorf("git show %s:%s: %w", ref, relPath, err)
	}
	return out, nil
}

// firstSpecAtRef returns the repo-relative path of the first OpenAPI-looking
// spec file in relDir at ref (skipping go.mod/go.sum/dotfiles).
func firstSpecAtRef(root, ref, relDir string) (string, error) {
	out, err := exec.Command("git", "-C", root, "ls-tree", "--name-only", ref, relDir+"/").Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" && isSpecCandidate(path.Base(line)) {
			return line, nil
		}
	}
	return "", fmt.Errorf("no spec file in %s at %s", relDir, ref)
}
