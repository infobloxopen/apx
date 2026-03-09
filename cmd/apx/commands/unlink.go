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
		Short: "Remove overlay and switch to released module",
		Long: `Remove the local overlay for a module and update go.mod to use the released version.

This transitions from local development mode (overlay) to consuming the released module.

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

	if err := updateGoModForReleased(modulePath); err != nil {
		ui.Error("Failed to update go.mod: %v", err)
		return err
	}

	printPythonUnlinkHint(modulePath)
	ui.Success("Unlinked %s - now using released module", modulePath)
	return nil
}

func updateGoModForReleased(modulePath string) error {
	goModPath := "go.mod"
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return nil
	}

	ui.Info("Note: Run 'go get github.com/<org>/apis/%s' to add released module", modulePath)
	return nil
}

func printPythonUnlinkHint(modulePath string) {
	cfg, _ := config.Load("")
	if cfg == nil || cfg.Org == "" {
		return
	}
	api, err := config.ParseAPIID(modulePath)
	if err != nil {
		return
	}
	distName := config.DerivePythonDistName(cfg.Org, api)
	ui.Info("Python: Run 'pip install %s' to install the released package", distName)
}
