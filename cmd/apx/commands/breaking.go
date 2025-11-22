package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/ui"
	"github.com/infobloxopen/apx/internal/validator"
	"github.com/urfave/cli/v2"
)

// BreakingCommand returns the breaking changes command
func BreakingCommand() *cli.Command {
	return &cli.Command{
		Name:      "breaking",
		Usage:     "Check for breaking changes in schema files",
		ArgsUsage: "[path]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "against",
				Usage:    "git reference or path to compare against",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Schema format (proto, openapi, avro, jsonschema, parquet)",
			},
		},
		Action: breakingAction,
	}
}

func breakingAction(c *cli.Context) error {
	// Get path from args or default to current directory
	path := "."
	if c.Args().Len() > 0 {
		path = c.Args().First()
	}

	// Get the baseline to compare against
	against := c.String("against")
	if against == "" {
		return fmt.Errorf("--against flag is required")
	}

	// Resolve absolute paths
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

	// Run breaking change detection
	ui.Info("Checking %s for breaking changes against: %s", format, against)

	err = v.Breaking(absPath, against, format)
	if err != nil {
		ui.Error("Breaking changes detected: %v", err)
		return err
	}

	ui.Success("✓ No breaking changes detected")
	return nil
}
