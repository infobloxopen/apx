package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/publisher"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/infobloxopen/apx/internal/validator"
	"github.com/spf13/cobra"
)

func newReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Manage releases through the release state machine",
		Long: `Release commands let you prepare, submit, and inspect releases.

A release progresses through explicit states:
  draft → validated → version-selected → prepared → submitted → ...

Use 'apx release prepare' to validate and build a release manifest.
Use 'apx release submit' to push the release to the canonical repo.
Use 'apx release inspect' to view the current release state.`,
	}
	cmd.AddCommand(
		newReleasePrepareCmd(),
		newReleaseSubmitCmd(),
		newReleaseInspectCmd(),
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
			return &publisher.PublishError{
				Code:    publisher.ErrCodeLifecycleMismatch,
				Message: err.Error(),
				Hint:    "Use --force to override lifecycle checks",
			}
		}
	}
	if config.LifecycleRequiresWarning(lifecycle) {
		ui.Warning("Publishing under deprecated lifecycle — consumers should migrate")
	}

	// Resolve source repo
	sourceRepo := canonicalRepo
	if sourceRepo == "" {
		sourceRepo = resolveSourceRepo(cmd)
		if sourceRepo == "github.com/<org>/<repo>" {
			return publisher.NewPublishError(
				publisher.ErrCodeMissingConfig,
				"cannot determine canonical repo; use --canonical-repo or configure org/repo in apx.yaml",
			)
		}
	}

	// Build identity
	api, source, release, langs, err := config.BuildIdentityBlock(apiID, sourceRepo, lifecycle, version)
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
							return &publisher.PublishError{
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
					return &publisher.PublishError{
						Code:    publisher.ErrCodeGoModMismatch,
						Message: fmt.Sprintf("invalid go.mod at %s: %v", goModDir, parseErr),
					}
				}
				if existingMod != goModulePath {
					manifest.Validation.GoMod = publisher.ValidationFailed
					manifest.Fail(string(publisher.ErrCodeGoModMismatch),
						fmt.Sprintf("got %q, expected %q", existingMod, goModulePath), "prepare")
					_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
					return &publisher.PublishError{
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
				return &publisher.PublishError{
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

	// Write manifest
	if err := publisher.WriteManifest(manifest, ".apx-release.yaml"); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	// Print summary
	ui.Success("Release prepared successfully")
	ui.Info("")
	report := config.FormatIdentityReport(api, source, release, langs)
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
		Long: `Submit pushes the prepared release (from 'apx release prepare') to
the canonical repository. It reads .apx-release.yaml and either
pushes via subtree or creates a pull request.

This operation is idempotent: if the same version with the same
content has already been published, it will report success without
changing anything.

Examples:
  apx release submit
  apx release submit --create-pr`,
		RunE: releaseSubmitAction,
	}
	cmd.Flags().Bool("create-pr", false, "Create a pull request instead of pushing directly")
	cmd.Flags().Bool("dry-run", false, "Show what would be submitted without actually doing it")
	return cmd
}

func releaseSubmitAction(cmd *cobra.Command, _ []string) error {
	createPR, _ := cmd.Flags().GetBool("create-pr")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Read manifest
	manifest, err := publisher.ReadManifest(".apx-release.yaml")
	if err != nil {
		return publisher.NewPublishError(
			publisher.ErrCodeMissingConfig,
			"no release manifest found — run 'apx release prepare' first",
		)
	}

	// Verify state
	if manifest.State != publisher.StatePrepared {
		if manifest.State == publisher.StatePackagePublished {
			ui.Success("Release already published — nothing to do")
			return nil
		}
		if manifest.State == publisher.StateFailed {
			return publisher.NewPublishError(
				publisher.ErrCodeValidationFailed,
				fmt.Sprintf("release is in failed state: %s", manifest.Error.Message),
			).WithHint("Fix the issue and re-run 'apx release prepare'")
		}
		return fmt.Errorf("unexpected manifest state %q — expected 'prepared'", manifest.State)
	}

	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return publisher.NewPublishError(publisher.ErrCodeNotGitRepo, "not in a git repository")
	}

	// Generate go.mod if needed
	goModulePath := manifest.GoModule
	if goModulePath != "" {
		api := &config.APIIdentity{
			ID: manifest.APIID, Format: manifest.Format,
			Domain: manifest.Domain, Name: manifest.Name, Line: manifest.Line,
		}
		goModDir := config.DeriveGoModDir(api)
		goModPath := filepath.Join(repoPath, goModDir, "go.mod")

		if _, readErr := os.ReadFile(goModPath); os.IsNotExist(readErr) {
			content, genErr := publisher.GenerateGoMod(goModulePath, "1.21")
			if genErr != nil {
				manifest.Fail(string(publisher.ErrCodeGoModMismatch), genErr.Error(), "submit")
				_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
				return fmt.Errorf("generating go.mod: %w", genErr)
			}
			if dryRun {
				ui.Info("Would generate go.mod at %s", goModDir)
			} else {
				if mkErr := os.MkdirAll(filepath.Join(repoPath, goModDir), 0o755); mkErr != nil {
					return fmt.Errorf("creating go.mod directory: %w", mkErr)
				}
				if writeErr := os.WriteFile(goModPath, content, 0o644); writeErr != nil {
					return fmt.Errorf("writing go.mod: %w", writeErr)
				}
				ui.Info("Generated go.mod at %s", goModDir)
			}
		}
	}

	if dryRun {
		ui.Info("Dry-run mode: showing what would be submitted")
		ui.Info("")
		fmt.Print(publisher.FormatManifestReport(manifest))
		ui.Info("")
		ui.Success("Would submit release successfully")
		return nil
	}

	// Publish via subtree
	ui.Info("Submitting release %s @ %s", manifest.APIID, manifest.RequestedVersion)

	subtreePublisher := publisher.NewSubtreePublisher(repoPath)
	commitHash, err := subtreePublisher.PublishModule(
		manifest.SourcePath, manifest.CanonicalRepo, manifest.RequestedVersion,
	)
	if err != nil {
		manifest.Fail(string(publisher.ErrCodeSubtreeFailed), err.Error(), "submit")
		_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
		return &publisher.PublishError{
			Code:    publisher.ErrCodeSubtreeFailed,
			Message: fmt.Sprintf("subtree publish failed: %v", err),
			Hint:    "Check git status and try 'apx release submit' again",
		}
	}

	if err := manifest.SetState(publisher.StateSubmitted); err != nil {
		return err
	}

	manifest.SourceCommit = commitHash
	_ = publisher.WriteManifest(manifest, ".apx-release.yaml")

	ui.Success("✓ Release submitted successfully")
	ui.Info("Commit:  %s", commitHash)
	ui.Info("Tag:     %s", manifest.Tag)

	if createPR {
		if err := manifest.SetState(publisher.StateCanonicalPROpen); err != nil {
			return err
		}
		_ = publisher.WriteManifest(manifest, ".apx-release.yaml")
		ui.Info("Creating pull request...")
		return publisher.NewPublishError(publisher.ErrCodePushFailed,
			"PR creation not yet implemented").WithHint("Push directly or implement PR flow")
	}

	return nil
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
	cmd.Flags().Bool("json", false, "Output in JSON format")
	return cmd
}

func releaseInspectAction(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")

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
