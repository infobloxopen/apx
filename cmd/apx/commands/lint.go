package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/ui"
	"github.com/infobloxopen/apx/internal/validator"
	"github.com/urfave/cli/v2"
)

// LintCommand returns the lint command for validating schema files
func LintCommand() *cli.Command {
	return &cli.Command{
		Name:      "lint",
		Usage:     "Validate schema files for syntax and style issues",
		ArgsUsage: "[path]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Schema format (proto, openapi, avro, jsonschema, parquet)",
			},
		},
		Action: lintAction,
	}
}

func lintAction(c *cli.Context) error {
	// Get path from args or default to current directory
	path := "."
	if c.Args().Len() > 0 {
		path = c.Args().First()
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	// Create toolchain resolver with default settings
	resolver := validator.NewToolchainResolver()

	// Create validator
	v := validator.NewValidator(resolver)

	// Detect or use specified format
	format := validator.FormatUnknown
	if formatStr := c.String("format"); formatStr != "" {
		format = validator.SchemaFormat(formatStr)
	} else {
		format = validator.DetectFormat(absPath)
	}

	// Validate format
	if format == validator.FormatUnknown {
		return fmt.Errorf("could not detect schema format for: %s\nPlease specify format with --format flag", absPath)
	}

	// Run lint
	ui.Info("Linting %s file: %s", format, absPath)

	err = v.Lint(absPath, format)
	if err != nil {
		ui.Error("Lint failed: %v", err)
		return err
	}

	ui.Success("✓ Lint passed")
	return nil
}
