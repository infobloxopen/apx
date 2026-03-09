package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

// UpgradeReport describes the result of analyzing a major-version upgrade.
type UpgradeReport struct {
	ModulePath     string `json:"module_path"`
	CurrentVersion string `json:"current_version"`
	TargetLine     string `json:"target_line"`
	TargetVersion  string `json:"target_version"`
	TargetModule   string `json:"target_module"`
	Lifecycle      string `json:"lifecycle,omitempty"`
	ImportChange   string `json:"import_change,omitempty"`
}

func newUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade <module-path>",
		Short: "Upgrade a dependency to a new API line (major version)",
		Long: `Upgrade a dependency to a new API line, handling the major version
transition. This replaces the old API line with the new one in apx.yaml
and apx.lock.

Use --to to specify the target API line (e.g. v2).
Use --dry-run to preview the upgrade without applying changes.`,
		Args: cobra.ExactArgs(1),
		RunE: upgradeAction,
	}

	cmd.Flags().String("to", "", "target API line (e.g. v2) — required")
	cmd.Flags().Bool("dry-run", false, "preview upgrade without applying")
	cmd.Flags().StringP("catalog", "c", "", "Path or URL to catalog file (default: catalog_url from apx.yaml, then catalog/catalog.yaml)")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func upgradeAction(cmd *cobra.Command, args []string) error {
	modulePath := args[0]
	targetLine, _ := cmd.Flags().GetString("to")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	catalogPath, _ := cmd.Flags().GetString("catalog")
	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")

	// Verify the current dependency exists
	mgr := config.NewDependencyManager("apx.yaml", "apx.lock", resolveSourceRepo(cmd))
	deps, err := mgr.List()
	if err != nil {
		return fmt.Errorf("failed to list dependencies: %w", err)
	}

	var currentDep *config.Dependency
	for _, dep := range deps {
		if dep.ModulePath == modulePath {
			d := dep
			currentDep = &d
			break
		}
	}
	if currentDep == nil {
		return fmt.Errorf("dependency not found: %s", modulePath)
	}

	// Parse current module path to extract the API line
	// Format: <format>/<domain>/<name>/<line>  e.g. proto/payments/ledger/v1
	parts := strings.Split(modulePath, "/")
	if len(parts) < 4 {
		return fmt.Errorf("invalid module path (expected <format>/<domain>/<name>/<line>): %s", modulePath)
	}
	currentLine := parts[len(parts)-1]

	// Normalize target line
	if !strings.HasPrefix(targetLine, "v") {
		targetLine = "v" + targetLine
	}

	if targetLine == currentLine {
		return fmt.Errorf("already on API line %s", targetLine)
	}

	// Build the target module path by replacing the line segment
	targetParts := make([]string, len(parts))
	copy(targetParts, parts)
	targetParts[len(targetParts)-1] = targetLine
	targetModulePath := strings.Join(targetParts, "/")

	// Look up the target in the catalog
	src := resolveCatalogSource(cmd, catalogPath)
	cat, err := src.Load()
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w (run apx catalog generate first)", err)
	}

	var targetMod *catalog.Module
	for _, m := range cat.Modules {
		if m.ID == targetModulePath {
			tm := m
			targetMod = &tm
			break
		}
	}
	if targetMod == nil {
		return fmt.Errorf("target API line not found in catalog: %s\n  Run 'apx search %s' to see available lines",
			targetModulePath, strings.Join(parts[:len(parts)-1], "/"))
	}

	// Determine target version
	targetVersion := targetMod.LatestStable
	if targetVersion == "" {
		targetVersion = targetMod.LatestPrerelease
	}
	if targetVersion == "" {
		targetVersion = targetMod.Version
	}
	targetVersion = normalizeVersion(targetVersion)

	// Build import change description
	importChange := fmt.Sprintf("%s → %s", modulePath, targetModulePath)

	report := UpgradeReport{
		ModulePath:     modulePath,
		CurrentVersion: currentDep.Version,
		TargetLine:     targetLine,
		TargetVersion:  targetVersion,
		TargetModule:   targetModulePath,
		Lifecycle:      targetMod.Lifecycle,
		ImportChange:   importChange,
	}

	// JSON output
	if jsonOut {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		if dryRun {
			return nil
		}
	}

	// Display upgrade plan
	ui.Info("Upgrade plan:")
	ui.Info("  Module:     %s → %s", modulePath, targetModulePath)
	ui.Info("  Version:    %s → %s", currentDep.Version, targetVersion)
	ui.Info("  Line:       %s → %s", currentLine, targetLine)
	ui.Info("  Lifecycle:  %s", report.Lifecycle)
	ui.Info("  Imports:    %s", importChange)

	if dryRun {
		ui.Info("\nRun without --dry-run to apply this upgrade")
		ui.Info("Tip: Run 'apx breaking' before and after to inspect breaking changes")
		return nil
	}

	// Apply: remove old dependency, add new one
	if err := mgr.Remove(modulePath); err != nil {
		ui.Warning("Could not remove old dependency (may not be in lock): %v", err)
	}

	// Look up provenance from catalog for the target module
	var provenance *config.ExternalProvenance
	if targetMod.Origin != "" {
		provenance = &config.ExternalProvenance{
			Origin:       targetMod.Origin,
			ManagedRepo:  targetMod.ManagedRepo,
			UpstreamRepo: targetMod.UpstreamRepo,
			UpstreamPath: targetMod.UpstreamPath,
			ImportMode:   targetMod.ImportMode,
		}
	}

	if err := mgr.AddWithProvenance(targetModulePath, targetVersion, provenance); err != nil {
		return fmt.Errorf("failed to add upgraded dependency: %w", err)
	}

	ui.Success("Upgraded %s → %s@%s", modulePath, targetModulePath, targetVersion)
	ui.Info("Run 'apx gen go && apx sync' to regenerate code with new imports")
	ui.Info("Tip: Update import paths in your code: %s", importChange)

	return nil
}
