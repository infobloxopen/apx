package commands

import (
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/urfave/cli/v2"
)

// AddCommand returns the add command for adding dependencies
func AddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add a dependency to apx.yaml and apx.lock",
		Description: `Add a schema module dependency to the project.

The dependency is added to apx.yaml and the version is locked in apx.lock.

Examples:
  apx add proto/payments/ledger/v1@v1.2.3
  apx add proto/payments/wallet/v1         # Uses latest version
  apx add openapi/customer/accounts/v2@v2.0.0`,
		ArgsUsage: "<module-path>[@version]",
		Action:    addAction,
	}
}

func addAction(c *cli.Context) error {
	if c.NArg() == 0 {
		ui.Error("Module path required")
		return fmt.Errorf("usage: apx add <module-path>[@version]")
	}

	arg := c.Args().First()

	// Parse module path and version
	var modulePath, version string
	if strings.Contains(arg, "@") {
		parts := strings.SplitN(arg, "@", 2)
		modulePath = parts[0]
		version = parts[1]
	} else {
		modulePath = arg
		version = "" // Will fetch latest
	}

	// Create dependency manager
	mgr := config.NewDependencyManager("apx.yaml", "apx.lock")

	// Add dependency
	if err := mgr.Add(modulePath, version); err != nil {
		ui.Error("Failed to add dependency: %v", err)
		return err
	}

	if version != "" {
		ui.Success("Added dependency: %s@%s", modulePath, version)
	} else {
		ui.Success("Added dependency: %s (latest version)", modulePath)
	}

	return nil
}
