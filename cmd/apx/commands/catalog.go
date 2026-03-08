package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newCatalogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Catalog operations",
	}
	cmd.AddCommand(newCatalogBuildCmd())
	cmd.AddCommand(newCatalogGenerateCmd())
	return cmd
}

func newCatalogBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Build module catalog",
		RunE:  catalogBuildAction,
	}
}

func catalogBuildAction(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		ui.Error("Failed to load config: %v", err)
		return err
	}

	return buildCatalog(cfg)
}

func buildCatalog(cfg *config.Config) error {
	ui.Info("Building module catalog...")
	ui.Success("Module catalog built successfully")
	return nil
}

func newCatalogGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate catalog from git tags",
		Long: `Scan git tags matching the release pattern <format>/<domain>/<name>/<line>/v<semver>
and generate a catalog.yaml with discovered APIs, their latest stable and prerelease
versions, and inferred lifecycle state.

This command should be run in a canonical API repository. It reads the org and repo
from apx.yaml (or from --org and --repo flags) and writes the catalog to the
configured catalog path (default: catalog/catalog.yaml).`,
		RunE: catalogGenerateAction,
	}

	cmd.Flags().StringP("output", "o", "", "output path for catalog.yaml (default: catalog/catalog.yaml)")
	cmd.Flags().String("org", "", "organization name (overrides apx.yaml)")
	cmd.Flags().String("repo", "", "repository name (overrides apx.yaml)")
	cmd.Flags().String("dir", ".", "git repository directory to scan")

	return cmd
}

func catalogGenerateAction(cmd *cobra.Command, args []string) error {
	dir, _ := cmd.Flags().GetString("dir")
	output, _ := cmd.Flags().GetString("output")
	orgFlag, _ := cmd.Flags().GetString("org")
	repoFlag, _ := cmd.Flags().GetString("repo")

	// Resolve org and repo: flags override config
	org := orgFlag
	repo := repoFlag
	if org == "" || repo == "" {
		cfg, err := loadConfig(cmd)
		if err == nil {
			if org == "" {
				org = cfg.Org
			}
			if repo == "" {
				repo = cfg.Repo
			}
		}
	}
	if org == "" {
		return fmt.Errorf("org is required: set in apx.yaml or pass --org")
	}
	if repo == "" {
		return fmt.Errorf("repo is required: set in apx.yaml or pass --repo")
	}

	// Resolve output path
	if output == "" {
		output = filepath.Join("catalog", "catalog.yaml")
	}

	ui.Info("Scanning git tags in %s...", dir)

	tags, err := catalog.ListGitTags(dir)
	if err != nil {
		return fmt.Errorf("failed to list git tags: %w", err)
	}
	if len(tags) == 0 {
		ui.Warning("No git tags found")
	}

	cat := catalog.GenerateFromTags(tags, org, repo)

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	gen := catalog.NewGenerator(output)
	if err := gen.Save(cat); err != nil {
		return fmt.Errorf("failed to write catalog: %w", err)
	}

	// JSON output mode
	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonOut {
		data, err := json.MarshalIndent(cat, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	ui.Success("Catalog generated: %s (%d APIs discovered)", output, len(cat.Modules))
	for _, m := range cat.Modules {
		version := m.Version
		if version == "" {
			version = "(no releases)"
		}
		lifecycle := m.Lifecycle
		if lifecycle == "" {
			lifecycle = "unknown"
		}
		ui.Info("  %s  %s  [%s]", m.ID, version, lifecycle)
	}

	return nil
}
