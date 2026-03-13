package commands

import (
	"fmt"

	"github.com/infobloxopen/apx/internal/language"
	"github.com/infobloxopen/apx/internal/overlay"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [language] [module-path]",
		Short: "Activate overlays for local development",
		Long: `Activate locally generated overlays in each language's package manager.

  Go:     updates go.work to reference generated overlay directories
  Python: runs 'pip install -e' for each Python overlay in the active virtualenv

Without a language argument, all supported languages are synced.

Use --clean to reverse the activation (deactivate overlays without deleting them):
  Go:     writes a minimal go.work with only the root module
  Python: runs 'pip uninstall' for each linked Python overlay

Examples:
  apx sync                                     # activate all languages
  apx sync go                                  # activate Go overlays only
  apx sync python                              # activate Python overlays only
  apx sync python proto/payments/ledger/v1     # activate one Python overlay
  apx sync --clean                             # deactivate all languages
  apx sync --clean python                      # deactivate Python only`,
		Args: cobra.MaximumNArgs(2),
		RunE: syncAction,
	}
	cmd.Flags().Bool("clean", false, "Deactivate overlays from package managers (reverse of sync)")
	cmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	return cmd
}

func syncAction(cmd *cobra.Command, args []string) error {
	clean, _ := cmd.Flags().GetBool("clean")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	var langFilter, moduleFilter string
	if len(args) > 0 {
		langFilter = args[0]
	}
	if len(args) > 1 {
		moduleFilter = args[1]
	}

	if langFilter != "" && language.Get(langFilter) == nil {
		return fmt.Errorf("unknown language %q", langFilter)
	}

	// Go: managed via go.work (not through the plugin Linker interface)
	if langFilter == "" || langFilter == "go" {
		if dryRun {
			if clean {
				ui.Info("[dry-run] Would clear go.work (deactivate Go overlays)")
			} else {
				ui.Info("[dry-run] Would update go.work with active Go overlays")
			}
		} else if clean {
			mgr := overlay.NewManager(".")
			if err := mgr.ClearWorkFile(); err != nil {
				return fmt.Errorf("clearing go.work: %w", err)
			}
			ui.Success("Go: go.work cleared")
		} else {
			mgr := overlay.NewManager(".")
			if err := mgr.SyncWorkFile(); err != nil {
				return fmt.Errorf("updating go.work: %w", err)
			}
			ui.Success("Go: go.work updated")
		}
	}

	// Other languages: use Linker / Unlinker plugin interfaces
	for _, p := range language.All() {
		if p.Name() == "go" {
			continue
		}
		if langFilter != "" && p.Name() != langFilter {
			continue
		}

		if clean {
			unlinker, ok := p.(language.Unlinker)
			if !ok {
				if langFilter == p.Name() {
					ui.Info("%s does not support sync --clean", p.Name())
				}
				continue
			}
			if dryRun {
				ui.Info("[dry-run] Would deactivate %s overlays", p.Name())
				continue
			}
			if err := unlinker.Unlink(".", moduleFilter); err != nil {
				if langFilter != "" {
					return err
				}
				ui.Warning("%s: %v", p.Name(), err)
			}
		} else {
			linker, ok := p.(language.Linker)
			if !ok {
				if langFilter == p.Name() {
					ui.Info("%s does not support sync (no package manager integration)", p.Name())
				}
				continue
			}
			if dryRun {
				ui.Info("[dry-run] Would activate %s overlays", p.Name())
				continue
			}
			if err := linker.Link(".", moduleFilter); err != nil {
				if langFilter != "" {
					return err
				}
				ui.Warning("%s: %v", p.Name(), err)
			}
		}
	}

	return nil
}
