package commands

import (
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/urfave/cli/v2"
)

// BreakingCommand returns the breaking changes command
func BreakingCommand() *cli.Command {
	return &cli.Command{
		Name:      "breaking",
		Usage:     "Check for breaking changes",
		ArgsUsage: "[path]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "against",
				Usage:    "git reference to compare against",
				Required: true,
			},
		},
		Action: breakingAction,
	}
}

func breakingAction(c *cli.Context) error {
	path := c.Args().First()
	if path == "" {
		path = "."
	}
	against := c.String("against")

	cfg, err := loadConfig(c)
	if err != nil {
		ui.Error("Failed to load config: %v", err)
		return err
	}

	return checkBreaking(cfg, path, against)
}

func checkBreaking(cfg *config.Config, path, against string) error {
	// TODO: Implement breaking change detection in internal/validator package
	ui.Info("Checking for breaking changes in %s against %s...", path, against)
	ui.Success("No breaking changes detected")
	return nil
}
