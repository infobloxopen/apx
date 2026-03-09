package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/language"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <api-id>",
		Short: "Show full identity and catalog data for an API",
		Long: `Display the full identity, derived coordinates, and catalog release data
for a given API ID.

This merges two data sources:
  1. Derived fields computed from the API ID (Go module, import path, tag pattern)
  2. Catalog fields read from catalog.yaml (latest stable/prerelease, lifecycle, owners)

The catalog can be a local file path or a remote URL (http:// or https://).
When --catalog is not specified, APX checks catalog_url from apx.yaml first,
then falls back to catalog/catalog.yaml.

The API ID format is: <format>/<domain>/<name>/<line>

Examples:
  apx show proto/payments/ledger/v1
  apx show openapi/billing/invoices/v2
  apx show --source-repo github.com/acme/apis proto/payments/ledger/v1
  apx --json show proto/payments/ledger/v1`,
		Args: cobra.ExactArgs(1),
		RunE: showAction,
	}

	cmd.Flags().String("source-repo", "", "Source repository (defaults to github.com/<org>/<repo> from apx.yaml)")
	cmd.Flags().String("catalog", "", "Path or URL to catalog.yaml (default: catalog_url from apx.yaml, then catalog/catalog.yaml)")

	return cmd
}

// showInfo holds everything we know about an API for display or JSON output.
type showInfo struct {
	API        *config.APIIdentity              `json:"api"`
	Source     *config.SourceIdentity           `json:"source"`
	Release    *showRelease                     `json:"release,omitempty"`
	Languages  map[string]config.LanguageCoords `json:"languages,omitempty"`
	Catalog    *showCatalog                     `json:"catalog,omitempty"`
	Lifecycle  *showLifecycle                   `json:"lifecycle,omitempty"`
	Provenance *showProvenance                  `json:"provenance,omitempty"`
}

type showRelease struct {
	LatestStable     string `json:"latest_stable,omitempty"`
	LatestPrerelease string `json:"latest_prerelease,omitempty"`
	TagPattern       string `json:"tag_pattern"`
}

type showCatalog struct {
	Lifecycle string   `json:"lifecycle,omitempty"`
	Owners    []string `json:"owners,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Version   string   `json:"version,omitempty"`
}

type showLifecycle struct {
	State                string `json:"state"`
	CompatibilityLevel   string `json:"compatibility_level"`
	CompatibilitySummary string `json:"compatibility_summary"`
	BreakingPolicy       string `json:"breaking_policy"`
	ProductionUse        string `json:"production_use"`
}

type showProvenance struct {
	Origin       string `json:"origin"`
	ImportMode   string `json:"import_mode"`
	ManagedRepo  string `json:"managed_repo"`
	ManagedPath  string `json:"managed_path"`
	UpstreamRepo string `json:"upstream_repo"`
	UpstreamPath string `json:"upstream_path"`
}

func showAction(cmd *cobra.Command, args []string) error {
	apiID := args[0]

	sourceRepo, _ := cmd.Flags().GetString("source-repo")
	catalogPath, _ := cmd.Flags().GetString("catalog")

	// Resolve source repo from config if not specified
	if sourceRepo == "" {
		sourceRepo = resolveSourceRepo(cmd)
	}
	importRoot := resolveImportRoot(cmd)
	org := resolveOrg(cmd)

	// Resolve catalog source
	src := resolveCatalogSource(cmd, catalogPath)

	// Build identity from API ID
	api, err := config.ParseAPIID(apiID)
	if err != nil {
		return err
	}

	source := &config.SourceIdentity{
		Repo: sourceRepo,
		Path: config.DeriveSourcePath(apiID),
	}

	langs, err := language.DeriveAllCoords(language.DerivationContext{
		SourceRepo: sourceRepo,
		ImportRoot: importRoot,
		Org:        org,
		API:        api,
	})
	if err != nil {
		return err
	}

	// Build output info
	info := &showInfo{
		API:       api,
		Source:    source,
		Languages: langs,
		Release: &showRelease{
			TagPattern: apiID + "/v*",
		},
	}

	// Try to enrich from catalog
	catalogFound := false
	cat, err := src.Load()
	if err == nil && len(cat.Modules) > 0 {
		for _, m := range cat.Modules {
			if m.ID == apiID {
				catalogFound = true
				info.Release.LatestStable = m.LatestStable
				info.Release.LatestPrerelease = m.LatestPrerelease

				info.Catalog = &showCatalog{
					Lifecycle: m.Lifecycle,
					Owners:    m.Owners,
					Tags:      m.Tags,
					Version:   m.Version,
				}

				// Add provenance for external APIs
				if m.Origin != "" {
					info.Provenance = &showProvenance{
						Origin:       m.Origin,
						ImportMode:   m.ImportMode,
						ManagedRepo:  m.ManagedRepo,
						ManagedPath:  m.Path,
						UpstreamRepo: m.UpstreamRepo,
						UpstreamPath: m.UpstreamPath,
					}
					// Override source for external APIs
					source.Repo = m.ManagedRepo
					source.Path = m.Path
				}

				// Enrich API lifecycle from catalog if not set
				if api.Lifecycle == "" && m.Lifecycle != "" {
					api.Lifecycle = m.Lifecycle
				}
				break
			}
		}
	}

	// JSON output
	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")

	// Build lifecycle section if lifecycle is known
	if api.Lifecycle != "" {
		promise := config.DeriveCompatibilityPromise(api.Line, api.Lifecycle)
		info.Lifecycle = &showLifecycle{
			State:                config.NormalizeLifecycle(api.Lifecycle),
			CompatibilityLevel:   promise.Level,
			CompatibilitySummary: promise.Summary,
			BreakingPolicy:       promise.BreakingPolicy,
			ProductionUse:        config.ProductionRecommendation(api.Lifecycle),
		}
	}

	if jsonOut {
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	// Text output
	printShowText(info, catalogFound)
	return nil
}

func printShowText(info *showInfo, catalogFound bool) {
	api := info.API
	source := info.Source

	ui.Info("API:        %s", api.ID)
	ui.Info("Format:     %s", api.Format)
	ui.Info("Domain:     %s", api.Domain)
	ui.Info("Name:       %s", api.Name)
	ui.Info("Line:       %s", api.Line)

	if api.Lifecycle != "" {
		ui.Info("Lifecycle:  %s", config.NormalizeLifecycle(api.Lifecycle))
	}

	// Provenance section for external APIs
	if info.Provenance != nil {
		ui.Info("")
		ui.Info("Provenance")
		ui.Info("  Origin:         %s", info.Provenance.Origin)
		ui.Info("  Import mode:    %s", info.Provenance.ImportMode)
		ui.Info("  Managed repo:   %s", info.Provenance.ManagedRepo)
		ui.Info("  Managed path:   %s", info.Provenance.ManagedPath)
		ui.Info("  Upstream repo:  %s", info.Provenance.UpstreamRepo)
		ui.Info("  Upstream path:  %s", info.Provenance.UpstreamPath)
	}

	if source != nil {
		ui.Info("Source:     %s/%s", source.Repo, source.Path)
	}

	// Lifecycle details
	if info.Lifecycle != nil {
		ui.Info("")
		ui.Info("Compatibility")
		ui.Info("  Level:    %s", info.Lifecycle.CompatibilityLevel)
		ui.Info("  Promise:  %s", info.Lifecycle.CompatibilitySummary)
		ui.Info("  Breaking: %s", info.Lifecycle.BreakingPolicy)
		ui.Info("  Use:      %s", info.Lifecycle.ProductionUse)
	}

	// Release info
	if info.Release != nil {
		ui.Info("")
		if info.Release.LatestStable != "" {
			ui.Info("Latest stable:      %s", info.Release.LatestStable)
		} else {
			ui.Info("Latest stable:      none")
		}
		if info.Release.LatestPrerelease != "" {
			ui.Info("Latest prerelease:  %s", info.Release.LatestPrerelease)
		}
	}

	// Language coordinates — iterate plugins in display order
	if len(info.Languages) > 0 {
		ui.Info("")
		for _, p := range language.All() {
			coords, ok := info.Languages[p.Name()]
			if !ok {
				continue
			}
			for _, rl := range p.ReportLines(coords) {
				ui.Info("%-12s%s", rl.Label+":", rl.Value)
			}
		}
	}

	// Owners from catalog
	if info.Catalog != nil && len(info.Catalog.Owners) > 0 {
		ui.Info("Owners:     %s", strings.Join(info.Catalog.Owners, ", "))
	}

	// Tags from catalog
	if info.Catalog != nil && len(info.Catalog.Tags) > 0 {
		ui.Info("Tags:       %s", strings.Join(info.Catalog.Tags, ", "))
	}

	if !catalogFound {
		ui.Info("")
		ui.Warning("No catalog data found. Run `apx catalog generate` for release data.")
	}
}
