package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/detector"
	"github.com/infobloxopen/apx/internal/schema/templates"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newWorkflowsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "workflows",
		Short:  "Manage GitHub Actions workflow files",
		Hidden: true,
	}
	cmd.AddCommand(newWorkflowsSyncCmd())
	return cmd
}

func newWorkflowsSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Regenerate GitHub Actions workflows from the latest APX templates",
		Long: `Overwrites the .github/workflows/ files with the latest templates
from this version of APX. Detects the repository type (canonical or app)
from the existing workflow files and reads org/repo from apx.yaml.`,
		RunE: workflowsSyncAction,
	}
	cmd.Flags().Bool("dry-run", false, "Show what would be written without modifying files")
	return cmd
}

func workflowsSyncAction(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	var org, repo string
	var moduleRoots []string

	cfg, err := config.Load("")
	if err != nil {
		// No apx.yaml — fall back to git remote detection
		ui.Info("No apx.yaml found, detecting org/repo from git remote...")
		defaults, detectErr := detector.GetSmartDefaults()
		if detectErr == nil {
			org = defaults.Org
			repo = defaults.Repo
		}
	} else {
		org = cfg.Org
		repo = cfg.Repo
		moduleRoots = cfg.ModuleRoots
	}

	if org == "" || repo == "" {
		return fmt.Errorf("could not determine org and repo.\nEither create an apx.yaml (apx init canonical) or ensure a git remote named 'origin' is configured.")
	}

	workflowDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %w", workflowDir, err)
	}

	// Detect repo type from existing workflow files.
	// Canonical repos have ci.yml and/or on-merge.yml.
	// App repos have apx-release.yml.
	// If we can't detect, check for canonical directory structure.
	isCanonical := false
	isApp := false

	if fileExists(filepath.Join(workflowDir, "ci.yml")) ||
		fileExists(filepath.Join(workflowDir, "on-merge.yml")) {
		isCanonical = true
	}
	if fileExists(filepath.Join(workflowDir, "apx-release.yml")) {
		isApp = true
	}

	// Fallback: check for canonical directory structure (proto/, openapi/, etc.)
	if !isCanonical && !isApp {
		for _, dir := range []string{"proto", "openapi", "avro", "catalog"} {
			if dirExists(dir) {
				isCanonical = true
				break
			}
		}
	}
	// Fallback: if module_roots are set, likely an app repo
	if !isCanonical && !isApp && len(moduleRoots) > 0 {
		isApp = true
	}

	if !isCanonical && !isApp {
		return fmt.Errorf("could not determine repository type (canonical or app).\n" +
			"Expected .github/workflows/ci.yml (canonical) or apx-release.yml (app).")
	}

	type workflowFile struct {
		path    string
		content string
	}

	var files []workflowFile

	if isCanonical {
		files = append(files,
			workflowFile{
				path:    filepath.Join(workflowDir, "ci.yml"),
				content: templates.GenerateCanonicalCI(),
			},
			workflowFile{
				path:    filepath.Join(workflowDir, "on-merge.yml"),
				content: templates.GenerateCanonicalOnMerge(org),
			},
		)
	}
	if isApp {
		// Determine the canonical repo name. For now use "apis" as default,
		// matching the template convention.
		canonicalRepo := "apis"
		files = append(files,
			workflowFile{
				path:    filepath.Join(workflowDir, "apx-release.yml"),
				content: templates.GenerateAppRelease(org, canonicalRepo),
			},
		)
	}

	for _, f := range files {
		if dryRun {
			ui.Info("Would write: %s", f.path)
			continue
		}
		if err := os.WriteFile(f.path, []byte(f.content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", f.path, err)
		}
		ui.Success("Updated %s", f.path)
	}

	if dryRun {
		ui.Info("\nDry run — no files were modified.")
	} else {
		ui.Success("\nWorkflows synced to latest APX templates.")
		ui.Info("Review the changes and commit them:")
		ui.Info("  git diff .github/workflows/")
		ui.Info("  git add .github/workflows/ && git commit -m 'chore: sync APX workflows'")
	}

	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
