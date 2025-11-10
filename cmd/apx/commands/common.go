package commands

import (
	"github.com/infobloxopen/apx/internal/config"
	"github.com/urfave/cli/v2"
)

// loadConfig loads the configuration file
func loadConfig(c *cli.Context) (*config.Config, error) {
	configPath := c.String("config")
	return config.Load(configPath)
}
