package commands

import (
	"encoding/json"
	"fmt"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for APIs in the catalog",
		Long: `Search the canonical repository catalog for available APIs.

Examples:
  apx search                    # List all APIs
  apx search ledger             # Search for APIs matching "ledger"
  apx search --format=proto     # Search for proto APIs only
  apx search --lifecycle=beta   # Search for beta APIs
  apx search --domain=payments  # Search by domain
  apx search payment --format=proto --lifecycle=stable`,
		Args: cobra.MaximumNArgs(1),
		RunE: searchAction,
	}
	cmd.Flags().StringP("format", "f", "", "Filter by schema format (proto, openapi, avro, jsonschema, parquet)")
	cmd.Flags().StringP("lifecycle", "l", "", "Filter by lifecycle (experimental, beta, stable, deprecated, sunset)")
	cmd.Flags().StringP("domain", "d", "", "Filter by domain (e.g. payments, billing)")
	cmd.Flags().String("api-line", "", "Filter by API line (e.g. v1, v2)")
	cmd.Flags().String("origin", "", "Filter by origin: first-party, external, forked")
	cmd.Flags().StringP("catalog", "c", "catalog/catalog.yaml", "Path to catalog file")
	return cmd
}

func searchAction(cmd *cobra.Command, args []string) error {
	query := ""
	if len(args) > 0 {
		query = args[0]
	}
	format, _ := cmd.Flags().GetString("format")
	lifecycle, _ := cmd.Flags().GetString("lifecycle")
	domain, _ := cmd.Flags().GetString("domain")
	apiLine, _ := cmd.Flags().GetString("api-line")
	origin, _ := cmd.Flags().GetString("origin")
	catalogPath, _ := cmd.Flags().GetString("catalog")

	gen := catalog.NewGenerator(catalogPath)
	modules, err := catalog.SearchModulesOpts(gen, catalog.SearchOptions{
		Query:     query,
		Format:    format,
		Lifecycle: lifecycle,
		Domain:    domain,
		APILine:   apiLine,
		Origin:    origin,
	})
	if err != nil {
		ui.Error("Failed to search catalog: %v", err)
		return err
	}

	if len(modules) == 0 {
		ui.Info("No APIs found matching query")
		return nil
	}

	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonOut {
		data, err := json.MarshalIndent(modules, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	ui.Info("Found %d API(s):", len(modules))
	fmt.Println()
	for _, module := range modules {
		if module.Origin != "" {
			fmt.Printf("  %-40s [%s]\n", module.DisplayName(), module.Origin)
		} else {
			fmt.Printf("  %s\n", module.DisplayName())
		}
		if module.Description != "" {
			fmt.Printf("    Description: %s\n", module.Description)
		}
		fmt.Printf("    Format: %s\n", module.Format)
		if module.Domain != "" {
			fmt.Printf("    Domain: %s\n", module.Domain)
		}
		if module.APILine != "" {
			fmt.Printf("    Line: %s\n", module.APILine)
		}
		if module.Lifecycle != "" {
			fmt.Printf("    Lifecycle: %s\n", module.Lifecycle)
		}
		if module.Version != "" {
			fmt.Printf("    Version: %s\n", module.Version)
		}
		if module.LatestStable != "" {
			fmt.Printf("    Latest stable: %s\n", module.LatestStable)
		}
		if module.LatestPrerelease != "" {
			fmt.Printf("    Latest prerelease: %s\n", module.LatestPrerelease)
		}
		if module.Origin != "" && module.ManagedRepo != "" {
			fmt.Printf("    Managed: %s\n", module.ManagedRepo)
			fmt.Printf("    Import: %s\n", module.ImportMode)
		}
		if len(module.Owners) > 0 {
			fmt.Printf("    Owners: %v\n", module.Owners)
		}
		fmt.Println()
	}

	return nil
}
