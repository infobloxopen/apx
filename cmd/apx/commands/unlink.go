package commands

import (
	"os"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/overlay"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newUnlinkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unlink <module-path>",
		Short: "Remove overlay and switch to published module",
		Long: `Remove the local overlay for a module and update go.mod to use the published version.

This transitions from local development mode (overlay) to consuming the published module.

Examples:
  apx unlink proto/payments/ledger/v1
  apx unlink openapi/customer/accounts/v2`,
		Args: cobra.ExactArgs(1),
		RunE: unlinkAction,
	}
}

func unlinkAction(cmd *cobra.Command, args []string) error {
	modulePath := args[0]

	depMgr := config.NewDependencyManager("apx.yaml", "apx.lock")

	ui.Info("Removing overlay for %s...", modulePath)
	if err := depMgr.Remove(modulePath); err != nil {
		ui.Error("Failed to remove dependency: %v", err)
		return err
	}

	mgr := overlay.NewManager(".")

	if err := mgr.Remove(modulePath); err != nil {
		ui.Error("Failed to remove overlay: %v", err)
		return err
	}

	if err := updateGoModForPublished(modulePath); err != nil {
		ui.Error("Failed to update go.mod: %v", err)
		return err
	}

	ui.Success("Unlinked %s - now using published module", modulePath)
	return nil
}

func updateGoModForPublished(modulePath string) error {
	goModPath := "go.mod"
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return nil
	}

	ui.Info("Note: Run 'go get github.com/<org>/apis/%s' to add published module", modulePath)
	return nil
}
