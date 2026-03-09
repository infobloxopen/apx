package commands

import (
	"strings"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <module-path>[@version]",
		Short: "Add a dependency to apx.yaml and apx.lock",
		Long: `Add a schema module dependency to the project.

The dependency is added to apx.yaml and the version is locked in apx.lock.

Examples:
  apx add proto/payments/ledger/v1@v1.2.3
  apx add proto/payments/wallet/v1         # Uses latest version
  apx add openapi/customer/accounts/v2@v2.0.0`,
		Args: cobra.ExactArgs(1),
		RunE: addAction,
	}
	cmd.Flags().StringP("catalog", "c", "", "Path or URL to catalog file (default: catalog_url from apx.yaml, then catalog/catalog.yaml)")
	return cmd
}

func addAction(cmd *cobra.Command, args []string) error {
	arg := args[0]

	var modulePath, version string
	if strings.Contains(arg, "@") {
		parts := strings.SplitN(arg, "@", 2)
		modulePath = parts[0]
		version = parts[1]
	} else {
		modulePath = arg
	}

	mgr := config.NewDependencyManager("apx.yaml", "apx.lock", resolveSourceRepo(cmd))

	// Look up the catalog to see if this is an external API
	var provenance *config.ExternalProvenance
	catalogPath, _ := cmd.Flags().GetString("catalog")
	if catalogPath == "" {
		catalogPath = resolveCatalogPath(cmd)
	}
	gen := catalog.NewGenerator(catalogPath)
	cat, err := gen.Load()
	if err == nil {
		for _, m := range cat.Modules {
			if m.ID == modulePath && m.Origin != "" {
				provenance = &config.ExternalProvenance{
					Origin:       m.Origin,
					ManagedRepo:  m.ManagedRepo,
					UpstreamRepo: m.UpstreamRepo,
					UpstreamPath: m.UpstreamPath,
					ImportMode:   m.ImportMode,
				}
				break
			}
		}
	}

	if err := mgr.AddWithProvenance(modulePath, version, provenance); err != nil {
		ui.Error("Failed to add dependency: %v", err)
		return err
	}

	if version != "" {
		ui.Success("Added dependency: %s@%s", modulePath, version)
	} else {
		ui.Success("Added dependency: %s (latest version)", modulePath)
	}

	if provenance != nil {
		ui.Info("  Source: %s (%s, %s)", provenance.ManagedRepo, provenance.Origin, provenance.ImportMode)
	}

	return nil
}
