package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect API identity, releases, and derived coordinates",
		Long: `Inspect commands let you query the canonical identity model:

  identity  - Show the full identity for an API ID
  release   - Show identity for a specific release version

Examples:
  apx inspect identity proto/payments/ledger/v1
  apx inspect release proto/payments/ledger/v1@v1.0.0-beta.1`,
	}

	cmd.AddCommand(newInspectIdentityCmd())
	cmd.AddCommand(newInspectReleaseCmd())

	return cmd
}

func newInspectIdentityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "identity <api-id>",
		Short: "Show canonical identity for an API",
		Long: `Display the full canonical identity for an API, including derived
Go module/import paths, source repo path, and lifecycle.

The API ID format is: <format>/<domain>/<name>/<line>

Examples:
  apx inspect identity proto/payments/ledger/v1
  apx inspect identity openapi/billing/invoices/v2
  apx inspect identity --source-repo github.com/acme/apis proto/payments/ledger/v1`,
		Args: cobra.ExactArgs(1),
		RunE: inspectIdentityAction,
	}
	cmd.Flags().String("source-repo", "", "Source repository (defaults to github.com/<org>/<repo> from apx.yaml)")
	cmd.Flags().String("lifecycle", "", "Lifecycle state (experimental, beta, stable, deprecated, sunset)")
	return cmd
}

func newInspectReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release <api-id>@<version>",
		Short: "Show identity for a specific API release",
		Long: `Display the canonical identity for a specific version of an API.

The format is: <api-id>@<version>

Examples:
  apx inspect release proto/payments/ledger/v1@v1.0.0-beta.1
  apx inspect release proto/payments/ledger/v2@v2.0.0`,
		Args: cobra.ExactArgs(1),
		RunE: inspectReleaseAction,
	}
	cmd.Flags().String("source-repo", "", "Source repository (defaults to github.com/<org>/<repo> from apx.yaml)")
	return cmd
}

func inspectIdentityAction(cmd *cobra.Command, args []string) error {
	apiID := args[0]
	sourceRepo, _ := cmd.Flags().GetString("source-repo")
	lifecycle, _ := cmd.Flags().GetString("lifecycle")

	if sourceRepo == "" {
		sourceRepo = resolveSourceRepo(cmd)
	}

	api, source, release, langs, err := config.BuildIdentityBlock(apiID, sourceRepo, lifecycle, "")
	if err != nil {
		return err
	}

	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonOut {
		return printIdentityJSON(api, source, release, langs)
	}

	report := config.FormatIdentityReport(api, source, release, langs)
	fmt.Print(report)
	return nil
}

func inspectReleaseAction(cmd *cobra.Command, args []string) error {
	input := args[0]

	// Parse "api-id@version"
	atIdx := strings.LastIndex(input, "@")
	if atIdx < 0 {
		return fmt.Errorf("expected format <api-id>@<version>, got %q", input)
	}
	apiID := input[:atIdx]
	version := input[atIdx+1:]

	if apiID == "" || version == "" {
		return fmt.Errorf("both API ID and version are required in <api-id>@<version>")
	}

	sourceRepo, _ := cmd.Flags().GetString("source-repo")
	if sourceRepo == "" {
		sourceRepo = resolveSourceRepo(cmd)
	}

	// Derive lifecycle from version prerelease if not explicit
	lifecycle := ""
	if strings.Contains(version, "-alpha") {
		lifecycle = "experimental"
	} else if strings.Contains(version, "-beta") {
		lifecycle = "beta"
	} else if strings.Contains(version, "-rc") {
		lifecycle = "beta"
	}

	api, source, release, langs, err := config.BuildIdentityBlock(apiID, sourceRepo, lifecycle, version)
	if err != nil {
		return err
	}

	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonOut {
		return printIdentityJSON(api, source, release, langs)
	}

	report := config.FormatIdentityReport(api, source, release, langs)
	fmt.Print(report)
	return nil
}

// resolveSourceRepo tries to read org/repo from apx.yaml config and build
// the source repo string. Falls back to a placeholder.
func resolveSourceRepo(cmd *cobra.Command) string {
	configPath, _ := cmd.Root().PersistentFlags().GetString("config")
	cfg, err := config.Load(configPath)
	if err == nil && cfg.Org != "" && cfg.Repo != "" {
		return fmt.Sprintf("github.com/%s/%s", cfg.Org, cfg.Repo)
	}
	return "github.com/<org>/<repo>"
}

func printIdentityJSON(api *config.APIIdentity, source *config.SourceIdentity, release *config.ReleaseInfo, langs map[string]config.LanguageCoords) error {
	out := map[string]interface{}{
		"api":    api,
		"source": source,
	}
	if release != nil {
		out["releases"] = release
		out["tag"] = config.DeriveTag(api.ID, release.Current)
	}
	if len(langs) > 0 {
		out["languages"] = langs
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func newExplainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "explain",
		Short: "Explain derived paths and coordinates for an API",
		Long: `Explain commands show how APX derives language-specific paths from an API ID.

Examples:
  apx explain go-path proto/payments/ledger/v1`,
	}

	cmd.AddCommand(newExplainGoPathCmd())

	return cmd
}

func newExplainGoPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "go-path <api-id>",
		Short: "Explain Go module and import path derivation",
		Long: `Show how APX derives Go module and import paths from an API identity.

This explains the Go module versioning rules:
  - v1: module path has no version suffix, import path includes /v1
  - v2+: both module and import path include /vN suffix

Examples:
  apx explain go-path proto/payments/ledger/v1
  apx explain go-path proto/payments/ledger/v2`,
		Args: cobra.ExactArgs(1),
		RunE: explainGoPathAction,
	}
	cmd.Flags().String("source-repo", "", "Source repository (defaults to github.com/<org>/<repo> from apx.yaml)")
	return cmd
}

func explainGoPathAction(cmd *cobra.Command, args []string) error {
	apiID := args[0]

	sourceRepo, _ := cmd.Flags().GetString("source-repo")
	if sourceRepo == "" {
		sourceRepo = resolveSourceRepo(cmd)
	}

	api, err := config.ParseAPIID(apiID)
	if err != nil {
		return err
	}

	goMod, err := config.DeriveGoModule(sourceRepo, api)
	if err != nil {
		return err
	}

	goImport, err := config.DeriveGoImport(sourceRepo, api)
	if err != nil {
		return err
	}

	major, _ := config.LineMajor(api.Line)

	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonOut {
		data, err := json.MarshalIndent(map[string]interface{}{
			"api_id":      apiID,
			"source_repo": sourceRepo,
			"go_module":   goMod,
			"go_import":   goImport,
			"major":       major,
			"rule":        goModuleRule(major),
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	ui.Info("API ID:      %s", apiID)
	ui.Info("Source repo: %s", sourceRepo)
	ui.Info("")
	ui.Info("Go module:   %s", goMod)
	ui.Info("Go import:   %s", goImport)
	ui.Info("")
	ui.Info("Rule: %s", goModuleRule(major))
	ui.Info("")

	if major == 1 {
		ui.Info("For v1, the Go module path does NOT include a version suffix.")
		ui.Info("The import path adds /v1 for the package directory.")
		ui.Info("")
		ui.Info("Usage in go.mod:")
		ui.Info("  require %s v1.x.x", goMod)
		ui.Info("")
		ui.Info("Usage in Go code:")
		ui.Info("  import \"%s\"", goImport)
	} else {
		ui.Info("For v%d, both the Go module and import path include /v%d.", major, major)
		ui.Info("This follows Go module major version suffix rules.")
		ui.Info("")
		ui.Info("Usage in go.mod:")
		ui.Info("  require %s v%d.x.x", goMod, major)
		ui.Info("")
		ui.Info("Usage in Go code:")
		ui.Info("  import \"%s\"", goImport)
	}

	return nil
}

func goModuleRule(major int) string {
	if major == 1 {
		return "v1: module path has no version suffix; import path includes /v1 directory"
	}
	return fmt.Sprintf("v%d+: both module and import path include /v%d suffix (Go major version rule)", major, major)
}
