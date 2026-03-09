package commands

import (
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/language"
	"github.com/infobloxopen/apx/internal/overlay"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

// GenerateOptions holds options for code generation
type GenerateOptions struct {
	Language  string
	Path      string
	OutputDir string
	Clean     bool
	Manifest  bool
}

func newGenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen <lang> [path]",
		Short: "Generate code",
		Long: fmt.Sprintf("Generate code for the specified language.\nSupported languages: %s",
			strings.Join(language.Names(), ", ")),
		Args: cobra.RangeArgs(1, 2),
		RunE: genAction,
	}
	cmd.Flags().String("out", "", "output directory")
	cmd.Flags().Bool("clean", false, "clean output directory before generation")
	cmd.Flags().Bool("manifest", false, "emit generation manifest")
	return cmd
}

func genAction(cmd *cobra.Command, args []string) error {
	lang := args[0]
	path := "."
	if len(args) > 1 {
		path = args[1]
	}

	outDir, _ := cmd.Flags().GetString("out")
	clean, _ := cmd.Flags().GetBool("clean")
	manifest, _ := cmd.Flags().GetBool("manifest")

	opts := GenerateOptions{
		Language:  lang,
		Path:      path,
		OutputDir: outDir,
		Clean:     clean,
		Manifest:  manifest,
	}

	return generateCode(opts)
}

func generateCode(opts GenerateOptions) error {
	ui.Info("Generating %s code from dependencies...", opts.Language)

	// Look up the plugin for scaffolding / post-gen hooks.
	plugin := language.Get(opts.Language)

	dm := config.NewDependencyManager("apx.yaml", "apx.lock", "")
	deps, err := dm.List()
	if err != nil {
		return fmt.Errorf("failed to list dependencies: %w", err)
	}

	if len(deps) == 0 {
		ui.Info("No dependencies found in apx.lock")
		return nil
	}

	cfg, _ := config.LoadRaw("")

	mgr := overlay.NewManager(".")
	for _, dep := range deps {
		ui.Info("Creating overlay for %s...", dep.ModulePath)
		ov, err := mgr.Create(dep.ModulePath, opts.Language)
		if err != nil {
			return fmt.Errorf("failed to create overlay: %w", err)
		}

		// Run Scaffolder if the plugin implements it.
		if scaffolder, ok := plugin.(language.Scaffolder); ok {
			api, parseErr := config.ParseAPIID(dep.ModulePath)
			if parseErr != nil {
				return fmt.Errorf("parsing API ID %s: %w", dep.ModulePath, parseErr)
			}
			org := ""
			importRoot := ""
			if cfg != nil {
				org = cfg.Org
				importRoot = cfg.ImportRoot
			}
			ctx := language.DerivationContext{
				SourceRepo: resolveSourceRepoFromConfig(cfg),
				ImportRoot: importRoot,
				Org:        org,
				API:        api,
			}
			if ctx.Org != "" {
				if err := scaffolder.Scaffold(ov.Path, ctx); err != nil {
					return fmt.Errorf("scaffolding %s for %s: %w", opts.Language, dep.ModulePath, err)
				}
			}
		}
	}

	// Run PostGenHook if the plugin implements it.
	if hook, ok := plugin.(language.PostGenHook); ok {
		if err := hook.PostGen("."); err != nil {
			return fmt.Errorf("post-generation hook for %s: %w", opts.Language, err)
		}
	}

	ui.Success("Code generation completed successfully")
	return nil
}

// resolveSourceRepoFromConfig builds the source repo string from a loaded config.
func resolveSourceRepoFromConfig(cfg *config.Config) string {
	if cfg != nil && cfg.Org != "" && cfg.Repo != "" {
		return fmt.Sprintf("github.com/%s/%s", cfg.Org, cfg.Repo)
	}
	return "github.com/<org>/<repo>"
}
