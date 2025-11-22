package commands

import (
	"fmt"

	"github.com/infobloxopen/apx/internal/overlay"
	"github.com/urfave/cli/v2"
)

// SyncCommand implements the apx sync command for overlay management
func SyncCommand() *cli.Command {
	return &cli.Command{
		Name:  "sync",
		Usage: "Synchronize go.work overlays with canonical imports",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "clean",
				Usage: "Remove all overlays before syncing",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Show what would be done without making changes",
			},
		},
		Action: syncAction,
	}
}

func syncAction(c *cli.Context) error {
	clean := c.Bool("clean")
	dryRun := c.Bool("dry-run")

	fmt.Printf("Synchronizing go.work overlays...\n")

	// Initialize overlay manager
	manager := overlay.NewManager(".")

	if clean {
		fmt.Printf("Cleaning existing overlays...\n")
		if !dryRun {
			if err := manager.CleanOverlays(); err != nil {
				return fmt.Errorf("failed to clean overlays: %w", err)
			}
		}
		fmt.Printf("✓ Overlays cleaned\n")
	}

	// List current overlays
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

	// Sync go.work file
	if !dryRun {
		fmt.Printf("Updating go.work...\n")
		if err := manager.SyncWorkFile(); err != nil {
			return fmt.Errorf("failed to sync go.work: %w", err)
		}
		fmt.Printf("✓ go.work updated\n")
	} else {
		fmt.Printf("Would update go.work (dry-run mode)\n")
	}

	fmt.Printf("✓ Sync complete\n")
	return nil
}
