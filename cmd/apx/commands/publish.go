package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/publisher"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/urfave/cli/v2"
)

// PublishCommand returns the publish command
func PublishCommand() *cli.Command {
	return &cli.Command{
		Name:      "publish",
		Usage:     "Publish a module to canonical repository",
		ArgsUsage: " ",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "module-path",
				Usage:    "Path to the module to publish (e.g., internal/apis/proto/payments/ledger/v1)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "canonical-repo",
				Usage:    "Canonical repository URL (e.g., github.com/myorg/apis)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "version",
				Usage: "Version to publish (auto-detected from git tags if not specified)",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Show what would be published without actually publishing",
			},
			&cli.BoolFlag{
				Name:  "create-pr",
				Usage: "Create a pull request instead of pushing directly",
			},
		},
		Action: publishAction,
	}
}

func publishAction(c *cli.Context) error {
	modulePath := c.String("module-path")
	canonicalRepo := c.String("canonical-repo")
	version := c.String("version")
	dryRun := c.Bool("dry-run")
	createPR := c.Bool("create-pr")

	// Validate module path exists
	absModulePath, err := filepath.Abs(modulePath)
	if err != nil {
		return fmt.Errorf("failed to resolve module path: %w", err)
	}

	if _, err := os.Stat(absModulePath); os.IsNotExist(err) {
		return fmt.Errorf("module path does not exist: %s", modulePath)
	}

	// Get current working directory (repo root)
	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Verify we're in a git repository
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

	// Create subtree publisher
	subtreePublisher := publisher.NewSubtreePublisher(repoPath)

	// TODO: Auto-detect version from git tags if not specified
	if version == "" {
		return fmt.Errorf("version auto-detection not yet implemented, please specify --version")
	}

	// Perform subtree split and publish
	ui.Info("Publishing module: %s", modulePath)
	ui.Info("Target repository: %s", canonicalRepo)
	ui.Info("Version: %s", version)

	commitHash, err := subtreePublisher.PublishModule(modulePath, canonicalRepo, version)
	if err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}

	ui.Success("✓ Module published successfully")
	ui.Info("Commit hash: %s", commitHash)

	if createPR {
		ui.Info("Creating pull request...")
		// TODO: Implement PR creation
		return fmt.Errorf("PR creation not yet implemented")
	}

	return nil
}
