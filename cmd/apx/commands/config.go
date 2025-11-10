package commands

import (
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/urfave/cli/v2"
)

// ConfigCommand returns the config command with subcommands
func ConfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Configuration operations",
		Subcommands: []*cli.Command{
			{
				Name:   "init",
				Usage:  "Initialize configuration file",
				Action: configInitAction,
			},
			{
				Name:   "validate",
				Usage:  "Validate configuration file",
				Action: configValidateAction,
			},
		},
	}
}

func configInitAction(c *cli.Context) error {
	return config.Init()
}

func configValidateAction(c *cli.Context) error {
	configPath := c.String("config")
	_, err := config.Load(configPath)
	if err != nil {
		ui.Error("Configuration validation failed: %v", err)
		return err
	}
	ui.Success("Configuration is valid")
	return nil
}
