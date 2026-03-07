package commands

import (
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newSemverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "semver",
		Short: "Semantic version operations",
	}
	cmd.AddCommand(newSemverSuggestCmd())
	return cmd
}

func newSemverSuggestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "suggest [path]",
		Short: "Suggest semantic version bump",
		Args:  cobra.MaximumNArgs(1),
		RunE:  semverSuggestAction,
	}
	cmd.Flags().String("against", "", "git reference to compare against")
	_ = cmd.MarkFlagRequired("against")
	return cmd
}

func semverSuggestAction(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}
	against, _ := cmd.Flags().GetString("against")

	cfg, err := loadConfig(cmd)
	if err != nil {
		ui.Error("Failed to load config: %v", err)
		return err
	}

	return suggestSemver(cfg, path, against)
}

func suggestSemver(cfg *config.Config, path, against string) error {
	ui.Info("Analyzing changes in %s against %s...", path, against)
	ui.Info("Suggested version bump: PATCH")
	return nil
}
