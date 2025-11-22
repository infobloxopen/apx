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
		Usage: "Initialize a new schema module or canonical repository",
		Description: "Create a new schema module with the specified kind and path, or initialize a canonical API repository.\n\n" +
			"SUBCOMMANDS:\n" +
			"  canonical - Initialize a canonical API repository structure\n" +
			"  app - Initialize an application repository with schema module\n\n" +
			"MODULE INIT:\n" +
			"  Supported kinds: proto, openapi, avro, jsonschema, parquet\n" +
			"  The command will interactively guide you through setup unless --non-interactive is used.\n" +
			"  If no arguments are provided, you'll be prompted to select the schema type and module path.",
		ArgsUsage: "[canonical|app|kind] [modulePath]",
		Subcommands: []*cli.Command{
			{
				Name:      "canonical",
				Usage:     "Initialize canonical API repository structure",
				ArgsUsage: " ",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "org",
						Usage:    "organization name",
						Required: false, // Made optional to support interactive mode
					},
					&cli.StringFlag{
						Name:     "repo",
						Usage:    "repository name",
						Required: false, // Made optional to support interactive mode
					},
					&cli.BoolFlag{
						Name:  "skip-git",
						Usage: "skip git initialization",
					},
					&cli.BoolFlag{
						Name:  "non-interactive",
						Usage: "disable interactive prompts and require all flags",
					},
				},
				Action: initCanonicalAction,
			},
			{
				Name:      "app",
				Usage:     "Initialize application repository with schema module",
				ArgsUsage: "<modulePath>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "org",
						Usage:    "organization name",
						Required: false,
					},
					&cli.BoolFlag{
						Name:  "non-interactive",
						Usage: "disable interactive prompts and require all flags",
					},
				},
				Action: initAppAction,
			},
		},
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

// initCanonicalAction initializes a canonical API repository
func initCanonicalAction(c *cli.Context) error {
	// Check both local and parent context for flags (supports parent flag inheritance)
	org := c.String("org")
	if org == "" && len(c.Lineage()) > 1 && c.Lineage()[1] != nil {
		org = c.Lineage()[1].String("org")
	}

	repo := c.String("repo")
	if repo == "" && len(c.Lineage()) > 1 && c.Lineage()[1] != nil {
		repo = c.Lineage()[1].String("repo")
	}

	skipGit := c.Bool("skip-git")

	nonInteractive := c.Bool("non-interactive")
	if !nonInteractive && len(c.Lineage()) > 1 && c.Lineage()[1] != nil {
		nonInteractive = c.Lineage()[1].Bool("non-interactive")
	} // If in non-interactive mode, both org and repo are required
	if nonInteractive && (org == "" || repo == "") {
		return fmt.Errorf("--org and --repo are required in non-interactive mode")
	}

	// Interactive mode: prompt for missing values
	if !nonInteractive {
		// Get smart defaults
		defaults, err := detector.GetSmartDefaults()
		if err != nil {
			ui.Warning("Could not detect project defaults: %v", err)
			defaults = &detector.ProjectDefaults{
				Org:  "your-org-name",
				Repo: "your-apis-repo",
			}
		}

		ui.Info("🚀 Initializing canonical API repository!")
		ui.Info("")

		// Prompt for org if not provided
		if org == "" {
			if err := interactive.PromptForString("Organization name:", defaults.Org, &org); err != nil {
				return fmt.Errorf("failed to get organization name: %w", err)
			}
		}

		// Prompt for repo if not provided
		if repo == "" {
			if err := interactive.PromptForString("Repository name:", defaults.Repo, &repo); err != nil {
				return fmt.Errorf("failed to get repository name: %w", err)
			}
		}
	}

	ui.Info("Initializing canonical API repository...")
	ui.Info("Organization: %s", org)
	ui.Info("Repository: %s", repo)
	ui.Info("")

	// Create scaffolder and generate structure
	scaffolder := schema.NewCanonicalScaffolder(org, repo)
	if err := scaffolder.Generate("."); err != nil {
		return fmt.Errorf("failed to generate canonical structure: %w", err)
	}

	ui.Success("✓ Created directory structure")
	ui.Success("✓ Generated buf.yaml")
	ui.Success("✓ Generated CODEOWNERS")
	ui.Success("✓ Generated catalog.yaml")
	ui.Success("✓ Generated README.md")

	if !skipGit {
		ui.Info("\nNext steps:")
		ui.Info("1. Initialize git: git init")
		ui.Info("2. Add files: git add .")
		ui.Info("3. Commit: git commit -m 'Initial canonical repository scaffold'")
		ui.Info("4. Create GitHub repository and set up branch protection:")
		ui.Info("   - Require pull request reviews")
		ui.Info("   - Require status checks (lint, breaking)")
		ui.Info("   - Require CODEOWNERS review")
		ui.Info("   - Restrict direct pushes to main")
		ui.Info("5. Push: git remote add origin <url> && git push -u origin main")
	}

	ui.Success("\n✓ Canonical API repository initialized successfully!")
	return nil
}

// initAppAction initializes an application repository with schema module
func initAppAction(c *cli.Context) error {
	modulePath := c.Args().First()

	// Check both local and parent context for flags (supports parent flag inheritance)
	org := c.String("org")
	if org == "" && len(c.Lineage()) > 1 && c.Lineage()[1] != nil {
		org = c.Lineage()[1].String("org")
	}

	nonInteractive := c.Bool("non-interactive")
	if !nonInteractive && len(c.Lineage()) > 1 && c.Lineage()[1] != nil {
		nonInteractive = c.Lineage()[1].Bool("non-interactive")
	} // Validate arguments
	if modulePath == "" {
		return fmt.Errorf("module path is required (e.g., internal/apis/proto/payments/ledger/v1)")
	}

	// If in non-interactive mode, org is required
	if nonInteractive && org == "" {
		return fmt.Errorf("--org is required in non-interactive mode")
	}

	// Interactive mode: prompt for missing values
	if org == "" {
		if nonInteractive {
			return fmt.Errorf("--org is required in non-interactive mode")
		}

		// Get smart defaults
		defaults, err := detector.GetSmartDefaults()
		if err != nil {
			ui.Warning("Could not detect project defaults: %v", err)
			defaults = &detector.ProjectDefaults{
				Org: "your-org-name",
			}
		}

		ui.Info("🚀 Initializing application repository with schema module!")
		ui.Info("")

		if err := interactive.PromptForString("Organization name:", defaults.Org, &org); err != nil {
			return fmt.Errorf("failed to get organization name: %w", err)
		}
	}

	ui.Info("Initializing application repository...")
	ui.Info("Module path: %s", modulePath)
	ui.Info("Organization: %s", org)

	// Create scaffolder and generate structure
	scaffolder := schema.NewAppScaffolder(modulePath, org)
	if err := scaffolder.Generate("."); err != nil {
		return fmt.Errorf("failed to generate app structure: %w", err)
	}

	ui.Success("✓ Created module directory structure")
	ui.Success("✓ Generated apx.yaml")
	ui.Success("✓ Generated example schema file")
	ui.Success("✓ Generated .gitignore")
	ui.Success("✓ Generated buf.work.yaml")

	ui.Info("\nNext steps:")
	ui.Info("1. Review and customize the generated schema file")
	ui.Info("2. Run lint checks: apx lint %s", modulePath)
	ui.Info("3. Commit your changes: git add . && git commit")
	ui.Info("4. Publish to canonical repo: apx publish --module-path=%s", modulePath)

	ui.Success("\n✓ Application repository initialized successfully!")
	return nil
}
