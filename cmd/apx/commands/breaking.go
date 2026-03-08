package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/infobloxopen/apx/internal/validator"
	"github.com/spf13/cobra"
)

func newBreakingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "breaking [path]",
		Short: "Check for breaking changes in schema files",
		Args:  cobra.MaximumNArgs(1),
		RunE:  breakingAction,
	}
	cmd.Flags().String("against", "", "git reference or path to compare against")
	_ = cmd.MarkFlagRequired("against")
	cmd.Flags().StringP("format", "f", "", "Schema format (proto, openapi, avro, jsonschema, parquet)")
	return cmd
}

func breakingAction(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	against, _ := cmd.Flags().GetString("against")

	// Try to resolve API ID (e.g. proto/payments/ledger/v1) to a path.
	// Falls back to treating the argument as a filesystem path.
	var apiFormat string
	cfg, _ := config.Load("")
	resolved, resolveErr := config.ResolveAPIPath(path, cfg)
	if resolveErr == nil {
		apiFormat = config.ResolveAPIFormat(path)
		path = resolved
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	resolver := validator.NewToolchainResolver()
	v := validator.NewValidator(resolver)

	format := validator.FormatUnknown
	if formatStr, _ := cmd.Flags().GetString("format"); formatStr != "" {
		format = validator.SchemaFormat(formatStr)
	} else if apiFormat != "" {
		format = validator.SchemaFormat(apiFormat)
	} else {
		format = validator.DetectFormat(absPath)
	}

	if format == validator.FormatUnknown {
		return fmt.Errorf("could not detect schema format for: %s\nPlease specify format with --format flag", absPath)
	}

	ui.Info("Checking %s for breaking changes against: %s", format, against)

	if err := v.Breaking(absPath, against, format); err != nil {
		ui.Error("Breaking changes detected: %v", err)
		return err
	}

	ui.Success("\u2713 No breaking changes detected")
	return nil
}
