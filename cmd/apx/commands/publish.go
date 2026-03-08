package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/publisher"
	"github.com/infobloxopen/apx/internal/ui"
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
	cmd.Flags().Bool("create-pr", false, "Create a pull request instead of pushing directly")
	return cmd
}

func publishAction(cmd *cobra.Command, args []string) error {
	modulePath, _ := cmd.Flags().GetString("module-path")
	canonicalRepo, _ := cmd.Flags().GetString("canonical-repo")
	version, _ := cmd.Flags().GetString("version")
	lifecycle, _ := cmd.Flags().GetString("lifecycle")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	createPR, _ := cmd.Flags().GetBool("create-pr")

	// Support positional API ID arg
	var apiID string
	if len(args) == 1 {
		apiID = args[0]
	}

	// If we have an API ID, use the identity model
	if apiID != "" {
		return publishWithIdentity(cmd, apiID, version, lifecycle, canonicalRepo, dryRun, createPR)
	}

	// Legacy publish path: require --module-path and --canonical-repo
	if modulePath == "" {
		return fmt.Errorf("either provide an API ID as argument or use --module-path flag")
	}
	if canonicalRepo == "" {
		return fmt.Errorf("--canonical-repo is required when using --module-path")
	}

	return publishLegacy(modulePath, canonicalRepo, version, dryRun, createPR)
}

func publishWithIdentity(cmd *cobra.Command, apiID, version, lifecycle, canonicalRepo string, dryRun, createPR bool) error {
	if version == "" {
		return fmt.Errorf("--version is required (e.g. v1.0.0-beta.1)")
	}

	// Validate lifecycle if provided
	if lifecycle != "" {
		if err := config.ValidateLifecycle(lifecycle); err != nil {
			return err
		}
	}

	// Resolve source repo from flag or config
	sourceRepo := canonicalRepo
	if sourceRepo == "" {
		sourceRepo = resolveSourceRepo(cmd)
		if sourceRepo == "github.com/<org>/<repo>" {
			return fmt.Errorf("cannot determine canonical repo; use --canonical-repo or configure org/repo in apx.yaml")
		}
	}

	api, source, release, langs, err := config.BuildIdentityBlock(apiID, sourceRepo, lifecycle, version)
	if err != nil {
		return err
	}

	tag := config.DeriveTag(apiID, version)

	if dryRun {
		ui.Info("Dry-run mode: showing what would be published")
		ui.Info("")
		report := config.FormatIdentityReport(api, source, release, langs)
		fmt.Print(report)
		ui.Info("Tag:        %s", tag)
		ui.Info("")
		ui.Success("Would publish module successfully")
		return nil
	}

	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not in a git repository (no .git directory found)")
	}

	subtreePublisher := publisher.NewSubtreePublisher(repoPath)

	ui.Info("Publishing API: %s", apiID)
	ui.Info("Version: %s", version)
	if lifecycle != "" {
		ui.Info("Lifecycle: %s", lifecycle)
	}
	ui.Info("Source: %s/%s", source.Repo, source.Path)
	if goCoords, ok := langs["go"]; ok {
		ui.Info("Go module: %s", goCoords.Module)
		ui.Info("Go import: %s", goCoords.Import)
	}
	ui.Info("Tag: %s", tag)

	commitHash, err := subtreePublisher.PublishModule(source.Path, sourceRepo, version)
	if err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}

	ui.Success("\u2713 Module published successfully")
	ui.Info("Commit: %s", commitHash)
	ui.Info("Tag: %s", tag)

	if createPR {
		ui.Info("Creating pull request...")
		return fmt.Errorf("PR creation not yet implemented")
	}

	return nil
}

func publishLegacy(modulePath, canonicalRepo, version string, dryRun, createPR bool) error {
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

	subtreePublisher := publisher.NewSubtreePublisher(repoPath)

	ui.Info("Publishing module: %s", modulePath)
	ui.Info("Target repository: %s", canonicalRepo)
	ui.Info("Version: %s", version)

	commitHash, err := subtreePublisher.PublishModule(modulePath, canonicalRepo, version)
	if err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}

	ui.Success("\u2713 Module published successfully")
	ui.Info("Commit hash: %s", commitHash)

	if createPR {
		ui.Info("Creating pull request...")
		return fmt.Errorf("PR creation not yet implemented")
	}

	return nil
}
