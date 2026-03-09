package commands

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

// UpdateCandidate describes a dependency that can be updated.
type UpdateCandidate struct {
	ModulePath     string `json:"module_path"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	Lifecycle      string `json:"lifecycle,omitempty"`
}

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [module-path]",
		Short: "Update dependencies to latest compatible versions",
		Long: `Check for compatible (same API line, higher minor/patch) updates
for pinned dependencies and apply them.

Without arguments, checks all dependencies. With a module path, updates
only that dependency.

Use --dry-run to preview what would be updated without modifying apx.lock.`,
		Args: cobra.MaximumNArgs(1),
		RunE: updateAction,
	}

	cmd.Flags().Bool("dry-run", false, "preview updates without applying them")
	cmd.Flags().StringP("catalog", "c", filepath.Join("catalog", "catalog.yaml"), "path to catalog file")

	return cmd
}

func updateAction(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	catalogPath, _ := cmd.Flags().GetString("catalog")
	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")

	// Load the catalog
	gen := catalog.NewGenerator(catalogPath)
	cat, err := gen.Load()
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w (run apx catalog generate first)", err)
	}

	// Build module index for quick lookup
	moduleIndex := make(map[string]catalog.Module)
	for _, m := range cat.Modules {
		moduleIndex[m.ID] = m
	}

	// Load current dependencies from lock file
	mgr := config.NewDependencyManager("apx.yaml", "apx.lock")
	deps, err := mgr.List()
	if err != nil {
		return fmt.Errorf("failed to list dependencies: %w", err)
	}
	if len(deps) == 0 {
		ui.Info("No dependencies in apx.lock")
		return nil
	}

	// If a specific module was requested, filter to just that one
	var targetModule string
	if len(args) > 0 {
		targetModule = args[0]
	}

	var candidates []UpdateCandidate
	for _, dep := range deps {
		if targetModule != "" && dep.ModulePath != targetModule {
			continue
		}

		mod, found := moduleIndex[dep.ModulePath]
		if !found {
			ui.Warning("  %s: not found in catalog (skipped)", dep.ModulePath)
			continue
		}

		latest := latestCompatible(dep.Version, mod)
		if latest == "" || latest == dep.Version {
			continue
		}

		// Verify this is actually a newer compatible version (same major)
		currentSV, err := config.ParseSemVer(dep.Version)
		if err != nil {
			continue
		}
		latestSV, err := config.ParseSemVer(latest)
		if err != nil {
			continue
		}
		if latestSV.Major != currentSV.Major {
			continue // Skip major version bumps — that's upgrade territory
		}
		if config.CompareSemVer(latestSV, currentSV) <= 0 {
			continue // Not actually newer
		}

		candidates = append(candidates, UpdateCandidate{
			ModulePath:     dep.ModulePath,
			CurrentVersion: dep.Version,
			LatestVersion:  latest,
			Lifecycle:      mod.Lifecycle,
		})
	}

	if targetModule != "" && len(candidates) == 0 {
		// Check if the module even exists in deps
		found := false
		for _, dep := range deps {
			if dep.ModulePath == targetModule {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("dependency not found: %s", targetModule)
		}
		ui.Success("%s is already at the latest compatible version", targetModule)
		return nil
	}

	if len(candidates) == 0 {
		ui.Success("All dependencies are up to date")
		return nil
	}

	// JSON output
	if jsonOut {
		data, err := json.MarshalIndent(candidates, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	// Display candidates
	if dryRun {
		ui.Info("Updates available (dry-run):")
	} else {
		ui.Info("Updating dependencies:")
	}

	for _, c := range candidates {
		ui.Info("  %s: %s → %s  [%s]", c.ModulePath, c.CurrentVersion, c.LatestVersion, c.Lifecycle)
	}

	if dryRun {
		ui.Info("\nRun without --dry-run to apply these updates")
		return nil
	}

	// Apply updates
	applied := 0
	for _, c := range candidates {
		if err := mgr.Add(c.ModulePath, c.LatestVersion); err != nil {
			ui.Error("  Failed to update %s: %v", c.ModulePath, err)
			continue
		}
		applied++
	}

	ui.Success("Updated %d dependencies", applied)
	ui.Info("Run 'apx gen go && apx sync' to regenerate code")

	return nil
}

// latestCompatible returns the best compatible version from the catalog module.
// It prefers LatestStable over LatestPrerelease, and falls back to Version.
func latestCompatible(currentVersion string, mod catalog.Module) string {
	// Prefer the latest stable version
	if mod.LatestStable != "" {
		return normalizeVersion(mod.LatestStable)
	}
	// Fall back to latest prerelease
	if mod.LatestPrerelease != "" {
		return normalizeVersion(mod.LatestPrerelease)
	}
	// Fall back to the module's Version field
	if mod.Version != "" {
		return normalizeVersion(mod.Version)
	}
	return ""
}

// normalizeVersion ensures a version has the "v" prefix.
func normalizeVersion(v string) string {
	if v == "" {
		return v
	}
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}
