package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/mattn/go-isatty"
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

	isTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

	bold := color.New(color.Bold).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	red := color.New(color.FgRed).SprintFunc()
	lifecycleColor := func(a ...interface{}) string {
		s := fmt.Sprint(a...)
		trimmed := strings.TrimSpace(s)
		switch trimmed {
		case "stable":
			return green(s)
		case "beta", "preview":
			return yellow(s)
		case "deprecated", "sunset":
			return red(s)
		default:
			return dim(s)
		}
	}

	// Column widths (visible characters, not bytes).
	const (
		colAPI       = 40
		colFormat    = 8
		colVersion   = 14
		colLifecycle = 13
		colOrigin    = 10
	)

	// pad returns s padded to width with spaces, then applies colorFn.
	// This ensures ANSI codes don't affect alignment.
	pad := func(s string, width int, colorFn func(...interface{}) string) string {
		padded := fmt.Sprintf("%-*s", width, s)
		if colorFn != nil {
			return colorFn(padded)
		}
		return padded
	}

	if isTTY {
		fmt.Fprintf(cmd.OutOrStdout(), "Found %d API(s):\n\n", len(modules))

		fmt.Fprintf(cmd.OutOrStdout(), "%s  %s  %s  %s  %s  %s\n",
			pad("API", colAPI, bold),
			pad("FORMAT", colFormat, bold),
			pad("VERSION", colVersion, bold),
			pad("LIFECYCLE", colLifecycle, bold),
			pad("ORIGIN", colOrigin, bold),
			bold("SOURCE"))
		fmt.Fprintf(cmd.OutOrStdout(), "%s  %s  %s  %s  %s  %s\n",
			dim(strings.Repeat("─", colAPI)),
			dim(strings.Repeat("─", colFormat)),
			dim(strings.Repeat("─", colVersion)),
			dim(strings.Repeat("─", colLifecycle)),
			dim(strings.Repeat("─", colOrigin)),
			dim(strings.Repeat("─", 30)))
	}

	for _, m := range modules {
		version := m.Version
		if version == "" {
			version = "(none)"
		}

		lifecycle := m.Lifecycle
		if lifecycle == "" {
			lifecycle = "—"
		}

		origin := m.Origin
		if origin == "" {
			origin = "local"
		}

		source := ""
		if m.ManagedRepo != "" {
			source = m.ManagedRepo
			if strings.HasPrefix(source, "github.com/") {
				source = strings.TrimPrefix(source, "github.com/")
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s  %-*s  %-*s  %s  %s  %s\n",
			pad(m.DisplayName(), colAPI, cyan),
			colFormat, m.Format,
			colVersion, version,
			pad(lifecycle, colLifecycle, lifecycleColor),
			pad(origin, colOrigin, dim),
			dim(source))
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
