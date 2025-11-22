package commands

import (
	"fmt"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/urfave/cli/v2"
)

// SearchCommand returns the search command for discovering APIs in the catalog
func SearchCommand() *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "Search for APIs in the catalog",
		Description: `Search the canonical repository catalog for available APIs.

Examples:
  apx search                    # List all APIs
  apx search ledger             # Search for APIs matching "ledger"
  apx search --format=proto     # Search for proto APIs only
  apx search payment --format=proto`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Filter by schema format (proto, openapi, avro, jsonschema, parquet)",
			},
			&cli.StringFlag{
				Name:    "catalog",
				Aliases: []string{"c"},
				Usage:   "Path to catalog file",
				Value:   "catalog/catalog.yaml",
			},
		},
		Action: searchAction,
	}
}

func searchAction(c *cli.Context) error {
	query := c.Args().First()
	format := c.String("format")
	catalogPath := c.String("catalog")

	// Load catalog
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

	// Display results
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
