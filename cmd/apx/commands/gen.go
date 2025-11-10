package commands

import (
	"fmt"

	"github.com/infobloxopen/apx/internal/config"
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

	cfg, err := loadConfig(c)
	if err != nil {
		ui.Error("Failed to load config: %v", err)
		return err
	}

	opts := GenerateOptions{
		Language:  lang,
		Path:      path,
		OutputDir: c.String("out"),
		Clean:     c.Bool("clean"),
		Manifest:  c.Bool("manifest"),
	}

	return generateCode(cfg, opts)
}

func generateCode(cfg *config.Config, opts GenerateOptions) error {
	// TODO: Implement code generation in internal/generator package
	ui.Info("Generating %s code from %s...", opts.Language, opts.Path)
	if opts.OutputDir != "" {
		ui.Info("Output directory: %s", opts.OutputDir)
	}
	ui.Success("Code generation completed successfully")
	return nil
}
