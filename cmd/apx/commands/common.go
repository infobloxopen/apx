package commands

import (
	"github.com/infobloxopen/apx/internal/config"
	"github.com/spf13/cobra"
)

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	configPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if configPath == "" {
		configPath = "apx.yaml"
	}
	return config.Load(configPath)
}
