package commands

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/detector"
	"github.com/infobloxopen/apx/internal/interactive"
	"github.com/infobloxopen/apx/internal/onboard"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newOnboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onboard <api-id>",
		Short: "Onboard an existing service schema into the APX publish-on-change workflow",
		Long: `Scaffold the files required to connect an existing service repository to the
APX publish-on-change workflow (apx-publish.sh). Unlike 'apx init app', this
command does not restructure your repository — it generates a minimal apx.yaml,
an .apx-publish.yaml manifest, a Makefile fds target, two GitHub workflow
callers, and the required .gitignore entries.

The <api-id> argument is the canonical identifier for your API module, e.g.:
  proto/infoblox.dev/host-activation/v1
  openapi/payments/ledger/v1

When a swagger file is provided (--swagger or auto-detected), an additional
openapi/ module is added to .apx-publish.yaml alongside the proto/ module.

See https://infobloxopen.github.io/apx/getting-started/brownfield-onboarding/ for details.`,
		Args: cobra.ExactArgs(1),
		RunE: onboardAction,
	}
	cmd.Flags().String("canonical-repo", "", "Canonical repo (e.g. github.com/Infoblox-CTO/apis)")
	cmd.Flags().String("schema-dir", "", "Directory to run buf build on (e.g. simpleproto); use --proto-file for atlas-gentool repos")
	cmd.Flags().String("proto-file", "", "Specific proto file for atlas-gentool Makefile target (e.g. pkg/pb/service.proto)")
	cmd.Flags().String("swagger", "", "Path to swagger/OpenAPI file; adds an openapi/ module alongside the proto/ module")
	cmd.Flags().String("lifecycle", "beta", "Lifecycle state: experimental|beta|stable|deprecated|sunset")
	cmd.Flags().String("audience", "public", "Audience: public|internal")
	cmd.Flags().String("fds-output", "api-v1.binpb", "Output filename for the FileDescriptorSet binary")
	cmd.Flags().Bool("non-interactive", false, "Disable interactive prompts; all required flags must be provided")
	cmd.Flags().String("org", "", "Org name (auto-detected from git remote)")
	cmd.Flags().String("repo", "", "Repo name (auto-detected from git remote)")
	cmd.Flags().String("default-branch", "", "Repo default branch for push trigger (auto-detected from git remote)")
	cmd.Flags().String("schema-file", "", "Path to schema file for non-FDS formats (e.g. api/event.schema.json for jsonschema, api/event.avsc for avro)")
	cmd.Flags().String("go-private", "", "GOPRIVATE pattern for private Go modules (e.g. github.com/my-org/*); only used when atlas-gentool + go.mod is detected")
	return cmd
}

func onboardAction(cmd *cobra.Command, args []string) error {
	apiID := args[0]
	if !isValidAPIID(apiID) {
		return fmt.Errorf("invalid api-id %q: must start with a known format (proto, openapi, avro, jsonschema, parquet, crd)", apiID)
	}

	canonicalRepo, _ := cmd.Flags().GetString("canonical-repo")
	schemaDir, _ := cmd.Flags().GetString("schema-dir")
	protoFile, _ := cmd.Flags().GetString("proto-file")
	swagger, _ := cmd.Flags().GetString("swagger")
	lifecycle, _ := cmd.Flags().GetString("lifecycle")
	audience, _ := cmd.Flags().GetString("audience")
	fdsOutput, _ := cmd.Flags().GetString("fds-output")
	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
	orgFlag, _ := cmd.Flags().GetString("org")
	repoFlag, _ := cmd.Flags().GetString("repo")
	defaultBranchFlag, _ := cmd.Flags().GetString("default-branch")
	schemaFile, _ := cmd.Flags().GetString("schema-file")
	goPrivate, _ := cmd.Flags().GetString("go-private")

	defaults, err := detector.GetSmartDefaults()
	if err != nil {
		defaults = &detector.ProjectDefaults{Org: "your-org", Repo: "your-repo"}
	}
	if orgFlag != "" {
		defaults.Org = orgFlag
	}
	if repoFlag != "" {
		defaults.Repo = repoFlag
	}

	if nonInteractive {
		if canonicalRepo == "" {
			return fmt.Errorf("--canonical-repo is required in non-interactive mode")
		}
		if defaults.Org == "" || defaults.Org == "your-org-name" {
			return fmt.Errorf("--org is required in non-interactive mode (could not auto-detect from git remote)")
		}
	} else {
		ui.Info("Onboarding %s into the APX publish-on-change workflow", apiID)
		ui.Info("")

		if canonicalRepo == "" {
			defaultCanonical := canonicalDefault(defaults)
			if err := interactive.PromptForString("Canonical repo:", defaultCanonical, &canonicalRepo); err != nil {
				return fmt.Errorf("failed to get canonical repo: %w", err)
			}
		}

		// Auto-detect swagger in schemaDir or protoFile's directory when not provided
		if swagger == "" {
			searchDir := schemaDir
			if searchDir == "" && protoFile != "" {
				// derive dir from proto file path
				parts := strings.Split(strings.ReplaceAll(protoFile, "\\", "/"), "/")
				if len(parts) > 1 {
					searchDir = strings.Join(parts[:len(parts)-1], "/")
				}
			}
			if detected := onboard.DetectSwagger(searchDir); detected != "" {
				var confirm string
				if err := interactive.PromptForString(
					fmt.Sprintf("Swagger file detected (%s) — include as openapi/ module? [Y/n]:", detected),
					"Y", &confirm,
				); err == nil && !strings.EqualFold(confirm, "n") {
					swagger = detected
				}
			}
		}
	}

	if canonicalRepo == "" {
		return fmt.Errorf("canonical-repo is required")
	}

	defaultBranch := defaultBranchFlag
	if defaultBranch == "" {
		defaultBranch = detectDefaultBranch()
	}

	opts := onboard.Options{
		APIID:         apiID,
		CanonicalRepo: canonicalRepo,
		SchemaDir:     schemaDir,
		ProtoFile:     protoFile,
		SchemaFile:    schemaFile,
		Swagger:       swagger,
		Lifecycle:     lifecycle,
		Audience:      audience,
		FDSOutput:     fdsOutput,
		DefaultBranch: defaultBranch,
		GoPrivate:     goPrivate,
		Org:           defaults.Org,
		Repo:          defaults.Repo,
	}

	result, err := onboard.New(opts).Generate(".")
	if err != nil {
		return fmt.Errorf("onboard failed: %w", err)
	}

	for _, f := range result.Created {
		ui.Success("✓ Generated %s", f)
	}
	for _, f := range result.Modified {
		ui.Success("✓ Updated %s", f)
	}
	for _, f := range result.Skipped {
		ui.Info("  skipped %s", f)
	}

	ui.Info("")
	ui.Info("Next steps:")
	step := 1
	if onboard.IsFDSFormat(apiID) {
		ui.Info("  %d. Run 'make fds' to generate the FileDescriptorSet binary", step)
		step++
	}
	ui.Info("  %d. Review and commit: git add apx.yaml .apx-publish.yaml Makefile .gitignore .github/", step)
	step++
	ui.Info("  %d. Push to GitHub — the api-catalog-publish workflow will open a catalog PR on merge", step)
	ui.Info("")
	ui.Info("For local testing, see:")
	ui.Info("  https://infobloxopen.github.io/apx/getting-started/brownfield-onboarding/#testing-locally")

	if globalCfg, loadErr := config.LoadGlobal(); loadErr == nil {
		globalCfg.AddOrg(defaults.Org, []string{defaults.Repo})
		_ = config.SaveGlobal(globalCfg)
	}

	ui.Success("\n✓ Onboarding scaffold complete!")
	return nil
}

func canonicalDefault(defaults *detector.ProjectDefaults) string {
	if defaults.Org == "" || defaults.Org == "your-org-name" {
		return ""
	}
	base := "github.com/" + defaults.Org + "/apis"
	if globalCfg, err := config.LoadGlobal(); err == nil {
		if orgEntry := globalCfg.FindOrg(defaults.Org); orgEntry != nil && len(orgEntry.Repos) > 0 {
			base = "github.com/" + defaults.Org + "/" + orgEntry.Repos[0]
		}
	}
	return base
}

// detectDefaultBranch reads the default branch from the git remote tracking ref.
// Falls back to "main" if the ref is not set.
func detectDefaultBranch() string {
	out, err := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD").Output()
	if err != nil {
		return "main"
	}
	// "refs/remotes/origin/master" → "master"
	ref := strings.TrimSpace(string(out))
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "main"
}

func isValidAPIID(id string) bool {
	for _, prefix := range []string{"proto/", "openapi/", "avro/", "jsonschema/", "parquet/", "crd/"} {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}
	return false
}
