package commands

import (
	"fmt"

	"github.com/infobloxopen/apx/internal/detector"
	"github.com/infobloxopen/apx/internal/interactive"
	"github.com/infobloxopen/apx/internal/schema"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/urfave/cli/v2"
)

// InitCommand returns the init command
func InitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize a new schema module",
		Description: "Create a new schema module with the specified kind and path.\n" +
			"Supported kinds: proto, openapi, avro, jsonschema, parquet\n\n" +
			"The command will interactively guide you through setup unless --non-interactive is used.\n" +
			"If no arguments are provided, you'll be prompted to select the schema type and module path.",
		ArgsUsage: "[kind] [modulePath]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "non-interactive",
				Usage: "disable interactive prompts and use defaults",
			},
			&cli.StringFlag{
				Name:     "org",
				Usage:    "organization name (auto-detected from git remote if available)",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "repo",
				Usage:    "repository name (auto-detected from current directory if available)",
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:  "languages",
				Usage: "target languages (auto-detected from project files if available)",
				Value: cli.NewStringSlice("go"), // default to go
			},
		},
		Action: initAction,
	}
}

// InitDefaults is no longer needed - moved to detector.ProjectDefaults

func initAction(c *cli.Context) error {
	var kind, modulePath string

	// Handle different argument scenarios
	switch c.NArg() {
	case 0:
		// No arguments - will be prompted in interactive mode or error in non-interactive
		if c.Bool("non-interactive") {
			return fmt.Errorf("kind and modulePath are required in non-interactive mode")
		}
	case 2:
		// Traditional usage with both arguments
		kind = c.Args().Get(0)
		modulePath = c.Args().Get(1)
	default:
		return fmt.Errorf("init requires either 0 arguments (interactive) or 2 arguments: <kind> <modulePath>")
	}

	// Get smart defaults
	defaults, err := detector.GetSmartDefaults()
	if err != nil {
		ui.Warning("Could not detect project defaults: %v", err)
		defaults = &detector.ProjectDefaults{
			Org:       "your-org-name",
			Repo:      "your-apis-repo",
			Languages: []string{"go"},
		}
	}

	// Override with command-line flags if provided
	if orgFlag := c.String("org"); orgFlag != "" {
		defaults.Org = orgFlag
	}
	if repoFlag := c.String("repo"); repoFlag != "" {
		defaults.Repo = repoFlag
	}
	if languages := c.StringSlice("languages"); len(languages) > 0 {
		defaults.Languages = languages
	}

	// Interactive mode unless --non-interactive is set
	if !c.Bool("non-interactive") && detector.IsInteractive() {
		var err error
		kind, modulePath, err = interactive.RunSetup(defaults, kind, modulePath)
		if err != nil {
			return fmt.Errorf("interactive setup failed: %w", err)
		}
	}

	// Validate that we have the required arguments
	if kind == "" || modulePath == "" {
		return fmt.Errorf("kind and modulePath are required")
	}

	// Create the module with the configured defaults
	initializer := schema.NewInitializer()
	opts := schema.InitOptions{
		Kind:       kind,
		ModulePath: modulePath,
		Defaults:   defaults,
	}
	return initializer.Initialize(opts)
}

// The init command implementation is now properly separated:
// - CLI logic stays in this file (clean and focused)
// - Business logic moved to internal/schema package
// - Interactive setup moved to internal/interactive package
// - Project detection moved to internal/detector package
