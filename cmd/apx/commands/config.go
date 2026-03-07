package commands

import (
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration operations",
	}
	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigValidateCmd())
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.Init()
		},
	}
}

func newConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Root().PersistentFlags().GetString("config")
			if configPath == "" {
				configPath = "apx.yaml"
			}
			_, err := config.Load(configPath)
			if err != nil {
				ui.Error("Configuration validation failed: %v", err)
				return err
			}
			ui.Success("Configuration is valid")
			return nil
		},
	}
}
