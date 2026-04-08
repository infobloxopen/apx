package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/language"
	"github.com/infobloxopen/apx/internal/policy"
	"github.com/infobloxopen/apx/internal/publisher"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/infobloxopen/apx/internal/validator"
	"github.com/infobloxopen/apx/pkg/githubauth"
	"github.com/spf13/cobra"
)

func newReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Manage releases through the release state machine",
		Long: `Release commands let you prepare, submit, finalize, and inspect releases.

A release progresses through explicit states:
  draft → validated → version-selected → prepared → submitted →
  canonical-validated → canonical-released → package-published

Use 'apx release prepare' to validate and build a release manifest.
Use 'apx release submit' to push the release to the canonical repo.
Use 'apx release finalize' to run canonical CI processing.
Use 'apx release inspect' to view the current release state.
Use 'apx release history' to list all releases for an API.
Use 'apx release promote' to promote an API to a new lifecycle.`,
	}
	cmd.AddCommand(
		newReleasePrepareCmd(),
		newReleaseSubmitCmd(),
		newReleaseFinalizeCmd(),
		newReleaseInspectCmd(),
		newReleaseHistoryCmd(),
		newReleasePromoteCmd(),
	)
	return cmd
}

// ---------------------------------------------------------------------------
// apx release prepare
// ---------------------------------------------------------------------------

func newReleasePrepareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prepare <api-id>",
		Short: "Validate and build a release manifest",
		Long: `Prepare a release by validating the schema, computing identity and
language coordinates, checking version-lifecycle compatibility, and
producing a machine-readable release manifest.

The manifest is written to .apx-release.yaml in the current directory.

Examples:
  apx release prepare proto/payments/ledger/v1 --version v1.0.0-beta.1 --lifecycle beta`,
		Args: cobra.ExactArgs(1),
		RunE: releasePrepareAction,
	}
	cmd.Flags().String("version", "", "Version to release (e.g. v1.0.0-beta.1)")
	cmd.Flags().String("lifecycle", "", "Lifecycle state (experimental, beta, stable, deprecated, sunset)")
	cmd.Flags().String("canonical-repo", "", "Canonical repository URL")
	cmd.Flags().Bool("strict", false, "Make go_package mismatches an error")
	cmd.Flags().Bool("skip-gomod", false, "Skip go.mod generation and validation")
	cmd.Flags().Bool("force", false, "Override sunset block")
	cmd.Flags().Bool("dry-run", false, "Show what would be prepared without writing the manifest")
	_ = cmd.MarkFlagRequired("version")
	return cmd
}

func releasePrepareAction(cmd *cobra.Command, args []string) error {
	apiID := args[0]
	version, _ := cmd.Flags().GetString("version")
	lifecycle, _ := cmd.Flags().GetString("lifecycle")
	canonicalRepo, _ := cmd.Flags().GetString("canonical-repo")
	strict, _ := cmd.Flags().GetBool("strict")
	skipGomod, _ := cmd.Flags().GetBool("skip-gomod")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	cfg, err := loadConfig(cmd)
	if err != nil {
		ui.Warning("Could not load config for policy check: %v", err)
		cfg = &config.Config{}
	}

	// Validate that the module path exists on disk before proceeding.
	if _, resolveErr := config.ResolveAPIPath(apiID, cfg); resolveErr != nil {
		return fmt.Errorf("module path does not exist: %s", apiID)
	}

	// --- State: draft ---
	ui.Info("Preparing release for %s @ %s", apiID, version)

	// Validate lifecycle
	if lifecycle != "" {
		if err := config.ValidateLifecycle(lifecycle); err != nil {
			return err
		}
	}

	// Validate version-lifecycle compatibility
	if lifecycle != "" && !force {
		if err := config.ValidateVersionLifecycle(version, lifecycle); err != nil {
			return &publisher.ReleaseError{
				Code:    publisher.ErrCodeLifecycleMismatch,
				Message: err.Error(),
				Hint:    "Use --force to override lifecycle checks",
			}
		}
	}
	if config.LifecycleRequiresWarning(lifecycle) {
		ui.Warning("Releasing under deprecated lifecycle — consumers should migrate")
	}

	// Validate version-line compatibility (major version must match API line)
	line := config.ParseLineFromID(apiID)
	if err := config.ValidateVersionLine(version, line); err != nil {
		return &publisher.ReleaseError{
			Code:    publisher.ErrCodeVersionLineMismatch,
			Message: err.Error(),
			Hint:    "Ensure version major matches the API line (e.g. v1.x.x for /v1)",
		}
	}

	// v0 line policy enforcement
	if config.IsV0Line(line) && lifecycle != "" && !force {
		if err := config.ValidateV0Lifecycle(lifecycle); err != nil {
			return &publisher.ReleaseError{
				Code:    publisher.ErrCodeLifecycleMismatch,
				Message: err.Error(),
				Hint:    "v0 lines must use 'experimental' or 'beta' lifecycle",
			}
		}
	}

	// Validate lifecycle transition (if previous lifecycle is known)
	if lifecycle != "" && !force {
		if prevLifecycle := resolveCurrentLifecycle(cmd, apiID); prevLifecycle != "" {
			if err := config.ValidateLifecycleTransition(prevLifecycle, lifecycle); err != nil {
				return &publisher.ReleaseError{
					Code:    publisher.ErrCodeIllegalTransition,
					Message: err.Error(),
					Hint:    "Lifecycle can only move forward: experimental → beta → stable → deprecated → sunset",
				}
			}
		}
	}

	// Resolve source repo
	sourceRepo := canonicalRepo
	if sourceRepo == "" {
		sourceRepo = resolveSourceRepo(cmd)
		if sourceRepo == "github.com/<org>/<repo>" {
			return publisher.NewReleaseError(
				publisher.ErrCodeMissingConfig,
				"cannot determine canonical repo; use --canonical-repo or configure org/repo in apx.yaml",
			)
		}
	}

	// Build identity
	api, source, release, err := config.BuildIdentityBlock(apiID, sourceRepo, lifecycle, version)
	if err != nil {
		return err
	}

	langs, err := language.DeriveAllCoords(language.DerivationContext{
		SourceRepo: sourceRepo,
		ImportRoot: resolveImportRoot(cmd),
		Org:        resolveOrg(cmd),
		API:        api,
	})
	if err != nil {
		return err
	}

	// Create manifest
	manifest := publisher.NewManifest(api, source, langs, version, sourceRepo)

	// --- State: validated (run validations) ---
	manifest.Validation = &publisher.ValidationResults{
		Lint:     publisher.ValidationSkipped,
		Breaking: publisher.ValidationSkipped,
		Policy:   publisher.ValidationSkipped,
	}

	// go_package validation (proto only)
	if api.Format == "proto" {
		manifest.Validation.GoPackage = publisher.ValidationSkipped
		repoPath, _ := os.Getwd()
		protoDir := filepath.Join(repoPath, source.Path)
		if info, statErr := os.Stat(protoDir); statErr == nil && info.IsDir() {
			protoFiles, globErr := validator.GlobProtoFiles(protoDir)
			if globErr != nil {
				ui.Warning("Could not scan for proto files: %v", globErr)
			}
			expectedImport := ""
			if goCoords, ok := langs["go"]; ok {
				expectedImport = goCoords.Import
			}
			goPackageOk := true
			if expectedImport != "" {
				for _, pf := range protoFiles {
					importPath, _, extractErr := validator.ExtractGoPackage(pf)
					if extractErr != nil {
						ui.Warning("Could not extract go_package from %s: %v", pf, extractErr)
						continue
					}
					if importPath == "" {
						continue
					}
					if valErr := config.ValidateGoPackage(importPath, expectedImport); valErr != nil {
						relPath, _ := filepath.Rel(repoPath, pf)
						if relPath == "" {
							relPath = pf
						}
						if strict {
							manifest.Validation.GoPackage = publisher.ValidationFailed
							manifest.Fail(string(publisher.ErrCodeGoPackageMismatch), valErr.Error(), "prepare")
							if writeErr := publisher.WriteManifest(manifest, ".apx-release.yaml"); writeErr != nil {
								ui.Warning("Could not write manifest: %v", writeErr)
							}
							return &publisher.ReleaseError{
								Code:    publisher.ErrCodeGoPackageMismatch,
								Message: fmt.Sprintf("%s: %v", relPath, valErr),
							}
						}
						ui.Warning("%s: %v", relPath, valErr)
						goPackageOk = false
					}
				}
			}
			if goPackageOk {
				manifest.Validation.GoPackage = publisher.ValidationPassed
			}
		}
	}

	// go.mod validation
	if !skipGomod {
		manifest.Validation.GoMod = publisher.ValidationSkipped
		repoPath, _ := os.Getwd()
		goModDir := config.DeriveGoModDir(api)
		goModPath := filepath.Join(repoPath, goModDir, "go.mod")
		goModulePath := ""
		if goCoords, ok := langs["go"]; ok {
			goModulePath = goCoords.Module
		}
		if goModulePath != "" {
			if existing, readErr := os.ReadFile(goModPath); readErr == nil {
				existingMod, parseErr := publisher.ParseGoModModule(existing)
				if parseErr != nil {
					manifest.Validation.GoMod = publisher.ValidationFailed
					manifest.Fail(string(publisher.ErrCodeGoModMismatch), parseErr.Error(), "prepare")
					_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
					return &publisher.ReleaseError{
						Code:    publisher.ErrCodeGoModMismatch,
						Message: fmt.Sprintf("invalid go.mod at %s: %v", goModDir, parseErr),
					}
				}
				if existingMod != goModulePath {
					manifest.Validation.GoMod = publisher.ValidationFailed
					manifest.Fail(string(publisher.ErrCodeGoModMismatch),
						fmt.Sprintf("got %q, expected %q", existingMod, goModulePath), "prepare")
					_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
					return &publisher.ReleaseError{
						Code:    publisher.ErrCodeGoModMismatch,
						Message: fmt.Sprintf("go.mod module mismatch at %s: got %q, expected %q", goModDir, existingMod, goModulePath),
					}
				}
				manifest.Validation.GoMod = publisher.ValidationPassed
				ui.Info("go.mod validated: %s", goModDir)
			} else if os.IsNotExist(readErr) {
				manifest.Validation.GoMod = publisher.ValidationPassed
				ui.Info("go.mod will be generated at %s during submit", goModDir)
			}
		}
	}

	// Policy validation
	{
		repoPath, _ := os.Getwd()
		schemaDir := filepath.Join(repoPath, source.Path)
		if _, statErr := os.Stat(schemaDir); statErr == nil {
			polResult, polErr := policy.Check(cfg.Policy, schemaDir)
			if polErr != nil {
				ui.Warning("Policy check error: %v", polErr)
				manifest.Validation.Policy = publisher.ValidationSkipped
			} else if !polResult.Passed() {
				manifest.Validation.Policy = publisher.ValidationFailed
				for _, v := range polResult.Violations {
					ui.Error("[%s] %s", v.Rule, v.Message)
				}
				manifest.Fail(string(publisher.ErrCodePolicyFailed),
					fmt.Sprintf("%d policy violation(s)", len(polResult.Violations)), "prepare")
				_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
				return &publisher.ReleaseError{
					Code:    publisher.ErrCodePolicyFailed,
					Message: fmt.Sprintf("policy check failed: %d violation(s)", len(polResult.Violations)),
				}
			} else {
				manifest.Validation.Policy = publisher.ValidationPassed
				ui.Info("Policy check passed (%d rule(s) evaluated)", polResult.Checked)
			}
		} else {
			manifest.Validation.Policy = publisher.ValidationSkipped
		}
	}

	if err := manifest.SetState(publisher.StateValidated); err != nil {
		return err
	}

	// --- State: version-selected ---
	if err := manifest.SetState(publisher.StateVersionSelected); err != nil {
		return err
	}

	// Capture source commit
	repoPath, _ := os.Getwd()
	if commitOut, gitErr := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").Output(); gitErr == nil {
		manifest.SourceCommit = strings.TrimSpace(string(commitOut))
	}

	// Idempotency check
	tag := config.DeriveTag(apiID, version)
	contentDir := filepath.Join(repoPath, source.Path)
	if _, statErr := os.Stat(contentDir); statErr == nil {
		result, idempErr := publisher.CheckIdempotency(repoPath, tag, contentDir)
		if idempErr != nil {
			ui.Warning("Idempotency check failed: %v", idempErr)
		} else {
			switch result {
			case publisher.ReleaseAlreadyPublished:
				ui.Success("Release %s already published with identical content — safe to skip", version)
				manifest.State = publisher.StatePackagePublished
				_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
				return nil
			case publisher.ReleaseConflict:
				manifest.Fail(string(publisher.ErrCodeVersionTaken),
					fmt.Sprintf("version %s already exists with different content", version), "prepare")
				_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
				return &publisher.ReleaseError{
					Code:    publisher.ErrCodeVersionTaken,
					Message: fmt.Sprintf("version %s already exists with different content", version),
					Hint:    "Choose a different version or investigate with 'apx release inspect'",
				}
			}
		}
	}

	// --- State: prepared ---
	if err := manifest.SetState(publisher.StatePrepared); err != nil {
		return err
	}

	// Dry-run: show what would be prepared without writing the manifest.
	if dryRun {
		ui.Info("Dry-run mode: showing what would be prepared")
		ui.Info("")
		report := language.FormatIdentityReport(api, source, release, langs)
		fmt.Print(report)
		ui.Info("Tag:         %s", tag)
		ui.Info("")
		ui.Info("(no manifest written in dry-run mode)")
		return nil
	}

	// Write manifest
	if err := publisher.WriteManifest(manifest, ".apx-release.yaml"); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	// Print summary
	ui.Success("Release prepared successfully")
	ui.Info("")
	report := language.FormatIdentityReport(api, source, release, langs)
	fmt.Print(report)
	ui.Info("Tag:         %s", tag)
	if manifest.SourceCommit != "" {
		ui.Info("Commit:      %s", manifest.SourceCommit)
	}
	ui.Info("Manifest:    .apx-release.yaml")
	ui.Info("")
	ui.Info("Next step:   apx release submit")

	return nil
}

// ---------------------------------------------------------------------------
// apx release submit
// ---------------------------------------------------------------------------

func newReleaseSubmitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit a prepared release to the canonical repo",
		Long: `Submit opens a pull request on the canonical repository with the
prepared release content (from 'apx release prepare'). It reads
.apx-release.yaml, clones the canonical repo, pushes the snapshot
to a release branch, and creates a PR.

This operation is idempotent: re-running after a partial failure will
detect existing branches and PRs, recovering gracefully without
creating duplicates.

Examples:
  apx release submit
  apx release submit --dry-run`,
		RunE: releaseSubmitAction,
	}
	cmd.Flags().Bool("dry-run", false, "Show what would be submitted without actually doing it")
	return cmd
}

func releaseSubmitAction(cmd *cobra.Command, _ []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Read manifest
	manifest, err := publisher.ReadManifest(".apx-release.yaml")
	if err != nil {
		return publisher.NewReleaseError(
			publisher.ErrCodeMissingConfig,
			"no release manifest found — run 'apx release prepare' first",
		)
	}

	// ── State guards ─────────────────────────────────────────────────
	switch manifest.State {
	case publisher.StatePrepared:
		// Expected state — proceed
	case publisher.StateCanonicalPROpen:
		// Already submitted — report existing PR and exit
		if manifest.PRURL != "" {
			ui.Success("Release already submitted — PR is open")
			ui.Info("PR:      %s", manifest.PRURL)
			if manifest.PRBranch != "" {
				ui.Info("Branch:  %s", manifest.PRBranch)
			}
			return nil
		}
		// PR metadata missing — fall through to re-submit
	case publisher.StatePackagePublished:
		ui.Success("Release already published — nothing to do")
		return nil
	case publisher.StateFailed:
		return publisher.NewReleaseError(
			publisher.ErrCodeValidationFailed,
			fmt.Sprintf("release is in failed state: %s", manifest.Error.Message),
		).WithHint("Fix the issue and re-run 'apx release prepare'")
	default:
		return fmt.Errorf("unexpected manifest state %q — expected 'prepared'", manifest.State)
	}

	// ── Dry-run path ─────────────────────────────────────────────────
	if dryRun {
		branch := publisher.ComputeReleaseBranchName(manifest.APIID, manifest.RequestedVersion)
		ui.Info("Dry-run mode: showing what would be submitted")
		ui.Info("")
		ui.Info("Branch:  %s", branch)
		ui.Info("")

		// List snapshot files
		snapshotDir := manifest.SourcePath
		if _, statErr := os.Stat(snapshotDir); statErr == nil {
			ui.Info("Snapshot files:")
			_ = filepath.Walk(snapshotDir, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil || info.IsDir() {
					return walkErr
				}
				rel, _ := filepath.Rel(snapshotDir, path)
				ui.Info("  %s", rel)
				return nil
			})
			ui.Info("")
		}

		// Show go.mod preview if applicable
		if goCoords, ok := manifest.Languages["go"]; ok && goCoords.Module != "" {
			content, genErr := publisher.GenerateGoMod(goCoords.Module, "1.21")
			if genErr == nil {
				ui.Info("go.mod preview:")
				ui.Info("%s", string(content))
			}
		}

		fmt.Print(publisher.FormatManifestReport(manifest))
		ui.Info("")
		ui.Success("Would submit release successfully")
		return nil
	}

	// ── Auth: ensure GitHub token ────────────────────────────────────
	org, orgErr := githubauth.DetectOrg()
	if orgErr != nil {
		return publisher.NewReleaseError(
			publisher.ErrCodePRCreationFailed,
			"Cannot detect GitHub org from git remote",
		).WithHint("Ensure you are in a git repository with a GitHub remote")
	}
	token, tokenErr := githubauth.EnsureToken(org)
	if tokenErr != nil {
		return publisher.NewReleaseError(
			publisher.ErrCodePRCreationFailed,
			fmt.Sprintf("GitHub authentication failed: %v", tokenErr),
		).WithHint("Run 'apx init canonical --setup-github' to set up GitHub authentication")
	}
	ghClient := githubauth.NewClient(token)

	// ── Submit via PR ────────────────────────────────────────────────
	ui.Info("Submitting release %s @ %s", manifest.APIID, manifest.RequestedVersion)

	// Build CI provenance extra for PR body
	prBodyExtra := buildCIProvenance()

	// Resolve the source path through module_roots and common fallbacks,
	// the same way releasePrepareAction does. manifest.SourcePath is the
	// bare API ID (e.g. "jsonschema/statexfer/canary-heartbeat/v1"), which
	// may not exist on disk when the app repo uses module_roots like
	// "internal/apis/jsonschema".
	cfg, _ := loadConfig(cmd)
	if cfg == nil {
		cfg = &config.Config{}
	}
	snapshotDir, resolveErr := config.ResolveAPIPath(manifest.SourcePath, cfg)
	if resolveErr != nil {
		snapshotDir = manifest.SourcePath // fall back to bare path
	}

	resp, err := publisher.SubmitReleaseWithPR(ghClient, manifest, snapshotDir, prBodyExtra)
	if err != nil {
		manifest.Fail(string(publisher.ErrCodePRCreationFailed), err.Error(), "submit")
		_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
		return &publisher.ReleaseError{
			Code:    publisher.ErrCodePRCreationFailed,
			Message: fmt.Sprintf("release submission failed: %v", err),
			Hint:    "Check authentication and try 'apx release submit' again",
		}
	}

	// ── Record PR metadata in manifest ───────────────────────────────
	manifest.PRNumber = resp.Number
	manifest.PRURL = resp.HTMLURL
	manifest.PRBranch = publisher.ComputeReleaseBranchName(manifest.APIID, manifest.RequestedVersion)

	// Record CI provenance if running in CI
	if prBodyExtra != "" {
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			manifest.CIProvider = "github-actions"
			serverURL := os.Getenv("GITHUB_SERVER_URL")
			repo := os.Getenv("GITHUB_REPOSITORY")
			runID := os.Getenv("GITHUB_RUN_ID")
			if serverURL != "" && repo != "" && runID != "" {
				manifest.CIRunURL = fmt.Sprintf("%s/%s/actions/runs/%s", serverURL, repo, runID)
			}
		} else if os.Getenv("GITLAB_CI") == "true" {
			manifest.CIProvider = "gitlab-ci"
			manifest.CIRunURL = os.Getenv("CI_PIPELINE_URL")
		} else if os.Getenv("JENKINS_URL") != "" {
			manifest.CIProvider = "jenkins"
			manifest.CIRunURL = os.Getenv("BUILD_URL")
		}
	}

	if err := manifest.SetState(publisher.StateCanonicalPROpen); err != nil {
		return err
	}
	if writeErr := publisher.WriteManifest(manifest, ".apx-release.yaml"); writeErr != nil {
		return fmt.Errorf("writing manifest: %w", writeErr)
	}

	ui.Success("✓ Release submitted successfully")
	ui.Info("PR:      %s", manifest.PRURL)
	if manifest.PRNumber != 0 {
		ui.Info("PR #:    %d", manifest.PRNumber)
	}
	ui.Info("Branch:  %s", manifest.PRBranch)
	ui.Info("Tag:     %s", manifest.Tag)

	return nil
}

// buildCIProvenance returns extra PR body content with CI provenance
// information, or an empty string if not running in CI.
func buildCIProvenance() string {
	// GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		serverURL := os.Getenv("GITHUB_SERVER_URL")
		repo := os.Getenv("GITHUB_REPOSITORY")
		runID := os.Getenv("GITHUB_RUN_ID")
		if serverURL != "" && repo != "" && runID != "" {
			runURL := fmt.Sprintf("%s/%s/actions/runs/%s", serverURL, repo, runID)
			return fmt.Sprintf("**CI**: github-actions\n**Run**: %s", runURL)
		}
		return "**CI**: github-actions"
	}

	// GitLab CI
	if os.Getenv("GITLAB_CI") == "true" {
		pipelineURL := os.Getenv("CI_PIPELINE_URL")
		if pipelineURL != "" {
			return fmt.Sprintf("**CI**: gitlab-ci\n**Run**: %s", pipelineURL)
		}
		return "**CI**: gitlab-ci"
	}

	// Jenkins
	if os.Getenv("JENKINS_URL") != "" {
		buildURL := os.Getenv("BUILD_URL")
		if buildURL != "" {
			return fmt.Sprintf("**CI**: jenkins\n**Run**: %s", buildURL)
		}
		return "**CI**: jenkins"
	}

	return ""
}

// ---------------------------------------------------------------------------
// apx release inspect
// ---------------------------------------------------------------------------

func newReleaseInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect [api-id]",
		Short: "Show the current release state",
		Long: `Inspect displays the current release state from the manifest file
(.apx-release.yaml) or from git tags for a given API ID.

Examples:
  apx release inspect
  apx release inspect proto/payments/ledger/v1`,
		Args: cobra.MaximumNArgs(1),
		RunE: releaseInspectAction,
	}
	return cmd
}

func releaseInspectAction(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")

	// Try reading manifest first
	manifest, err := publisher.ReadManifest(".apx-release.yaml")
	if err == nil {
		// Filter by API ID if provided
		if len(args) == 1 && args[0] != manifest.APIID {
			ui.Info("Manifest is for %s, not %s", manifest.APIID, args[0])
		}

		if jsonOut {
			data, err := publisher.MarshalManifest(manifest)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Print(publisher.FormatManifestReport(manifest))
		return nil
	}

	// No manifest — show tag-based info if API ID is provided
	if len(args) == 0 {
		return fmt.Errorf("no .apx-release.yaml found; provide an API ID to inspect tags")
	}

	apiID := args[0]
	api, parseErr := config.ParseAPIID(apiID)
	if parseErr != nil {
		return parseErr
	}

	repoPath, _ := os.Getwd()
	tm := publisher.NewTagManager(repoPath, "")

	// List tags matching the API ID prefix
	tagPrefix := apiID + "/"
	out, gitErr := exec.Command("git", "-C", repoPath, "tag", "-l", tagPrefix+"*").Output()
	if gitErr != nil {
		return fmt.Errorf("listing tags: %w", gitErr)
	}

	tags := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(tags) == 0 || (len(tags) == 1 && tags[0] == "") {
		ui.Info("No releases found for %s", apiID)
		return nil
	}

	_ = tm // satisfy linter
	ui.Info("Releases for %s (%s):", apiID, api.Format)
	for _, t := range tags {
		if t == "" {
			continue
		}
		version := strings.TrimPrefix(t, tagPrefix)
		ui.Info("  %s", version)
	}

	return nil
}

// resolveCurrentLifecycle attempts to determine the current lifecycle state
// for an API. It checks:
// 1. An existing .apx-release.yaml manifest
// 2. The config's API section
// Returns empty string if unknown (first-time publish).
func resolveCurrentLifecycle(cmd *cobra.Command, apiID string) string {
	// Check existing manifest
	manifest, err := publisher.ReadManifest(".apx-release.yaml")
	if err == nil && manifest != nil && manifest.Lifecycle != "" {
		return manifest.Lifecycle
	}

	// Check config
	cfg, err := loadConfig(cmd)
	if err == nil && cfg != nil && cfg.API != nil && cfg.API.Lifecycle != "" {
		return cfg.API.Lifecycle
	}

	return ""
}

// ---------------------------------------------------------------------------
// apx release finalize
// ---------------------------------------------------------------------------

func newReleaseFinalizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "finalize",
		Short: "Run canonical CI release processing",
		Long: `Finalize is run by canonical CI after a release has been submitted.
It re-validates the schema, creates the official canonical tag,
updates the catalog, records artifact metadata, and emits an
immutable release record.

Go modules are published implicitly via the subdirectory tag — the Go
module proxy picks them up automatically. Other language packages
(Maven, wheels, OCI) require separate CI workflow steps that teams
configure outside APX.

The manifest must be in 'submitted' or 'canonical-pr-open' state.

Examples:
  apx release finalize
  apx release finalize --catalog catalog.yaml
  apx release finalize --skip-packages`,
		RunE: releaseFinalizeAction,
	}
	cmd.Flags().String("catalog", "catalog.yaml", "Path to catalog.yaml")
	cmd.Flags().Bool("skip-packages", false, "Skip recording Go module artifact metadata")
	cmd.Flags().Bool("skip-catalog", false, "Skip catalog update")
	cmd.Flags().String("record-path", ".apx-release-record.yaml", "Path to write the release record")
	return cmd
}

func releaseFinalizeAction(cmd *cobra.Command, _ []string) error {
	catalogPath, _ := cmd.Flags().GetString("catalog")
	skipPackages, _ := cmd.Flags().GetBool("skip-packages")
	skipCatalog, _ := cmd.Flags().GetBool("skip-catalog")
	recordPath, _ := cmd.Flags().GetString("record-path")

	cfg, err := loadConfig(cmd)
	if err != nil {
		ui.Warning("Could not load config for policy check: %v", err)
		cfg = &config.Config{}
	}

	// Read manifest
	manifest, err := publisher.ReadManifest(".apx-release.yaml")
	if err != nil {
		return publisher.NewReleaseError(
			publisher.ErrCodeMissingConfig,
			"no release manifest found — run 'apx release prepare' and 'apx release submit' first",
		)
	}

	// Verify state: must be submitted or canonical-pr-open
	switch manifest.State {
	case publisher.StateSubmitted, publisher.StateCanonicalPROpen:
		// OK
	case publisher.StatePackagePublished:
		ui.Success("Release already finalized — nothing to do")
		return nil
	case publisher.StateFailed:
		return publisher.NewReleaseError(
			publisher.ErrCodeValidationFailed,
			fmt.Sprintf("release is in failed state: %s", manifest.Error.Message),
		).WithHint("Fix the issue and re-run the release pipeline")
	default:
		return fmt.Errorf("unexpected manifest state %q — expected 'submitted' or 'canonical-pr-open'", manifest.State)
	}

	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	ui.Info("Finalizing release %s @ %s", manifest.APIID, manifest.RequestedVersion)

	// --- Re-validation (canonical CI validates again) ---
	ui.Info("Re-validating schema in canonical repo...")

	schemaDir := filepath.Join(repoPath, manifest.CanonicalPath)
	if manifest.Validation == nil {
		manifest.Validation = &publisher.ValidationResults{}
	}

	// Create a validator instance for re-validation
	resolver := validator.NewToolchainResolver()
	v := validator.NewValidator(resolver)
	schemaFormat := validator.SchemaFormat(manifest.Format)

	// Re-run lint
	manifest.Validation.Lint = publisher.ValidationSkipped
	if info, statErr := os.Stat(schemaDir); statErr == nil && info.IsDir() {
		if lintErr := v.Lint(schemaDir, schemaFormat); lintErr != nil {
			manifest.Validation.Lint = publisher.ValidationFailed
			manifest.Fail(string(publisher.ErrCodeValidationFailed), lintErr.Error(), "finalize")
			_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
			return &publisher.ReleaseError{
				Code:    publisher.ErrCodeValidationFailed,
				Message: fmt.Sprintf("lint re-validation failed: %v", lintErr),
				Hint:    "Fix lint issues and re-submit",
			}
		}
		manifest.Validation.Lint = publisher.ValidationPassed
	}

	// Re-run breaking check (against previous tag if it exists)
	manifest.Validation.Breaking = publisher.ValidationSkipped
	if info, statErr := os.Stat(schemaDir); statErr == nil && info.IsDir() {
		finalizeTM := publisher.NewTagManager(repoPath, "")
		versions, _ := finalizeTM.ListVersionsForAPI(manifest.APIID)
		if len(versions) > 0 {
			// Find the latest previous version to check against
			lineMajor, _ := config.LineMajor(manifest.Line)
			latestPrev, _ := config.LatestVersion(versions, lineMajor)
			if latestPrev != "" && latestPrev != manifest.RequestedVersion {
				prevTag := config.DeriveTag(manifest.APIID, latestPrev)
				if breakErr := v.Breaking(schemaDir, prevTag, schemaFormat); breakErr != nil {
					manifest.Validation.Breaking = publisher.ValidationFailed
					manifest.Fail(string(publisher.ErrCodeBreakingChange), breakErr.Error(), "finalize")
					_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
					return &publisher.ReleaseError{
						Code:    publisher.ErrCodeBreakingChange,
						Message: fmt.Sprintf("breaking change detected against %s: %v", prevTag, breakErr),
						Hint:    "Create a new API line for breaking changes",
					}
				}
				manifest.Validation.Breaking = publisher.ValidationPassed
			}
		}
	}

	// Policy re-validation during finalize
	{
		schemaDir := filepath.Join(repoPath, manifest.SourcePath)
		if _, statErr := os.Stat(schemaDir); statErr == nil {
			polResult, polErr := policy.Check(cfg.Policy, schemaDir)
			if polErr != nil {
				ui.Warning("Policy re-check error: %v", polErr)
				manifest.Validation.Policy = publisher.ValidationSkipped
			} else if !polResult.Passed() {
				manifest.Validation.Policy = publisher.ValidationFailed
				manifest.Fail(string(publisher.ErrCodePolicyFailed),
					fmt.Sprintf("%d policy violation(s)", len(polResult.Violations)), "finalize")
				_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
				return &publisher.ReleaseError{
					Code:    publisher.ErrCodePolicyFailed,
					Message: fmt.Sprintf("policy check failed: %d violation(s)", len(polResult.Violations)),
				}
			} else {
				manifest.Validation.Policy = publisher.ValidationPassed
			}
		} else {
			manifest.Validation.Policy = publisher.ValidationSkipped
		}
	}

	// Transition to canonical-validated
	if err := manifest.SetState(publisher.StateCanonicalValidated); err != nil {
		return err
	}
	_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
	ui.Success("Re-validation passed")

	// --- Create canonical tag ---
	ui.Info("Creating canonical tag %s...", manifest.Tag)

	tm := publisher.NewTagManager(repoPath, "")
	exists, err := tm.TagExists(manifest.Tag)
	if err != nil {
		return fmt.Errorf("checking tag existence: %w", err)
	}

	if !exists {
		commitHash := "HEAD"
		if manifest.SourceCommit != "" {
			commitHash = manifest.SourceCommit
		}
		message := fmt.Sprintf("Release %s %s\n\nLifecycle: %s\nSource: %s/%s",
			manifest.APIID, manifest.RequestedVersion,
			manifest.Lifecycle, manifest.SourceRepo, manifest.SourcePath)
		if err := tm.CreateTag(manifest.Tag, message, commitHash); err != nil {
			manifest.Fail(string(publisher.ErrCodePushFailed), err.Error(), "finalize")
			_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
			return &publisher.ReleaseError{
				Code:    publisher.ErrCodePushFailed,
				Message: fmt.Sprintf("tag creation failed: %v", err),
			}
		}
		if err := tm.PushTag(manifest.Tag, ""); err != nil {
			ui.Warning("Tag created locally but push failed: %v", err)
		}
	} else {
		ui.Info("Tag %s already exists — skipping creation", manifest.Tag)
	}

	// Transition to canonical-released
	if err := manifest.SetState(publisher.StateCanonicalReleased); err != nil {
		return err
	}
	_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
	ui.Success("Canonical tag created")

	// --- Build release record ---
	record := publisher.NewReleaseRecord(manifest)
	record.DetectCI()

	// --- Catalog update ---
	if !skipCatalog {
		ui.Info("Updating catalog at %s...", catalogPath)
		gen := catalog.NewGenerator(catalogPath)
		cat, loadErr := gen.Load()
		if loadErr != nil {
			// Create a new catalog if it doesn't exist
			cat = &catalog.Catalog{
				Version: 1,
				Modules: []catalog.Module{},
			}
		}

		// Find or create the module entry
		found := false
		for i, mod := range cat.Modules {
			if mod.ID == manifest.APIID || mod.DisplayName() == manifest.APIID {
				cat.Modules[i].Version = manifest.RequestedVersion
				cat.Modules[i].Lifecycle = manifest.Lifecycle
				cat.Modules[i].LatestStable = updateLatestStable(cat.Modules[i].LatestStable, manifest.RequestedVersion, manifest.Lifecycle)
				cat.Modules[i].LatestPrerelease = updateLatestPrerelease(cat.Modules[i].LatestPrerelease, manifest.RequestedVersion, manifest.Lifecycle)
				found = true
				break
			}
		}
		if !found {
			mod := catalog.Module{
				ID:        manifest.APIID,
				Format:    manifest.Format,
				Domain:    manifest.Domain,
				APILine:   manifest.Line,
				Version:   manifest.RequestedVersion,
				Lifecycle: manifest.Lifecycle,
				Path:      manifest.CanonicalPath,
			}
			mod.LatestStable = updateLatestStable("", manifest.RequestedVersion, manifest.Lifecycle)
			mod.LatestPrerelease = updateLatestPrerelease("", manifest.RequestedVersion, manifest.Lifecycle)
			cat.Modules = append(cat.Modules, mod)
		}

		if saveErr := gen.Save(cat); saveErr != nil {
			ui.Warning("Catalog update failed: %v", saveErr)
			record.CatalogUpdated = false
		} else {
			record.CatalogUpdated = true
			record.CatalogPath = catalogPath
			ui.Success("Catalog updated")
		}
	}

	// --- Package publication ---
	if goCoords, ok := manifest.Languages["go"]; ok && goCoords.Module != "" {
		if !skipPackages {
			ui.Info("Recording Go module artifact: %s", goCoords.Module)
			record.AddArtifact("go-module", goCoords.Module, manifest.RequestedVersion, "published")
		} else {
			record.AddArtifact("go-module", goCoords.Module, manifest.RequestedVersion, "skipped")
		}
	}

	// Transition to package-published (terminal success)
	if err := manifest.SetState(publisher.StatePackagePublished); err != nil {
		return err
	}
	_ = publisher.WriteManifest(manifest, ".apx-release.yaml")

	// Capture canonical commit
	if commitOut, gitErr := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").Output(); gitErr == nil {
		record.CanonicalCommit = strings.TrimSpace(string(commitOut))
	}

	// Write release record
	if err := publisher.WriteReleaseRecord(record, recordPath); err != nil {
		ui.Warning("Could not write release record: %v", err)
	}

	ui.Success("✓ Release finalized successfully")
	ui.Info("")
	fmt.Print(publisher.FormatRecordReport(record))
	ui.Info("")
	ui.Info("Release record: %s", recordPath)

	return nil
}

// updateLatestStable returns the latest stable version string.
func updateLatestStable(current, version, lifecycle string) string {
	if lifecycle != "stable" && lifecycle != "" {
		return current
	}
	// For stable releases, use the newer version
	if current == "" {
		return version
	}
	sv1, err1 := config.ParseSemVer(current)
	sv2, err2 := config.ParseSemVer(version)
	if err1 != nil || err2 != nil {
		return version
	}
	if config.CompareSemVer(sv2, sv1) > 0 {
		return version
	}
	return current
}

// updateLatestPrerelease returns the latest prerelease version string.
func updateLatestPrerelease(current, version, lifecycle string) string {
	if lifecycle == "stable" || lifecycle == "" {
		return current
	}
	if current == "" {
		return version
	}
	sv1, err1 := config.ParseSemVer(current)
	sv2, err2 := config.ParseSemVer(version)
	if err1 != nil || err2 != nil {
		return version
	}
	if config.CompareSemVer(sv2, sv1) > 0 {
		return version
	}
	return current
}

// ---------------------------------------------------------------------------
// apx release history
// ---------------------------------------------------------------------------

func newReleaseHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history <api-id>",
		Short: "List all releases for an API",
		Long: `History shows all published versions for a given API ID, extracted
from git tags. Versions are sorted newest-first.

Examples:
  apx release history proto/payments/ledger/v1
  apx release history proto/payments/ledger/v1 --format json`,
		Args: cobra.ExactArgs(1),
		RunE: releaseHistoryAction,
	}
	cmd.Flags().String("format", "table", "Output format: table, json")
	return cmd
}

func releaseHistoryAction(cmd *cobra.Command, args []string) error {
	apiID := args[0]
	format, _ := cmd.Flags().GetString("format")

	if _, parseErr := config.ParseAPIID(apiID); parseErr != nil {
		return parseErr
	}

	repoPath, _ := os.Getwd()
	tm := publisher.NewTagManager(repoPath, "")

	versions, err := tm.ListVersionsForAPI(apiID)
	if err != nil {
		return fmt.Errorf("listing versions: %w", err)
	}

	if len(versions) == 0 {
		ui.Info("No releases found for %s", apiID)
		return nil
	}

	// Parse and sort versions (newest first)
	type versionEntry struct {
		Version   string `json:"version"`
		Tag       string `json:"tag"`
		Lifecycle string `json:"lifecycle"`
	}

	entries := make([]versionEntry, 0, len(versions))
	for _, v := range versions {
		lifecycle := inferLifecycleFromVersion(v)
		entries = append(entries, versionEntry{
			Version:   v,
			Tag:       config.DeriveTag(apiID, v),
			Lifecycle: lifecycle,
		})
	}

	// Sort newest first using semver comparison
	sort.Slice(entries, func(i, j int) bool {
		sv1, err1 := config.ParseSemVer(entries[i].Version)
		sv2, err2 := config.ParseSemVer(entries[j].Version)
		if err1 != nil || err2 != nil {
			return entries[i].Version > entries[j].Version
		}
		return config.CompareSemVer(sv1, sv2) > 0
	})

	if format == "json" {
		data, _ := json.MarshalIndent(struct {
			APIID    string         `json:"api_id"`
			Versions []versionEntry `json:"versions"`
		}{
			APIID:    apiID,
			Versions: entries,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Table format
	ui.Info("Release history for %s:", apiID)
	ui.Info("")
	ui.Info("  %-20s %-14s %s", "VERSION", "LIFECYCLE", "TAG")
	ui.Info("  %-20s %-14s %s", "-------", "---------", "---")
	for _, e := range entries {
		lc := e.Lifecycle
		if lc == "" {
			lc = "-"
		}
		ui.Info("  %-20s %-14s %s", e.Version, lc, e.Tag)
	}
	ui.Info("")
	ui.Info("Total: %d release(s)", len(entries))

	return nil
}

// inferLifecycleFromVersion guesses the lifecycle from version prerelease tags.
func inferLifecycleFromVersion(version string) string {
	sv, err := config.ParseSemVer(version)
	if err != nil {
		return ""
	}
	if sv.Prerelease == "" {
		return "stable"
	}
	if strings.HasPrefix(sv.Prerelease, "alpha") {
		return "experimental"
	}
	if strings.HasPrefix(sv.Prerelease, "beta") || strings.HasPrefix(sv.Prerelease, "rc") {
		return "beta"
	}
	return ""
}

// ---------------------------------------------------------------------------
// apx release promote
// ---------------------------------------------------------------------------

func newReleasePromoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "promote <api-id>",
		Short: "Promote an API to a new lifecycle state",
		Long: `Promote creates a new release that moves an API forward in its
lifecycle (e.g. beta → stable). It determines the appropriate new
version and validates the lifecycle transition.

The promotion creates a new release manifest ready for submit.

Examples:
  apx release promote proto/payments/ledger/v1 --to stable --version v1.0.0
  apx release promote proto/payments/ledger/v1 --to deprecated`,
		Args: cobra.ExactArgs(1),
		RunE: releasePromoteAction,
	}
	cmd.Flags().String("to", "", "Target lifecycle (beta, stable, deprecated, sunset)")
	cmd.Flags().String("version", "", "Version for the promoted release (required for stable promotion)")
	cmd.Flags().String("canonical-repo", "", "Canonical repository URL")
	cmd.Flags().Bool("force", false, "Override lifecycle checks")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func releasePromoteAction(cmd *cobra.Command, args []string) error {
	apiID := args[0]
	targetLifecycle, _ := cmd.Flags().GetString("to")
	version, _ := cmd.Flags().GetString("version")
	canonicalRepo, _ := cmd.Flags().GetString("canonical-repo")
	force, _ := cmd.Flags().GetBool("force")

	cfg, err := loadConfig(cmd)
	if err != nil {
		ui.Warning("Could not load config for policy check: %v", err)
		cfg = &config.Config{}
	}

	// Validate target lifecycle
	if err := config.ValidateLifecycle(targetLifecycle); err != nil {
		return err
	}

	// Parse API ID
	if _, parseErr := config.ParseAPIID(apiID); parseErr != nil {
		return parseErr
	}

	// Determine current lifecycle
	currentLifecycle := resolveCurrentLifecycle(cmd, apiID)
	if currentLifecycle == "" {
		// Try to infer from latest tag
		repoPath, _ := os.Getwd()
		tm := publisher.NewTagManager(repoPath, "")
		versions, _ := tm.ListVersionsForAPI(apiID)
		if len(versions) > 0 {
			lineMajor, _ := config.LineMajor(config.ParseLineFromID(apiID))
			latest, _ := config.LatestVersion(versions, lineMajor)
			currentLifecycle = inferLifecycleFromVersion(latest)
		}
	}

	ui.Info("Promoting %s: %s → %s", apiID, currentLifecycleLabel(currentLifecycle), targetLifecycle)

	// Validate transition
	if !force {
		if err := config.ValidateLifecycleTransition(currentLifecycle, targetLifecycle); err != nil {
			return &publisher.ReleaseError{
				Code:    publisher.ErrCodeIllegalTransition,
				Message: err.Error(),
				Hint:    "Use --force to override lifecycle checks",
			}
		}

		// v0 line policy enforcement for promote target
		line := config.ParseLineFromID(apiID)
		if config.IsV0Line(line) {
			if err := config.ValidateV0Lifecycle(targetLifecycle); err != nil {
				return &publisher.ReleaseError{
					Code:    publisher.ErrCodeLifecycleMismatch,
					Message: err.Error(),
					Hint:    "v0 lines cannot be promoted to stable; create a v1 line instead",
				}
			}
		}
	}

	// Determine version for the promotion
	if version == "" {
		// Auto-derive version based on lifecycle
		repoPath, _ := os.Getwd()
		tm := publisher.NewTagManager(repoPath, "")
		versions, _ := tm.ListVersionsForAPI(apiID)
		promoteLineMajor, _ := config.LineMajor(config.ParseLineFromID(apiID))

		if targetLifecycle == "stable" {
			// For stable promotion, strip the prerelease from latest pre-release version
			latest, _ := config.LatestVersion(versions, promoteLineMajor)
			if latest != "" {
				sv, err := config.ParseSemVer(latest)
				if err == nil && sv.Prerelease != "" {
					version = fmt.Sprintf("v%d.%d.%d", sv.Major, sv.Minor, sv.Patch)
				} else if err == nil {
					// Already stable — bump patch
					version = fmt.Sprintf("v%d.%d.%d", sv.Major, sv.Minor, sv.Patch+1)
				}
			}
		} else if targetLifecycle == "beta" {
			latest, _ := config.LatestVersion(versions, promoteLineMajor)
			if latest != "" {
				sv, err := config.ParseSemVer(latest)
				if err == nil {
					version = fmt.Sprintf("v%d.%d.%d-beta.1", sv.Major, sv.Minor, sv.Patch)
				}
			}
		}

		if version == "" {
			return fmt.Errorf("cannot auto-determine version for %s promotion; use --version", targetLifecycle)
		}
		ui.Info("Auto-derived version: %s", version)
	}

	// Validate version-lifecycle compatibility
	if !force {
		if err := config.ValidateVersionLifecycle(version, targetLifecycle); err != nil {
			return &publisher.ReleaseError{
				Code:    publisher.ErrCodeLifecycleMismatch,
				Message: err.Error(),
				Hint:    "Use --force to override or choose a compatible version",
			}
		}
	}

	// Validate version-line compatibility
	if err := config.ValidateVersionLine(version, config.ParseLineFromID(apiID)); err != nil {
		return &publisher.ReleaseError{
			Code:    publisher.ErrCodeVersionLineMismatch,
			Message: err.Error(),
		}
	}

	// Resolve source repo
	sourceRepo := canonicalRepo
	if sourceRepo == "" {
		sourceRepo = resolveSourceRepo(cmd)
		if sourceRepo == "github.com/<org>/<repo>" {
			return publisher.NewReleaseError(
				publisher.ErrCodeMissingConfig,
				"cannot determine canonical repo; use --canonical-repo or configure org/repo in apx.yaml",
			)
		}
	}

	// Build identity
	api, source, _, err := config.BuildIdentityBlock(apiID, sourceRepo, targetLifecycle, version)
	if err != nil {
		return err
	}

	langs, err := language.DeriveAllCoords(language.DerivationContext{
		SourceRepo: sourceRepo,
		ImportRoot: resolveImportRoot(cmd),
		Org:        resolveOrg(cmd),
		API:        api,
	})
	if err != nil {
		return err
	}

	// Create manifest
	manifest := publisher.NewManifest(api, source, langs, version, sourceRepo)
	manifest.Lifecycle = targetLifecycle

	// Skip to prepared (promotion is a lifecycle change, not a content change)
	manifest.Validation = &publisher.ValidationResults{
		Lint:     publisher.ValidationSkipped,
		Breaking: publisher.ValidationSkipped,
		Policy:   publisher.ValidationSkipped,
	}

	// Policy check for promote
	{
		promoteRepoPath, _ := os.Getwd()
		schemaDir := filepath.Join(promoteRepoPath, source.Path)
		if _, statErr := os.Stat(schemaDir); statErr == nil {
			polResult, polErr := policy.Check(cfg.Policy, schemaDir)
			if polErr != nil {
				ui.Warning("Policy check error: %v", polErr)
			} else if !polResult.Passed() {
				manifest.Validation.Policy = publisher.ValidationFailed
				for _, v := range polResult.Violations {
					ui.Error("[%s] %s", v.Rule, v.Message)
				}
				manifest.Fail(string(publisher.ErrCodePolicyFailed),
					fmt.Sprintf("%d policy violation(s)", len(polResult.Violations)), "promote")
				_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
				return &publisher.ReleaseError{
					Code:    publisher.ErrCodePolicyFailed,
					Message: fmt.Sprintf("policy check failed: %d violation(s)", len(polResult.Violations)),
				}
			} else {
				manifest.Validation.Policy = publisher.ValidationPassed
			}
		}
	}

	// Capture source commit
	promoteRepoPath, _ := os.Getwd()
	if commitOut, gitErr := exec.Command("git", "-C", promoteRepoPath, "rev-parse", "HEAD").Output(); gitErr == nil {
		manifest.SourceCommit = strings.TrimSpace(string(commitOut))
	}

	if err := manifest.SetState(publisher.StateValidated); err != nil {
		return err
	}
	if err := manifest.SetState(publisher.StateVersionSelected); err != nil {
		return err
	}
	if err := manifest.SetState(publisher.StatePrepared); err != nil {
		return err
	}

	// Write manifest
	if err := publisher.WriteManifest(manifest, ".apx-release.yaml"); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	ui.Success("✓ Promotion prepared: %s → %s @ %s", currentLifecycleLabel(currentLifecycle), targetLifecycle, version)
	ui.Info("Tag:         %s", manifest.Tag)
	ui.Info("Manifest:    .apx-release.yaml")
	ui.Info("")
	ui.Info("Next step:   apx release submit")

	return nil
}

// currentLifecycleLabel returns a display label for the current lifecycle.
func currentLifecycleLabel(lc string) string {
	if lc == "" {
		return "(unknown)"
	}
	return lc
}

// ---------------------------------------------------------------------------
// Dependents
// ---------------------------------------------------------------------------

// FindDependents returns a list of API IDs from the catalog that depend on
// the given API ID. It searches the dependency lock files or catalog cross-
// references found in the repository.
func FindDependents(repoPath, apiID, catalogPath string) ([]string, error) {
	gen := catalog.NewGenerator(catalogPath)
	cat, err := gen.Load()
	if err != nil {
		return nil, fmt.Errorf("loading catalog: %w", err)
	}

	// Search for modules that list apiID in their dependencies.
	// This is a heuristic based on the catalog tags.
	var dependents []string
	for _, mod := range cat.Modules {
		if mod.ID == apiID {
			continue
		}
		// Check if any tag references the target API
		for _, tag := range mod.Tags {
			if tag == "depends:"+apiID || strings.HasPrefix(tag, "depends:"+apiID+"/") {
				dependents = append(dependents, mod.ID)
				break
			}
		}
	}

	// Also search for lock file references
	lockDeps, _ := findLockFileDependents(repoPath, apiID)
	for _, d := range lockDeps {
		if !containsString(dependents, d) {
			dependents = append(dependents, d)
		}
	}

	return dependents, nil
}

// findLockFileDependents scans apx.lock files in the repo for references
// to the given API ID.
func findLockFileDependents(repoPath, apiID string) ([]string, error) {
	lockPath := filepath.Join(repoPath, "apx.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, nil // No lock file — not an error
	}

	var dependents []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, apiID) {
			// Extract the module that depends on apiID
			// Lock files list dependencies as keys
			if strings.Contains(line, ":") {
				parts := strings.SplitN(line, ":", 2)
				dep := strings.TrimSpace(parts[0])
				if dep != "" && dep != apiID && !containsString(dependents, dep) {
					dependents = append(dependents, dep)
				}
			}
		}
	}
	return dependents, nil
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
