package commands

import (
	"encoding/json"
	"fmt"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newCatalogResolveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve <resource-type>",
		Short: "Resolve an AIP-122 resource type to the module that serves it",
		Long: `Resolve a resource type (from a google.api.resource_reference.type, e.g.
"iam.example.com/User") to the catalog module that serves it, returning path
coordinates: module ID, domain, API line, version, and lifecycle.

apx resolves type -> module only. The API path is derivable from the domain and
API line; the concrete network host stays consumer/environment-supplied (apx is
a schema catalog, not a service registry).

Resolution fails loud:
  - an unknown type (no module serves it) is an error
  - an ambiguous type (more than one module claims it) is an error, listing the
    claimants — apx never silently picks one

A type that is declared but has no serving surface resolves successfully with a
"no serving surface" warning.

The catalog source is resolved like other read commands: --catalog first, then
apx.yaml / global config, then local catalog/catalog.yaml.

Examples:
  apx catalog resolve iam.example.com/User
  apx --json catalog resolve iam.example.com/User
  apx catalog resolve --catalog=https://raw.githubusercontent.com/org/apis/main/catalog/catalog.yaml iam.example.com/User`,
		Args: cobra.ExactArgs(1),
		RunE: catalogResolveAction,
	}
	cmd.Flags().StringP("catalog", "c", "", "Path or URL to catalog file (default: catalog_url from apx.yaml, then catalog/catalog.yaml)")
	return cmd
}

func catalogResolveAction(cmd *cobra.Command, args []string) error {
	resourceType := args[0]
	catalogPath, _ := cmd.Flags().GetString("catalog")

	src := resolveCatalogSource(cmd, catalogPath)
	cat, err := src.Load()
	if err != nil {
		return fmt.Errorf("failed to load catalog from %s: %w", src.Name(), err)
	}

	res, err := catalog.ResolveType(cat, resourceType)
	if err != nil {
		return err
	}

	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonOut {
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		if res.Warning != "" {
			ui.Warning("%s", res.Warning)
		}
		return nil
	}

	ui.Info("Type:       %s", res.Type)
	ui.Info("Module:     %s", res.ModuleID)
	if res.Domain != "" {
		ui.Info("Domain:     %s", res.Domain)
	}
	if res.APILine != "" {
		ui.Info("API line:   %s", res.APILine)
	}
	if res.Version != "" {
		ui.Info("Version:    %s", res.Version)
	}
	if res.Lifecycle != "" {
		ui.Info("Lifecycle:  %s", res.Lifecycle)
	}
	if res.Origin != "" {
		ui.Info("Origin:     %s", res.Origin)
	}
	if res.ManagedRepo != "" {
		ui.Info("Managed by: %s", res.ManagedRepo)
	}
	if res.Warning != "" {
		ui.Warning("%s", res.Warning)
	}
	return nil
}
