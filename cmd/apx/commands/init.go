package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/detector"
	gh "github.com/infobloxopen/apx/internal/github"
	"github.com/infobloxopen/apx/internal/interactive"
	"github.com/infobloxopen/apx/internal/schema"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/infobloxopen/apx/pkg/githubauth"
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
	cmd.Flags().String("import-root", "", "Custom public Go import prefix (e.g. go.acme.dev/apis)")
	cmd.Flags().String("site-url", "", "Custom domain for the catalog site (e.g. apis.internal.infoblox.dev)")
	cmd.Flags().Bool("skip-git", false, "Skip git initialization")
	cmd.Flags().Bool("non-interactive", false, "Disable interactive prompts and require all flags")
	cmd.Flags().Bool("setup-github", false, "Configure GitHub repo settings (apps, branch/tag protection, org secrets)")
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
	cmd.Flags().String("import-root", "", "Custom public Go import prefix (e.g. go.acme.dev/apis)")
	cmd.Flags().Bool("non-interactive", false, "Disable interactive prompts and require all flags")
	cmd.Flags().Bool("setup-github", false, "Configure GitHub repo settings (branch protection)")
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
	importRoot, _ := cmd.Flags().GetString("import-root")
	siteURL, _ := cmd.Flags().GetString("site-url")
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

		// Prompt for import_root if not provided via flag
		if importRoot == "" {
			ui.Info("")
			ui.Info("Custom import root (optional):")
			ui.Info("  Decouples public Go import paths from your Git hosting URL.")
			ui.Info("  Example: go.%s.dev/apis → consumers import go.%s.dev/apis/proto/...", org, org)
			ui.Info("  Leave blank to use github.com/%s/%s as the import root.", org, repo)
			ui.Info("")
			if err := interactive.PromptForString("Import root (blank to skip):", "", &importRoot); err != nil {
				return fmt.Errorf("failed to get import root: %w", err)
			}
		}

		// Prompt for site_url if not provided via flag
		if siteURL == "" {
			defaultSiteURL := strings.ToLower(org) + ".github.io/" + repo
			ui.Info("")
			ui.Info("Catalog site URL (optional):")
			ui.Info("  Where the API catalog explorer will be hosted via GitHub Pages.")
			ui.Info("  Default: %s", defaultSiteURL)
			ui.Info("  Set a custom domain to use your own URL (e.g. apis.internal.%s.dev)", strings.ToLower(org))
			ui.Info("")
			if err := interactive.PromptForString("Site URL (blank for default):", "", &siteURL); err != nil {
				return fmt.Errorf("failed to get site URL: %w", err)
			}
		}
	}

	ui.Info("Initializing canonical API repository...")
	ui.Info("Organization: %s", org)
	ui.Info("Repository: %s", repo)
	if importRoot != "" {
		ui.Info("Import root: %s", importRoot)
	}
	if siteURL != "" {
		ui.Info("Site URL: %s", siteURL)
	}
	ui.Info("")

	scaffolder := schema.NewCanonicalScaffolder(org, repo, importRoot, siteURL)
	if err := scaffolder.Generate("."); err != nil {
		return fmt.Errorf("failed to generate canonical structure: %w", err)
	}

	ui.Success("\u2713 Created directory structure")
	ui.Success("\u2713 Generated buf.yaml")
	ui.Success("\u2713 Generated CODEOWNERS")
	ui.Success("\u2713 Generated catalog/Dockerfile")
	ui.Success("\u2713 Generated README.md")
	ui.Success("\u2713 Generated apx.yaml")
	ui.Success("\u2713 Generated .github/workflows/ci.yml")
	ui.Success("\u2713 Generated .github/workflows/on-merge.yml")

	// --setup-github: configure GitHub repo settings
	setupGitHub, _ := cmd.Flags().GetBool("setup-github")
	if setupGitHub {
		appID, _ := cmd.Flags().GetString("app-id")
		pemFile, _ := cmd.Flags().GetString("app-pem-file")

		// ── Step 1: User App (least-privilege, daily use) ──────────
		// Creates apx-{org}-user with minimal permissions for device-flow
		// auth, catalog discovery, releases, and pull requests.
		userClientID := gh.GetCachedUserAppClientID(org)
		if userClientID == "" {
			if nonInteractive {
				return fmt.Errorf("user app not configured for org %q; run interactively first", org)
			}
			ui.Info("\nCreating user app %q via GitHub App manifest flow...", gh.UserAppName(org))
			creds, createErr := gh.CreateAppViaManifest(org, gh.UserAppName(org), gh.UserAppPermissions)
			if createErr != nil {
				return fmt.Errorf("failed to create user app: %w", createErr)
			}
			if err := gh.CacheUserAppClientID(org, creds.ClientID); err != nil {
				return fmt.Errorf("failed to cache user app client ID: %w", err)
			}
			if err := gh.CacheUserAppID(org, fmt.Sprintf("%d", creds.ID)); err != nil {
				return fmt.Errorf("failed to cache user app ID: %w", err)
			}
			if creds.Slug != "" {
				if err := gh.CacheUserAppSlug(org, creds.Slug); err != nil {
					ui.Warning("Failed to cache user app slug: %v", err)
				}
			}
			userClientID = creds.ClientID
			ui.Success("User app created! Client ID: %s", userClientID)
			ui.Info("  Permissions: contents:write, pull_requests:write, metadata:read, packages:read")

			// The GitHub App manifest flow cannot enable device flow.
			// It must be enabled manually in the App settings UI.
			settingsURL := fmt.Sprintf("https://github.com/organizations/%s/settings/apps/%s", org, creds.Slug)
			ui.Info("")
			ui.Warning("Device flow must be enabled manually on the user app.")
			ui.Info("  1. Open: %s", settingsURL)
			ui.Info("  2. Check \"Enable Device Flow\" under \"Identifying and authorizing users\"")
			ui.Info("  3. Save changes")
			ui.Info("")
			if !nonInteractive {
				ui.Info("Opening browser to App settings...")
				_ = gh.OpenBrowser(settingsURL)
				ui.Info("Press Enter after enabling Device Flow...")
				fmt.Scanln() //nolint:errcheck
			}
		} else {
			ui.Info("User app already configured (client_id cached).")
		}

		// ── Step 2: Admin App (elevated, setup-only) ────────────────
		// Creates apx-{org}-admin with elevated permissions for branch
		// protection, org secrets, and GitHub Pages. Only used during
		// --setup-github, not for daily operations.
		adminClientID := gh.GetCachedAdminAppClientID(org)
		if adminClientID == "" {
			if nonInteractive {
				return fmt.Errorf("admin app not configured for org %q; run interactively first", org)
			}
			ui.Info("\nCreating admin app %q via GitHub App manifest flow...", gh.AdminAppName(org))
			ui.Info("  This app has elevated permissions for one-time repo setup.")
			creds, createErr := gh.CreateAppViaManifest(org, gh.AdminAppName(org), gh.AdminAppPermissions)
			if createErr != nil {
				return fmt.Errorf("failed to create admin app: %w", createErr)
			}
			if err := gh.CacheAdminAppClientID(org, creds.ClientID); err != nil {
				return fmt.Errorf("failed to cache admin app client ID: %w", err)
			}
			if err := gh.CacheAdminAppID(org, fmt.Sprintf("%d", creds.ID)); err != nil {
				return fmt.Errorf("failed to cache admin app ID: %w", err)
			}
			if creds.Slug != "" {
				if err := gh.CacheAdminAppSlug(org, creds.Slug); err != nil {
					ui.Warning("Failed to cache admin app slug: %v", err)
				}
			}
			adminClientID = creds.ClientID
			ui.Success("Admin app created! Client ID: %s", adminClientID)
		} else {
			ui.Info("Admin app already configured (client_id cached).")
		}

		// ── Step 3: Device flow login (user app) ────────────────────
		// Authenticate the user via the user app for daily-use token.
		token, tokenErr := githubauth.EnsureToken(org)
		if tokenErr != nil && githubauth.IsDeviceFlowDisabled(tokenErr) && !nonInteractive {
			slug := gh.GetCachedUserAppSlug(org)
			if slug == "" {
				slug = gh.UserAppName(org)
			}
			settingsURL := fmt.Sprintf("https://github.com/organizations/%s/settings/apps/%s", org, slug)
			ui.Warning("Device flow is not enabled on the %q GitHub App.", slug)
			ui.Info("")
			ui.Info("  1. Open: %s", settingsURL)
			ui.Info("  2. Check \"Enable Device Flow\" under \"Identifying and authorizing users\"")
			ui.Info("  3. Save changes")
			ui.Info("")
			ui.Info("Opening browser to App settings...")
			_ = gh.OpenBrowser(settingsURL)
			ui.Info("Press Enter after enabling Device Flow...")
			fmt.Scanln() //nolint:errcheck
			token, tokenErr = githubauth.EnsureToken(org)
		}
		if tokenErr != nil {
			return fmt.Errorf("GitHub authentication failed: %w", tokenErr)
		}
		client := githubauth.NewClient(token)

		// ── Step 4: Ensure apps are installed ────────────────────────
		userAppIDStr := gh.GetCachedUserAppID(org)
		userAppSlug := gh.GetCachedUserAppSlug(org)
		if userAppIDStr != "" && userAppSlug != "" {
			userAppIDInt, _ := strconv.Atoi(userAppIDStr)
			if userAppIDInt > 0 {
				if err := gh.EnsureAppInstalled(client, org, userAppIDInt, userAppSlug); err != nil {
					ui.Warning("Could not verify user app installation: %v", err)
				}
			}
		}
		adminAppIDStr := gh.GetCachedAdminAppID(org)
		adminAppSlug := gh.GetCachedAdminAppSlug(org)
		if adminAppIDStr != "" && adminAppSlug != "" {
			adminAppIDInt, _ := strconv.Atoi(adminAppIDStr)
			if adminAppIDInt > 0 {
				if err := gh.EnsureAppInstalled(client, org, adminAppIDInt, adminAppSlug); err != nil {
					ui.Warning("Could not verify admin app installation: %v", err)
				}
			}
		}

		// ── Step 5: CI App ──────────────────────────────────────────
		// Create the CI GitHub App (apx-{repo}-{org}) if not cached.
		if appID == "" {
			appID = gh.GetCachedAppID(org)
		}
		pemCached := false
		if cachePath, pemErr := gh.PEMCachePath(org); pemErr == nil {
			if _, statErr := os.Stat(cachePath); statErr == nil {
				pemCached = true
			}
		}

		needsApp := appID == "" || (!pemCached && pemFile == "")
		if needsApp && !nonInteractive {
			ui.Info("\nCreating CI app %q via GitHub App manifest flow...", gh.CIAppName(repo, org))
			creds, createErr := gh.CreateAppViaManifest(org, gh.CIAppName(repo, org), gh.CIAppPermissions)
			if createErr != nil {
				return fmt.Errorf("failed to create CI app: %w", createErr)
			}
			if err := gh.CachePEMFromContents(org, creds.PEM); err != nil {
				return fmt.Errorf("failed to cache PEM: %w", err)
			}
			if err := gh.CacheAppID(org, fmt.Sprintf("%d", creds.ID)); err != nil {
				return fmt.Errorf("failed to cache CI app ID: %w", err)
			}
			if creds.Slug != "" {
				if err := gh.CacheAppSlug(org, creds.Slug); err != nil {
					ui.Warning("Failed to cache CI app slug: %v", err)
				}
			}
			appID = fmt.Sprintf("%d", creds.ID)
			ui.Success("CI app created! App ID: %s", appID)

			// Ensure CI app is installed on the org.
			if creds.Slug != "" {
				if err := gh.EnsureAppInstalled(client, org, creds.ID, creds.Slug); err != nil {
					ui.Warning("Could not verify CI app installation: %v", err)
				}
			}
		} else if needsApp {
			return fmt.Errorf("--app-id and --app-pem-file are required with --setup-github in non-interactive mode")
		} else {
			// CI app already exists – ensure it is installed on the org.
			ciSlug := gh.GetCachedAppSlug(org)
			if ciSlug != "" {
				appIDInt, _ := strconv.Atoi(appID)
				if appIDInt > 0 {
					if err := gh.EnsureAppInstalled(client, org, appIDInt, ciSlug); err != nil {
						ui.Warning("Could not verify CI app installation: %v", err)
					}
				}
			}
		}

		// ── Step 5: Set up canonical repo ───────────────────────────
		// Resolve siteURL from config if not provided via flag.
		if siteURL == "" {
			if cfg, cfgErr := config.LoadRaw("apx.yaml"); cfgErr == nil && cfg.SiteURL != "" {
				siteURL = cfg.SiteURL
			}
		}

		ui.Info("\nConfiguring GitHub repository...")
		res, setupErr := gh.SetupCanonicalRepo(client, org, repo, appID, pemFile, siteURL)
		if setupErr != nil {
			return fmt.Errorf("GitHub setup failed: %w", setupErr)
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
		ui.Info("Or re-run with --setup-github to configure automatically.")
	}

	// Seed global config with this org/repo
	if globalCfg, loadErr := config.LoadGlobal(); loadErr == nil {
		globalCfg.AddOrg(org, []string{repo})
		globalCfg.SetDefaultOrg(org)
		_ = config.SaveGlobal(globalCfg)
	}

	ui.Success("\n\u2713 Canonical API repository initialized successfully!")
	return nil
}

func initAppAction(cmd *cobra.Command, args []string) error {
	modulePath := args[0]

	org, _ := cmd.Flags().GetString("org")
	repo, _ := cmd.Flags().GetString("repo")
	importRoot, _ := cmd.Flags().GetString("import-root")
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
		// Auto-inherit import_root from canonical repo when not explicitly set
		if importRoot == "" {
			importRoot = config.FetchRemoteImportRoot(org, repo)
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

		// Prompt for import_root if not provided via flag
		if importRoot == "" {
			ui.Info("")
			ui.Info("Custom import root (optional):")
			ui.Info("  Decouples public Go import paths from your Git hosting URL.")
			ui.Info("  Example: go.%s.dev/apis → consumers import go.%s.dev/apis/proto/...", org, org)

			// Try to detect import_root from canonical repo or cached catalog
			defaultRoot := config.FetchRemoteImportRoot(org, repo)
			if defaultRoot != "" {
				ui.Success("  Detected from canonical repo: %s", defaultRoot)
			} else {
				ui.Info("  Leave blank to use github.com/%s/%s as the import root.", org, repo)
			}
			ui.Info("")
			if err := interactive.PromptForString("Import root (blank to skip):", defaultRoot, &importRoot); err != nil {
				return fmt.Errorf("failed to get import root: %w", err)
			}
		}
	}

	ui.Info("Initializing application repository...")
	ui.Info("Module path: %s", modulePath)
	ui.Info("Organization: %s", org)
	ui.Info("Repository: %s", repo)
	if importRoot != "" {
		ui.Info("Import root: %s", importRoot)
	}

	scaffolder := schema.NewAppScaffolder(modulePath, org, repo, importRoot)
	if err := scaffolder.Generate("."); err != nil {
		return fmt.Errorf("failed to generate app structure: %w", err)
	}

	ui.Success("\u2713 Created module directory structure")
	ui.Success("\u2713 Generated apx.yaml")
	ui.Success("\u2713 Generated example schema file")
	ui.Success("\u2713 Generated .gitignore")
	ui.Success("\u2713 Generated .github/workflows/apx-release.yml")

	// --setup-github: configure GitHub repo settings
	setupGitHub, _ := cmd.Flags().GetBool("setup-github")
	if setupGitHub {
		token, tokenErr := githubauth.EnsureToken(org)
		if tokenErr != nil {
			return fmt.Errorf("GitHub authentication failed: %w", tokenErr)
		}
		client := githubauth.NewClient(token)

		ui.Info("\nConfiguring GitHub repository...")
		res, setupErr := gh.SetupAppRepo(client, org, repo)
		if setupErr != nil {
			return fmt.Errorf("GitHub setup failed: %w", setupErr)
		}
		res.Print()
	}

	ui.Info("\nNext steps:")
	ui.Info("1. Review and customize the generated schema file")
	ui.Info("2. Run lint checks: apx lint %s", modulePath)
	ui.Info("3. Commit your changes: git add . && git commit")
	ui.Info("4. Release to canonical repo: apx release submit --module-path=%s", modulePath)

	// Seed global config with this org/repo
	if globalCfg, loadErr := config.LoadGlobal(); loadErr == nil {
		globalCfg.AddOrg(org, []string{repo})
		_ = config.SaveGlobal(globalCfg)
	}

	ui.Success("\n\u2713 Application repository initialized successfully!")
	return nil
}
