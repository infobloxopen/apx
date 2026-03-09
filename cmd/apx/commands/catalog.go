package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newCatalogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Catalog operations",
	}
	cmd.AddCommand(newCatalogGenerateCmd())
	cmd.AddCommand(newCatalogPublishCmd())
	return cmd
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

	// Merge external API registrations from apx.yaml
	cfg, cfgErr := loadConfig(cmd)
	if cfgErr == nil && len(cfg.ExternalAPIs) > 0 {
		if err := catalog.MergeExternalAPIs(cat, cfg.ExternalAPIs); err != nil {
			return fmt.Errorf("failed to merge external APIs: %w", err)
		}
	}

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

	firstParty, external := catalog.ExternalModuleCount(cat)
	if external > 0 {
		ui.Success("Catalog generated: %d modules (%d first-party, %d external)", len(cat.Modules), firstParty, external)
	} else {
		ui.Success("Catalog generated: %s (%d APIs discovered)", output, len(cat.Modules))
	}
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

func newCatalogPublishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish catalog to GHCR as an OCI artifact",
		Long: `Push the local catalog.yaml to GitHub Container Registry as a data-only OCI artifact.
The image is published to ghcr.io/<org>/<repo>-catalog:<tag>.

This command is typically run by CI after 'apx catalog generate' on the canonical
API repository. It requires 'gh' to be installed and authenticated with write:packages scope.

Examples:
  apx catalog publish                     # Push catalog/catalog.yaml to GHCR
  apx catalog publish --catalog path.yaml # Push a specific catalog file
  apx catalog publish --tag v1            # Push with a specific tag
  apx catalog publish --dry-run           # Preview without pushing`,
		RunE: catalogPublishAction,
	}

	cmd.Flags().String("catalog", "", "path to catalog.yaml (default: catalog/catalog.yaml)")
	cmd.Flags().String("tag", "latest", "OCI image tag")
	cmd.Flags().String("org", "", "organization name (overrides apx.yaml)")
	cmd.Flags().String("repo", "", "repository name (overrides apx.yaml)")
	cmd.Flags().Bool("dry-run", false, "show what would be published without pushing")

	return cmd
}

func catalogPublishAction(cmd *cobra.Command, args []string) error {
	catalogPath, _ := cmd.Flags().GetString("catalog")
	tag, _ := cmd.Flags().GetString("tag")
	orgFlag, _ := cmd.Flags().GetString("org")
	repoFlag, _ := cmd.Flags().GetString("repo")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Resolve catalog path
	if catalogPath == "" {
		catalogPath = filepath.Join("catalog", "catalog.yaml")
	}

	// Load the catalog
	gen := catalog.NewGenerator(catalogPath)
	cat, err := gen.Load()
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	// Resolve org and repo: flags > catalog fields > apx.yaml
	org := orgFlag
	repo := repoFlag
	if org == "" {
		org = cat.Org
	}
	if repo == "" {
		repo = cat.Repo
	}
	if org == "" || repo == "" {
		cfg, cfgErr := loadConfig(cmd)
		if cfgErr == nil {
			if org == "" {
				org = cfg.Org
			}
			if repo == "" {
				repo = cfg.Repo
			}
		}
	}
	if org == "" {
		return fmt.Errorf("org is required: set in apx.yaml, catalog.yaml, or pass --org")
	}
	if repo == "" {
		return fmt.Errorf("repo is required: set in apx.yaml, catalog.yaml, or pass --repo")
	}

	imageRef := fmt.Sprintf("ghcr.io/%s/%s%s:%s", org, repo, catalog.CatalogImageSuffix, tag)

	if dryRun {
		ui.Info("Dry run: would publish catalog to %s", imageRef)
		ui.Info("  Org:     %s", org)
		ui.Info("  Repo:    %s", repo)
		ui.Info("  Modules: %d", len(cat.Modules))
		for _, m := range cat.Modules {
			ui.Info("    %s", m.ID)
		}
		return nil
	}

	ui.Info("Publishing catalog to %s...", imageRef)

	if err := catalog.PushCatalog(cat, catalog.PushOptions{
		Org:  org,
		Repo: repo,
		Tag:  tag,
	}); err != nil {
		return fmt.Errorf("failed to publish catalog: %w", err)
	}

	ui.Success("Catalog published: %s (%d APIs)", imageRef, len(cat.Modules))
	return nil
}
