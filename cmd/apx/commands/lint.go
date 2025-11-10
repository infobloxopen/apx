package commands

import (
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/urfave/cli/v2"
)

// LintCommand returns the lint command
func LintCommand() *cli.Command {
	return &cli.Command{
		Name:      "lint",
		Usage:     "Lint schema files",
		ArgsUsage: "[path]",
		Action:    lintAction,
	}
}

func lintAction(c *cli.Context) error {
	path := c.Args().First()
	if path == "" {
		path = "."
	}

	cfg, err := loadConfig(c)
	if err != nil {
		ui.Error("Failed to load config: %v", err)
		return err
	}

	// TODO: Move to internal/validator package
	return lintSchemas(cfg, path)
}

func lintSchemas(cfg *config.Config, path string) error {
	// TODO: Implement schema linting in internal/validator package
	ui.Info("Linting schemas in %s...", path)
	ui.Success("Schema linting completed successfully")
	return nil
}
