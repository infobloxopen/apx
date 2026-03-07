package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/publisher"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newPublishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish a module to canonical repository",
		RunE:  publishAction,
	}
	cmd.Flags().String("module-path", "", "Path to the module to publish")
	cmd.Flags().String("canonical-repo", "", "Canonical repository URL")
	cmd.Flags().String("version", "", "Version to publish (auto-detected from git tags if not specified)")
	cmd.Flags().Bool("dry-run", false, "Show what would be published without actually publishing")
	cmd.Flags().Bool("create-pr", false, "Create a pull request instead of pushing directly")
	_ = cmd.MarkFlagRequired("module-path")
	_ = cmd.MarkFlagRequired("canonical-repo")
	return cmd
}

func publishAction(cmd *cobra.Command, args []string) error {
	modulePath, _ := cmd.Flags().GetString("module-path")
	canonicalRepo, _ := cmd.Flags().GetString("canonical-repo")
	version, _ := cmd.Flags().GetString("version")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	createPR, _ := cmd.Flags().GetBool("create-pr")

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

	subtreePublisher := publisher.NewSubtreePublisher(repoPath)

	if version == "" {
		return fmt.Errorf("version auto-detection not yet implemented, please specify --version")
	}

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
