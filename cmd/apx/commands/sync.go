package commands

import (
	"fmt"

	"github.com/infobloxopen/apx/internal/overlay"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize go.work overlays with canonical imports",
		RunE:  syncAction,
	}
	cmd.Flags().Bool("clean", false, "Remove all overlays before syncing")
	cmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	return cmd
}

func syncAction(cmd *cobra.Command, args []string) error {
	clean, _ := cmd.Flags().GetBool("clean")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	fmt.Printf("Synchronizing go.work overlays...\n")

	manager := overlay.NewManager(".")

	if clean {
		fmt.Printf("Cleaning existing overlays...\n")
		if !dryRun {
			if err := manager.CleanOverlays(); err != nil {
				return fmt.Errorf("failed to clean overlays: %w", err)
			}
		}
		fmt.Printf("\u2713 Overlays cleaned\n")
	}

	overlays, err := manager.ListOverlays()
	if err != nil {
		return fmt.Errorf("failed to list overlays: %w", err)
	}

	if len(overlays) == 0 {
		fmt.Printf("No overlays found\n")
	} else {
		fmt.Printf("Active overlays:\n")
		for _, o := range overlays {
			fmt.Printf("  - %s\n", o)
		}
	}

	if !dryRun {
		fmt.Printf("Updating go.work...\n")
		if err := manager.SyncWorkFile(); err != nil {
			return fmt.Errorf("failed to sync go.work: %w", err)
		}
		fmt.Printf("\u2713 go.work updated\n")
	} else {
		fmt.Printf("Would update go.work (dry-run mode)\n")
	}

	fmt.Printf("\u2713 Sync complete\n")
	return nil
}
