package commands

import (
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <module-path>[@version]",
		Short: "Add a dependency to apx.yaml and apx.lock",
		Long: `Add a schema module dependency to the project.

The dependency is added to apx.yaml and the version is locked in apx.lock.

Examples:
  apx add proto/payments/ledger/v1@v1.2.3
  apx add proto/payments/wallet/v1         # Uses latest version
  apx add openapi/customer/accounts/v2@v2.0.0

Unreleased overrides (local hot-loop) let you build against a dependency's
schema BEFORE it is released. They are fail-closed: releases are blocked while
any override is present (replace with a released version via apx update /
apx unlink before releasing).

  apx add openapi/billing/invoices/v2 --path ../billing-api
  apx add openapi/billing/invoices/v2 --git github.com/acme/apis --ref feature-branch`,
		Args: cobra.ExactArgs(1),
		RunE: addAction,
	}
	cmd.Flags().StringP("catalog", "c", "", "Path or URL to catalog file (default: catalog_url from apx.yaml, then catalog/catalog.yaml)")
	cmd.Flags().String("path", "", "local directory override: read this dependency's schema from here (unreleased)")
	cmd.Flags().String("git", "", "git repo override (URL or github.com/org/repo): read schema from a branch/fork (unreleased)")
	cmd.Flags().String("ref", "", "git branch/tag/commit for --git (required with --git)")
	return cmd
}

func addAction(cmd *cobra.Command, args []string) error {
	arg := args[0]

	var modulePath, version string
	if strings.Contains(arg, "@") {
		parts := strings.SplitN(arg, "@", 2)
		modulePath = parts[0]
		version = parts[1]
	} else {
		modulePath = arg
	}

	mgr := config.NewDependencyManager("apx.yaml", "apx.lock", resolveSourceRepo(cmd))

	// Unreleased override path: --path (local dir) or --git+--ref (branch/fork).
	// --path and --git are mutually exclusive. When neither is set, behavior is
	// unchanged (catalog/version pin below).
	pathOverride, _ := cmd.Flags().GetString("path")
	gitOverride, _ := cmd.Flags().GetString("git")
	refOverride, _ := cmd.Flags().GetString("ref")

	if pathOverride != "" || gitOverride != "" {
		if pathOverride != "" && gitOverride != "" {
			err := fmt.Errorf("--path and --git are mutually exclusive")
			ui.Error("%v", err)
			return err
		}
		if gitOverride != "" && refOverride == "" {
			err := fmt.Errorf("--git requires --ref (branch, tag, or commit)")
			ui.Error("%v", err)
			return err
		}
		ov := config.DependencyLock{
			Ref:    version, // may be empty → recorded as "override"
			Path:   pathOverride,
			Git:    gitOverride,
			GitRef: refOverride,
		}
		if err := mgr.AddWithOverride(modulePath, ov); err != nil {
			ui.Error("Failed to add dependency override: %v", err)
			return err
		}
		if pathOverride != "" {
			ui.Success("Added dependency: %s (unreleased override → path %s)", modulePath, pathOverride)
		} else {
			ui.Success("Added dependency: %s (unreleased override → git %s#%s)", modulePath, gitOverride, refOverride)
		}
		ui.Warning("%s is pinned to an UNRELEASED override; releases are blocked until it is replaced with a released version (apx update / apx unlink).", modulePath)
		return nil
	}

	// Look up the catalog to see if this is an external API
	var provenance *config.ExternalProvenance
	catalogPath, _ := cmd.Flags().GetString("catalog")
	src := resolveCatalogSource(cmd, catalogPath)
	cat, err := src.Load()
	if err == nil {
		for _, m := range cat.Modules {
			if m.ID == modulePath && m.Origin != "" {
				provenance = &config.ExternalProvenance{
					Origin:       m.Origin,
					ManagedRepo:  m.ManagedRepo,
					UpstreamRepo: m.UpstreamRepo,
					UpstreamPath: m.UpstreamPath,
					ImportMode:   m.ImportMode,
				}
				break
			}
		}
	}

	if err := mgr.AddWithProvenance(modulePath, version, provenance); err != nil {
		ui.Error("Failed to add dependency: %v", err)
		return err
	}

	if version != "" {
		ui.Success("Added dependency: %s@%s", modulePath, version)
	} else {
		ui.Success("Added dependency: %s (latest version)", modulePath)
	}

	if provenance != nil {
		ui.Info("  Source: %s (%s, %s)", provenance.ManagedRepo, provenance.Origin, provenance.ImportMode)
	}

	return nil
}
