package commands

import (
	"fmt"
	"os"
	"path/filepath"

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
	} else {
		format = validator.DetectFormat(absPath)
	}

	if format == validator.FormatUnknown {
		return fmt.Errorf("could not detect schema format for: %s\nPlease specify format with --format flag", absPath)
	}

	ui.Info("Linting %s files in: %s", format, absPath)

	if err := v.Lint(absPath, format); err != nil {
		ui.Error("Lint failed: %v", err)
		return err
	}

	ui.Success("\u2713 All files passed lint checks")
	return nil
}
