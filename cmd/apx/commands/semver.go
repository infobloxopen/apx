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

func newSemverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "semver",
		Short: "Semantic version operations",
	}
	cmd.AddCommand(newSemverSuggestCmd())
	return cmd
}

func newSemverSuggestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "suggest [path]",
		Short: "Suggest semantic version bump based on breaking-change analysis",
		Long: `Analyze schema changes and suggest the appropriate version bump.

Rules:
  - Breaking change detected → reject (must create new major API line)
  - Non-breaking additive changes → MINOR bump
  - No schema changes (bugfix/docs/generator-only) → PATCH bump

Lifecycle pre-release mapping:
  - experimental → -alpha.N
  - beta         → -beta.N  (preview accepted as alias for beta)
  - stable       → normal semver (no prerelease)
  - deprecated   → allowed with warning
  - sunset       → blocked`,
		Args: cobra.MaximumNArgs(1),
		RunE: semverSuggestAction,
	}
	cmd.Flags().String("against", "", "Git reference or path to compare against (required)")
	_ = cmd.MarkFlagRequired("against")
	cmd.Flags().String("api-id", "", "API ID (e.g. proto/payments/ledger/v1)")
	cmd.Flags().String("lifecycle", "", "Lifecycle state (experimental, beta, stable, deprecated, sunset)")
	cmd.Flags().StringP("format", "f", "", "Schema format (proto, openapi, avro, jsonschema, parquet)")
	return cmd
}

func semverSuggestAction(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}
	against, _ := cmd.Flags().GetString("against")
	apiID, _ := cmd.Flags().GetString("api-id")
	lifecycle, _ := cmd.Flags().GetString("lifecycle")
	formatStr, _ := cmd.Flags().GetString("format")

	cfg, _ := loadConfig(cmd)

	// Resolve API ID to real path if provided
	var apiFormat string
	if apiID != "" {
		resolved, err := config.ResolveAPIPath(apiID, cfg)
		if err == nil {
			apiFormat = config.ResolveAPIFormat(apiID)
			path = resolved
		}
	}

	return suggestSemver(cfg, path, against, apiID, lifecycle, apiFormat, formatStr)
}

func suggestSemver(cfg *config.Config, path, against, apiID, lifecycle, apiFormat, formatFlag string) error {
	ui.Info("Analyzing changes in %s against %s...", path, against)

	// --- Determine schema format ---
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	var format validator.SchemaFormat
	switch {
	case formatFlag != "":
		format = validator.SchemaFormat(formatFlag)
	case apiFormat != "":
		format = validator.SchemaFormat(apiFormat)
	default:
		format = validator.DetectFormat(absPath)
	}

	if format == validator.FormatUnknown && cfg != nil {
		format = validator.DetectFormatFromModuleRoots(cfg.ModuleRoots)
	}

	if format == validator.FormatUnknown {
		return fmt.Errorf("could not detect schema format for: %s\nPlease specify format with --format flag", absPath)
	}

	// --- Validate lifecycle ---
	if lifecycle != "" {
		if err := config.ValidateLifecycle(lifecycle); err != nil {
			return err
		}
	}

	// --- Parse API line from api-id ---
	line := "v1" // default
	if apiID != "" {
		api, err := config.ParseAPIID(apiID)
		if err != nil {
			return err
		}
		line = api.Line
	}

	// --- Run breaking-change detection ---
	resolver := validator.NewToolchainResolver()
	v := validator.NewValidator(resolver)

	hasBreaking := false
	breakingErr := v.Breaking(absPath, against, format)
	if breakingErr != nil {
		hasBreaking = true
		ui.Warning("Breaking changes detected: %v", breakingErr)
	} else {
		ui.Success("No breaking changes detected")
	}

	// --- Determine current latest version ---
	current := ""
	if apiID != "" {
		cwd, _ := os.Getwd()
		tm := publisher.NewTagManager(cwd, "")
		versions, listErr := tm.ListVersionsForAPI(apiID)
		if listErr != nil {
			ui.Warning("Could not list tags: %v", listErr)
		} else if len(versions) > 0 {
			major, _ := config.LineMajor(line)
			latest, latestErr := config.LatestVersion(versions, major)
			if latestErr == nil && latest != "" {
				current = latest
			}
		}
	}

	// Determine if there are additive changes (non-breaking but meaningful).
	// If the breaking check passed without error but the against ref differs,
	// treat as additive changes. If caller explicitly knows there are no
	// schema changes, they would use a different workflow.
	hasChanges := true // conservative: assume additive changes unless proven otherwise

	// --- Suggest version ---
	suggestion, suggestErr := config.SuggestVersion(current, hasBreaking, hasChanges, lifecycle, line)
	if suggestion != nil {
		ui.Info("\n%s", config.FormatSuggestionReport(suggestion))
	}
	if suggestErr != nil {
		return &publisher.PublishError{
			Code:    publisher.ErrCodeBreakingChange,
			Message: suggestErr.Error(),
			Hint:    fmt.Sprintf("Create a new API line (e.g. apx init --line v%s)", nextLine(line)),
		}
	}

	return nil
}

// nextLine increments a line version string for hint messages.
func nextLine(line string) string {
	major, err := config.LineMajor(line)
	if err != nil {
		return "N+1"
	}
	return fmt.Sprintf("%d", major+1)
}
