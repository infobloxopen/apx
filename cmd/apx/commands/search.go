package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newCatalogSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for APIs in the catalog",
		Long: `Search the canonical repository catalog for available APIs.

The catalog can be a local file path or a remote URL (http:// or https://).
When --catalog is not specified, APX resolves the catalog source in order:
  1. catalog_registries from apx.yaml
  2. Auto-discover from org in apx.yaml (queries GHCR packages API)
  3. Known orgs/repos from ~/.config/apx/config.yaml (seeded by apx auth login)
  4. catalog_url from apx.yaml
  5. Local catalog/catalog.yaml

Examples:
  apx catalog search                    # List all APIs
  apx catalog search ledger             # Search for APIs matching "ledger"
  apx catalog search --format=proto     # Search for proto APIs only
  apx catalog search --lifecycle=beta   # Search for beta APIs
  apx catalog search --domain=payments  # Search by domain
  apx catalog search --tag=public       # Search by tag
  apx catalog search payment --format=proto --lifecycle=stable
  apx catalog search --catalog=https://raw.githubusercontent.com/org/apis/main/catalog/catalog.yaml`,
		Args: cobra.MaximumNArgs(1),
		RunE: searchAction,
	}
	cmd.Flags().StringP("format", "f", "", "Filter by schema format (proto, openapi, avro, jsonschema, parquet)")
	cmd.Flags().StringP("lifecycle", "l", "", "Filter by lifecycle (experimental, beta, stable, deprecated, sunset)")
	cmd.Flags().StringP("domain", "d", "", "Filter by domain (e.g. payments, billing)")
	cmd.Flags().String("api-line", "", "Filter by API line (e.g. v1, v2)")
	cmd.Flags().String("origin", "", "Filter by origin: first-party, external, forked")
	cmd.Flags().String("tag", "", "Filter by tag")
	cmd.Flags().StringP("catalog", "c", "", "Path or URL to catalog file (default: catalog_url from apx.yaml, then catalog/catalog.yaml)")
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
	tag, _ := cmd.Flags().GetString("tag")
	catalogPath, _ := cmd.Flags().GetString("catalog")

	// Resolve catalog source: explicit flag > registry sources > local default
	src := resolveCatalogSource(cmd, catalogPath)
	gen := catalog.NewGenerator("") // only used for search API compat
	gen.Source = src

	modules, err := catalog.SearchModulesOpts(gen, catalog.SearchOptions{
		Query:     query,
		Format:    format,
		Lifecycle: lifecycle,
		Domain:    domain,
		APILine:   apiLine,
		Origin:    origin,
		Tag:       tag,
	})
	if err != nil {
		// If the error is auth-related, suggest running apx auth login.
		errStr := err.Error()
		if strings.Contains(errStr, "auth") || strings.Contains(errStr, "403") ||
			strings.Contains(errStr, "token") || strings.Contains(errStr, "user app") {
			ui.Error("Catalog search failed due to authentication.")
			ui.Info("Run `apx auth login` to authenticate and discover catalogs.")
			return err
		}
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

	bold := color.New(color.Bold).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	lifecycleColor := func(lc string) string {
		switch lc {
		case "stable":
			return green(lc)
		case "beta", "preview":
			return yellow(lc)
		case "deprecated", "sunset":
			return color.New(color.FgRed).Sprint(lc)
		default:
			return dim(lc)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Found %d API(s):\n\n", len(modules))

	// Table header
	fmt.Fprintf(cmd.OutOrStdout(), "%-40s  %-8s  %-14s  %-13s  %-10s  %s\n",
		bold("API"), bold("FORMAT"), bold("VERSION"), bold("LIFECYCLE"), bold("ORIGIN"), bold("SOURCE"))
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n",
		dim("────────────────────────────────────────  ────────  ──────────────  ─────────────  ──────────  ──────────────────────────────"))

	for _, m := range modules {
		version := m.Version
		if version == "" {
			version = dim("(none)")
		}

		lifecycle := m.Lifecycle
		if lifecycle != "" {
			lifecycle = lifecycleColor(lifecycle)
		} else {
			lifecycle = dim("—")
		}

		origin := ""
		if m.Origin != "" {
			origin = dim(m.Origin)
		} else {
			origin = dim("local")
		}

		source := ""
		if m.ManagedRepo != "" {
			// Show just org/repo, not the full github.com/ prefix
			source = m.ManagedRepo
			if strings.HasPrefix(source, "github.com/") {
				source = strings.TrimPrefix(source, "github.com/")
			}
			source = dim(source)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%-40s  %-8s  %-14s  %-13s  %-10s  %s\n",
			cyan(m.DisplayName()), m.Format, version, lifecycle, origin, source)
	}

	return nil
}

// resolveCatalogSource returns the best CatalogSource by checking:
//  1. Explicit --catalog flag (path or URL) → SourceFor
//  2. Local apx.yaml + global config → ResolveSourceWithGlobal
//  3. Global config alone (when no local apx.yaml) → ResolveSourceWithGlobal
//  4. Local catalog/catalog.yaml fallback
func resolveCatalogSource(cmd *cobra.Command, catalogFlag string) catalog.CatalogSource {
	// 1. Explicit flag always wins
	if catalogFlag != "" {
		return catalog.SourceFor(catalogFlag)
	}

	// Load global config (always attempted, regardless of local config)
	globalCfg, _ := config.LoadGlobal()

	// 2. Try config-based resolution (registries, auto-discover, global, catalog_url)
	configPath, _ := cmd.Root().PersistentFlags().GetString("config")
	cfg, err := config.LoadRaw(configPath)
	if err == nil {
		return catalog.ResolveSourceWithGlobal(cfg, globalCfg)
	}

	// 3. No local config — still try global config
	if globalCfg != nil && len(globalCfg.Orgs) > 0 {
		return catalog.ResolveSourceWithGlobal(&config.Config{}, globalCfg)
	}

	// 4. Fallback to local file
	return &catalog.LocalSource{Path: "catalog/catalog.yaml"}
}
