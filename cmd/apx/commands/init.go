package commands

import (
	"fmt"

	"github.com/infobloxopen/apx/internal/detector"
	gh "github.com/infobloxopen/apx/internal/github"
	"github.com/infobloxopen/apx/internal/interactive"
	"github.com/infobloxopen/apx/internal/schema"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [kind] [modulePath]",
		Short: "Initialize a new schema module or canonical repository",
		Long: `Create a new schema module with the specified kind and path, or initialize a canonical API repository.

SUBCOMMANDS:
  canonical - Initialize a canonical API repository structure
  app       - Initialize an application repository with schema module

MODULE INIT:
  Supported kinds: proto, openapi, avro, jsonschema, parquet
  The command will interactively guide you through setup unless --non-interactive is used.
  If no arguments are provided, you'll be prompted to select the schema type and module path.`,
		Args: cobra.MaximumNArgs(2),
		RunE: initAction,
	}
	cmd.Flags().Bool("non-interactive", false, "Disable interactive prompts and use defaults")
	cmd.Flags().String("org", "", "Organization name (auto-detected from git remote if available)")
	cmd.Flags().String("repo", "", "Repository name (auto-detected from current directory if available)")
	cmd.Flags().StringSlice("languages", []string{"go"}, "Target languages (auto-detected from project files if available)")

	cmd.AddCommand(newInitCanonicalCmd())
	cmd.AddCommand(newInitAppCmd())
	return cmd
}

func newInitCanonicalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "canonical",
		Short: "Initialize canonical API repository structure",
		RunE:  initCanonicalAction,
	}
	cmd.Flags().String("org", "", "Organization name")
	cmd.Flags().String("repo", "", "Repository name")
	cmd.Flags().Bool("skip-git", false, "Skip git initialization")
	cmd.Flags().Bool("non-interactive", false, "Disable interactive prompts and require all flags")
	cmd.Flags().Bool("setup-github", false, "Configure GitHub repo settings (branch/tag protection, org secrets) via gh CLI")
	cmd.Flags().String("app-id", "", "GitHub App ID for org secrets (used with --setup-github)")
	cmd.Flags().String("app-pem-file", "", "Path to GitHub App private key PEM file (used with --setup-github)")
	return cmd
}

func newInitAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app <modulePath>",
		Short: "Initialize application repository with schema module",
		Args:  cobra.ExactArgs(1),
		RunE:  initAppAction,
	}
	cmd.Flags().String("org", "", "Organization name")
	cmd.Flags().String("repo", "", "Repository name")
	cmd.Flags().Bool("non-interactive", false, "Disable interactive prompts and require all flags")
	cmd.Flags().Bool("setup-github", false, "Configure GitHub repo settings (branch protection) via gh CLI")
	return cmd
}

func initAction(cmd *cobra.Command, args []string) error {
	var kind, modulePath string

	switch len(args) {
	case 0:
		nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
		if nonInteractive {
			return fmt.Errorf("kind and modulePath are required in non-interactive mode")
		}
	case 2:
		kind = args[0]
		modulePath = args[1]
	default:
		return fmt.Errorf("init requires either 0 arguments (interactive) or 2 arguments: <kind> <modulePath>")
	}

	defaults, err := detector.GetSmartDefaults()
	if err != nil {
		ui.Warning("Could not detect project defaults: %v", err)
		defaults = &detector.ProjectDefaults{
			Org:       "your-org-name",
			Repo:      "apis",
			Languages: []string{"go"},
		}
	}

	if orgFlag, _ := cmd.Flags().GetString("org"); orgFlag != "" {
		defaults.Org = orgFlag
	}
	if repoFlag, _ := cmd.Flags().GetString("repo"); repoFlag != "" {
		defaults.Repo = repoFlag
	}
	if languages, _ := cmd.Flags().GetStringSlice("languages"); len(languages) > 0 {
		defaults.Languages = languages
	}

	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
	if !nonInteractive && detector.IsInteractive() {
		kind, modulePath, err = interactive.RunSetup(defaults, kind, modulePath)
		if err != nil {
			return fmt.Errorf("interactive setup failed: %w", err)
		}
	}

	if kind == "" || modulePath == "" {
		return fmt.Errorf("kind and modulePath are required")
	}

	initializer := schema.NewInitializer()
	opts := schema.InitOptions{
		Kind:       kind,
		ModulePath: modulePath,
		Defaults:   defaults,
	}
	return initializer.Initialize(opts)
}

func initCanonicalAction(cmd *cobra.Command, args []string) error {
	org, _ := cmd.Flags().GetString("org")
	repo, _ := cmd.Flags().GetString("repo")
	skipGit, _ := cmd.Flags().GetBool("skip-git")
	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

	// Also check parent flags
	if org == "" {
		org, _ = cmd.Parent().Flags().GetString("org")
	}
	if repo == "" {
		repo, _ = cmd.Parent().Flags().GetString("repo")
	}
	if !nonInteractive {
		ni, _ := cmd.Parent().Flags().GetBool("non-interactive")
		nonInteractive = ni
	}

	// Auto-detect defaults from git remote / environment
	defaults, err := detector.GetSmartDefaults()
	if err != nil {
		ui.Warning("Could not detect project defaults: %v", err)
		defaults = &detector.ProjectDefaults{
			Org:  "your-org-name",
			Repo: "apis",
		}
	}

	// Flag values take precedence over auto-detection
	if org != "" {
		defaults.Org = org
	}
	if repo != "" {
		defaults.Repo = repo
	}

	if nonInteractive {
		// Use detected/flag values directly
		org = defaults.Org
		repo = defaults.Repo
		if org == "" || org == "your-org-name" {
			return fmt.Errorf("--org is required in non-interactive mode (could not auto-detect from git remote)")
		}
		if repo == "" {
			return fmt.Errorf("--repo is required in non-interactive mode")
		}
	} else {
		ui.Info("\U0001f680 Initializing canonical API repository!")
		ui.Info("")

		// Prompt for org if not provided via flag (default pre-filled from detection)
		if org == "" {
			if err := interactive.PromptForString("Organization name:", defaults.Org, &org); err != nil {
				return fmt.Errorf("failed to get organization name: %w", err)
			}
		}

		// Prompt for repo if not provided via flag (default pre-filled from detection)
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

	scaffolder := schema.NewCanonicalScaffolder(org, repo)
	if err := scaffolder.Generate("."); err != nil {
		return fmt.Errorf("failed to generate canonical structure: %w", err)
	}

	ui.Success("\u2713 Created directory structure")
	ui.Success("\u2713 Generated buf.yaml")
	ui.Success("\u2713 Generated CODEOWNERS")
	ui.Success("\u2713 Generated catalog.yaml")
	ui.Success("\u2713 Generated README.md")
	ui.Success("\u2713 Generated .github/workflows/ci.yml")
	ui.Success("\u2713 Generated .github/workflows/on-merge.yml")

	// --setup-github: configure GitHub repo settings via gh CLI
	setupGitHub, _ := cmd.Flags().GetBool("setup-github")
	if setupGitHub {
		appID, _ := cmd.Flags().GetString("app-id")
		pemFile, _ := cmd.Flags().GetString("app-pem-file")

		if appID == "" && !nonInteractive {
			if err := interactive.PromptForString("GitHub App ID:", "", &appID); err != nil {
				return fmt.Errorf("failed to get app ID: %w", err)
			}
		}
		if appID == "" {
			return fmt.Errorf("--app-id is required with --setup-github")
		}

		// pemFile is optional if there's a cached PEM
		if pemFile == "" && !nonInteractive {
			cachePath, _ := gh.PEMCachePath(org)
			if err := interactive.PromptForString(
				fmt.Sprintf("Path to App private key PEM (leave empty if cached at %s):", cachePath),
				"", &pemFile,
			); err != nil {
				return fmt.Errorf("failed to get PEM path: %w", err)
			}
		}

		ui.Info("\nConfiguring GitHub repository...")
		res, err := gh.SetupCanonicalRepo(org, repo, appID, pemFile)
		if err != nil {
			return fmt.Errorf("GitHub setup failed: %w", err)
		}
		res.Print()
	}

	if !skipGit && !setupGitHub {
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
		ui.Info("")
		ui.Info("Or re-run with --setup-github to configure automatically via gh CLI.")
	}

	ui.Success("\n\u2713 Canonical API repository initialized successfully!")
	return nil
}

func initAppAction(cmd *cobra.Command, args []string) error {
	modulePath := args[0]

	org, _ := cmd.Flags().GetString("org")
	repo, _ := cmd.Flags().GetString("repo")
	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

	// Also check parent flags
	if org == "" {
		org, _ = cmd.Parent().Flags().GetString("org")
	}
	if repo == "" {
		repo, _ = cmd.Parent().Flags().GetString("repo")
	}
	if !nonInteractive {
		ni, _ := cmd.Parent().Flags().GetBool("non-interactive")
		nonInteractive = ni
	}

	// Auto-detect defaults from git remote / environment
	defaults, err := detector.GetSmartDefaults()
	if err != nil {
		ui.Warning("Could not detect project defaults: %v", err)
		defaults = &detector.ProjectDefaults{
			Org:  "your-org-name",
			Repo: "apis",
		}
	}

	// Flag values take precedence over auto-detection
	if org != "" {
		defaults.Org = org
	}
	if repo != "" {
		defaults.Repo = repo
	}

	if nonInteractive {
		// Use detected/flag values directly
		org = defaults.Org
		repo = defaults.Repo
		if org == "" || org == "your-org-name" {
			return fmt.Errorf("--org is required in non-interactive mode (could not auto-detect from git remote)")
		}
		if repo == "" {
			return fmt.Errorf("--repo is required in non-interactive mode")
		}
	} else if org == "" || repo == "" {
		ui.Info("\U0001f680 Initializing application repository with schema module!")
		ui.Info("")

		// Prompt for org if not provided via flag (default pre-filled from detection)
		if org == "" {
			if err := interactive.PromptForString("Organization name:", defaults.Org, &org); err != nil {
				return fmt.Errorf("failed to get organization name: %w", err)
			}
		}

		// Prompt for repo if not provided via flag (default pre-filled from detection)
		if repo == "" {
			if err := interactive.PromptForString("Repository name:", defaults.Repo, &repo); err != nil {
				return fmt.Errorf("failed to get repository name: %w", err)
			}
		}
	}

	ui.Info("Initializing application repository...")
	ui.Info("Module path: %s", modulePath)
	ui.Info("Organization: %s", org)
	ui.Info("Repository: %s", repo)

	scaffolder := schema.NewAppScaffolder(modulePath, org, repo)
	if err := scaffolder.Generate("."); err != nil {
		return fmt.Errorf("failed to generate app structure: %w", err)
	}

	ui.Success("\u2713 Created module directory structure")
	ui.Success("\u2713 Generated apx.yaml")
	ui.Success("\u2713 Generated example schema file")
	ui.Success("\u2713 Generated .gitignore")
	ui.Success("\u2713 Generated buf.work.yaml")
	ui.Success("\u2713 Generated .github/workflows/apx-publish.yml")

	// --setup-github: configure GitHub repo settings via gh CLI
	setupGitHub, _ := cmd.Flags().GetBool("setup-github")
	if setupGitHub {
		ui.Info("\nConfiguring GitHub repository...")
		res, err := gh.SetupAppRepo(org, repo)
		if err != nil {
			return fmt.Errorf("GitHub setup failed: %w", err)
		}
		res.Print()
	}

	ui.Info("\nNext steps:")
	ui.Info("1. Review and customize the generated schema file")
	ui.Info("2. Run lint checks: apx lint %s", modulePath)
	ui.Info("3. Commit your changes: git add . && git commit")
	ui.Info("4. Publish to canonical repo: apx publish --module-path=%s", modulePath)

	ui.Success("\n\u2713 Application repository initialized successfully!")
	return nil
}
