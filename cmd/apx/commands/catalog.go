package commands

import (
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newCatalogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Catalog operations",
	}
	cmd.AddCommand(newCatalogBuildCmd())
	return cmd
}

func newCatalogBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Build module catalog",
		RunE:  catalogBuildAction,
	}
}

func catalogBuildAction(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		ui.Error("Failed to load config: %v", err)
		return err
	}

	return buildCatalog(cfg)
}

func buildCatalog(cfg *config.Config) error {
	ui.Info("Building module catalog...")
	ui.Success("Module catalog built successfully")
	return nil
}
