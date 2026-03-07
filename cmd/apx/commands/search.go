package commands

import (
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
  apx search payment --format=proto`,
		Args: cobra.MaximumNArgs(1),
		RunE: searchAction,
	}
	cmd.Flags().StringP("format", "f", "", "Filter by schema format (proto, openapi, avro, jsonschema, parquet)")
	cmd.Flags().StringP("catalog", "c", "catalog/catalog.yaml", "Path to catalog file")
	return cmd
}

func searchAction(cmd *cobra.Command, args []string) error {
	query := ""
	if len(args) > 0 {
		query = args[0]
	}
	format, _ := cmd.Flags().GetString("format")
	catalogPath, _ := cmd.Flags().GetString("catalog")

	gen := catalog.NewGenerator(catalogPath)
	modules, err := catalog.SearchModules(gen, query, format)
	if err != nil {
		ui.Error("Failed to search catalog: %v", err)
		return err
	}

	if len(modules) == 0 {
		ui.Info("No APIs found matching query")
		return nil
	}

	ui.Info("Found %d API(s):", len(modules))
	fmt.Println()
	for _, module := range modules {
		fmt.Printf("  %s\n", module.Name)
		if module.Description != "" {
			fmt.Printf("    Description: %s\n", module.Description)
		}
		fmt.Printf("    Format: %s\n", module.Format)
		if module.Version != "" {
			fmt.Printf("    Version: %s\n", module.Version)
		}
		if len(module.Owners) > 0 {
			fmt.Printf("    Owners: %v\n", module.Owners)
		}
		fmt.Println()
	}

	return nil
}
