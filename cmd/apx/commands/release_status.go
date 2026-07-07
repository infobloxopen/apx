package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/publisher"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

// ErrNotInSync is returned by `apx release status --exit-code` when the module
// is not in sync with the catalog (changed or absent). main maps it to exit
// code 2 and prints nothing extra — the status was already reported to stdout.
// (Returning a sentinel rather than exiting the process directly keeps the
// "only main() exits" principle and lets deferred cleanup of the clone run.)
var ErrNotInSync = errors.New("module not in sync with catalog")

// DriftStatus is the outcome of an `apx release status` check.
type DriftStatus string

const (
	// DriftUnchanged means the local module content matches the latest
	// published version in the canonical catalog — nothing to publish.
	DriftUnchanged DriftStatus = "unchanged"
	// DriftChanged means a version is published but the local content differs —
	// the API changed and needs a new release.
	DriftChanged DriftStatus = "changed"
	// DriftAbsent means no version of this module is published yet — first
	// publish.
	DriftAbsent DriftStatus = "absent"
)

// driftResult is the machine-readable result of a status check.
type driftResult struct {
	APIID            string      `json:"api_id"`
	Status           DriftStatus `json:"status"`
	PublishedVersion string      `json:"published_version,omitempty"`
	PublishedTag     string      `json:"published_tag,omitempty"`
	CompareRef       string      `json:"compare_ref,omitempty"`
	LocalHash        string      `json:"local_hash,omitempty"`
	PublishedHash    string      `json:"published_hash,omitempty"`
	CanonicalRepo    string      `json:"canonical_repo,omitempty"`
	LocalPath        string      `json:"local_path,omitempty"`
}

func newReleaseStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <api-id>",
		Short: "Report whether the local module matches what is published in the catalog",
		Long: `Status answers "is my current API already published?" by comparing the
local module content for an API against the latest version published in the
canonical catalog. It reports one of:

  unchanged  local content matches the latest published version (in sync)
  changed    a version is published but local content differs (needs release)
  absent     no version is published yet (first publish)

The comparison is scoped to schema content: generated packaging such as
go.mod and apx sidecar files are ignored, so an unchanged API is not
reported as drift.

Content is compared against the canonical repo's default branch (HEAD) by
default — the live catalog state — because release tags can lag the published
content. Use --against to compare a specific tag or branch instead. The
latest release tag is reported as the published version.

The canonical repo is read only — no PR is created, so this is safe to run
in PR CI and on merge. Provide an existing local clone with --canonical-dir
(no network, works with any forge including Gitea) or a repo URL with
--canonical-repo to have apx clone it.

Examples:
  apx release status openapi/csp.infoblox.com/identity/v2 --canonical-dir ../apis
  apx release status openapi/csp.infoblox.com/identity/v2 --canonical-repo github.com/acme/apis --format json
  apx release status openapi/csp.infoblox.com/identity/v2 --canonical-dir ../apis --exit-code`,
		Args: cobra.ExactArgs(1),
		RunE: releaseStatusAction,
		// --exit-code returns ErrNotInSync as a clean signal; don't let cobra
		// print "Error:"/usage for it (the status is already on stdout).
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.Flags().String("canonical-repo", "", "Canonical repository URL to clone read-only (falls back to apx.yaml org/repo)")
	cmd.Flags().String("canonical-dir", "", "Path to an existing local clone of the canonical repo (skips cloning)")
	cmd.Flags().String("against", "HEAD", "Canonical git ref to compare content against (default: default-branch HEAD)")
	cmd.Flags().String("format", "table", "Output format: table, json")
	cmd.Flags().Bool("exit-code", false, "Exit with code 2 when the module is not in sync (changed or absent)")
	return cmd
}

func releaseStatusAction(cmd *cobra.Command, args []string) error {
	apiID := args[0]
	format, _ := cmd.Flags().GetString("format")
	canonicalRepo, _ := cmd.Flags().GetString("canonical-repo")
	canonicalDir, _ := cmd.Flags().GetString("canonical-dir")
	against, _ := cmd.Flags().GetString("against")
	wantExitCode, _ := cmd.Flags().GetBool("exit-code")

	if _, parseErr := config.ParseAPIID(apiID); parseErr != nil {
		return parseErr
	}

	cfg, err := loadConfig(cmd)
	if err != nil {
		cfg = &config.Config{}
	}

	// Resolve the local module directory (module_roots + well-known fallbacks).
	localPath, err := config.ResolveAPIPath(apiID, cfg)
	if err != nil {
		return fmt.Errorf("local module not found for %s: %w", apiID, err)
	}

	// Resolve a canonical repo string for reporting and clone fallback.
	if canonicalRepo == "" {
		canonicalRepo = resolveSourceRepo(cmd)
	}

	// Obtain a local clone (with tags) of the canonical repo to read from.
	repoDir := canonicalDir
	cleanup := func() {}
	defer func() { cleanup() }()
	if repoDir == "" {
		if canonicalRepo == "" || canonicalRepo == "github.com/<org>/<repo>" {
			return publisher.NewReleaseError(
				publisher.ErrCodeMissingConfig,
				"cannot determine canonical repo; use --canonical-dir, --canonical-repo, or configure org/repo in apx.yaml",
			)
		}
		cloned, cl, cloneErr := clonePublishedRepo(canonicalRepo)
		if cloneErr != nil {
			return cloneErr
		}
		cleanup = cl
		repoDir = cloned
	}

	result, err := computeDriftStatus(apiID, localPath, repoDir, canonicalRepo, against)
	if err != nil {
		return err
	}

	if format == "json" {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		printDriftTable(result)
	}

	if wantExitCode && result.Status != DriftUnchanged {
		return ErrNotInSync // main maps this to exit 2; deferred cleanup still runs.
	}
	return nil
}

// computeDriftStatus compares the local module content against the canonical
// repo's content at ref (default-branch HEAD unless overridden). It is the
// testable core of the status command, independent of cobra and cloning.
//
// "Published" content is taken from ref rather than the release tag because
// release tags can lag the catalog's live content (a manual-runbook / G3
// artifact). The latest release tag is still reported as the published version.
func computeDriftStatus(apiID, localPath, repoDir, canonicalRepo, ref string) (*driftResult, error) {
	if ref == "" {
		ref = "HEAD"
	}
	// git treats a leading-dash ref as an option; reject rather than risk it.
	if strings.HasPrefix(ref, "-") {
		return nil, fmt.Errorf("invalid ref %q", ref)
	}
	res := &driftResult{APIID: apiID, CanonicalRepo: canonicalRepo, LocalPath: localPath, CompareRef: ref}

	// Compare only schema content for the module's format (an allowlist), so
	// generated packaging (go.mod), apx sidecars (.apx-*.yaml), catalog.yaml, or
	// a stray README in the module dir never register as drift. skip==true means
	// "not schema, exclude from the hash".
	format := "openapi"
	if api, perr := config.ParseAPIID(apiID); perr == nil && api.Format != "" {
		format = api.Format
	}
	skip := func(rel string) bool { return !isSchemaFile(rel, format) }

	localHash, err := publisher.HashDirectoryFiltered(localPath, skip)
	if err != nil {
		return nil, fmt.Errorf("hashing local module %s: %w", localPath, err)
	}
	res.LocalHash = localHash

	if !refExists(repoDir, ref) {
		return nil, fmt.Errorf("canonical ref %q not found in %s", ref, repoDir)
	}

	// Report the latest release tag as the published version (informational).
	tm := publisher.NewTagManager(repoDir, "")
	if versions, verr := tm.ListVersionsForAPI(apiID); verr == nil && len(versions) > 0 {
		res.PublishedVersion = latestSemver(versions)
		res.PublishedTag = config.DeriveTag(apiID, res.PublishedVersion)
	}

	// The module's canonical path is the api-id (see manifest.CanonicalPath).
	contentDir := filepath.Join(repoDir, filepath.FromSlash(apiID))
	hasSchema, err := moduleHasSchemaAtRef(repoDir, ref, apiID, skip)
	if err != nil {
		return nil, err
	}
	if !hasSchema {
		// No schema files published for this module at ref → first publish.
		res.Status = DriftAbsent
		return res, nil
	}

	pubHash, err := publisher.HashGitTreeAtTagFiltered(repoDir, ref, contentDir, skip)
	if err != nil {
		return nil, fmt.Errorf("reading published content at %s: %w", ref, err)
	}
	res.PublishedHash = pubHash

	if localHash == pubHash {
		res.Status = DriftUnchanged
	} else {
		res.Status = DriftChanged
	}
	return res, nil
}

// refExists reports whether a git ref resolves to a commit in repoDir.
func refExists(repoDir, ref string) bool {
	_, err := runGitStatus("-C", repoDir, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	return err == nil
}

// moduleHasSchemaAtRef reports whether the module path has any schema file (per
// the skip predicate) in the tree at ref. It scopes "published?" to schema
// content, so a module dir holding only generated packaging reads as absent
// rather than erroring.
func moduleHasSchemaAtRef(repoDir, ref, modulePath string, skip func(rel string) bool) (bool, error) {
	out, err := runGitStatus("-C", repoDir, "ls-tree", "-r", "--name-only", ref, "--", modulePath)
	if err != nil {
		return false, fmt.Errorf("listing %s at %s: %s", modulePath, ref, strings.TrimSpace(out))
	}
	prefix := modulePath + "/"
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		rel := strings.TrimPrefix(line, prefix)
		if !skip(rel) {
			return true, nil
		}
	}
	return false, nil
}

// isSchemaFile reports whether a module-relative slash path is a schema file of
// the given API format (an allowlist). Dotfiles (apx sidecars like
// .apx-release.yaml) and known packaging (go.mod/go.sum/catalog.yaml) are never
// schema even when their extension matches, so they cannot cause false drift.
func isSchemaFile(rel, format string) bool {
	base := path.Base(rel)
	if strings.HasPrefix(base, ".") {
		return false
	}
	switch base {
	case "go.mod", "go.sum", "catalog.yaml", "apx.lock":
		return false
	}
	ext := strings.ToLower(path.Ext(base))
	switch format {
	case "proto":
		return ext == ".proto"
	case "avro":
		return ext == ".avsc"
	case "jsonschema":
		return ext == ".json"
	case "parquet":
		return ext == ".parquet"
	default: // openapi, crd
		return ext == ".yaml" || ext == ".yml" || ext == ".json"
	}
}

// latestSemver returns the newest version from a list, falling back to string
// ordering for entries that do not parse as semver.
func latestSemver(versions []string) string {
	sorted := append([]string(nil), versions...)
	sort.Slice(sorted, func(i, j int) bool {
		a, ea := config.ParseSemVer(sorted[i])
		b, eb := config.ParseSemVer(sorted[j])
		if ea != nil || eb != nil {
			return sorted[i] > sorted[j]
		}
		return config.CompareSemVer(a, b) > 0
	})
	return sorted[0]
}

// clonePublishedRepo clones the canonical repo (with tags) into a temp dir for
// read-only inspection. It returns the clone dir and a cleanup func. The repo
// may be "github.com/org/repo", a full URL, "git@host:org/repo", or
// "host/org/repo" for any forge; a bare "host/org/repo" is cloned over https.
func clonePublishedRepo(repo string) (string, func(), error) {
	tmp, err := os.MkdirTemp("", "apx-drift-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("creating temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }

	dest := filepath.Join(tmp, "canonical")
	url := normalizeCloneURL(repo)
	if out, cloneErr := runGitStatus("clone", "--quiet", url, dest); cloneErr != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("cloning canonical repo %s: %s", url, strings.TrimSpace(out))
	}
	return dest, cleanup, nil
}

func normalizeCloneURL(repo string) string {
	if strings.Contains(repo, "://") || strings.HasPrefix(repo, "git@") {
		return repo
	}
	repo = strings.TrimSuffix(strings.TrimSuffix(repo, "/"), ".git")
	return "https://" + repo + ".git"
}

func runGitStatus(args ...string) (string, error) {
	out, err := exec.Command("git", args...).CombinedOutput()
	return string(out), err
}

func printDriftTable(r *driftResult) {
	ui.Info("Drift status for %s:", r.APIID)
	ui.Info("")
	pub := r.PublishedVersion
	if pub == "" {
		pub = "(untagged)"
	}
	switch r.Status {
	case DriftUnchanged:
		ui.Success("  unchanged — in sync with the catalog (published version %s)", pub)
	case DriftChanged:
		ui.Warning("  changed — local content differs from the catalog (published version %s); needs a new release", pub)
	case DriftAbsent:
		ui.Warning("  absent — not yet in the catalog; first release")
	}
	if r.CanonicalRepo != "" {
		ui.Info("  canonical:      %s", r.CanonicalRepo)
	}
	if r.CompareRef != "" {
		ui.Info("  compared @:     %s", r.CompareRef)
	}
	if r.PublishedHash != "" {
		ui.Info("  local hash:     %s", shortHash(r.LocalHash))
		ui.Info("  published hash: %s", shortHash(r.PublishedHash))
	}
}

func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}
