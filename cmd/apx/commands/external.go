package commands

import (
	"encoding/json"
	"fmt"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newExternalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "external",
		Short: "Manage external API registrations",
		Long: `Register, list, and manage third-party external APIs in your APX catalog.

External APIs are APIs from outside your organization that you want to include
in your catalog for discovery and dependency management.`,
	}
	cmd.AddCommand(newExternalRegisterCmd())
	cmd.AddCommand(newExternalListCmd())
	cmd.AddCommand(newExternalTransitionCmd())
	return cmd
}

func newExternalRegisterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register <api-id>",
		Short: "Register an external API",
		Long: `Register an external API in the organization's APX catalog.

Example:
  apx external register proto/google/pubsub/v1 \
    --managed-repo github.com/Infoblox-CTO/apis-contrib-google \
    --managed-path google/pubsub/v1 \
    --upstream-repo github.com/googleapis/googleapis \
    --upstream-path google/pubsub/v1 \
    --description "Google Cloud Pub/Sub API" \
    --lifecycle stable`,
		Args: cobra.ExactArgs(1),
		RunE: externalRegisterAction,
	}
	cmd.Flags().String("managed-repo", "", "Internal repository hosting curated snapshots (required)")
	cmd.Flags().String("managed-path", "", "Filesystem path in managed repository (required)")
	cmd.Flags().String("upstream-repo", "", "Original external repository URL (required)")
	cmd.Flags().String("upstream-path", "", "Path in upstream repository (required)")
	cmd.Flags().String("import-mode", "preserve", "Import path handling: preserve, rewrite")
	cmd.Flags().String("description", "", "Human-readable description")
	cmd.Flags().String("lifecycle", "", "Lifecycle state (experimental, beta, stable, deprecated, sunset)")
	cmd.Flags().String("version", "", "Current version of the managed snapshot")
	cmd.Flags().StringSlice("owners", nil, "Comma-separated list of owners")
	cmd.Flags().StringSlice("tags", nil, "Comma-separated list of tags")
	_ = cmd.MarkFlagRequired("managed-repo")
	_ = cmd.MarkFlagRequired("managed-path")
	_ = cmd.MarkFlagRequired("upstream-repo")
	_ = cmd.MarkFlagRequired("upstream-path")
	return cmd
}

func externalRegisterAction(cmd *cobra.Command, args []string) error {
	apiID := args[0]
	managedRepo, _ := cmd.Flags().GetString("managed-repo")
	managedPath, _ := cmd.Flags().GetString("managed-path")
	upstreamRepo, _ := cmd.Flags().GetString("upstream-repo")
	upstreamPath, _ := cmd.Flags().GetString("upstream-path")
	importMode, _ := cmd.Flags().GetString("import-mode")
	description, _ := cmd.Flags().GetString("description")
	lifecycle, _ := cmd.Flags().GetString("lifecycle")
	version, _ := cmd.Flags().GetString("version")
	owners, _ := cmd.Flags().GetStringSlice("owners")
	tags, _ := cmd.Flags().GetStringSlice("tags")

	reg := &config.ExternalRegistration{
		ID:           apiID,
		ManagedRepo:  managedRepo,
		ManagedPath:  managedPath,
		UpstreamRepo: upstreamRepo,
		UpstreamPath: upstreamPath,
		ImportMode:   importMode,
		Description:  description,
		Lifecycle:    lifecycle,
		Version:      version,
		Owners:       owners,
		Tags:         tags,
	}

	configPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if configPath == "" {
		configPath = "apx.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		ui.Error("Failed to load config: %v", err)
		return err
	}

	// Collect existing module paths for conflict detection
	var existingPaths []string
	// We don't have the catalog here, so just use paths from existing externals
	for _, ext := range cfg.ExternalAPIs {
		existingPaths = append(existingPaths, ext.ManagedPath)
	}

	if err := config.AddExternal(cfg, reg, existingPaths); err != nil {
		ui.Error("Registration failed: %v", err)
		return err
	}

	if err := config.SaveConfig(cfg, configPath); err != nil {
		ui.Error("Failed to save config: %v", err)
		return err
	}

	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonOut {
		data, err := json.MarshalIndent(reg, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	ui.Success("Registered external API: %s", apiID)
	fmt.Printf("  Managed:  %s :: %s\n", reg.ManagedRepo, reg.ManagedPath)
	fmt.Printf("  Upstream: %s :: %s\n", reg.UpstreamRepo, reg.UpstreamPath)
	fmt.Printf("  Import:   %s\n", reg.ImportMode)
	fmt.Printf("  Origin:   %s\n", reg.Origin)

	return nil
}

func newExternalListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered external APIs",
		RunE:  externalListAction,
	}
	cmd.Flags().String("origin", "", "Filter by origin: external, forked")
	return cmd
}

func externalListAction(cmd *cobra.Command, args []string) error {
	originFilter, _ := cmd.Flags().GetString("origin")

	configPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if configPath == "" {
		configPath = "apx.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		ui.Error("Failed to load config: %v", err)
		return err
	}

	regs := config.ListExternals(cfg, originFilter)

	if len(regs) == 0 {
		ui.Info("No external APIs registered.")
		return nil
	}

	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonOut {
		data, err := json.MarshalIndent(regs, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	ui.Info("External APIs (%d registered):", len(regs))
	fmt.Println()
	for _, reg := range regs {
		fmt.Printf("  %-35s [%s] %s\n", reg.ID, reg.Origin, reg.ImportMode)
		fmt.Printf("    Managed:  %s :: %s\n", reg.ManagedRepo, reg.ManagedPath)
		fmt.Printf("    Upstream: %s :: %s\n", reg.UpstreamRepo, reg.UpstreamPath)
		fmt.Println()
	}

	return nil
}

func newExternalTransitionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transition <api-id>",
		Short: "Transition an external API between external and forked",
		Long: `Transition an external API between "external" (preserve imports) and
"forked" (rewrite imports).

When transitioning to forked, import_mode automatically changes to "rewrite".
When transitioning to external, import_mode automatically changes to "preserve".`,
		Args: cobra.ExactArgs(1),
		RunE: externalTransitionAction,
	}
	cmd.Flags().String("to", "", "Target classification: external or forked (required)")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func externalTransitionAction(cmd *cobra.Command, args []string) error {
	apiID := args[0]
	targetOrigin, _ := cmd.Flags().GetString("to")

	configPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if configPath == "" {
		configPath = "apx.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		ui.Error("Failed to load config: %v", err)
		return err
	}

	// Get the current state before transition for display
	existing, err := config.FindExternalByID(cfg, apiID)
	if err != nil {
		ui.Error("Transition failed: %v", err)
		return err
	}
	previousOrigin := existing.Origin
	previousMode := existing.ImportMode

	if err := config.TransitionExternal(cfg, apiID, targetOrigin); err != nil {
		ui.Error("Transition failed: %v", err)
		return err
	}

	if err := config.SaveConfig(cfg, configPath); err != nil {
		ui.Error("Failed to save config: %v", err)
		return err
	}

	jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonOut {
		updated, _ := config.FindExternalByID(cfg, apiID)
		data, err := json.MarshalIndent(updated, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	updated, _ := config.FindExternalByID(cfg, apiID)
	ui.Success("Transitioned %s: %s → %s", apiID, previousOrigin, targetOrigin)
	if previousMode != updated.ImportMode {
		fmt.Printf("  Import mode changed: %s → %s\n", previousMode, updated.ImportMode)
	}
	if targetOrigin == config.OriginForked {
		fmt.Println("  Upstream origin retained for provenance.")
	}

	return nil
}
