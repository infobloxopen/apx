package commands

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/infobloxopen/apx/internal/language"
	"github.com/infobloxopen/apx/internal/site"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newCatalogSiteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "site",
		Short: "API catalog explorer site",
	}
	cmd.AddCommand(newCatalogSiteGenerateCmd())
	cmd.AddCommand(newCatalogSiteServeCmd())
	return cmd
}

func newCatalogSiteGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a static API catalog explorer site",
		Long: `Generate a complete static website from catalog.yaml that lets teams
browse all APIs, filter by format/lifecycle/domain, and view language-specific
import coordinates.

The generated site can be deployed directly to GitHub Pages. All HTML, CSS, and
JavaScript assets are embedded in the APX binary — no additional tools required.

Example workflow for a canonical API repository:

  apx catalog generate            # build catalog.yaml from git tags
  apx catalog site generate       # generate the static site

The output directory contains a self-contained static site ready for deployment.`,
		RunE: catalogSiteGenerateAction,
	}

	cmd.Flags().StringP("output", "o", "_site", "output directory for the generated site")
	cmd.Flags().String("catalog", "", "path or URL to catalog.yaml (default: same resolution as apx search)")
	cmd.Flags().String("base-path", "", "URL base path for asset references (e.g. /catalog)")

	return cmd
}

func catalogSiteGenerateAction(cmd *cobra.Command, args []string) error {
	outputDir, _ := cmd.Flags().GetString("output")
	catalogFlag, _ := cmd.Flags().GetString("catalog")
	basePath, _ := cmd.Flags().GetString("base-path")

	// Resolve context from config (same helpers as show/search commands).
	sourceRepo := resolveSourceRepo(cmd)
	importRoot := resolveImportRoot(cmd)
	org := resolveOrg(cmd)

	// Resolve and load catalog.
	src := resolveCatalogSource(cmd, catalogFlag)
	cat, err := src.Load()
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	if len(cat.Modules) == 0 {
		ui.Warning("Catalog has no modules — generating empty site")
	}

	// Override org/repo/importRoot from catalog if not available from config.
	if org == "" && cat.Org != "" {
		org = cat.Org
	}
	if sourceRepo == "github.com/<org>/<repo>" && cat.Org != "" && cat.Repo != "" {
		sourceRepo = fmt.Sprintf("github.com/%s/%s", cat.Org, cat.Repo)
	}
	if importRoot == "" && cat.ImportRoot != "" {
		importRoot = cat.ImportRoot
	}

	ui.Info("Building site data for %d APIs...", len(cat.Modules))
	ui.Info("  Org: %s  Repo: %s", cat.Org, cat.Repo)
	if importRoot != "" {
		ui.Info("  Import root: %s", importRoot)
	}
	ui.Info("  Languages: %d plugins registered", len(language.All()))

	data := site.BuildSiteData(cat, sourceRepo, importRoot, org)

	ui.Info("Generating site to %s...", outputDir)
	if err := site.Generate(data, outputDir, basePath); err != nil {
		return fmt.Errorf("generating site: %w", err)
	}

	ui.Success("Catalog site generated: %s (%d APIs)", outputDir, len(data.APIs))
	return nil
}

// defaultServePort is 10451 — an homage to Fahrenheit 451.
const defaultServePort = 10451

func newCatalogSiteServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Generate and serve the catalog site locally",
		Long: `Generate the catalog site and serve it over HTTP for local preview.

This is a convenience command that combines 'catalog site generate' with a
local HTTP server. It generates the site to a temporary directory, starts
a server on port 10451, and opens the browser.

Press Ctrl+C to stop the server.`,
		RunE: catalogSiteServeAction,
	}

	cmd.Flags().IntP("port", "p", defaultServePort, "port to serve on")
	cmd.Flags().String("catalog", "", "path or URL to catalog.yaml (default: same resolution as apx search)")
	cmd.Flags().Bool("no-open", false, "don't open the browser automatically")

	return cmd
}

func catalogSiteServeAction(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	catalogFlag, _ := cmd.Flags().GetString("catalog")
	noOpen, _ := cmd.Flags().GetBool("no-open")

	// Generate to a temp directory.
	tmpDir, err := os.MkdirTemp("", "apx-catalog-site-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Resolve context.
	sourceRepo := resolveSourceRepo(cmd)
	importRoot := resolveImportRoot(cmd)
	org := resolveOrg(cmd)

	// Load catalog.
	src := resolveCatalogSource(cmd, catalogFlag)
	cat, err := src.Load()
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	// Override from catalog.
	if org == "" && cat.Org != "" {
		org = cat.Org
	}
	if sourceRepo == "github.com/<org>/<repo>" && cat.Org != "" && cat.Repo != "" {
		sourceRepo = fmt.Sprintf("github.com/%s/%s", cat.Org, cat.Repo)
	}
	if importRoot == "" && cat.ImportRoot != "" {
		importRoot = cat.ImportRoot
	}

	ui.Info("Building site data for %d APIs...", len(cat.Modules))
	data := site.BuildSiteData(cat, sourceRepo, importRoot, org)

	// Generate with empty base path (served from root).
	if err := site.Generate(data, tmpDir, ""); err != nil {
		return fmt.Errorf("generating site: %w", err)
	}

	// Resolve the actual directory to serve (filepath.Clean for safety).
	serveDir := filepath.Clean(tmpDir)

	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listening on port %d: %w", port, err)
	}

	url := fmt.Sprintf("http://localhost:%d", port)
	ui.Success("Serving %d APIs at %s", len(data.APIs), url)
	ui.Info("Press Ctrl+C to stop")

	// Open browser unless --no-open.
	if !noOpen {
		openBrowser(url)
	}

	// Serve until interrupted.
	server := &http.Server{
		Handler: http.FileServer(http.Dir(serveDir)),
	}
	return server.Serve(listener)
}

// openBrowser opens the given URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}
