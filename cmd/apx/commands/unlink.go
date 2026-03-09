package commands

import (
	"fmt"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/language"
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

	depMgr := config.NewDependencyManager("apx.yaml", "apx.lock", "")

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

	// Print unlink hints from all available plugins.
	printUnlinkHints(modulePath)
	ui.Success("Unlinked %s - now using released module", modulePath)
	return nil
}

func printUnlinkHints(modulePath string) {
	api, err := config.ParseAPIID(modulePath)
	if err != nil {
		return
	}

	cfg, _ := config.LoadRaw("")
	sourceRepo := "github.com/<org>/<repo>"
	org := ""
	importRoot := ""
	if cfg != nil {
		if cfg.Org != "" && cfg.Repo != "" {
			sourceRepo = fmt.Sprintf("github.com/%s/%s", cfg.Org, cfg.Repo)
		}
		org = cfg.Org
		importRoot = cfg.ImportRoot
	}

	ctx := language.DerivationContext{
		SourceRepo: sourceRepo,
		ImportRoot: importRoot,
		Org:        org,
		API:        api,
	}

	for _, p := range language.Available(ctx) {
		hint := p.UnlinkHint(ctx)
		if hint != nil {
			ui.Info("%s", hint.Message)
		}
	}
}
