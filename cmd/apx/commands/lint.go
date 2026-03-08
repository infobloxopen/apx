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

func newLintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint [path]",
		Short: "Validate schema files for syntax and style issues",
		Args:  cobra.MaximumNArgs(1),
		RunE:  lintAction,
	}
	cmd.Flags().StringP("format", "f", "", "Schema format (proto, openapi, avro, jsonschema, parquet)")
	return cmd
}

func lintAction(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

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

	var format validator.SchemaFormat
	if formatStr, _ := cmd.Flags().GetString("format"); formatStr != "" {
		format = validator.SchemaFormat(formatStr)
	} else if apiFormat != "" {
		format = validator.SchemaFormat(apiFormat)
	} else {
		format = validator.DetectFormat(absPath)
	}

	if format == validator.FormatUnknown && cfg != nil {
		format = validator.DetectFormatFromModuleRoots(cfg.ModuleRoots)
	}

	if format == validator.FormatUnknown {
		// No explicit --format and we couldn't detect one. If this is a
		// directory, it may simply contain no schema files yet (e.g. a
		// freshly-scaffolded canonical repo). In that case succeed with
		// a helpful message instead of erroring.
		if info, statErr := os.Stat(absPath); statErr == nil && info.IsDir() {
			ui.Info("No schema files found in: %s", absPath)
			ui.Info("Nothing to lint. Add schema files or specify --format.")
			return nil
		}
		return fmt.Errorf("could not detect schema format for: %s\nPlease specify format with --format flag", absPath)
	}

	ui.Info("Linting %s files in: %s", format, absPath)

	if err := v.Lint(absPath, format); err != nil {
		ui.Error("Lint failed: %v", err)
		return err
	}

	// For proto files, check go_package for deprecated apis-go import root.
	if format == validator.FormatProto {
		if warnings := validator.CheckGoPackageCanonical(absPath); len(warnings) > 0 {
			for _, w := range warnings {
				ui.Warning("%s", w)
			}
		}
	}

	ui.Success("\u2713 All files passed lint checks")
	return nil
}
