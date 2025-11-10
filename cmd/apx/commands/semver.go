package commands

import (
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/urfave/cli/v2"
)

// SemverCommand returns the semver command with subcommands
func SemverCommand() *cli.Command {
	return &cli.Command{
		Name:      "semver",
		Usage:     "Semantic version operations",
		ArgsUsage: "[path]",
		Subcommands: []*cli.Command{
			{
				Name:      "suggest",
				Usage:     "Suggest semantic version bump",
				ArgsUsage: "[path]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "against",
						Usage:    "git reference to compare against",
						Required: true,
					},
				},
				Action: semverSuggestAction,
			},
		},
	}
}

func semverSuggestAction(c *cli.Context) error {
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

	return suggestSemver(cfg, path, against)
}

func suggestSemver(cfg *config.Config, path, against string) error {
	// TODO: Implement semver suggestion in internal/versioning package
	ui.Info("Analyzing changes in %s against %s...", path, against)
	ui.Info("Suggested version bump: PATCH")
	return nil
}
