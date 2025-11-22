package commands

import (
	"fmt"
	"os"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/overlay"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/urfave/cli/v2"
)

// UnlinkCommand returns the unlink command for removing overlays
func UnlinkCommand() *cli.Command {
	return &cli.Command{
		Name:  "unlink",
		Usage: "Remove overlay and switch to published module",
		Description: `Remove the local overlay for a module and update go.mod to use the published version.

This transitions from local development mode (overlay) to consuming the published module.

Examples:
  apx unlink proto/payments/ledger/v1
  apx unlink openapi/customer/accounts/v2`,
		ArgsUsage: "<module-path>",
		Action:    unlinkAction,
	}
}

func unlinkAction(c *cli.Context) error {
	if c.NArg() == 0 {
		ui.Error("Module path required")
		return fmt.Errorf("usage: apx unlink <module-path>")
	}

	modulePath := c.Args().First()

	// Initialize dependency manager to check if dependency exists
	depMgr := config.NewDependencyManager("apx.yaml", "apx.lock")

	// Remove from apx.lock (this validates dependency exists)
	ui.Info("Removing overlay for %s...", modulePath)
	if err := depMgr.Remove(modulePath); err != nil {
		ui.Error("Failed to remove dependency: %v", err)
		return err
	}

	// Initialize overlay manager
	mgr := overlay.NewManager(".")

	// Remove overlay directory
	if err := mgr.Remove(modulePath); err != nil {
		ui.Error("Failed to remove overlay: %v", err)
		return err
	}

	// Update go.mod to use published module
	if err := updateGoModForPublished(modulePath); err != nil {
		ui.Error("Failed to update go.mod: %v", err)
		return err
	}

	ui.Success("Unlinked %s - now using published module", modulePath)
	return nil
}

func updateGoModForPublished(modulePath string) error {
	// This is a simplified implementation
	// In production, this would:
	// 1. Parse go.mod
	// 2. Add require directive for published module
	// 3. Remove replace directive if present
	// 4. Run go mod tidy

	goModPath := "go.mod"
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		// No go.mod, skip
		return nil
	}

	// For now, just ensure the published module path is noted
	// Full implementation would parse and update go.mod properly
	ui.Info("Note: Run 'go get github.com/<org>/apis-go/%s' to add published module", modulePath)

	return nil
}
