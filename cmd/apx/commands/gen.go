package commands

import (
	"fmt"

	"github.com/infobloxopen/apx/internal/config"
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
		Long:  "Generate code for the specified language.\nSupported languages: go, python, java",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  genAction,
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

	dm := config.NewDependencyManager("apx.yaml", "apx.lock")
	deps, err := dm.List()
	if err != nil {
		return fmt.Errorf("failed to list dependencies: %w", err)
	}

	if len(deps) == 0 {
		ui.Info("No dependencies found in apx.lock")
		return nil
	}

	mgr := overlay.NewManager(".")
	for _, dep := range deps {
		ui.Info("Creating overlay for %s...", dep.ModulePath)
		if _, err := mgr.Create(dep.ModulePath, opts.Language); err != nil {
			return fmt.Errorf("failed to create overlay: %w", err)
		}
	}

	if opts.Language == "go" {
		if err := mgr.Sync(); err != nil {
			return fmt.Errorf("failed to sync go.work: %w", err)
		}
	}

	ui.Success("Code generation completed successfully")
	return nil
}
