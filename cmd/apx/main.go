package main

import (
	"context"
	"fmt"
	"os"

	"github.com/infobloxopen/apx/cmd/apx/commands"
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/urfave/cli/v2"
)

var (
	version = "dev"
	commit  = "none"    //nolint:unused // Set by build
	date    = "unknown" //nolint:unused // Set by build
)

func main() {
	ui.InitializeFromEnv()
	app := NewApp()
	err := app.RunContext(context.Background(), os.Args)
	os.Exit(exitCode(err))
}

// NewApp creates a new CLI application
func NewApp() *cli.App {
	app := &cli.App{
		Name:    "apx",
		Usage:   "API Publishing eXperience CLI",
		Version: fmt.Sprintf("apx %s (%s) %s", version, commit, date),
		Action: func(c *cli.Context) error {
			// Default action - show help
			return cli.ShowAppHelp(c)
		},
		Before: func(c *cli.Context) error {
			// Set global flags
			if c.Bool("quiet") {
				ui.SetQuiet(true)
			}
			if c.Bool("verbose") {
				ui.SetVerbose(true)
			}
			if c.Bool("json") {
				ui.SetJSONOutput(true)
			}
			if c.Bool("no-color") {
				ui.SetColorEnabled(false)
			}
			return nil
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "suppress output",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "verbose output",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "output in JSON format",
			},
			&cli.BoolFlag{
				Name:  "no-color",
				Usage: "disable colored output",
			},
			&cli.StringFlag{
				Name:  "config",
				Usage: "config file path",
				Value: "apx.yaml",
			},
		},
		Commands: []*cli.Command{
			commands.InitCommand(),
			commands.LintCommand(),
			commands.BreakingCommand(),
			commands.SemverCommand(),
			commands.GenCommand(),
			commands.PolicyCommand(),
			commands.CatalogCommand(),
			commands.PublishCommand(),
			commands.ConfigCommand(),
		},
	}

	return app
}

// exitCode maps errors to exit codes
func exitCode(err error) int {
	if err == nil {
		return 0
	}

	// Handle config errors
	if config.IsValidationError(err) {
		return 6
	}

	// Default error
	return 1
}
