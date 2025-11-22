package commands

import (
	"fmt"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/overlay"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/urfave/cli/v2"
)

// GenCommand returns the code generation command
func GenCommand() *cli.Command {
	return &cli.Command{
		Name:      "gen",
		Usage:     "Generate code",
		ArgsUsage: "<lang> [path]",
		Description: "Generate code for the specified language.\n" +
			"Supported languages: go, python, java",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "out",
				Usage: "output directory",
			},
			&cli.BoolFlag{
				Name:  "clean",
				Usage: "clean output directory before generation",
			},
			&cli.BoolFlag{
				Name:  "manifest",
				Usage: "emit generation manifest",
			},
		},
		Action: genAction,
	}
}

// GenerateOptions holds options for code generation
type GenerateOptions struct {
	Language  string
	Path      string
	OutputDir string
	Clean     bool
	Manifest  bool
}

func genAction(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("gen requires at least 1 argument: <lang>")
	}

	lang := c.Args().Get(0)
	path := c.Args().Get(1)
	if path == "" {
		path = "."
	}

	opts := GenerateOptions{
		Language:  lang,
		Path:      path,
		OutputDir: c.String("out"),
		Clean:     c.Bool("clean"),
		Manifest:  c.Bool("manifest"),
	}

	return generateCode(opts)
}

func generateCode(opts GenerateOptions) error {
	// For now, this creates overlay structure for dependencies in apx.lock
	// Full codegen implementation would use buf/openapi-generator/etc.

	ui.Info("Generating %s code from dependencies...", opts.Language)

	// Load dependencies
	dm := config.NewDependencyManager("apx.yaml", "apx.lock")
	deps, err := dm.List()
	if err != nil {
		return fmt.Errorf("failed to list dependencies: %w", err)
	}

	if len(deps) == 0 {
		ui.Info("No dependencies found in apx.lock")
		return nil
	}

	// Create overlays for each dependency
	mgr := overlay.NewManager(".")
	for _, dep := range deps {
		ui.Info("Creating overlay for %s...", dep.ModulePath)
		if _, err := mgr.Create(dep.ModulePath, opts.Language); err != nil {
			return fmt.Errorf("failed to create overlay: %w", err)
		}
	}

	// For Go, automatically sync go.work to make overlays immediately usable
	if opts.Language == "go" {
		if err := mgr.Sync(); err != nil {
			return fmt.Errorf("failed to sync go.work: %w", err)
		}
	}

	ui.Success("Code generation completed successfully")
	return nil
}
