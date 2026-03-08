package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/publisher"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/infobloxopen/apx/internal/validator"
	"github.com/spf13/cobra"
)

func newPublishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish [api-id]",
		Short: "Publish a module to canonical repository",
		Long: `Publish an API module to the canonical repository.

When an API ID is provided (e.g. proto/payments/ledger/v1), APX derives
the canonical source path, language coordinates, and tag automatically.

Examples:
  apx publish proto/payments/ledger/v1 --version v1.0.0-beta.1 --lifecycle beta
  apx publish --module-path proto/payments/ledger/v1 --canonical-repo github.com/acme/apis --version v1.0.0`,
		Args: cobra.MaximumNArgs(1),
		RunE: publishAction,
	}
	cmd.Flags().String("module-path", "", "Path to the module to publish (legacy; prefer positional api-id)")
	cmd.Flags().String("canonical-repo", "", "Canonical repository URL (auto-derived from apx.yaml if available)")
	cmd.Flags().String("version", "", "Version to publish (e.g. v1.0.0-beta.1)")
	cmd.Flags().String("lifecycle", "", "Lifecycle state (experimental, beta, stable, deprecated, sunset)")
	cmd.Flags().Bool("dry-run", false, "Show what would be published without actually publishing")
	cmd.Flags().Bool("strict", false, "Make go_package mismatches an error instead of a warning")
	cmd.Flags().Bool("skip-gomod", false, "Skip go.mod generation and validation")
	return cmd
}

// publishOpts holds all options for the identity-based publish flow.
type publishOpts struct {
	APIID         string
	Version       string
	Lifecycle     string
	CanonicalRepo string
	DryRun        bool
	Strict        bool
	SkipGomod     bool
}

func publishAction(cmd *cobra.Command, args []string) error {
	modulePath, _ := cmd.Flags().GetString("module-path")
	canonicalRepo, _ := cmd.Flags().GetString("canonical-repo")
	version, _ := cmd.Flags().GetString("version")
	lifecycle, _ := cmd.Flags().GetString("lifecycle")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	strict, _ := cmd.Flags().GetBool("strict")
	skipGomod, _ := cmd.Flags().GetBool("skip-gomod")

	// Support positional API ID arg
	var apiID string
	if len(args) == 1 {
		apiID = args[0]
	}

	// If we have an API ID, use the identity model
	if apiID != "" {
		return publishWithIdentity(cmd, publishOpts{
			APIID:         apiID,
			Version:       version,
			Lifecycle:     lifecycle,
			CanonicalRepo: canonicalRepo,
			DryRun:        dryRun,
			Strict:        strict,
			SkipGomod:     skipGomod,
		})
	}

	// Legacy publish path: require --module-path and --canonical-repo
	if modulePath == "" {
		return fmt.Errorf("either provide an API ID as argument or use --module-path flag")
	}
	if canonicalRepo == "" {
		return fmt.Errorf("--canonical-repo is required when using --module-path")
	}

	return publishLegacy(modulePath, canonicalRepo, version, dryRun)
}

func publishWithIdentity(cmd *cobra.Command, opts publishOpts) error {
	if opts.Version == "" {
		return publisher.NewPublishError(publisher.ErrCodeInvalidVersion,
			"--version is required (e.g. v1.0.0-beta.1)")
	}

	// Validate lifecycle if provided
	if opts.Lifecycle != "" {
		if err := config.ValidateLifecycle(opts.Lifecycle); err != nil {
			return err
		}
		// Validate version-lifecycle compatibility
		if err := config.ValidateVersionLifecycle(opts.Version, opts.Lifecycle); err != nil {
			return &publisher.PublishError{
				Code:    publisher.ErrCodeLifecycleMismatch,
				Message: err.Error(),
				Hint:    "Use 'apx release prepare --force' to override",
			}
		}
		if config.LifecycleRequiresWarning(opts.Lifecycle) {
			ui.Warning("Publishing under deprecated lifecycle — consumers should migrate")
		}
	}

	// Validate version-line compatibility
	if err := config.ValidateVersionLine(opts.Version, config.ParseLineFromID(opts.APIID)); err != nil {
		return &publisher.PublishError{
			Code:    publisher.ErrCodeVersionLineMismatch,
			Message: err.Error(),
			Hint:    "Ensure version major matches the API line (e.g. v1.x.x for /v1)",
		}
	}

	// Resolve source repo from flag or config
	sourceRepo := opts.CanonicalRepo
	if sourceRepo == "" {
		sourceRepo = resolveSourceRepo(cmd)
		if sourceRepo == "github.com/<org>/<repo>" {
			return publisher.NewPublishError(publisher.ErrCodeMissingConfig,
				"cannot determine canonical repo; use --canonical-repo or configure org/repo in apx.yaml")
		}
	}

	api, source, _, langs, err := config.BuildIdentityBlock(opts.APIID, sourceRepo, opts.Lifecycle, opts.Version)
	if err != nil {
		return err
	}

	tag := config.DeriveTag(opts.APIID, opts.Version)

	// Build release manifest for tracking
	manifest := publisher.NewManifest(api, source, langs, opts.Version, sourceRepo)

	// -------------------------------------------------------------------
	// Phase 4: go_package validation (proto format only)
	// -------------------------------------------------------------------
	if api.Format == "proto" {
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
			if expectedImport != "" {
				for _, pf := range protoFiles {
					importPath, _, extractErr := validator.ExtractGoPackage(pf)
					if extractErr != nil {
						ui.Warning("Could not extract go_package from %s: %v", pf, extractErr)
						continue
					}
					if importPath == "" {
						continue // no go_package option — skip
					}
					if valErr := config.ValidateGoPackage(importPath, expectedImport); valErr != nil {
						relPath, _ := filepath.Rel(repoPath, pf)
						if relPath == "" {
							relPath = pf
						}
						if opts.Strict {
							manifest.Fail(string(publisher.ErrCodeGoPackageMismatch), valErr.Error(), "publish")
							return &publisher.PublishError{
								Code:    publisher.ErrCodeGoPackageMismatch,
								Message: fmt.Sprintf("%s: %v", relPath, valErr),
							}
						}
						ui.Warning("%s: %v", relPath, valErr)
					}
				}
			}
		}
	}

	// -------------------------------------------------------------------
	// Phase 5: go.mod generation / validation
	// -------------------------------------------------------------------
	if !opts.SkipGomod {
		repoPath, _ := os.Getwd()
		goModDir := config.DeriveGoModDir(api)
		goModPath := filepath.Join(repoPath, goModDir, "go.mod")

		goModulePath := ""
		if goCoords, ok := langs["go"]; ok {
			goModulePath = goCoords.Module
		}

		if goModulePath != "" {
			if existing, readErr := os.ReadFile(goModPath); readErr == nil {
				// go.mod exists — validate module directive
				existingMod, parseErr := publisher.ParseGoModModule(existing)
				if parseErr != nil {
					manifest.Fail(string(publisher.ErrCodeGoModMismatch), parseErr.Error(), "publish")
					return &publisher.PublishError{
						Code:    publisher.ErrCodeGoModMismatch,
						Message: fmt.Sprintf("invalid go.mod at %s: %v", goModDir, parseErr),
					}
				}
				if existingMod != goModulePath {
					manifest.Fail(string(publisher.ErrCodeGoModMismatch),
						fmt.Sprintf("got %q, expected %q", existingMod, goModulePath), "publish")
					return &publisher.PublishError{
						Code:    publisher.ErrCodeGoModMismatch,
						Message: fmt.Sprintf("go.mod module mismatch at %s: got %q, expected %q", goModDir, existingMod, goModulePath),
					}
				}
				ui.Info("go.mod validated: %s", goModDir)
			} else if os.IsNotExist(readErr) {
				// go.mod missing — generate minimal go.mod
				content, genErr := publisher.GenerateGoMod(goModulePath, "1.21")
				if genErr != nil {
					return fmt.Errorf("generating go.mod: %w", genErr)
				}
				if opts.DryRun {
					ui.Info("Would generate go.mod at %s", goModDir)
					ui.Info("  module %s", goModulePath)
				} else {
					if mkErr := os.MkdirAll(filepath.Join(repoPath, goModDir), 0o755); mkErr != nil {
						return fmt.Errorf("creating go.mod directory: %w", mkErr)
					}
					if writeErr := os.WriteFile(goModPath, content, 0o644); writeErr != nil {
						return fmt.Errorf("writing go.mod: %w", writeErr)
					}
					ui.Info("Generated go.mod at %s", goModDir)
				}
			} else {
				return fmt.Errorf("checking go.mod at %s: %w", goModDir, readErr)
			}
		}
	}

	if opts.DryRun {
		ui.Info("Dry-run mode: showing what would be published")
		ui.Info("")
		fmt.Print(publisher.FormatManifestReport(manifest))
		ui.Info("")
		ui.Success("Would publish module successfully")
		ui.Info("")
		ui.Info("Tip: use 'apx release prepare' + 'apx release submit' for full state tracking")
		return nil
	}

	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return publisher.NewPublishError(publisher.ErrCodeNotGitRepo,
			"not in a git repository (no .git directory found)")
	}

	ui.Info("Publishing API: %s", opts.APIID)
	ui.Info("Version: %s", opts.Version)
	if opts.Lifecycle != "" {
		ui.Info("Lifecycle: %s", opts.Lifecycle)
	}
	ui.Info("Source: %s/%s", source.Repo, source.Path)
	if goCoords, ok := langs["go"]; ok {
		ui.Info("Go module: %s", goCoords.Module)
		ui.Info("Go import: %s", goCoords.Import)
	}
	ui.Info("Tag: %s", tag)

	// -------------------------------------------------------------------
	// PR-based publish: clone canonical, copy files, push branch, open PR
	// -------------------------------------------------------------------
	canonicalNWO, nwoErr := publisher.ParseCanonicalNWO(sourceRepo)
	if nwoErr != nil {
		return &publisher.PublishError{
			Code:    publisher.ErrCodePRCreationFailed,
			Message: nwoErr.Error(),
			Hint:    "Provide --canonical-repo as github.com/<owner>/<repo>",
		}
	}

	localModuleDir := filepath.Join(repoPath, source.Path)
	if info, statErr := os.Stat(localModuleDir); statErr != nil || !info.IsDir() {
		return publisher.NewPublishError(publisher.ErrCodePRCreationFailed,
			fmt.Sprintf("module directory not found: %s", source.Path))
	}

	goModulePath := ""
	if goCoords, ok := langs["go"]; ok {
		goModulePath = goCoords.Module
	}

	ui.Info("Creating pull request on %s …", canonicalNWO)
	prResp, prErr := publisher.PublishModuleWithPR(
		localModuleDir, canonicalNWO, source.Path,
		opts.APIID, opts.Version, goModulePath,
	)
	if prErr != nil {
		manifest.Fail(string(publisher.ErrCodePRCreationFailed), prErr.Error(), "publish")
		return &publisher.PublishError{
			Code:    publisher.ErrCodePRCreationFailed,
			Message: prErr.Error(),
			Hint:    "Ensure gh is authenticated and you have push access to the canonical repo",
		}
	}

	_ = manifest.SetState(publisher.StateSubmitted)
	ui.Success("✓ Pull request created")
	if prResp.HTMLURL != "" {
		ui.Info("PR: %s", prResp.HTMLURL)
	}
	if prResp.Number > 0 {
		ui.Info("PR #%d", prResp.Number)
	}
	return nil
}

func publishLegacy(modulePath, canonicalRepo, version string, dryRun bool) error {
	absModulePath, err := filepath.Abs(modulePath)
	if err != nil {
		return fmt.Errorf("failed to resolve module path: %w", err)
	}

	if _, err := os.Stat(absModulePath); os.IsNotExist(err) {
		return fmt.Errorf("module path does not exist: %s", modulePath)
	}

	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not in a git repository (no .git directory found)")
	}

	if dryRun {
		ui.Info("Dry-run mode: showing what would be published")
		ui.Info("Module path: %s", modulePath)
		ui.Info("Canonical repo: %s", canonicalRepo)
		if version != "" {
			ui.Info("Version: %s", version)
		} else {
			ui.Info("Version: (would auto-detect from git tags)")
		}
		ui.Success("Would publish module successfully")
		return nil
	}

	if version == "" {
		return fmt.Errorf("version auto-detection not yet implemented, please specify --version")
	}

	ui.Info("Publishing module: %s", modulePath)
	ui.Info("Target repository: %s", canonicalRepo)
	ui.Info("Version: %s", version)

	// PR-based publish
	canonicalNWO, nwoErr := publisher.ParseCanonicalNWO(canonicalRepo)
	if nwoErr != nil {
		return fmt.Errorf("cannot parse canonical repo: %w", nwoErr)
	}

	apiID := modulePath // best-effort label for legacy path
	ui.Info("Creating pull request on %s …", canonicalNWO)
	prResp, prErr := publisher.PublishModuleWithPR(
		absModulePath, canonicalNWO, modulePath,
		apiID, version, "",
	)
	if prErr != nil {
		return fmt.Errorf("PR creation failed: %w", prErr)
	}
	ui.Success("✓ Pull request created")
	if prResp.HTMLURL != "" {
		ui.Info("PR: %s", prResp.HTMLURL)
	}
	return nil
}
