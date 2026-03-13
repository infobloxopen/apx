package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/infobloxopen/apx/pkg/githubauth"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate and discover API catalogs",
		Long: `Authenticate with GitHub and discover API catalogs across your organizations.

The auth command manages GitHub authentication via OAuth device flow and
discovers canonical API repositories (catalog images) in your organizations.
Discovered orgs and repos are saved to ~/.config/apx/config.yaml so that
apx catalog search works from any directory without per-repo configuration.`,
	}
	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthStatusCmd())
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in and discover API catalogs",
		Long: `Authenticate with GitHub via device flow, discover your organizations,
and find all published API catalog images.

If --org is provided, authentication targets that specific org.
Otherwise, the org is auto-detected from the current git remote.

After authentication, apx queries the GitHub Packages API to find
all container images ending with "-catalog" in each accessible org.
Results are saved to ~/.config/apx/config.yaml for future use.`,
		RunE: authLoginAction,
	}
	cmd.Flags().String("org", "", "Target organization (auto-detected from git remote if omitted)")
	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication and discovery status",
		RunE:  authStatusAction,
	}
}

func authLoginAction(cmd *cobra.Command, args []string) error {
	orgFlag, _ := cmd.Flags().GetString("org")

	// Determine the target org for initial authentication
	org := orgFlag
	if org == "" {
		detected, err := githubauth.DetectOrg()
		if err != nil {
			ui.Info("Could not detect org from git remote.")
			return fmt.Errorf("--org is required when not inside a git repository with a GitHub remote")
		}
		org = detected
		ui.Info("Detected org from git remote: %s", org)
	}

	// Authenticate via device flow
	ui.Info("Authenticating with GitHub for org %q...", org)
	token, err := githubauth.EnsureToken(org)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	ui.Success("Authenticated successfully.")

	client := githubauth.NewClient(token)

	// Discover user's organizations
	ui.Info("Discovering your organizations...")
	userOrgs, err := client.ListUserOrgs()
	if err != nil {
		ui.Warning("Could not list organizations: %v", err)
		userOrgs = []string{org} // fall back to just the target org
	}

	// Ensure the target org is in the list
	found := false
	for _, o := range userOrgs {
		if strings.EqualFold(o, org) {
			found = true
			break
		}
	}
	if !found {
		userOrgs = append([]string{org}, userOrgs...)
	}

	ui.Info("Found %d organization(s): %s", len(userOrgs), strings.Join(userOrgs, ", "))

	// Load or create global config
	globalCfg, err := config.LoadGlobal()
	if err != nil {
		globalCfg = &config.GlobalConfig{Version: 1}
	}

	// Discover catalog images in each org
	totalRepos := 0
	for _, orgName := range userOrgs {
		repos := discoverCatalogRepos(orgName)
		globalCfg.AddOrg(orgName, repos)
		if len(repos) > 0 {
			totalRepos += len(repos)
			ui.Success("  %s: %s", orgName, strings.Join(repos, ", "))
		} else {
			ui.Info("  %s: no catalog images found", orgName)
		}
	}

	// Set default org
	globalCfg.SetDefaultOrg(org)

	// Save global config
	if err := config.SaveGlobal(globalCfg); err != nil {
		return fmt.Errorf("failed to save global config: %w", err)
	}

	ui.Success("\nDiscovery complete: %d org(s), %d catalog repo(s) saved to global config.", len(userOrgs), totalRepos)

	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonOut {
		data, err := json.MarshalIndent(globalCfg, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
	}

	return nil
}

// discoverCatalogRepos uses the GitHub Packages API to find catalog images
// for a given org. Returns repo names (with the "-catalog" suffix stripped).
func discoverCatalogRepos(org string) []string {
	sources := catalog.DiscoverRegistries(org)
	if len(sources) == 0 {
		return nil
	}

	var repos []string
	for _, src := range sources {
		// CachedSource wraps RegistrySource — extract the name to derive the repo.
		name := src.Name()
		// Name format: "ghcr.io/<org>/<repo>-catalog:latest (cached)"
		// Strip suffix to get repo name.
		name = strings.TrimSuffix(name, " (cached)")
		parts := strings.Split(name, "/")
		if len(parts) >= 3 {
			repoTag := parts[len(parts)-1]
			repo := strings.TrimSuffix(repoTag, catalog.CatalogImageSuffix+":latest")
			if repo != "" {
				repos = append(repos, repo)
			}
		}
	}
	return repos
}

func authStatusAction(cmd *cobra.Command, args []string) error {
	globalCfg, err := config.LoadGlobal()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonOut {
		data, err := json.MarshalIndent(globalCfg, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	if len(globalCfg.Orgs) == 0 {
		ui.Info("No organizations configured. Run `apx auth login` to get started.")
		return nil
	}

	if globalCfg.DefaultOrg != "" {
		ui.Info("Default org: %s", globalCfg.DefaultOrg)
	}
	fmt.Println()

	for _, org := range globalCfg.Orgs {
		// Check token status
		tok, _ := githubauth.LoadToken(org.Name)
		status := "not authenticated"
		if tok != nil {
			status = fmt.Sprintf("authenticated (since %s)", tok.CreatedAt.Format("2006-01-02"))
		}

		ui.Info("  %s  [%s]", org.Name, status)
		if len(org.Repos) > 0 {
			ui.Info("    Catalog repos: %s", strings.Join(org.Repos, ", "))
		} else {
			ui.Info("    No catalog repos discovered")
		}
		fmt.Println()
	}

	return nil
}
