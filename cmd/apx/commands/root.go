package commands

import (
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root cobra command with all subcommands registered.
func NewRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apx",
		Short: "API Publishing eXperience CLI",
		Long:  "apx is a CLI tool for managing, publishing, and consuming API schemas across organizations.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			quiet, _ := cmd.Flags().GetBool("quiet")
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonOut, _ := cmd.Flags().GetBool("json")
			noColor, _ := cmd.Flags().GetBool("no-color")

			if quiet {
				ui.SetQuiet(true)
			}
			if verbose {
				ui.SetVerbose(true)
			}
			if jsonOut {
				ui.SetJSONOutput(true)
			}
			if noColor {
				ui.SetColorEnabled(false)
			}
			return nil
		},
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.SetVersionTemplate("{{.Version}}\n")

	// Global persistent flags
	cmd.PersistentFlags().BoolP("quiet", "q", false, "suppress output")
	cmd.PersistentFlags().Bool("verbose", false, "verbose output")
	cmd.PersistentFlags().Bool("json", false, "output in JSON format")
	cmd.PersistentFlags().Bool("no-color", false, "disable colored output")
	cmd.PersistentFlags().String("config", "apx.yaml", "config file path")

	// Register all subcommands
	cmd.AddCommand(
		newInitCmd(),
		newLintCmd(),
		newBreakingCmd(),
		newSemverCmd(),
		newGenCmd(),
		newPolicyCmd(),
		newCatalogCmd(),
		newPublishCmd(),
		newReleaseCmd(),
		newSearchCmd(),
		newShowCmd(),
		newAddCmd(),
		newSyncCmd(),
		newUnlinkCmd(),
		newConfigCmd(),
		newFetchCmd(),
		newInspectCmd(),
		newExplainCmd(),
		newWorkflowsCmd(),
	)

	return cmd
}
